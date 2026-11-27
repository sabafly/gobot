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
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

type modalRespondEvent interface {
	Modal(modalCreate discord.ModalCreate, opts ...rest.RequestOpt) error
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
	UpdateInteractionResponse(messageUpdate discord.MessageUpdate, opts ...rest.RequestOpt) (*discord.Message, error)
	CreateFollowupMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) (*discord.Message, error)
}

func (s *System) respondConsentModal(
	e modalRespondEvent,
	createdAt time.Time,
	loc discord.Locale,
	userID snowflake.ID,
	kind pendingKind,
	targetID uuid.UUID,
) {
	token := "none"
	if kind != "" && targetID != uuid.Nil {
		token = s.savePending(userID, kind, targetID)
	}
	modal := discordsync.RenderConsentModal(s.Service.Policy, loc, token)
	sr := discordsync.NewSafeResponder(createdAt)
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open consent modal", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
}

func (s *System) handleConsentViewDoc(e *handler.ComponentEvent) error {
	loc := e.Locale()
	kind := bbscore.PolicyDocKind(e.Vars["doc"])
	switch kind {
	case bbscore.PolicyDocTerms, bbscore.PolicyDocGuidelines, bbscore.PolicyDocPrivacy:
	default:
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.invalid_input"),
			i18n.TranslateText(loc, "components.bbs.consent.doc.invalid"),
		)
		return nil
	}

	msg := discordsync.RenderPolicyDocumentView(s.Service.Policy, kind, loc)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleConsentOpenModal(e *handler.ComponentEvent) error {
	loc := e.Locale()
	sr := discordsync.NewSafeResponder(e.ID().Time())

	versionTok := e.Vars["version"]
	if versionTok == "" || versionTok != bbscore.PolicyVersionToken(s.Service.Policy.CurrentVersion) {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.consent.accept_failed"),
			describeBBSError(loc, bbscore.ErrStalePolicyVersion),
		))
		return nil
	}

	token := e.Vars["token"]
	if token == "" {
		token = "none"
	}
	modal := discordsync.RenderConsentModal(s.Service.Policy, loc, token)
	if err := sr.RespondModal(e, modal); err != nil {
		slog.Warn("bbs failed to open consent modal from component", "err", err)
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.modal_failed"),
			describeBBSError(loc, err),
		))
	}
	return nil
}

func hasAllConsentChecks(data discord.ModalSubmitInteractionData) bool {
	comp, ok := data.Component(discordsync.ConsentCheckboxGroupCustomID)
	if !ok {
		return false
	}
	cb, ok := comp.(discord.CheckboxGroupComponent)
	if !ok {
		return false
	}
	return slices.Contains(cb.Values, discordsync.ConsentOptTerms) &&
		slices.Contains(cb.Values, discordsync.ConsentOptGuidelines) &&
		slices.Contains(cb.Values, discordsync.ConsentOptPrivacy)
}

