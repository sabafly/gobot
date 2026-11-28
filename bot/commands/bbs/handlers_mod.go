/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package bbs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

func buildReportPostModal(loc discord.Locale, postID uuid.UUID) discord.ModalCreate {
	return discord.NewModalCreate(
		fmt.Sprintf("/bbs/report/postmodal/%s", postID),
		discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.report.modal_post_title"), discordsync.MaxModalTitleRunes),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.report.modal_post_label"), discordsync.MaxLabelTextRunes),
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(300),
		),
	)
}

func buildReportThreadModal(loc discord.Locale, threadID uuid.UUID) discord.ModalCreate {
	return discord.NewModalCreate(
		fmt.Sprintf("/bbs/report/threadmodal/%s", threadID),
		discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.report.modal_thread_title"), discordsync.MaxModalTitleRunes),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.report.modal_thread_label"), discordsync.MaxLabelTextRunes),
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(300),
		),
	)
}

func (s *System) loadPostForGuild(ctx context.Context, guildID snowflake.ID, postID uuid.UUID) (*bbscore.Board, *bbscore.Thread, *bbscore.Post, *bbscore.BoardMount, error) {
	post, err := s.Service.GetPost(ctx, postID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	thread, err := s.Service.GetThread(ctx, post.ThreadID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	board, err := s.Service.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var mount *bbscore.BoardMount
	switch board.Scope {
	case bbscore.ScopeLocal:
		if board.GuildID == nil || *board.GuildID != guildID {
			return nil, nil, nil, nil, bbscore.ErrUnauthorized
		}
	case bbscore.ScopeGlobal:
		var mErr error
		mount, mErr = s.Service.Store.GetBoardMountByGuildAndBoard(ctx, guildID, board.ID)
		if mErr != nil || mount == nil {
			mount = nil
			pm, pErr := s.Service.Store.GetPostMirrorByPostAndGuild(ctx, post.ID, guildID)
			if pErr != nil || pm == nil {
				return nil, nil, nil, nil, bbscore.ErrUnauthorized
			}
		}
	default:
		return nil, nil, nil, nil, bbscore.ErrUnauthorized
	}
	return board, thread, post, mount, nil
}

func (s *System) handlePostMenuButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.report.invalid_post_id"),
		)
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.errors.guild_only"),
		)
		return nil
	}

	board, thread, post, mount, err := s.loadPostForGuild(e.Ctx, *guildID, postID)
	if err != nil {
		errTitle := i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed")
		if errors.Is(err, bbscore.ErrUnauthorized) {
			errTitle = i18n.TranslateText(loc, "components.bbs.errors.title.permission")
		}
		s.replyComponentError(e, errTitle, describeBBSError(loc, err))
		return nil
	}

	var canReply bool
	if !post.Abone && thread.Status == bbscore.ThreadStatusActive && thread.ResCount < thread.Rules.Normalize().MaxRes {
		if mount == nil && board.Scope == bbscore.ScopeLocal {
			if m, mErr := s.Service.Store.GetBoardMountByGuildAndBoard(e.Ctx, *guildID, board.ID); mErr == nil {
				mount = m
			}
		}
		if mount != nil && mount.Active {
			if _, gErr := s.Service.EnsureGuildEnabled(e.Ctx, *guildID); gErr == nil {
				if mirror, tmErr := s.Service.Store.GetThreadMirrorByMountAndThread(e.Ctx, mount.ID, thread.ID); tmErr == nil && mirror != nil {
					canReply = true
				}
			}
		}
	}
	var hideReport bool
	if p, pErr := s.Service.Registry.Get(board.ProviderKey); pErr == nil && p != nil {
		canReply = canReply && p.Capabilities().CanCreatePost
		if fp, ok := p.(bbscore.FederatedProvider); ok && !fp.CanReport() {
			hideReport = true
		}
	}

	isAdmin := s.hasGuildAdmin(e.Ctx, e.Member(), guildID, e.User(), e.Client())
	canLocalMod := isAdmin && board.Scope == bbscore.ScopeLocal && board.GuildID != nil && *board.GuildID == *guildID
	showGlobalModNote := isAdmin && board.Scope == bbscore.ScopeGlobal

	msg := discordsync.RenderPostActionMenu(discordsync.PostActionMenuOptions{
		Board:             *board,
		Thread:            *thread,
		Post:              *post,
		CanReply:          canReply,
		HideReport:        hideReport,
		CanLocalMod:       canLocalMod,
		ShowGlobalModNote: showGlobalModNote,
	}, loc)

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleReportPostButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.report.invalid_post_id"),
		)
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.errors.guild_only"),
		)
		return nil
	}

	board, _, _, _, err := s.loadPostForGuild(e.Ctx, *guildID, postID)
	if err != nil {
		errTitle := i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed")
		if errors.Is(err, bbscore.ErrUnauthorized) {
			errTitle = i18n.TranslateText(loc, "components.bbs.errors.title.permission")
		}
		s.replyComponentError(e, errTitle, describeBBSError(loc, err))
		return nil
	}

	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		if errors.Is(err, bbscore.ErrConsentRequired) {
			s.respondConsentModal(e, e.ID().Time(), loc, e.User().ID, pendingReportPost, postID)
			return nil
		}
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if p, pErr := s.Service.Registry.Get(board.ProviderKey); pErr == nil {
		if fp, ok := p.(bbscore.FederatedProvider); ok && !fp.CanReport() {
			s.replyComponentError(e,
				i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
				describeBBSError(loc, bbscore.ErrFederationReadOnly),
			)
			return nil
		}
	}
	if !s.ensureRemoteDelegation(e, *board, fmt.Sprintf("/bbs/report/post/%s", postID)) {
		return nil
	}

	modal := buildReportPostModal(loc, postID)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open report post modal from button", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (s *System) handleReportThreadButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.threads.invalid_id"),
		)
		return nil
	}
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		if errors.Is(err, bbscore.ErrConsentRequired) {
			s.respondConsentModal(e, e.ID().Time(), loc, e.User().ID, pendingReportThread, threadID)
			return nil
		}
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if th, tErr := s.Service.GetThread(e.Ctx, threadID); tErr == nil && th != nil {
		if b, bErr := s.Service.GetBoard(e.Ctx, th.BoardID); bErr == nil && b != nil {
			if p, pErr := s.Service.Registry.Get(b.ProviderKey); pErr == nil {
				if fp, ok := p.(bbscore.FederatedProvider); ok && !fp.CanReport() {
					s.replyComponentError(e,
						i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
						describeBBSError(loc, bbscore.ErrFederationReadOnly),
					)
					return nil
				}
			}
			if !s.ensureRemoteDelegation(e, *b, fmt.Sprintf("/bbs/report/thread/%s", threadID)) {
				return nil
			}
		}
	} else if tErr != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, tErr),
		)
		return nil
	}
	modal := buildReportThreadModal(loc, threadID)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open report thread modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (s *System) handleReportPostModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	invalidTitle := i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input")
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyModalError(e, invalidTitle, i18n.TranslateText(loc, "components.bbs.report.invalid_post_id"))
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		s.replyModalError(e, invalidTitle, i18n.TranslateText(loc, "components.bbs.errors.guild_only"))
		return nil
	}
	report, err := s.Service.CreateReport(e.Ctx, *guildID, e.User().ID, postID, e.Data.Text("reason"))
	if err != nil {
		if errors.Is(err, bbscore.ErrFederationAuthRequired) {
			if p, pErr := s.Service.GetPost(e.Ctx, postID); pErr == nil && p != nil {
				if th, tErr := s.Service.GetThread(e.Ctx, p.ThreadID); tErr == nil && th != nil {
					s.respondFederationAuthPrompt(e, e.ID().Time(), loc, th.ProviderKey, e.User().ID, e.ApplicationID(), e.Token(), fmt.Sprintf("/bbs/report/post/%s", postID))
					return nil
				}
			}
		}
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.report.failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.report.accepted_title"),
		i18n.TranslateText(loc, "components.bbs.report.accepted_body", map[string]any{"id": report.ID.String()}),
		discordsync.ColorNotice,
	).WithEphemeral(true))
	repCopy := *report
	go s.notifyReportLog(e.Client(), &repCopy)
	return nil
}

