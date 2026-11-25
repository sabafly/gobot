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
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/google/uuid"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
)

func (s *System) handleReportMessageCommand(data discord.MessageCommandInteractionData, e *handler.CommandEvent) error {
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		msg := discordsync.RenderConsentGate(s.Service.Policy, e.Locale(), "none")
		sr := discordsync.NewSafeResponder(e.ID().Time())
		_ = sr.RespondMessageCreate(e, msg)
		return nil
	}

	targetMsg := data.TargetMessage()
	pm, err := s.Service.Store.GetPostMirrorByDiscordMessage(e.Ctx, targetMsg.ID)
	if err != nil {
		s.replyCommandError(e, "通報対象外", "このメッセージはBBSレスのミラーではありません。")
		return nil
	}

	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/report/postmodal/%s", pm.PostID),
		"レスを通報",
		discord.NewLabel(
			"通報理由 (荒らし・個人情報・規約違反など)",
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(300),
		),
	)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.RespondModal(e, modal); err != nil {
		s.replyCommandError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (s *System) handleReportThreadButton(e *handler.ComponentEvent) error {
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		msg := discordsync.RenderConsentGate(s.Service.Policy, e.Locale(), "none")
		sr := discordsync.NewSafeResponder(e.ID().Time())
		_ = sr.RespondMessageCreate(e, msg)
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/report/threadmodal/%s", threadID),
		"スレッドを通報",
		discord.NewLabel(
			"通報理由 (具体的なレス番号や内容を記載)",
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(300),
		),
	)
	if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err != nil {
		s.replyComponentError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (s *System) handleReportPostModal(e *handler.ModalEvent) error {
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyModalError(e, errTitleInvalidInput, "レスIDが不正です。")
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		s.replyModalError(e, errTitleInvalidInput, "サーバー内から実行してください。")
		return nil
	}
	report, err := s.Service.CreateReport(e.Ctx, *guildID, e.User().ID, postID, e.Data.Text("reason"))
	if err != nil {
		s.replyModalError(e, "通報失敗", describeBBSError(err))
		return nil
	}
	s.notifyReportLog(e.Client(), report)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("通報を受け付けました", fmt.Sprintf("ReportID: %s\n内容を確認のうえ対処します。", report.ID), discordsync.ColorNotice))
	return nil
}

func (s *System) handleReportThreadModal(e *handler.ModalEvent) error {
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyModalError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		s.replyModalError(e, errTitleInvalidInput, "サーバー内から実行してください。")
		return nil
	}
	posts, err := s.Service.ListPosts(e.Ctx, threadID, 0, 1)
	if err != nil || len(posts) == 0 {
		s.replyModalError(e, "通報失敗", "対象スレッドのレスが見つかりません。")
		return nil
	}
	report, err := s.Service.CreateReport(e.Ctx, *guildID, e.User().ID, posts[0].ID, e.Data.Text("reason"))
	if err != nil {
		s.replyModalError(e, "通報失敗", describeBBSError(err))
		return nil
	}
	s.notifyReportLog(e.Client(), report)
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("スレッド通報を受け付けました", fmt.Sprintf("ReportID: %s", report.ID), discordsync.ColorNotice))
	return nil
}

func (s *System) notifyReportLog(client *bot.Client, report *bbscore.Report) {
	if client == nil || s.LogChannel == nil || *s.LogChannel == 0 {
		return
	}
	msg := discordsync.RenderNotice(
		"新規BBS通報を受信しました",
		fmt.Sprintf("ReportID: %s\nScope: %s / Guild: %d\nPostID: %s\n理由: %s\n対応: /bbs-admin report inspect",
			report.ID, report.Scope, report.GuildID, report.PostID, discordsync.TruncateRunes(report.Reason, 200)),
		discordsync.ColorError,
	).WithEphemeral(false)
	validated, err := s.Syncer.Validator.ValidateAndClampMessageCreate(msg)
	if err != nil {
		return
	}
	_, _ = client.Rest.CreateMessage(*s.LogChannel, validated)
}

