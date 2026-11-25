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
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
)

func buildNewThreadModal(mountID uuid.UUID, boardName string) discord.ModalCreate {
	title := "新規スレッド作成"
	if boardName != "" {
		title = discordsync.TruncateRunes(fmt.Sprintf("%s / 新規スレッド", bbscore.ClearFormat(boardName)), 45)
	}
	return discord.NewModalCreate(
		fmt.Sprintf("/bbs/thread/modal/%s", mountID),
		title,
		discord.NewLabel(
			"スレッドタイトル",
			discord.NewShortTextInput("title").WithRequired(true).WithMinLength(1).WithMaxLength(bbscore.MaxTitleRunes).WithPlaceholder("例: 雑談スレ Part1"),
		),
		discord.NewLabel(
			"名前欄 (省略可 / #トリップ / !id:on / !slip:vvvvv)",
			discord.NewShortTextInput("name").WithRequired(false).WithMaxLength(bbscore.MaxNameRunes).WithPlaceholder("省略時はデフォルト名無し"),
		),
		discord.NewLabel(
			"メール欄 (省略可 / sage)",
			discord.NewShortTextInput("mail").WithRequired(false).WithMaxLength(32).WithPlaceholder("sage"),
		),
		discord.NewLabel(
			"本文 (>>1 や画像・動画URL可)",
			discord.NewParagraphTextInput("body").WithRequired(true).WithMinLength(1).WithMaxLength(bbscore.MaxBodyRunes),
		),
	)
}

func buildNewPostModal(mirrorID uuid.UUID) discord.ModalCreate {
	return discord.NewModalCreate(
		fmt.Sprintf("/bbs/post/modal/%s", mirrorID),
		"レスを書き込む",
		discord.NewLabel(
			"名前欄 (省略可 / #トリップ / !id:on / !slip:vvvvv)",
			discord.NewShortTextInput("name").WithRequired(false).WithMaxLength(bbscore.MaxNameRunes),
		),
		discord.NewLabel(
			"メール欄 (省略可 / sage)",
			discord.NewShortTextInput("mail").WithRequired(false).WithMaxLength(32).WithPlaceholder("sage"),
		),
		discord.NewLabel(
			"本文 (>>1 等アンカー・画像・動画URL可)",
			discord.NewParagraphTextInput("body").WithRequired(true).WithMinLength(1).WithMaxLength(bbscore.MaxBodyRunes),
		),
	)
}

func combineNameAndMail(rawName, rawMail string) string {
	name := strings.TrimSpace(rawName)
	mail := strings.TrimSpace(rawMail)
	if !strings.EqualFold(mail, "sage") {
		return name
	}
	if name == "" {
		return "sage"
	}
	if strings.Contains(name, "#") {
		before, after, _ := strings.Cut(name, "#")
		return strings.TrimSpace("!sage "+before) + "#" + after
	}
	return "!sage " + name
}

func (s *System) handleNewThreadButton(e *handler.ComponentEvent) error {
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "マウントIDが不正です。")
		return nil
	}
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		s.replyComponentError(e, "操作不可", "この板マウントは現在無効です。")
		return nil
	}
	if _, err := s.Service.EnsureGuildEnabled(e.Ctx, mount.GuildID); err != nil {
		s.replyComponentError(e, "BBS無効", describeBBSError(err))
		return nil
	}
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		token := s.savePending(e.User().ID, pendingNewThread, mountID, "")
		sr := discordsync.NewSafeResponder(e.ID().Time())
		_ = sr.RespondMessageCreate(e, discordsync.RenderConsentGate(s.Service.Policy, e.Locale(), token))
		return nil
	}
	boardName := ""
	if b, bErr := s.Service.GetBoard(e.Ctx, mount.BoardID); bErr == nil {
		boardName = b.Name
	}
	modal := buildNewThreadModal(mountID, boardName)
	if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err != nil {
		s.replyComponentError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (s *System) handleNewThreadModal(e *handler.ModalEvent) error {
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyModalError(e, errTitleInvalidInput, "マウントIDが不正です。")
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs new thread defer failed", "err", err)
	}

	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("スレッド作成失敗", "板マウントが見つからないか無効です。"))
		return nil
	}

	title := e.Data.Text("title")
	rawName := combineNameAndMail(e.Data.Text("name"), e.Data.Text("mail"))
	rawBody := e.Data.Text("body")

	board, thread, firstPost, err := s.Service.CreateThread(e.Ctx, bbscore.CreateThreadInput{
		BoardID: mount.BoardID,
		GuildID: mount.GuildID,
		UserID:  e.User().ID,
		Title:   title,
		RawName: rawName,
		RawBody: rawBody,
		Now:     time.Now(),
	})
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("スレッド作成失敗", describeBBSError(err)))
		return nil
	}

	threadChID, msgID, syncErr := s.Syncer.PublishThread(e.Ctx, e.Client(), mount.ID, *board, *thread, *firstPost, e.Locale())
	if syncErr != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(
			"スレッドを作成しました (Discord同期一部遅延)",
			fmt.Sprintf("スレタイ: %s\nDiscord側同期中にエラーが発生しました: %v", thread.Title, syncErr),
			discordsync.ColorNotice,
		))
		return nil
	}
	url := discord.MessageURL(mount.GuildID, threadChID, msgID)
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(
		"スレッドを作成しました",
		fmt.Sprintf("スレタイ: %s\n%s", thread.Title, url),
		discordsync.ColorLocalBoard,
	))
	return nil
}