func (s *System) handleReportThreadModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	invalidTitle := i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input")
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyModalError(e, invalidTitle, i18n.TranslateText(loc, "components.bbs.threads.invalid_id"))
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		s.replyModalError(e, invalidTitle, i18n.TranslateText(loc, "components.bbs.errors.guild_only"))
		return nil
	}
	reportFail := i18n.TranslateText(loc, "components.bbs.report.failed")
	posts, err := s.Service.ListPosts(e.Ctx, threadID, 0, 1)
	if err != nil || len(posts) == 0 {
		s.replyModalError(e, reportFail, i18n.TranslateText(loc, "components.bbs.report.no_posts"))
		return nil
	}
	report, err := s.Service.CreateReport(e.Ctx, *guildID, e.User().ID, posts[0].ID, e.Data.Text("reason"))
	if err != nil {
		if errors.Is(err, bbscore.ErrFederationAuthRequired) {
			if th, tErr := s.Service.GetThread(e.Ctx, threadID); tErr == nil && th != nil {
				s.respondFederationAuthPrompt(e, e.ID().Time(), loc, th.ProviderKey, e.User().ID, e.ApplicationID(), e.Token(), fmt.Sprintf("/bbs/report/thread/%s", threadID))
				return nil
			}
		}
		s.replyModalError(e, reportFail, describeBBSError(loc, err))
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.report.accepted_title"),
		i18n.TranslateText(loc, "components.bbs.report.accepted_body", map[string]any{"id": report.ID.String()}),
		discordsync.ColorNotice,
	).WithEphemeral(true))
	repCopy := *report
	go s.notifyReportLog(e.Client(), &repCopy)
	return nil
}