func (s *System) handleConsentModal(e *handler.ModalEvent) error {
	loc := e.Locale()
	sr := discordsync.NewSafeResponder(e.ID().Time())

	if !hasAllConsentChecks(e.Data) {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.consent.accept_failed"),
			i18n.TranslateText(loc, "components.bbs.consent.modal.all_required"),
		))
		return nil
	}

	consent, err := s.Service.AcceptPolicyVersion(e.Ctx, e.User().ID, e.Vars["version"])
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.consent.accept_failed"),
			describeBBSError(loc, err),
		))
		return nil
	}

	if act, ok := s.popPending(e.Vars["token"], e.User().ID); ok {
		switch act.Kind {
		case pendingWatchThread:
			thread, tErr := s.Service.GetThread(e.Ctx, act.TargetID)
			var wErr error
			if tErr == nil {
				wErr = s.Service.Store.SaveThreadWatch(e.Ctx, &bbscore.ThreadWatch{
					ThreadID:  act.TargetID,
					UserID:    e.User().ID,
					Locale:    string(loc),
					CreatedAt: time.Now(),
				})
			}
			if tErr == nil && wErr == nil {
				_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
					i18n.TranslateText(loc, "components.bbs.consent.accepted.title"),
					i18n.TranslateText(loc, "components.bbs.consent.accepted.watch_done", map[string]any{
						"version": consent.PolicyVersion,
						"title":   thread.Title,
					}),
					discordsync.ColorLocalBoard,
				).WithEphemeral(true))
				return nil
			}
			watchErr := tErr
			if watchErr == nil {
				watchErr = wErr
			}
			_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
				i18n.TranslateText(loc, "components.bbs.consent.accepted.title"),
				i18n.TranslateText(loc, "components.bbs.consent.accepted.watch_failed", map[string]any{
					"version": consent.PolicyVersion,
					"err":     describeBBSError(loc, watchErr),
				}),
				discordsync.ColorNotice,
			).WithEphemeral(true))
			return nil

		case pendingNewThread, pendingNewPost, pendingReplyPost, pendingReportThread, pendingReportPost:
			var btnLabelKey, btnCustomID string
			switch act.Kind {
			case pendingNewThread:
				btnLabelKey = "components.bbs.consent.continue.new_thread"
				btnCustomID = fmt.Sprintf("/bbs/thread/new/%s", act.TargetID)
			case pendingNewPost:
				btnLabelKey = "components.bbs.consent.continue.new_post"
				btnCustomID = fmt.Sprintf("/bbs/post/new/%s", act.TargetID)
			case pendingReplyPost:
				btnLabelKey = "components.bbs.consent.continue.reply_post"
				btnCustomID = fmt.Sprintf("/bbs/post/reply/%s", act.TargetID)
			case pendingReportThread:
				btnLabelKey = "components.bbs.consent.continue.report_thread"
				btnCustomID = fmt.Sprintf("/bbs/report/thread/%s", act.TargetID)
			case pendingReportPost:
				btnLabelKey = "components.bbs.consent.continue.report_post"
				btnCustomID = fmt.Sprintf("/bbs/report/post/%s", act.TargetID)
			}
			msg := discord.NewMessageCreateV2(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.accepted.title")),
					discord.NewSmallSeparator(),
					discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.accepted.continue_body", map[string]any{
						"version": consent.PolicyVersion,
					})),
					discord.NewLargeSeparator(),
					discord.NewActionRow(
						discord.NewPrimaryButton(
							discordsync.TruncateRunes(i18n.TranslateText(loc, btnLabelKey), discordsync.MaxButtonLabelRunes),
							btnCustomID,
						),
					),
				).WithAccentColor(discordsync.ColorLocalBoard),
			).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})
			_ = sr.RespondMessageCreate(e, msg)
			return nil
		}
	}

	container := discord.NewContainer(
		discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.accepted.title")),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(i18n.TranslateText(loc, "components.bbs.consent.accepted.body", map[string]any{
			"version": consent.PolicyVersion,
		})),
	).WithAccentColor(discordsync.ColorLocalBoard)

	if e.Message != nil && e.Message.Flags.Has(discord.MessageFlagEphemeral) {
		upd := discord.NewMessageUpdateV2(container).WithAllowedMentions(&discord.AllowedMentions{})
		if uErr := sr.UpdateComponentMessage(e, upd); uErr == nil {
			return nil
		}
	}

	msg := discord.NewMessageCreateV2(container).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})
	_ = sr.RespondMessageCreate(e, msg)
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

func (s *System) handleConsentRoleDeprecated(e *handler.ComponentEvent) error {
	loc := e.Locale()
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
		i18n.TranslateText(loc, "components.bbs.consent.role.deprecated_title"),
		i18n.TranslateText(loc, "components.bbs.consent.role.deprecated_body"),
		discordsync.ColorNotice,
	).WithEphemeral(true))
	return nil
}
