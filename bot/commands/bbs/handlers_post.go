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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

const (
	capSelectCustomID       = "cap"
	userFlagActiveDeveloper = discord.UserFlags(1 << 22)
)

func buildSlipContext(e *handler.ModalEvent, guildID snowflake.ID) bbscore.SlipContext {
	user := e.User()
	sc := bbscore.SlipContext{
		Username:        user.Username,
		UserCreatedAt:   user.CreatedAt(),
		IsDefaultAvatar: user.Avatar == nil,
		HasProfileDecor: user.Banner != nil || user.AvatarDecorationData != nil || user.PrimaryGuild != nil || user.Collectibles != nil,
		UserLocale:      string(e.Locale()),
	}
	if user.GlobalName != nil {
		sc.GlobalName = *user.GlobalName
	}
	switch {
	case user.PublicFlags.Has(discord.UserFlagHouseBravery):
		sc.HypeSquadHouse = "bravery"
	case user.PublicFlags.Has(discord.UserFlagHouseBrilliance):
		sc.HypeSquadHouse = "brilliance"
	case user.PublicFlags.Has(discord.UserFlagHouseBalance):
		sc.HypeSquadHouse = "balance"
	}
	specialMask := discord.UserFlagDiscordEmployee |
		discord.UserFlagPartneredServerOwner |
		discord.UserFlagHypeSquadEvents |
		discord.UserFlagBugHunterLevel1 |
		discord.UserFlagBugHunterLevel2 |
		discord.UserFlagEarlySupporter |
		discord.UserFlagEarlyVerifiedBotDeveloper |
		discord.UserFlagDiscordCertifiedModerator |
		userFlagActiveDeveloper
	sc.HasSpecialBadge = (user.PublicFlags & specialMask) != 0

	if member := e.Member(); member != nil {
		if member.Nick != nil && strings.TrimSpace(*member.Nick) != "" {
			sc.MemberNick = *member.Nick
			sc.HasServerNick = true
		}
		sc.HasServerAvatar = member.Avatar != nil || member.Banner != nil || member.AvatarDecorationData != nil
		sc.MemberJoinedAt = member.JoinedAt
		sc.IsGuildBooster = member.PremiumSince != nil
		sc.RoleCount = len(member.RoleIDs)
		sc.HasZeroRoles = len(member.RoleIDs) == 0
	}

	if client := e.Client(); client != nil && guildID != 0 {
		if vs, ok := client.Caches.VoiceState(guildID, user.ID); ok && vs.ChannelID != nil {
			sc.InVoiceChannel = true
			sc.IsVoiceStreaming = vs.SelfStream || vs.SelfVideo
			sc.IsVoiceMutedOrDeaf = vs.SelfMute || vs.GuildMute || vs.SelfDeaf || vs.GuildDeaf
		}
		if g, ok := client.Caches.Guild(guildID); ok {
			sc.GuildCreatedAt = g.CreatedAt()
			sc.GuildMemberCount = g.MemberCount
			sc.GuildBoostTier = int(g.PremiumTier)
			sc.GuildIsCommunity = slices.Contains(g.Features, discord.GuildFeatureCommunity)
			sc.GuildHasVanityURL = (g.VanityURLCode != nil && *g.VanityURLCode != "") || slices.Contains(g.Features, discord.GuildFeatureVanityURL)
		}
	}
	return sc
}

func buildCapSelectLabel(locale discord.Locale, caps []bbscore.Cap) discord.LabelComponent {
	options := make([]discord.StringSelectMenuOption, 0, min(len(caps)+1, discordsync.MaxSelectOptions))
	options = append(options, discord.NewStringSelectMenuOption(
		discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.field.option.cap_none"), discordsync.MaxSelectTextRunes),
		bbscore.CapSelectNoneValue,
	).WithDefault(true))

	maxCaps := discordsync.MaxSelectOptions - 1
	for i, c := range caps {
		if i >= maxCaps {
			break
		}
		opt := discord.NewStringSelectMenuOption(
			discordsync.TruncateRunes(c.Name, discordsync.MaxSelectTextRunes),
			c.ID,
		)
		if desc := strings.TrimSpace(c.Description); desc != "" {
			opt = opt.WithDescription(discordsync.TruncateRunes(desc, discordsync.MaxSelectTextRunes))
		}
		options = append(options, opt)
	}

	menu := discord.NewStringSelectMenu(
		capSelectCustomID,
		discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.field.placeholder.cap"), discordsync.MaxSelectPlaceholderRunes),
		options...,
	).WithRequired(false).WithMinValues(0).WithMaxValues(1)

	return discord.NewLabel(
		discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.field.label.cap"), discordsync.MaxLabelTextRunes),
		menu,
	)
}