func (s *System) notifyReportLog(client *bot.Client, report *bbscore.Report) {
	if client == nil || s.LogChannel == nil || *s.LogChannel == 0 || report == nil {
		return
	}
	var logGuildID snowflake.ID
	if ch, ok := client.Caches.Channel(*s.LogChannel); ok {
		logGuildID = ch.GuildID()
	}
	var logLocale discord.Locale
	if s != nil && s.Syncer != nil {
		logLocale = s.Syncer.ResolveGuildLocale(context.Background(), client, logGuildID, i18n.DefaultLocale())
	} else {
		logLocale = discordsync.ResolveGuildLocale(client, logGuildID, i18n.DefaultLocale())
	}

	msg := discordsync.RenderNotice(logLocale,
		i18n.TranslateText(logLocale, "components.bbs.report.log_title"),
		i18n.TranslateText(logLocale, "components.bbs.report.log_body", map[string]any{
			"id":       report.ID.String(),
			"scope":    string(report.Scope),
			"guild_id": report.GuildID,
			"post_id":  report.PostID.String(),
			"reason":   discordsync.TruncateRunes(report.Reason, 200),
		}),
		discordsync.ColorError,
	).WithEphemeral(false).WithAllowedMentions(&discord.AllowedMentions{})
	if err := s.Syncer.Validator.ValidateMessageCreate(msg); err != nil {
		slog.Warn("bbs failed to validate report log message", "report_id", report.ID, "err", err)
		return
	}
	if _, err := client.Rest.CreateMessage(*s.LogChannel, msg); err != nil {
		slog.Warn("bbs failed to send report log message", "report_id", report.ID, "err", err)
	}
}

