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
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
)

func (s *System) handleConsentViewDoc(e *handler.ComponentEvent) error {
	kind := bbscore.PolicyDocKind(e.Vars["doc"])
	var title string
	switch kind {
	case bbscore.PolicyDocTerms:
		title = "利用規約"
	case bbscore.PolicyDocGuidelines:
		title = "コミュニティガイドライン"
	case bbscore.PolicyDocPrivacy:
		title = "プライバシーポリシー"
	default:
		s.replyComponentError(e, errTitleInvalidInput, "不明な文書種別です。")
		return nil
	}

	docText := s.Service.Policy.GetDocument(kind, string(e.Locale()))
	msg := discordsync.RenderNotice(title, docText, discordsync.ColorLocalBoard)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleConsentAccept(e *handler.ComponentEvent) error {
	consent, err := s.Service.AcceptPolicy(e.Ctx, e.User().ID)
	if err != nil {
		s.replyComponentError(e, "同意登録失敗", describeBBSError(err))
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
			modal := buildNewThreadModal(act.TargetID, boardName)
			if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err == nil {
				return nil
			}
		case pendingNewPost:
			modal := buildNewPostModal(act.TargetID)
			if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err == nil {
				return nil
			}
		case pendingOpenReader:
			if mu, rErr := s.buildReaderMessageUpdate(e.Ctx, e.GuildID(), act.TargetID, act.Page); rErr == nil {
				sr := discordsync.NewSafeResponder(e.ID().Time())
				_ = sr.UpdateComponentMessage(e, mu)
				return nil
			}
		}
	}

	upd := discord.NewMessageUpdateV2(
		discord.NewContainer(
			discord.NewTextDisplay("### ■ 同意が完了しました"),
			discord.NewSmallSeparator(),
			discord.NewTextDisplayf("BBS利用規約・ガイドライン・プライバシーポリシー (version: %s) への同意を記録しました。\n元のボタンを押して操作を続けてください。", consent.PolicyVersion),
		).WithAccentColor(discordsync.ColorLocalBoard),
	).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.UpdateComponentMessage(e, upd)
	return nil
}

func (s *System) handleConsentRevoke(e *handler.ComponentEvent) error {
	if err := s.Service.RevokePolicy(e.Ctx, e.User().ID); err != nil {
		s.replyComponentError(e, "撤回失敗", describeBBSError(err))
		return nil
	}
	upd := discord.NewMessageUpdateV2(
		discord.NewContainer(
			discord.NewTextDisplay("### ■ ポリシー同意を撤回しました"),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay("BBSポリシーへの同意を撤回しました。今後の投稿・閲覧操作には再同意が必要です。"),
		).WithAccentColor(discordsync.ColorNotice),
	).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.UpdateComponentMessage(e, upd)
	return nil
}

func (s *System) handleConsentRole(e *handler.ComponentEvent) error {
	gidVal, err := strconv.ParseUint(e.Vars["guild_id"], 10, 64)
	if err != nil || gidVal == 0 {
		s.replyComponentError(e, errTitleInvalidInput, "サーバーIDが不正です。")
		return nil
	}
	guildID := snowflake.ID(gidVal)

	if _, err := s.Service.AcceptPolicy(e.Ctx, e.User().ID); err != nil {
		s.replyComponentError(e, "同意記録失敗", describeBBSError(err))
		return nil
	}

	cfg, err := s.Service.EnsureGuildEnabled(e.Ctx, guildID)
	if err != nil {
		s.replyComponentError(e, "ロール付与失敗", describeBBSError(err))
		return nil
	}
	if cfg.ViewerRoleID == nil || *cfg.ViewerRoleID == 0 {
		s.replyComponentError(e, "ロール未設定", "このサーバーには閲覧ロールが設定されていません。")
		return nil
	}

	checker := discordsync.NewPreflightChecker(discordsync.NewClientGuildStateReader(e.Client()))
	if err := checker.CheckRoleAssign(guildID, *cfg.ViewerRoleID); err != nil {
		s.replyComponentError(e, "ロール付与不可", err.Error())
		return nil
	}
	if err := e.Client().Rest.AddMemberRole(guildID, e.User().ID, *cfg.ViewerRoleID); err != nil {
		s.replyComponentError(e, "ロール付与失敗", fmt.Sprintf("ロール付与中にエラーが発生しました: %v", err))
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(
		"同意・閲覧ロール付与完了",
		fmt.Sprintf("ポリシー同意を記録し、<@&%d> を付与しました。", *cfg.ViewerRoleID),
		discordsync.ColorLocalBoard,
	))
	return nil
}