func selectedCapID(data discord.ModalSubmitInteractionData) string {
	vals := data.StringValues(capSelectCustomID)
	if len(vals) == 0 || vals[0] == bbscore.CapSelectNoneValue {
		return ""
	}
	return vals[0]
}

func buildNewThreadModal(locale discord.Locale, mountID uuid.UUID, boardName string, caps []bbscore.Cap) discord.ModalCreate {
	title := i18n.TranslateText(locale, "components.bbs.modal.new_thread.default_title")
	if boardName != "" {
		title = i18n.TranslateText(locale, "components.bbs.modal.new_thread.board_title", map[string]any{
			"board": bbscore.ClearFormat(boardName),
		})
	}
	title = discordsync.TruncateRunes(title, discordsync.MaxModalTitleRunes)

	comps := []discord.LayoutComponent{
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.new_thread.label.title"), discordsync.MaxLabelTextRunes),
			discord.NewShortTextInput("title").
				WithRequired(true).
				WithMinLength(1).
				WithMaxLength(bbscore.MaxTitleRunes).
				WithPlaceholder(discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.new_thread.placeholder.title"), discordsync.MaxTextInputPlaceholder)),
		),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.field.label.name"), discordsync.MaxLabelTextRunes),
			discord.NewShortTextInput("name").
				WithRequired(false).
				WithMaxLength(bbscore.MaxNameRunes).
				WithPlaceholder(discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.field.placeholder.name"), discordsync.MaxTextInputPlaceholder)),
		),
	}
	if len(caps) > 0 {
		comps = append(comps, buildCapSelectLabel(locale, caps))
	}
	comps = append(comps, discord.NewLabel(
		discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.new_thread.label.body"), discordsync.MaxLabelTextRunes),
		discord.NewParagraphTextInput("body").
			WithRequired(true).
			WithMinLength(1).
			WithMaxLength(bbscore.MaxBodyRunes),
	))

	return discord.NewModalCreate(
		fmt.Sprintf("/bbs/thread/modal/%s", mountID),
		title,
		comps...,
	)
}

func buildNewPostModal(locale discord.Locale, mirrorID uuid.UUID, caps []bbscore.Cap) discord.ModalCreate {
	comps := []discord.LayoutComponent{
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.field.label.name"), discordsync.MaxLabelTextRunes),
			discord.NewShortTextInput("name").
				WithRequired(false).
				WithMaxLength(bbscore.MaxNameRunes).
				WithPlaceholder(discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.field.placeholder.name"), discordsync.MaxTextInputPlaceholder)),
		),
	}
	if len(caps) > 0 {
		comps = append(comps, buildCapSelectLabel(locale, caps))
	}
	comps = append(comps, discord.NewLabel(
		discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.new_post.label.body"), discordsync.MaxLabelTextRunes),
		discord.NewParagraphTextInput("body").WithRequired(true).WithMinLength(1).WithMaxLength(bbscore.MaxBodyRunes),
	))

	return discord.NewModalCreate(
		fmt.Sprintf("/bbs/post/modal/%s", mirrorID),
		discordsync.TruncateRunes(i18n.TranslateText(locale, "components.bbs.modal.new_post.title"), discordsync.MaxModalTitleRunes),
		comps...,
	)
}