func (s *System) handleNewPostButton(e *handler.ComponentEvent) error {
	mirrorID, err := uuid.Parse(e.Vars["mirror_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドミラーIDが不正です。")
		return nil
	}
	mirror, err := s.Service.Store.GetThreadMirror(e.Ctx, mirrorID)
	if err != nil {
		s.replyComponentError(e, "操作不可", "スレッドミラー情報が見つかりません。")
		return nil
	}
	if _, err := s.Service.EnsureGuildEnabled(e.Ctx, mirror.GuildID); err != nil {
		s.replyComponentError(e, "BBS無効", describeBBSError(err))
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, mirror.ThreadID)
	if err != nil {
		s.replyComponentError(e, "操作不可", describeBBSError(err))
		return nil
	}
	if thread.Status != bbscore.ThreadStatusActive {
		s.replyComponentError(e, "書き込み不可", "このスレッドは過去ログ倉庫に格納されています。")
		return nil
	}
	if thread.ResCount >= thread.Rules.Normalize().MaxRes {
		s.replyComponentError(e, "書き込み不可", "上限レス数に達したため書き込みできません。")
		return nil
	}
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		token := s.savePending(e.User().ID, pendingNewPost, mirrorID, "")
		sr := discordsync.NewSafeResponder(e.ID().Time())
		_ = sr.RespondMessageCreate(e, discordsync.RenderConsentGate(s.Service.Policy, e.Locale(), token))
		return nil
	}
	if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, buildNewPostModal(mirrorID)); err != nil {
		s.replyComponentError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (s *System) handleNewPostModal(e *handler.ModalEvent) error {
	mirrorID, err := uuid.Parse(e.Vars["mirror_id"])
	if err != nil {
		s.replyModalError(e, errTitleInvalidInput, "スレッドミラーIDが不正です。")
		return nil
	}
	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs new post defer failed", "err", err)
	}

	mirror, err := s.Service.Store.GetThreadMirror(e.Ctx, mirrorID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("書き込み失敗", "スレッドミラー情報が見つかりません。"))
		return nil
	}

	rawName := combineNameAndMail(e.Data.Text("name"), e.Data.Text("mail"))
	rawBody := e.Data.Text("body")

	board, thread, post, backRefs, err := s.Service.CreatePost(e.Ctx, bbscore.CreatePostInput{
		ThreadID: mirror.ThreadID,
		GuildID:  mirror.GuildID,
		UserID:   e.User().ID,
		RawName:  rawName,
		RawBody:  rawBody,
		Now:      time.Now(),
	})
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("書き込み失敗", describeBBSError(err)))
		return nil
	}

	thChID, msgID, syncErr := s.Syncer.PublishPost(e.Ctx, e.Client(), mirror.BoardMountID, *board, *thread, *post, backRefs, e.Locale())
	if syncErr != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(
			fmt.Sprintf(">>%d に書き込みました", post.ResNumber),
			fmt.Sprintf("投稿は保存されましたが、Discord反映中にエラーが発生しました: %v", syncErr),
			discordsync.ColorNotice,
		))
		return nil
	}
	url := discord.MessageURL(mirror.GuildID, thChID, msgID)
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice(
		fmt.Sprintf(">>%d に書き込みました", post.ResNumber),
		fmt.Sprintf("ID:%s\n%s", post.AnonID, url),
		discordsync.ColorLocalBoard,
	))
	return nil
}