func (s *System) handleModAboneButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.admin_required"),
		)
		return nil
	}
	postID := e.Vars["post_id"]
	if _, err := uuid.Parse(postID); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.report.invalid_post_id"),
		)
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/mod/abonemodal/%s", postID),
		discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.abone_modal_title"), discordsync.MaxModalTitleRunes),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.abone_modal_label"), discordsync.MaxLabelTextRunes),
			discord.NewShortTextInput("reason").
				WithRequired(false).
				WithMaxLength(120).
				WithPlaceholder(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.abone_modal_placeholder"), discordsync.MaxTextInputPlaceholder)),
		),
	)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open mod abone modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (s *System) handleModAboneModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) || e.GuildID() == nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.admin_required"),
		)
		return nil
	}
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.report.invalid_post_id"),
		)
		return nil
	}
	post, err := s.Service.AbonePostLocal(e.Ctx, *e.GuildID(), postID, e.Data.Text("reason"))
	if err != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.mod.abone_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if thread, tErr := s.Service.GetThread(e.Ctx, post.ThreadID); tErr == nil {
		go func() {
			if syncErr := s.Syncer.SyncAbone(s.Components.Ctx(), e.Client(), *post, *thread); syncErr != nil {
				slog.Warn("bbs failed to sync local abone", "post_id", post.ID, "err", syncErr)
			}
		}()
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.mod.aboned_title"),
		i18n.TranslateText(loc, "components.bbs.mod.aboned_body", map[string]any{"res": post.ResNumber}),
		discordsync.ColorNotice,
	).WithEphemeral(true))
	return nil
}

func (s *System) handleModArchiveButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) || e.GuildID() == nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.admin_required"),
		)
		return nil
	}
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.threads.invalid_id"),
		)
		return nil
	}
	thread, err := s.Service.ArchiveThreadLocal(e.Ctx, *e.GuildID(), threadID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.mod.archive_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if board, bErr := s.Service.GetBoard(e.Ctx, thread.BoardID); bErr == nil {
		go s.Syncer.SyncArchive(s.Components.Ctx(), e.Client(), *board, *thread, false)
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.mod.archived_title"),
		i18n.TranslateText(loc, "components.bbs.mod.archived_body", map[string]any{"title": thread.Title}),
		discordsync.ColorNotice,
	).WithEphemeral(true))
	return nil
}

func (s *System) handleModBanButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.admin_required"),
		)
		return nil
	}
	postID := e.Vars["post_id"]
	if _, err := uuid.Parse(postID); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.report.invalid_post_id"),
		)
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/mod/banmodal/%s", postID),
		discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.ban_modal_title"), discordsync.MaxModalTitleRunes),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.ban_hours_label"), discordsync.MaxLabelTextRunes),
			discord.NewShortTextInput("hours").
				WithRequired(false).
				WithMaxLength(10).
				WithPlaceholder(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.ban_hours_placeholder"), discordsync.MaxTextInputPlaceholder)),
		),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.ban_reason_label"), discordsync.MaxLabelTextRunes),
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(200),
		),
	)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open mod ban modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (s *System) handleModBanModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) || e.GuildID() == nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.admin_required"),
		)
		return nil
	}
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.report.invalid_post_id"),
		)
		return nil
	}
	fetchFail := i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed")
	post, err := s.Service.GetPost(e.Ctx, postID)
	if err != nil {
		s.replyModalError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, post.ThreadID)
	if err != nil {
		s.replyModalError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, thread.BoardID)
	if err != nil {
		s.replyModalError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}
	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *e.GuildID() {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.mod.local_ban_only"),
		)
		return nil
	}

	hoursRaw := strings.TrimSpace(e.Data.Text("hours"))
	var dur *time.Duration
	if hoursRaw != "" && hoursRaw != "0" {
		hrs, pErr := strconv.Atoi(hoursRaw)
		if pErr != nil || hrs < 0 || hrs > bbscore.MaxBanHours {
			s.replyModalError(e,
				i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
				i18n.TranslateText(loc, "components.bbs.mod.invalid_hours_range", map[string]any{"max": bbscore.MaxBanHours}),
			)
			return nil
		}
		if hrs > 0 {
			d := time.Duration(hrs) * time.Hour
			dur = &d
		}
	}

	ban, err := s.Service.BanLocal(e.Ctx, *e.GuildID(), e.User().ID, post.PosterHash, e.Data.Text("reason"), dur)
	if err != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.mod.ban_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.mod.banned_title"),
		i18n.TranslateText(loc, "components.bbs.mod.banned_body", map[string]any{
			"hash": ban.PosterHash,
			"id":   ban.ID.String(),
		}),
		discordsync.ColorError,
	).WithEphemeral(true))
	return nil
}