func (s *System) handleNewThreadButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.post_ui.invalid_mount"),
		)
		return nil
	}
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			i18n.TranslateText(loc, "components.bbs.post_ui.mount_inactive"),
		)
		return nil
	}
	if _, err := s.Service.EnsureGuildEnabled(e.Ctx, mount.GuildID); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		if errors.Is(err, bbscore.ErrConsentRequired) {
			s.respondConsentModal(e, e.ID().Time(), loc, e.User().ID, pendingNewThread, mountID)
			return nil
		}
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}
	boardName := ""
	if b, bErr := s.Service.GetBoard(e.Ctx, mount.BoardID); bErr == nil {
		boardName = b.Name
	}
	caps := s.Service.ListAvailableCaps(e.Ctx, e.User().ID, mount.BoardID)
	modal := buildNewThreadModal(loc, mountID, boardName, caps)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open new thread modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (s *System) handleNewThreadModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.post_ui.invalid_mount"),
		)
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs new thread defer failed", "err", err)
	}

	failTitle := i18n.TranslateText(loc, "components.bbs.post_ui.thread_create_failed")
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, failTitle, i18n.TranslateText(loc, "components.bbs.post_ui.mount_missing_or_inactive")))
		return nil
	}
	if _, err := s.Service.EnsureGuildEnabled(e.Ctx, mount.GuildID); err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, failTitle, describeBBSError(loc, err)))
		return nil
	}

	title := e.Data.Text("title")
	rawName := e.Data.Text("name")
	capID := selectedCapID(e.Data)
	rawBody := e.Data.Text("body")

	board, thread, firstPost, err := s.Service.CreateThread(e.Ctx, bbscore.CreateThreadInput{
		BoardID:     mount.BoardID,
		GuildID:     mount.GuildID,
		UserID:      e.User().ID,
		Title:       title,
		RawName:     rawName,
		CapID:       capID,
		RawBody:     rawBody,
		SlipContext: buildSlipContext(e, mount.GuildID),
		Now:         time.Now(),
	})
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, failTitle, describeBBSError(loc, err)))
		return nil
	}

	mountLocale := s.Syncer.ResolveMountLocale(e.Ctx, e.Client(), *mount)
	threadChID, msgID, syncErr := s.Syncer.PublishThread(e.Ctx, e.Client(), mount.ID, *board, *thread, *firstPost, mountLocale)
	if syncErr != nil {
		slog.Warn("bbs publish thread sync failed", "thread_id", thread.ID, "err", syncErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
			i18n.TranslateText(loc, "components.bbs.post_ui.thread_created_partial_title"),
			i18n.TranslateText(loc, "components.bbs.post_ui.thread_created_partial_body", map[string]any{
				"title": thread.Title,
				"err":   describeBBSError(loc, syncErr),
			}),
			discordsync.ColorNotice,
		).WithEphemeral(true))
		return nil
	}
	url := discord.MessageURL(mount.GuildID, threadChID, msgID)
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.post_ui.thread_created_title"),
		i18n.TranslateText(loc, "components.bbs.post_ui.thread_created_body", map[string]any{
			"title": thread.Title,
			"url":   url,
		}),
		discordsync.ColorLocalBoard,
	).WithEphemeral(true))
	return nil
}

