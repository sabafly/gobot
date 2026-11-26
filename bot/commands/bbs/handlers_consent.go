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
	"log/slog"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

func (s *System) handleConsentViewDoc(e *handler.ComponentEvent) error {
	loc := e.Locale()
	kind := bbscore.PolicyDocKind(e.Vars["doc"])
	var titleKey string
	switch kind {
	case bbscore.PolicyDocTerms:
		titleKey = "components.bbs.consent.btn.terms"
	case bbscore.PolicyDocGuidelines:
		titleKey = "components.bbs.consent.btn.guidelines"
	case bbscore.PolicyDocPrivacy:
		titleKey = "components.bbs.consent.btn.privacy"
	default:
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.consent.doc.invalid"),
		)
		return nil
	}

	docText := s.Service.Policy.GetDocument(kind, string(loc))
	msg := discordsync.RenderNotice(loc, i18n.TranslateText(loc, titleKey), docText, discordsync.ColorLocalBoard).WithEphemeral(true)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleConsentAccept(e *handler.ComponentEvent) error {
	loc := e.Locale()
	sr := discordsync.NewSafeResponder(e.ID().Time())

	versionTok := e.Vars["version"]
	consent, err := s.Service.AcceptPolicyVersion(e.Ctx, e.User().ID, versionTok)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.consent.accept_failed"),
			describeBBSError(loc, err),
		))
		return nil
	}

	if act, ok := s.popPending(e.Vars["token"], e.User().ID); ok {
		switch act.Kind {
		case pendingNewThread:
			boardName := ""
			if mount, mErr := s.Service.Store.GetBoardMount(e.Ctx, act.TargetID); mErr == nil {
				if board, bErr := s.Service.GetBoard(e.Ctx, mount.BoardID); bErr == nil {
					boardName = board.Name
				}
			}
			modal := buildNewThreadModal(loc, act.TargetID, boardName)
			if err := sr.RespondModal(e, modal); err == nil {
				return nil
			}
		case pendingNewPost:
			modal := buildNewPostModal(loc, act.TargetID)
			if err := sr.RespondModal(e, modal); err == nil {
				return nil
			}
		case pendingOpenReader:
			if mu, rErr := s.buildReaderMessageUpdate(e.Ctx, e.GuildID(), act.TargetID, act.Page, loc); rErr == nil {
				_ = sr.UpdateComponentMessage(e, mu)
				return nil
			}
		}
	}

	upd := discord.NewMessageUpdateV2(
		discord.NewContainer(
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.accepted.title")),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.accepted.body", map[string]any{
				"version": consent.PolicyVersion,
			})),
		).WithAccentColor(discordsync.ColorLocalBoard),
	).WithAllowedMentions(&discord.AllowedMentions{})

	_ = sr.UpdateComponentMessage(e, upd)
	return nil
}

func (s *System) handleConsentRevoke(e *handler.ComponentEvent) error {
	loc := e.Locale()
	if err := s.Service.RevokePolicy(e.Ctx, e.User().ID); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.consent.revoke_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	upd := discord.NewMessageUpdateV2(
		discord.NewContainer(
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.revoked.title")),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.revoked.body")),
		).WithAccentColor(discordsync.ColorNotice),
	).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.UpdateComponentMessage(e, upd)
	return nil
}

func (s *System) handleConsentRole(e *handler.ComponentEvent) error {
	loc := e.Locale()
	gidVal, err := strconv.ParseUint(e.Vars["guild_id"], 10, 64)
	if err != nil || gidVal == 0 {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.consent.role.invalid_guild"),
		)
		return nil
	}
	guildID := snowflake.ID(gidVal)

	versionTok := e.Vars["version"]
	if versionTok == "" || versionTok != bbscore.PolicyVersionToken(s.Service.Policy.CurrentVersion) {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.consent.accept_failed"),
			describeBBSError(loc, bbscore.ErrStalePolicyVersion),
		)
		return nil
	}

	cfg, err := s.Service.EnsureGuildEnabled(e.Ctx, guildID)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.consent.role.failed"),
			describeBBSError(loc, err),
		)
		return nil
	}
	if cfg.ViewerRoleID == nil || *cfg.ViewerRoleID == 0 {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.consent.role.not_configured_title"),
			i18n.TranslateText(loc, "components.bbs.consent.role.not_configured_body"),
		)
		return nil
	}

	checker := discordsync.NewPreflightChecker(discordsync.NewClientGuildStateReader(e.Client()))
	if err := checker.CheckRoleAssign(guildID, *cfg.ViewerRoleID); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.consent.role.unassignable_title"),
			describeBBSError(loc, err),
		)
		return nil
	}

	if _, err := s.Service.AcceptPolicyVersion(e.Ctx, e.User().ID, versionTok); err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.consent.accept_failed"),
			describeBBSError(loc, err),
		)
		return nil
	}

	if err := e.Client().Rest.AddMemberRole(guildID, e.User().ID, *cfg.ViewerRoleID); err != nil {
		slog.Warn("bbs failed to assign viewer role", "guild_id", guildID, "role_id", *cfg.ViewerRoleID, "err", err)
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.consent.role.failed"),
			i18n.TranslateText(loc, "components.bbs.consent.role.error_body", map[string]any{"err": describeBBSError(loc, err)}),
		)
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.consent.role.success_title"),
		i18n.TranslateText(loc, "components.bbs.consent.role.success_body", map[string]any{"role_id": *cfg.ViewerRoleID}),
		discordsync.ColorLocalBoard,
	).WithEphemeral(true))
	return nil
}
