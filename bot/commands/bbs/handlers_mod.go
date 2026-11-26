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

func (s *System) handleReportMessageCommand(data discord.MessageCommandInteractionData, e *handler.CommandEvent) error {
	loc := e.Locale()
	targetMsg := data.TargetMessage()
	guildID := e.GuildID()
	pm, err := s.Service.Store.GetPostMirrorByDiscordMessage(e.Ctx, targetMsg.ID)
	if err != nil || guildID == nil || pm.GuildID != *guildID {
		s.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.report.not_target_title"),
			i18n.TranslateText(loc, "components.bbs.report.not_target_body"),
		)
		return nil
	}

	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		if errors.Is(err, bbscore.ErrConsentRequired) {
			s.respondConsentModal(e, e.ID().Time(), loc, e.User().ID, pendingReportPost, pm.PostID)
			return nil
		}
		s.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}

	modal := buildReportPostModal(loc, pm.PostID)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open report post modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
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

	post, err := s.Service.GetPost(e.Ctx, postID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, post.ThreadID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, thread.BoardID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if board.Scope == bbscore.ScopeLocal && (board.GuildID == nil || *board.GuildID != *guildID) {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.permission"),
			i18n.TranslateText(loc, "components.bbs.errors.unauthorized"),
		)
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
	logLocale := discordsync.ResolveGuildLocale(client, logGuildID, discord.LocaleJapanese)

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

func (s *System) handleModerateMessageCommand(data discord.MessageCommandInteractionData, e *handler.CommandEvent) error {
	if !s.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}
	loc := e.Locale()

	targetMsg := data.TargetMessage()
	pm, err := s.Service.Store.GetPostMirrorByDiscordMessage(e.Ctx, targetMsg.ID)
	if err != nil || pm.GuildID != *guildID {
		s.replyCommandError(e,
			i18n.TranslateText(loc, "components.bbs.mod.not_target_title"),
			i18n.TranslateText(loc, "components.bbs.mod.not_target_body"),
		)
		return nil
	}
	fetchFail := i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed")
	post, err := s.Service.GetPost(e.Ctx, pm.PostID)
	if err != nil {
		s.replyCommandError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, post.ThreadID)
	if err != nil {
		s.replyCommandError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, thread.BoardID)
	if err != nil {
		s.replyCommandError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}

	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *guildID {
		globalInfo := i18n.TranslateText(loc, "components.bbs.mod.global_info", map[string]any{
			"board":   board.Name,
			"scope":   string(board.Scope),
			"thread":  thread.Title,
			"res":     post.ResNumber,
			"anon_id": post.AnonID,
		})
		s.replyCommandNotice(e,
			i18n.TranslateText(loc, "components.bbs.mod.global_notice_title"),
			i18n.TranslateText(loc, "components.bbs.mod.global_notice_body", map[string]any{"info": globalInfo}),
			discordsync.ColorGlobalBoard,
		)
		return nil
	}

	// Strict anonymity boundary: guild moderators only ever inspect local-scoped PosterHash, never AuthorUserID.
	info := i18n.TranslateText(loc, "components.bbs.mod.info", map[string]any{
		"board":   board.Name,
		"scope":   string(board.Scope),
		"thread":  thread.Title,
		"res":     post.ResNumber,
		"hash":    post.PosterHash,
		"anon_id": post.AnonID,
	})

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.mod.local_panel_title")),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(info),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewDangerButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.btn.abone"), discordsync.MaxButtonLabelRunes), fmt.Sprintf("/bbs/mod/abone/%s", post.ID)),
				discord.NewSecondaryButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.btn.archive"), discordsync.MaxButtonLabelRunes), fmt.Sprintf("/bbs/mod/archive/%s", thread.ID)),
				discord.NewDangerButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.mod.btn.ban"), discordsync.MaxButtonLabelRunes), fmt.Sprintf("/bbs/mod/ban/%s", post.ID)),
			),
		).WithAccentColor(discordsync.ColorError),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
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