func (s *System) handleNewPostButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	mirrorID, err := uuid.Parse(e.Vars["mirror_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.post_ui.invalid_mirror"),
		)
		return nil
	}
	mirror, err := s.Service.Store.GetThreadMirror(e.Ctx, mirrorID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			i18n.TranslateText(loc, "components.bbs.post_ui.mirror_not_found"),
		)
		return nil
	}
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mirror.BoardMountID)
	if err != nil || !mount.Active {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			i18n.TranslateText(loc, "components.bbs.post_ui.mount_missing_or_inactive"),
		)
		return nil
	}
	if _, err := s.Service.EnsureGuildEnabled(e.Ctx, mirror.GuildID); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, mirror.ThreadID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}
	cannotPostTitle := i18n.TranslateText(loc, "components.bbs.post_ui.cannot_post_title")
	if thread.Status != bbscore.ThreadStatusActive {
		s.replyComponentError(e, cannotPostTitle, i18n.TranslateText(loc, "components.bbs.errors.thread_archived"))
		return nil
	}
	if thread.ResCount >= thread.Rules.Normalize().MaxRes {
		s.replyComponentError(e, cannotPostTitle, i18n.TranslateText(loc, "components.bbs.errors.max_res_reached"))
		return nil
	}
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		if errors.Is(err, bbscore.ErrConsentRequired) {
			s.respondConsentModal(e, e.ID().Time(), loc, e.User().ID, pendingNewPost, mirrorID)
			return nil
		}
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return nil
	}
	caps := s.Service.ListAvailableCaps(e.Ctx, e.User().ID, thread.BoardID)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, buildNewPostModal(loc, mirrorID, caps)); err != nil {
		slog.Warn("bbs failed to open new post modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (s *System) handleNewPostModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	mirrorID, err := uuid.Parse(e.Vars["mirror_id"])
	if err != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.post_ui.invalid_mirror"),
		)
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs new post defer failed", "err", err)
	}

	failTitle := i18n.TranslateText(loc, "components.bbs.post_ui.post_failed")
	mirror, err := s.Service.Store.GetThreadMirror(e.Ctx, mirrorID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, failTitle, i18n.TranslateText(loc, "components.bbs.post_ui.mirror_not_found")))
		return nil
	}
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mirror.BoardMountID)
	if err != nil || !mount.Active {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, failTitle, i18n.TranslateText(loc, "components.bbs.post_ui.mount_missing_or_inactive")))
		return nil
	}

	rawName := e.Data.Text("name")
	capID := selectedCapID(e.Data)
	rawBody := e.Data.Text("body")

	board, thread, post, backRefs, err := s.Service.CreatePost(e.Ctx, bbscore.CreatePostInput{
		ThreadID:    mirror.ThreadID,
		GuildID:     mirror.GuildID,
		UserID:      e.User().ID,
		RawName:     rawName,
		CapID:       capID,
		RawBody:     rawBody,
		SlipContext: buildSlipContext(e, mirror.GuildID),
		Now:         time.Now(),
	})
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, failTitle, describeBBSError(loc, err)))
		return nil
	}

	mountLocale := s.Syncer.ResolveMountLocale(e.Ctx, e.Client(), *mount)
	thChID, msgID, syncErr := s.Syncer.PublishPost(e.Ctx, e.Client(), mirror.BoardMountID, *board, *thread, *post, backRefs, mountLocale)
	if syncErr != nil {
		slog.Warn("bbs publish post sync failed", "thread_id", thread.ID, "res", post.ResNumber, "err", syncErr)
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
			i18n.TranslateText(loc, "components.bbs.post_ui.post_created_partial_title", map[string]any{"res": post.ResNumber}),
			i18n.TranslateText(loc, "components.bbs.post_ui.post_created_partial_body", map[string]any{"err": describeBBSError(loc, syncErr)}),
			discordsync.ColorNotice,
		).WithEphemeral(true))
		return nil
	}
	url := discord.MessageURL(mirror.GuildID, thChID, msgID)
	noticeBody := url
	if post.AnonID != "" {
		noticeBody = fmt.Sprintf("ID:%s\n%s", post.AnonID, url)
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.post_ui.post_created_title", map[string]any{"res": post.ResNumber}),
		noticeBody,
		discordsync.ColorLocalBoard,
	).WithEphemeral(true))
	return nil
}