func (s *System) handleBoardThreads(e *handler.ComponentEvent) error {
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "マウントIDが不正です。")
		return nil
	}
	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		s.replyComponentError(e, "取得失敗", "板マウントが見つからないか無効です。")
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, mount.BoardID)
	if err != nil {
		s.replyComponentError(e, "取得失敗", describeBBSError(err))
		return nil
	}
	status := bbscore.ThreadStatusActive
	threads, err := s.Service.ListThreads(e.Ctx, bbscore.ThreadFilter{
		BoardID: board.ID,
		Status:  &status,
		Limit:   25,
	})
	if err != nil {
		s.replyComponentError(e, "取得失敗", describeBBSError(err))
		return nil
	}

	subComps := []discord.ContainerSubComponent{
		discord.NewTextDisplayf("### ■ %s板 スレッド一覧 (最大25件)", bbscore.ClearFormat(board.Name)),
		discord.NewSmallSeparator(),
	}
	if len(threads) == 0 {
		subComps = append(subComps, discord.NewTextDisplay("現在アクティブなスレッドはありません。上の「スレッドを作成」から立てることができます。"))
	} else {
		var lines []string
		var options []discord.StringSelectMenuOption
		for i, th := range threads {
			cleanTitle := discordsync.TruncateRunes(bbscore.ClearFormat(th.Title), 65)
			lines = append(lines, fmt.Sprintf("%d. %s (%dレス)", i+1, cleanTitle, th.ResCount))
			options = append(options, discord.NewStringSelectMenuOption(
				discordsync.TruncateRunes(cleanTitle, 90),
				th.ID.String(),
			).WithDescription(fmt.Sprintf("%dレス", th.ResCount)))
		}
		subComps = append(subComps, discord.NewTextDisplay(discordsync.TruncateRunes(strings.Join(lines, "\n"), 1800)))
		if len(options) > 0 {
			subComps = append(subComps, discord.NewActionRow(
				discord.NewStringSelectMenu(fmt.Sprintf("/bbs/board/jump/%s", mount.ID), "スレッドを選択して開く / 展開する", options...),
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
	mountID, err := uuid.Parse(e.Vars["mount_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "マウントIDが不正です。")
		return nil
	}
	vals := e.StringSelectMenuInteractionData().Values
	if len(vals) == 0 {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドが選択されていません。")
		return nil
	}
	threadID, err := uuid.Parse(vals[0])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if err := sr.DeferCreate(e, true); err != nil {
		slog.Warn("bbs board jump defer failed", "err", err)
	}

	mount, err := s.Service.Store.GetBoardMount(e.Ctx, mountID)
	if err != nil || !mount.Active {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("操作不可", "板マウントが見つかりません。"))
		return nil
	}
	board, err := s.Service.GetBoard(e.Ctx, mount.BoardID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("取得失敗", describeBBSError(err)))
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, threadID)
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("取得失敗", describeBBSError(err)))
		return nil
	}

	mirror, err := s.Syncer.EnsureThreadMirror(e.Ctx, e.Client(), *mount, *board, *thread, 0, e.Locale())
	if err != nil {
		_ = sr.RespondMessageCreate(e, discordsync.RenderError("スレッド展開失敗", describeBBSError(err)))
		return nil
	}

	msg := discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplayf("### ■ %s", discordsync.TruncateRunes(bbscore.ClearFormat(thread.Title), 80)),
			discord.NewSmallSeparator(),
			discord.NewTextDisplayf("スレッドチャンネル: <#%d> (全%dレス)", mirror.DiscordThreadID, thread.ResCount),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewSecondaryButton("過去レスを読む", fmt.Sprintf("/bbs/reader/%s/1", thread.ID)),
			),
		).WithAccentColor(discordsync.ColorLocalBoard),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})

	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleWatchToggle(e *handler.ComponentEvent) error {
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, threadID)
	if err != nil {
		s.replyComponentError(e, "取得失敗", describeBBSError(err))
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if existing, gErr := s.Service.Store.GetThreadWatch(e.Ctx, threadID, e.User().ID); gErr == nil && existing != nil {
		if err := s.Service.Store.DeleteThreadWatch(e.Ctx, threadID, e.User().ID); err != nil {
			s.replyComponentError(e, "購読解除失敗", describeBBSError(err))
			return nil
		}
		_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("スレッド購読を解除しました", fmt.Sprintf("対象: %s", thread.Title), discordsync.ColorNotice))
		return nil
	}

	if err := s.Service.Store.SaveThreadWatch(e.Ctx, &bbscore.ThreadWatch{
		ThreadID:  threadID,
		UserID:    e.User().ID,
		CreatedAt: time.Now(),
	}); err != nil {
		s.replyComponentError(e, "購読登録失敗", describeBBSError(err))
		return nil
	}
	_ = sr.RespondMessageCreate(e, discordsync.RenderNotice("スレッドを購読しました", fmt.Sprintf("対象: %s\n新着レスが付いた際にDMでお知らせします (3分間隔制限)。", thread.Title), discordsync.ColorLocalBoard))
	return nil
}

