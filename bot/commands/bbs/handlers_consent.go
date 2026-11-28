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
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/federation"
	"github.com/sabafly/gobot/internal/i18n"
)

type messageRespondEvent interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
	UpdateInteractionResponse(messageUpdate discord.MessageUpdate, opts ...rest.RequestOpt) (*discord.Message, error)
	CreateFollowupMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) (*discord.Message, error)
}

type modalRespondEvent interface {
	Modal(modalCreate discord.ModalCreate, opts ...rest.RequestOpt) error
	messageRespondEvent
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

func (s *System) respondFederationAuthPrompt(
	e messageRespondEvent,
	createdAt time.Time,
	loc discord.Locale,
	providerKey string,
	userID, appID snowflake.ID,
	interactionToken, continueCustomID string,
) {
	sr := discordsync.NewSafeResponder(createdAt)
	if s.FederationAuth == nil || !s.FederationAuth.Enabled() {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, bbscore.ErrFederationReadOnly),
		))
		return
	}
	authURL, label, err := s.FederationAuth.BeginUserAuth(context.Background(), providerKey, userID, appID, interactionToken, continueCustomID, loc)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError(loc,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		))
		return
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderFederationAuthPrompt(label, authURL, continueCustomID, loc))
}

// ensureRemoteDelegation verifies that userID has an active OAuth delegation if board belongs to a FederatedProvider.
// Returns true if execution may proceed immediately, or false if an error/auth prompt response was sent.
func (s *System) ensureRemoteDelegation(
	e *handler.ComponentEvent,
	board bbscore.Board,
	continueCustomID string,
) bool {
	loc := e.Locale()
	p, err := s.Service.Registry.Get(board.ProviderKey)
	if err != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, err),
		)
		return false
	}
	fp, ok := p.(bbscore.FederatedProvider)
	if !ok {
		return true
	}
	hasDelegation, dErr := fp.HasUserDelegation(e.Ctx, e.User().ID)
	if dErr != nil {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.errors.title.unavailable"),
			describeBBSError(loc, dErr),
		)
		return false
	}
	if hasDelegation {
		return true
	}
	s.respondFederationAuthPrompt(e, e.ID().Time(), loc, fp.Key(), e.User().ID, e.ApplicationID(), e.Token(), continueCustomID)
	return false
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

func (s *System) handleFederationRevokeSelect(e *handler.ComponentEvent) error {
	if s.Service == nil || s.Service.Store == nil {
		return nil
	}
	loc := e.Locale()
	sr := discordsync.NewSafeResponder(e.ID().Time())
	vals := e.StringSelectMenuInteractionData().Values
	if len(vals) == 0 {
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoke_failed"),
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_invalid_select"),
		)
		return nil
	}
	selected := vals[0]

	switch {
	case strings.HasPrefix(selected, "origin:"):
		rawID := strings.TrimPrefix(selected, "origin:")
		grantID, err := uuid.Parse(rawID)
		if err != nil {
			s.replyComponentError(e,
				i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoke_failed"),
				i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_invalid_select"),
			)
			return nil
		}
		grant, err := s.Service.Store.GetFederationGrant(e.Ctx, grantID)
		if err != nil || grant == nil || grant.UserID != e.User().ID {
			if err == nil {
				err = bbscore.ErrNotFound
			}
			s.replyComponentError(e,
				i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoke_failed"),
				describeBBSError(loc, err),
			)
			return nil
		}
		if err := s.Service.Store.RevokeFederationGrant(e.Ctx, grantID, e.User().ID, time.Now().UTC()); err != nil {
			s.replyComponentError(e,
				i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoke_failed"),
				describeBBSError(loc, err),
			)
			return nil
		}
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoked_title"),
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoked_origin_body", map[string]any{
				"peer_id": grant.PeerID,
			}),
			discordsync.ColorNotice,
		).WithEphemeral(true))
		return nil

	case strings.HasPrefix(selected, federation.RemoteKeyPrefix):
		remoteKey := strings.TrimPrefix(selected, federation.RemoteKeyPrefix)
		if strings.TrimSpace(remoteKey) == "" {
			s.replyComponentError(e,
				i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoke_failed"),
				i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_invalid_select"),
			)
			return nil
		}
		label := remoteKey
		var revokeErr error
		if p, pErr := s.Service.Registry.Get(federation.RemoteKeyPrefix + remoteKey); pErr == nil && p != nil {
			if fp, ok := p.(bbscore.FederatedProvider); ok {
				if l := strings.TrimSpace(fp.DisplayLabel()); l != "" {
					label = l
				}
				revokeErr = fp.RevokeUserDelegation(e.Ctx, e.User().ID)
			} else {
				revokeErr = s.Service.Store.DeleteFederationUserToken(e.Ctx, remoteKey, e.User().ID)
			}
		} else {
			revokeErr = s.Service.Store.DeleteFederationUserToken(e.Ctx, remoteKey, e.User().ID)
		}
		if revokeErr != nil {
			s.replyComponentError(e,
				i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoke_failed"),
				describeBBSError(loc, revokeErr),
			)
			return nil
		}
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(loc,
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoked_title"),
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoked_remote_body", map[string]any{
				"label": label,
			}),
			discordsync.ColorNotice,
		).WithEphemeral(true))
		return nil

	default:
		s.replyComponentError(e,
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_revoke_failed"),
			i18n.TranslateText(loc, "components.bbs.cmd.policy.federation_invalid_select"),
		)
		return nil
	}
}