func (s *System) handleBoardThreads(e *handler.ComponentEvent) error {
	loc := e.Locale()
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.post_ui.invalid_mount"),
		)
		return nil
	}
	fetchFail := i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed")
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		s.replyComponentError(e, fetchFail, i18n.TranslateText(loc, "components.bbs.post_ui.mount_missing_or_inactive"))
		return nil
	}
	if _, err := s.Service.EnsureGuildEnabled(e.Ctx, mount.GuildID); err != nil {
		s.replyComponentError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, mount.BoardID)
	if err != nil {
		s.replyComponentError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}
	status := bbscore.ThreadStatusActive
	threads, err := s.Service.ListThreads(e.Ctx, bbscore.ThreadFilter{
		BoardID: board.ID,
		Status:  &status,
		Limit:   25,
	})
	if err != nil {
		s.replyComponentError(e, fetchFail, describeBBSError(loc, err))
		return nil
	}

	subComps := []discord.ContainerSubComponent{
		discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.threads.list_title", map[string]any{
			"board": bbscore.ClearFormat(board.Name),
		})),
		discord.NewSmallSeparator(),
	}
	if len(threads) == 0 {
		subComps = append(subComps, discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.threads.empty")))
	} else {
		var lines []string
		var options []discord.StringSelectMenuOption
		for i, th := range threads {
			cleanTitle := discordsync.TruncateRunes(bbscore.ClearFormat(th.Title), 65)
			lines = append(lines, i18n.TranslateText(loc, "components.bbs.threads.line", map[string]any{
				"index": i + 1,
				"title": cleanTitle,
				"count": th.ResCount,
			}))
			options = append(options, discord.NewStringSelectMenuOption(
				discordsync.TruncateRunes(cleanTitle, 90),
				th.ID.String(),
			).WithDescription(i18n.TranslateText(loc, "components.bbs.threads.desc", map[string]any{
				"count": th.ResCount,
			})))
		}
		subComps = append(subComps, discord.NewTextDisplay(discordsync.TruncateRunes(strings.Join(lines, "\n"), 1800)))
		if len(options) > 0 {
			subComps = append(subComps, discord.NewActionRow(
				discord.NewStringSelectMenu(
					fmt.Sprintf("/bbs/board/jump/%s", mount.ID),
					discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.threads.select_placeholder"), discordsync.MaxSelectPlaceholderRunes),
					options...,
				),
			))
		}
	}

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(subComps...).WithAccentColor(discordsync.ColorLocalBoard),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleBoardJumpSelect(e *handler.ComponentEvent) error {
	loc := e.Locale()
	invalidTitle := i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input")
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyComponentError(e, invalidTitle, i18n.TranslateText(loc, "components.bbs.post_ui.invalid_mount"))
		return nil
	}
	vals := e.StringSelectMenuInteractionData().Values
	if len(vals) == 0 {
		s.replyComponentError(e, invalidTitle, i18n.TranslateText(loc, "components.bbs.threads.none_selected"))
		return nil
	}
	threadID, err := uuid.Parse(vals[0])
	if err != nil {
		s.replyComponentError(e, invalidTitle, i18n.TranslateText(loc, "components.bbs.threads.invalid_id"))
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs board jump defer failed", "err", err)
	}

	fetchFail := i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed")
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			i18n.TranslateText(loc, "components.bbs.threads.mount_not_found"),
		))
		return nil
	}
	if _, err := s.Service.EnsureGuildEnabled(e.Ctx, mount.GuildID); err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		))
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, mount.BoardID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, fetchFail, describeBBSError(loc, err)))
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, threadID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc, fetchFail, describeBBSError(loc, err)))
		return nil
	}
	if thread.BoardID != board.ID {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.errors.unauthorized"),
		))
		return nil
	}

	mountLocale := s.Syncer.ResolveMountLocale(e.Ctx, e.Client(), *mount)
	mirror, err := s.Syncer.EnsureThreadMirror(e.Ctx, e.Client(), *mount, *board, *thread, 0, mountLocale)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.threads.mirror_failed"),
			describeBBSError(loc, err),
		))
		return nil
	}

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplayf("### ■ %s", discordsync.TruncateRunes(bbscore.ClearFormat(thread.Title), 80)),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.threads.jump_body", map[string]any{
				"channel_id": mirror.DiscordThreadID,
				"count":      thread.ResCount,
			})),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewSecondaryButton(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.control_panel.btn.reader"), discordsync.MaxButtonLabelRunes), fmt.Sprintf("/bbs/reader/%s/1", thread.ID)),
			),
		).WithAccentColor(discordsync.ColorLocalBoard),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleWatchToggle(e *handler.ComponentEvent) error {
	loc := e.Locale()
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.threads.invalid_id"),
		)
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, threadID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.fetch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if existing, gErr := s.Service.Store.GetThreadWatch(e.Ctx, threadID, e.User().ID); gErr == nil && existing != nil {
		if err := s.Service.Store.DeleteThreadWatch(e.Ctx, threadID, e.User().ID); err != nil {
			s.replyComponentError(e,
				i18n.TranslateText(loc, "components.bbs.watch.unwatch_failed"),
				describeBBSError(loc, err),
			)
			return nil
		}
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
			i18n.TranslateText(loc, "components.bbs.watch.unwatched_title"),
			i18n.TranslateText(loc, "components.bbs.watch.unwatched_body", map[string]any{"title": thread.Title}),
			discordsync.ColorNotice,
		).WithEphemeral(true))
		return nil
	}

	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		if errors.Is(err, bbscore.ErrConsentRequired) {
			s.respondConsentModal(e, e.ID().Time(), loc, e.User().ID, pendingWatchThread, threadID)
			return nil
		}
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.watch.watch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}

	if err := s.Service.Store.SaveThreadWatch(e.Ctx, &bbscore.ThreadWatch{
		ThreadID:  threadID,
		UserID:    e.User().ID,
		Locale:    string(loc),
		CreatedAt: time.Now(),
	}); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.watch.watch_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.watch.watched_title"),
		i18n.TranslateText(loc, "components.bbs.watch.watched_body", map[string]any{"title": thread.Title}),
		discordsync.ColorLocalBoard,
	).WithEphemeral(true))
	return nil
}