func (s *System) handleReaderPage(e *handler.ComponentEvent) error {
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}
	pageSpec := e.Vars["page"]
	if _, err := s.Service.CheckConsent(e.Ctx, e.User().ID); err != nil {
		token := s.savePending(e.User().ID, pendingOpenReader, threadID, pageSpec)
		sr := discordsync.NewSafeResponder(e.ID().Time())
		_ = sr.RespondMessageCreate(e, discordsync.RenderConsentGate(s.Service.Policy, e.Locale(), token))
		return nil
	}

	comps, err := s.renderReaderComponents(e.Ctx, e.GuildID(), threadID, pageSpec)
	if err != nil {
		s.replyComponentError(e, "表示失敗", describeBBSError(err))
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	if e.Message.Flags.Has(discord.MessageFlagEphemeral) {
		upd := discord.NewMessageUpdateV2(comps...).WithAllowedMentions(&discord.AllowedMentions{})
		_ = sr.UpdateComponentMessage(e, upd)
		return nil
	}
	msg := discord.NewMessageCreateV2(comps...).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})
	_ = sr.RespondMessageCreate(e, msg)
	return nil
}

func (s *System) handleReaderJumpButton(e *handler.ComponentEvent) error {
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyComponentError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}
	modal := discord.NewModalCreate(
		fmt.Sprintf("/bbs/reader/jumpmodal/%s", threadID),
		"レス番指定ジャンプ",
		discord.NewLabel(
			"レス番号 または lN (最新N件)",
			discord.NewShortTextInput("target").WithRequired(true).WithMaxLength(16).WithPlaceholder("例: 85 または l10"),
		),
	)
	if err := discordsync.SafeRespondComponentModal(e.ComponentInteractionCreate, modal); err != nil {
		s.replyComponentError(e, "モーダル表示失敗", err.Error())
	}
	return nil
}

func (s *System) handleReaderJumpModal(e *handler.ModalEvent) error {
	threadID, err := uuid.Parse(e.Vars["thread_id"])
	if err != nil {
		s.replyModalError(e, errTitleInvalidInput, "スレッドIDが不正です。")
		return nil
	}
	thread, err := s.Service.GetThread(e.Ctx, threadID)
	if err != nil {
		s.replyModalError(e, "表示失敗", describeBBSError(err))
		return nil
	}
	target := discordsync.ParseReaderJumpInput(e.Data.Text("target"), thread.ResCount)
	comps, err := s.renderReaderComponents(e.Ctx, e.GuildID(), threadID, strconv.FormatUint(uint64(target), 10))
	if err != nil {
		s.replyModalError(e, "表示失敗", describeBBSError(err))
		return nil
	}

	sr := discordsync.NewSafeResponder(e.ID().Time())
	upd := discord.NewMessageUpdateV2(comps...).WithAllowedMentions(&discord.AllowedMentions{})
	if uErr := sr.UpdateComponentMessage(e, upd); uErr != nil {
		msg := discord.NewMessageCreateV2(comps...).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})
		_ = sr.RespondMessageCreate(e, msg)
	}
	return nil
}

func (s *System) buildReaderMessageUpdate(ctx context.Context, guildID *snowflake.ID, threadID uuid.UUID, pageSpec string) (discord.MessageUpdate, error) {
	comps, err := s.renderReaderComponents(ctx, guildID, threadID, pageSpec)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return discord.NewMessageUpdateV2(comps...).WithAllowedMentions(&discord.AllowedMentions{}), nil
}

func (s *System) renderReaderComponents(ctx context.Context, guildID *snowflake.ID, threadID uuid.UUID, pageSpec string) ([]discord.LayoutComponent, error) {
	thread, err := s.Service.GetThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	posts, err := s.Service.ListPosts(ctx, threadID, 0, int(max(thread.ResCount, 1000)))
	if err != nil {
		return nil, err
	}

	startIndex := 0
	if pageSpec == "latest" {
		startIndex = discordsync.ComputeLatestReaderStartIndex(*thread, posts)
	} else {
		n, pErr := strconv.ParseUint(strings.TrimSpace(pageSpec), 10, 32)
		if pErr == nil && n > 1 {
			targetRes := uint32(n)
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

	var resolveURL func(resNum uint32) (string, bool)
	if guildID != nil && len(posts) > 0 {
		if pm, pErr := s.Service.Store.GetPostMirrorByPostAndGuild(ctx, posts[0].ID, *guildID); pErr == nil {
			if tm, tErr := s.Service.Store.GetThreadMirror(ctx, pm.ThreadMirrorID); tErr == nil {
				gid := *guildID
				tid := tm.DiscordThreadID
				resolveURL = func(resNum uint32) (string, bool) {
					m, mErr := s.Service.Store.GetPostMirrorByResAndGuild(ctx, threadID, resNum, gid)
					if mErr != nil || m.DiscordMessageID == 0 {
						return "", false
					}
					return discord.MessageURL(gid, tid, m.DiscordMessageID), true
				}
			}
		}
	}

	comps, _, _ := discordsync.RenderReaderPage(*thread, posts, startIndex, resolveURL)
	return comps, nil
}