func (s *System) handleModerateMessageCommand(data discord.MessageCommandInteractionData, e *handler.CommandEvent) error {
	if !s.requireCmdGuildAdmin(e) {
		return nil
	}
	guildID := e.GuildID()
	if guildID == nil {
		return nil
	}

	targetMsg := data.TargetMessage()
	pm, err := s.Service.Store.GetPostMirrorByDiscordMessage(e.Ctx, targetMsg.ID)
	if err != nil || pm.GuildID != *guildID {
		s.replyCommandError(e, "対象外", "このサーバーに紐づくBBSレスミラーが見つかりません。")
		return nil
	}
	post, err := s.Service.GetPost(e.Ctx, pm.PostID)
	if err != nil {
		s.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, post.ThreadID)
	if err != nil {
		s.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, thread.BoardID)
	if err != nil {
		s.replyCommandError(e, "取得失敗", describeBBSError(err))
		return nil
	}

	// Strict anonymity boundary: guild moderators only ever inspect PosterHash, never AuthorUserID.
	info := fmt.Sprintf("板: %s (%s)\nスレッド: %s (>>%d)\n投稿者識別ハッシュ (匿名): %s\n日替わりID: %s",
		board.Name, board.Scope, thread.Title, post.ResNumber, post.PosterHash, post.AnonID)

	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *guildID {
		s.replyCommandNotice(
			e,
			"匿名モデレーション情報 (グローバル板)",
			info+"\n\n注意: グローバル板のあぼーん・過去ログ化・BANは gobot 運営のみ実施可能です。問題投稿は「BBS - レスを通報」からご報告ください。",
			discordsync.ColorGlobalBoard,
		)
		return nil
	}

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay("### ■ ローカル板 匿名モデレーション"),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(info),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewDangerButton("このレスをあぼーん", fmt.Sprintf("/bbs/mod/abone/%s", post.ID)),
				discord.NewSecondaryButton("スレッドを過去ログ化", fmt.Sprintf("/bbs/mod/archive/%s", thread.ID)),
				discord.NewDangerButton("PosterHashをローカルBAN", fmt.Sprintf("/bbs/mod/ban/%s", post.ID)),
			),
		).WithAccentColor(discordsync.ColorError),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleModAboneButton(e *handler.ComponentEvent) error {
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) {
		s.replyComponentError(e, errTitlePermission, "サーバー管理権限が必要です。")
		return nil
	}
	postID := e.Vars["post_id"]
	if _, err := uuid.Parse(postID); err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "レスIDが不正です。")
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/mod/abonemodal/%s", postID),
		"レスをあぼーん",
		discord.NewLabel(
			"あぼーん理由 (任意)",
			discord.NewShortTextInput("reason").WithRequired(false).WithMaxLength(120).WithPlaceholder("例: 荒らし・規約違反"),
		),
	)
	if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err != nil {
		s.replyComponentError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (s *System) handleModAboneModal(e *handler.ModalEvent) error {
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) || e.GuildID() == nil {
		s.replyModalError(e, errTitlePermission, "サーバー管理権限が必要です。")
		return nil
	}
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyModalError(e, errTitleInvalidInput, "レスIDが不正です。")
		return nil
	}
	post, err := s.Service.AbonePostLocal(e.Ctx, *e.GuildID(), postID, e.Data.Text("reason"))
	if err != nil {
		s.replyModalError(e, "あぼーん失敗", describeBBSError(err))
		return nil
	}
	if thread, tErr := s.Service.GetThread(e.Ctx, post.ThreadID); tErr == nil {
		go s.Syncer.SyncAbone(s.Components.Ctx(), e.Client(), *post, *thread)
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("あぼーんを実行しました", fmt.Sprintf(">>%d をあぼーん表示に更新しました (レス番号は維持)。", post.ResNumber), discordsync.ColorNotice))
	return nil
}

func (s *System) handleModArchiveButton(e *handler.ComponentEvent) error {
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) || e.GuildID() == nil {
		s.replyComponentError(e, errTitlePermission, "サーバー管理権限が必要です。")
		return nil
	}
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}
	thread, err := s.Service.ArchiveThreadLocal(e.Ctx, *e.GuildID(), threadID)
	if err != nil {
		s.replyComponentError(e, "過去ログ化失敗", describeBBSError(err))
		return nil
	}
	if board, bErr := s.Service.GetBoard(e.Ctx, thread.BoardID); bErr == nil {
		go s.Syncer.SyncArchive(s.Components.Ctx(), e.Client(), *board, *thread, false)
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("過去ログ倉庫へ格納しました", fmt.Sprintf("スレッド: %s をロック+アーカイブします。", thread.Title), discordsync.ColorNotice))
	return nil
}

func (s *System) handleModBanButton(e *handler.ComponentEvent) error {
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) {
		s.replyComponentError(e, errTitlePermission, "サーバー管理権限が必要です。")
		return nil
	}
	postID := e.Vars["post_id"]
	if _, err := uuid.Parse(postID); err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "レスIDが不正です。")
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/mod/banmodal/%s", postID),
		"PosterHash ローカルBAN",
		discord.NewLabel(
			"期間 (時間数・空欄または0で無期限)",
			discord.NewShortTextInput("hours").WithRequired(false).WithMaxLength(10).WithPlaceholder("例: 24"),
		),
		discord.NewLabel(
			"理由",
			discord.NewParagraphTextInput("reason").WithRequired(true).WithMaxLength(200),
		),
	)
	if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err != nil {
		s.replyComponentError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (s *System) handleModBanModal(e *handler.ModalEvent) error {
	if !s.hasGuildAdmin(e.Ctx, e.Member(), e.GuildID(), e.User(), e.Client()) || e.GuildID() == nil {
		s.replyModalError(e, errTitlePermission, "サーバー管理権限が必要です。")
		return nil
	}
	postID, err := uuid.Parse(e.Vars["post_id"])
	if err != nil {
		s.replyModalError(e, errTitleInvalidInput, "レスIDが不正です。")
		return nil
	}
	post, err := s.Service.GetPost(e.Ctx, postID)
	if err != nil {
		s.replyModalError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, post.ThreadID)
	if err != nil {
		s.replyModalError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, thread.BoardID)
	if err != nil {
		s.replyModalError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	if board.Scope != bbscore.ScopeLocal || board.GuildID == nil || *board.GuildID != *e.GuildID() {
		s.replyModalError(e, errTitlePermission, "ローカルBANはこのサーバー所有のローカル板投稿にのみ適用できます。")
		return nil
	}

	hoursRaw := strings.TrimSpace(e.Data.Text("hours"))
	var dur *time.Duration
	if hoursRaw != "" && hoursRaw != "0" {
		hrs, pErr := strconv.Atoi(hoursRaw)
		if pErr != nil || hrs < 0 {
			s.replyModalError(e, errTitleInvalidInput, "期間は0以上の整数(時間数)で入力してください。")
			return nil
		}
		if hrs > 0 {
			d := time.Duration(hrs) * time.Hour
			dur = &d
		}
	}

	ban, err := s.Service.BanLocal(e.Ctx, *e.GuildID(), e.User().ID, post.PosterHash, e.Data.Text("reason"), dur)
	if err != nil {
		s.replyModalError(e, "ローカルBAN失敗", describeBBSError(err))
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("ローカルBANを登録しました", fmt.Sprintf("PosterHash: %s\nBanID: %s", ban.PosterHash, ban.ID), discordsync.ColorError))
	return nil
}