func (s *System) handleReaderPage(e *handler.ComponentEvent) error {
	loc := e.Locale()
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.threads.invalid_id"),
		)
		return nil
	}
	pageSpec := e.Vars["page"]

	comps, err := s.renderReaderComponents(e.Ctx, e.GuildID(), threadID, pageSpec, loc)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.reader.render_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if e.Message.Flags.Has(discord.MessageFlagEphemeral) {
		upd := discord.NewMessageUpdateV2(comps...).WithAllowedMentions(&discord.AllowedMentions{})
		if err := sr.UpdateComponentMessage(e, upd); err != nil {
			slog.Warn("bbs failed to update reader page", "thread_id", threadID, "page", pageSpec, "err", err)
			if err2 := sr.RespondMessageCreate(e, discordsync.RenderError(loc,
				i18n.TranslateText(loc, "components.bbs.reader.render_failed"),
				describeBBSError(loc, err),
			)); err2 != nil {
				slog.Warn("bbs failed to send reader update fallback error", "thread_id", threadID, "err", err2)
			}
		}
		return nil
	}
	msg := discord.NewMessageCreateV2(comps...).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})
	if err := sr.RespondMessageCreate(e, msg); err != nil {
		slog.Warn("bbs failed to open reader page", "thread_id", threadID, "page", pageSpec, "err", err)
		if err2 := sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.reader.render_failed"),
			describeBBSError(loc, err),
		)); err2 != nil {
			slog.Warn("bbs failed to send reader open fallback error", "thread_id", threadID, "err", err2)
		}
	}
	return nil
}

func (s *System) handleReaderJumpButton(e *handler.ComponentEvent) error {
	loc := e.Locale()
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.threads.invalid_id"),
		)
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/reader/jumpmodal/%s", threadID),
		discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.modal.jump.title"), discordsync.MaxModalTitleRunes),
		discord.NewLabel(
			discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.modal.jump.label"), discordsync.MaxLabelTextRunes),
			discord.NewShortTextInput("target").
				WithRequired(true).
				WithMaxLength(16).
				WithPlaceholder(discordsync.TruncateRunes(i18n.TranslateText(loc, "components.bbs.modal.jump.placeholder"), discordsync.MaxTextInputPlaceholder)),
		),
	)
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open reader jump modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func (s *System) handleReaderJumpModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.threads.invalid_id"),
		)
		return nil
	}
	renderFail := i18n.TranslateText(loc, "components.bbs.reader.render_failed")
	thread, err := s.Service.GetThread(e.Ctx, threadID)
	if err != nil {
		s.replyModalError(e, renderFail, describeBBSError(loc, err))
		return nil
	}
	target, jErr := discordsync.ParseReaderJumpInput(e.Data.Text("target"), thread.ResCount)
	if jErr != nil {
		s.replyModalError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			describeBBSError(loc, jErr),
		)
		return nil
	}
	comps, err := s.renderReaderComponents(e.Ctx, e.GuildID(), threadID, strconv.FormatUint(uint64(target), 10), loc)
	if err != nil {
		s.replyModalError(e, renderFail, describeBBSError(loc, err))
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	upd := discord.NewMessageUpdateV2(comps...).WithAllowedMentions(&discord.AllowedMentions{})
	if uErr := sr.UpdateComponentMessage(e, upd); uErr != nil {
		slog.Warn("bbs reader jump update fell back to ephemeral create", "thread_id", threadID, "err", uErr)
		msg := discord.NewMessageCreateV2(comps...).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})
		if cErr := sr.RespondMessageCreate(e, msg); cErr != nil {
			slog.Warn("bbs failed to send reader jump page", "thread_id", threadID, "err", cErr)
			if err2 := sr.RespondMessageCreate(e, discordsync.RenderError(loc, renderFail, describeBBSError(loc, cErr))); err2 != nil {
				slog.Warn("bbs failed to send reader jump fallback error", "thread_id", threadID, "err", err2)
			}
		}
	}
	return nil
}

func (s *System) renderReaderComponents(ctx context.Context, guildID *snowflake.ID, threadID uuid.UUID, pageSpec string, locale discord.Locale) ([]discord.LayoutComponent, error) {
	thread, err := s.Service.GetThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	posts, err := s.Service.ListPosts(ctx, threadID, 0, int(max(thread.ResCount, 1000)))
	if err != nil {
		return nil, err
	}

	var resolveURL func(resNum uint32) (string, bool)
	if guildID != nil && len(posts) > 0 {
		if pm, pErr := s.Service.Store.GetPostMirrorByPostAndGuild(ctx, posts[0].ID, *guildID); pErr == nil {
			if tm, tErr := s.Service.Store.GetThreadMirror(ctx, pm.ThreadMirrorID); tErr == nil {
				gid := *guildID
				tid := tm.DiscordThreadID
				urlCache := make(map[uint32]string)
				urlMissed := make(map[uint32]bool)
				resolveURL = func(resNum uint32) (string, bool) {
					if u, ok := urlCache[resNum]; ok {
						return u, true
					}
					if urlMissed[resNum] {
						return "", false
					}
					m, mErr := s.Service.Store.GetPostMirrorByResAndGuild(ctx, threadID, resNum, gid)
					if mErr != nil || m.DiscordMessageID == 0 {
						urlMissed[resNum] = true
						return "", false
					}
					u := discord.MessageURL(gid, tid, m.DiscordMessageID)
					urlCache[resNum] = u
					return u, true
				}
			}
		}
	}

	var maxEndRes uint32
	startIndex := 0
	trimmedSpec := strings.TrimSpace(pageSpec)
	switch trimmedSpec {
	case "latest":
		startIndex = discordsync.ComputeLatestReaderStartIndex(*thread, posts, resolveURL, locale)
	case "first":
		startIndex = 0
	default:
		startStr, endStr, isRange := strings.Cut(trimmedSpec, "-")
		var targetRes uint32
		if isRange {
			sVal, sErr := strconv.ParseUint(startStr, 10, 32)
			eVal, eErr := strconv.ParseUint(endStr, 10, 32)
			if sErr != nil || eErr != nil || sVal == 0 || eVal < sVal || (len(posts) > 0 && uint32(eVal) > thread.ResCount) {
				return nil, bbscore.ErrInvalidReaderJump
			}
			targetRes = uint32(sVal)
			maxEndRes = uint32(eVal)
		} else {
			n, pErr := strconv.ParseUint(trimmedSpec, 10, 32)
			if pErr != nil || n == 0 || (len(posts) > 0 && uint32(n) > thread.ResCount) {
				return nil, bbscore.ErrInvalidReaderJump
			}
			targetRes = uint32(n)
		}
		if len(posts) > 0 && targetRes > 1 {
			startIndex = len(posts) - 1
			for i, p := range posts {
				if p.ResNumber >= targetRes {
					startIndex = i
					break
				}
			}
			if startIndex < 0 {
				startIndex = 0
			}
		}
	}

	comps, _, _ := discordsync.RenderReaderPageBounded(*thread, posts, startIndex, maxEndRes, resolveURL, locale)
	return comps, nil
}
