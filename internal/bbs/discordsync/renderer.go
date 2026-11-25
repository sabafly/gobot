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

package discordsync

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

const (
	ColorLocalBoard  = 0x2F5496
	ColorGlobalBoard = 0x3B7A57
	ColorNotice      = 0x4A5568
	ColorError       = 0xB83227
)

var weekdayJP = [...]string{"日", "月", "火", "水", "木", "金", "土"}

// FormatTimestamp5ch formats time in classic 5ch JST style: 2026/11/25(水) 22:40:00.000
func FormatTimestamp5ch(t time.Time) string {
	jst := t.In(bbs.JSTLocation)
	wd := weekdayJP[jst.Weekday()]
	return fmt.Sprintf("%s(%s) %s.%03d",
		jst.Format("2006/01/02"),
		wd,
		jst.Format("15:04:05"),
		jst.Nanosecond()/1e6,
	)
}

// RenderNotice builds a pure text-based Components V2 notice container (no emojis).
func RenderNotice(title string, body string, color int) discord.MessageCreate {
	if color == 0 {
		color = ColorNotice
	}
	titleText := TruncateRunes(strings.TrimSpace(title), 200)
	if titleText == "" {
		titleText = "お知らせ"
	}
	bodyText := TruncateRunes(strings.TrimSpace(body), 3400)
	var comps []discord.ContainerSubComponent
	comps = append(comps, discord.NewTextDisplayf("### ■ %s", titleText))
	if bodyText != "" {
		comps = append(comps, discord.NewSmallSeparator(), discord.NewTextDisplay(bodyText))
	}
	return discord.NewMessageCreateV2(discord.NewContainer(comps...).WithAccentColor(color))
}

// RenderError builds a pure text-based Components V2 ephemeral error message (no emojis).
func RenderError(title string, body string) discord.MessageCreate {
	mc := RenderNotice(title, body, ColorError)
	return mc.WithEphemeral(true)
}

// RenderBoardGuide creates the pinned board panel / guide post in a board channel.
func RenderBoardGuide(board bbs.Board, mountID uuid.UUID, locale discord.Locale) discord.MessageCreate {
	isEN := strings.HasPrefix(strings.ToLower(string(locale)), "en")

	scopeBadge := "[ローカル板]"
	color := ColorLocalBoard
	if board.Scope == bbs.ScopeGlobal {
		scopeBadge = "[グローバル板]"
		color = ColorGlobalBoard
		if isEN {
			scopeBadge = "[Global Board]"
		}
	} else if isEN {
		scopeBadge = "[Local Board]"
	}

	idMode := "ID表示"
	if board.Rules.HideID {
		idMode = "ID非表示"
	}
	slipMode := "SLIPなし"
	if board.Rules.SLIP == bbs.SLIPVVVVV {
		slipMode = "ﾜｯﾁｮｲ有効"
	}
	if isEN {
		idMode = "ID: Shown"
		if board.Rules.HideID {
			idMode = "ID: Hidden"
		}
		slipMode = "SLIP: Off"
		if board.Rules.SLIP == bbs.SLIPVVVVV {
			slipMode = "SLIP: On"
		}
	}

	desc := strings.TrimSpace(board.Description)
	if desc == "" {
		if isEN {
			desc = "Use the buttons below to create a new thread or browse active threads."
		} else {
			desc = "下のボタンから新規スレッド作成またはスレッド一覧の確認が行えます。"
		}
	}

	createBtnLabel := "スレッドを作成"
	listBtnLabel := "スレッド一覧"
	policyBtnLabel := "利用規約・ポリシー"
	ruleLine := fmt.Sprintf("名無し: `%s` | %s | %s | 上限: %dレス",
		bbs.ClearFormat(board.Rules.Normalize().DefaultName), idMode, slipMode, board.Rules.Normalize().MaxRes)
	if isEN {
		createBtnLabel = "New Thread"
		listBtnLabel = "Thread List"
		policyBtnLabel = "Terms & Policy"
		ruleLine = fmt.Sprintf("Default Name: `%s` | %s | %s | Max: %d",
			bbs.ClearFormat(board.Rules.Normalize().DefaultName), idMode, slipMode, board.Rules.Normalize().MaxRes)
	}

	container := discord.NewContainer(
		discord.NewTextDisplayf("## %s %s", scopeBadge, bbs.ClearFormat(board.Name)),
		discord.NewTextDisplay(TruncateRunes(bbs.ClearFormat(desc), 1500)),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(ruleLine),
		discord.NewLargeSeparator(),
		discord.NewActionRow(
			discord.NewPrimaryButton(createBtnLabel, fmt.Sprintf("/bbs/thread/new/%s", mountID)),
			discord.NewSecondaryButton(listBtnLabel, fmt.Sprintf("/bbs/board/threads/%s", mountID)),
			discord.NewSecondaryButton(policyBtnLabel, "/bbs/consent/view/terms"),
		),
		discord.NewTextDisplayf("-# board:%s mount:%s", board.ID, mountID),
	).WithAccentColor(color)

	return discord.NewMessageCreateV2(container).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderPostHeader formats the 5ch-style header line for a single post.
func RenderPostHeader(post bbs.Post, threadRules bbs.BoardRules) string {
	if post.Abone {
		return fmt.Sprintf("%d :あぼーん:あぼーん ID:あぼーん", post.ResNumber)
	}
	name := bbs.ClearFormat(post.Name)
	if name == "" {
		name = bbs.ClearFormat(threadRules.Normalize().DefaultName)
	}
	var namePart strings.Builder
	namePart.WriteString(name)
	if post.Trip != "" {
		namePart.WriteByte(' ')
		namePart.WriteString(bbs.ClearFormat(post.Trip))
	}
	if post.SlipText != "" {
		namePart.WriteString(" (ﾜｯﾁｮｲ ")
		namePart.WriteString(bbs.ClearFormat(post.SlipText))
		namePart.WriteByte(')')
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d :%s:%s", post.ResNumber, namePart.String(), FormatTimestamp5ch(post.CreatedAt))
	if post.AnonID != "" {
		fmt.Fprintf(&b, " ID:%s", post.AnonID)
		if post.IsOP {
			b.WriteString(" (>>1)")
		}
	}
	return b.String()
}

// RenderPostComponents constructs the Components V2 layout for a single post message.
func RenderPostComponents(
	post bbs.Post,
	thread bbs.Thread,
	resolveURL func(resNum uint32) (string, bool),
	includeExtDat bool,
) []discord.LayoutComponent {
	header := RenderPostHeader(post, thread.Rules)
	if post.Abone {
		return []discord.LayoutComponent{
			discord.NewTextDisplay(header),
			discord.NewTextDisplay("あぼーん"),
			discord.NewLargeSeparator(),
		}
	}

	maxRes := thread.Rules.Normalize().MaxRes
	formattedBody := bbs.RenderFormattedBody(post.Body, maxRes, resolveURL)
	maxBodyAllowed := SafeTextDisplayRunesTotal - utf8.RuneCountInString(header) - 120
	if maxBodyAllowed < 200 {
		maxBodyAllowed = 200
	}
	formattedBody = TruncateRunes(formattedBody, maxBodyAllowed)

	comps := []discord.LayoutComponent{
		discord.NewTextDisplay(header),
		discord.NewTextDisplay(formattedBody),
	}

	mediaURLs := bbs.ExtractMediaURLs(post.Body)
	if len(mediaURLs) > 0 {
		items := make([]discord.MediaGalleryItem, 0, len(mediaURLs))
		for _, u := range mediaURLs {
			items = append(items, discord.MediaGalleryItem{
				Media: discord.UnfurledMediaItem{URL: u},
			})
		}
		comps = append(comps, discord.NewMediaGallery(items...))
	}

	comps = append(comps, discord.NewLargeSeparator())
	if includeExtDat && post.ResNumber == 1 && thread.ExtDat != "" {
		comps = append(comps, discord.NewTextDisplay(thread.ExtDat))
	}
	return comps
}

// RenderPostMessageCreate wraps post components in a validated MessageCreate.
func RenderPostMessageCreate(
	post bbs.Post,
	thread bbs.Thread,
	resolveURL func(resNum uint32) (string, bool),
) discord.MessageCreate {
	comps := RenderPostComponents(post, thread, resolveURL, post.ResNumber == 1)
	return discord.NewMessageCreateV2(comps...).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderPostMessageUpdate builds a MessageUpdate for anchor back-patching or abone updates.
func RenderPostMessageUpdate(
	post bbs.Post,
	thread bbs.Thread,
	resolveURL func(resNum uint32) (string, bool),
) discord.MessageUpdate {
	comps := RenderPostComponents(post, thread, resolveURL, post.ResNumber == 1)
	flags := discord.MessageFlagIsComponentsV2
	return discord.MessageUpdate{
		Flags:           &flags,
		Components:      &comps,
		AllowedMentions: &discord.AllowedMentions{},
	}
}

// RenderLazyCatchupBanner builds the banner inserted after >>1 when mirroring a pre-existing global thread.
func RenderLazyCatchupBanner(threadID uuid.UUID, omittedFrom, omittedTo uint32) discord.MessageCreate {
	return discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplayf("--- %d〜%dレス目は購読前の書き込みです ---", omittedFrom, omittedTo),
			discord.NewActionRow(
				discord.NewSecondaryButton(
					TruncateRunes(fmt.Sprintf("過去レスを読む (%d-%d)", omittedFrom, omittedTo), MaxButtonLabelRunes),
					fmt.Sprintf("/bbs/reader/%s/%d", threadID, omittedFrom),
				),
			),
		).WithAccentColor(ColorNotice),
	).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderThreadControlPanel creates the floating action panel placed at the bottom of active threads.
func RenderThreadControlPanel(mirrorID uuid.UUID, thread bbs.Thread) discord.MessageCreate {
	container := discord.NewContainer(
		discord.NewTextDisplayf("スレッド操作 (%s / %dレス)", TruncateRunes(bbs.ClearFormat(thread.Title), 60), thread.ResCount),
		discord.NewSmallSeparator(),
		discord.NewActionRow(
			discord.NewPrimaryButton("書き込み", fmt.Sprintf("/bbs/post/new/%s", mirrorID)),
			discord.NewSecondaryButton("購読", fmt.Sprintf("/bbs/watch/toggle/%s", thread.ID)),
			discord.NewSecondaryButton("過去レスを読む", fmt.Sprintf("/bbs/reader/%s/1", thread.ID)),
			discord.NewDangerButton("通報", fmt.Sprintf("/bbs/report/thread/%s", thread.ID)),
		),
	)
	return discord.NewMessageCreateV2(container).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderOver1000Message builds the classic 1001 Over 1000 Thread notice.
func RenderOver1000Message(thread bbs.Thread) discord.MessageCreate {
	life := time.Since(thread.CreatedAt).Round(time.Second)
	lifeStr := fmt.Sprintf("%d日%d時間%d分", int(life.Hours())/24, int(life.Hours())%24, int(life.Minutes())%60)
	return discord.NewMessageCreateV2(
		discord.NewTextDisplay("1001 :1001:Over 1000 Thread"),
		discord.NewTextDisplayf("このスレッドは%dを超えました。\n新しいスレッドを立ててください。\nlife time: %s", thread.Rules.Normalize().MaxRes, lifeStr),
	).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderArchivedNotice builds the archive warehouse message when a thread locks.
func RenderArchivedNotice() discord.MessageCreate {
	return discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay("### ■ このスレッドは過去ログ倉庫に格納されています"),
		).WithAccentColor(ColorNotice),
	).WithAllowedMentions(&discord.AllowedMentions{})
}

// ParseReaderJumpInput parses user input from the jump modal ("85" or "l10") into a 1-based start res number.
func ParseReaderJumpInput(input string, totalRes uint32) uint32 {
	if totalRes == 0 {
		return 1
	}
	s := strings.ToLower(strings.TrimSpace(input))
	if strings.HasPrefix(s, "l") {
		n, err := strconv.ParseUint(strings.TrimPrefix(s, "l"), 10, 32)
		if err != nil || n == 0 {
			return 1
		}
		count := uint32(n)
		if count >= totalRes {
			return 1
		}
		return totalRes - count + 1
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil || n == 0 {
		return 1
	}
	res := uint32(n)
	if res > totalRes {
		return totalRes
	}
	return res
}

// RenderReaderPage packs posts dynamically into a single ephemeral read.cgi view
// strictly bounded by SafeReaderTextDisplayRunes (3500) and SafeReaderMaxComponents (32).
func RenderReaderPage(
	thread bbs.Thread,
	posts []bbs.Post,
	startIndex int,
	resolveURL func(resNum uint32) (string, bool),
) (components []discord.LayoutComponent, startRes uint32, endRes uint32) {
	if len(posts) == 0 {
		c := discord.NewContainer(
			discord.NewTextDisplayf("### ■ %s", TruncateRunes(bbs.ClearFormat(thread.Title), 80)),
			discord.NewTextDisplay("レスがありません。"),
		)
		return []discord.LayoutComponent{c}, 1, 1
	}
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex >= len(posts) {
		startIndex = len(posts) - 1
	}

	titleLine := fmt.Sprintf("### ■ %s (全%dレス)", TruncateRunes(bbs.ClearFormat(thread.Title), 80), thread.ResCount)
	usedRunes := utf8.RuneCountInString(titleLine)
	usedComponents := 2 // outer container + titleLine

	subComps := []discord.ContainerSubComponent{
		discord.NewTextDisplay(titleLine),
		discord.NewSmallSeparator(),
	}
	usedComponents++ // separator

	startRes = posts[startIndex].ResNumber
	endRes = startRes
	shownCount := 0
	maxRes := thread.Rules.Normalize().MaxRes

	for i := startIndex; i < len(posts); i++ {
		p := posts[i]
		header := RenderPostHeader(p, thread.Rules)
		body := "あぼーん"
		var mediaURLs []string
		if !p.Abone {
			body = bbs.RenderFormattedBody(p.Body, maxRes, resolveURL)
			mediaURLs = bbs.ExtractMediaURLs(p.Body)
		}

		if shownCount == 0 {
			// Ensure at least the first post fits even if long.
			budget := SafeReaderTextDisplayRunes - usedRunes - utf8.RuneCountInString(header) - 40
			if budget < 200 {
				budget = 200
			}
			body = TruncateRunes(body, budget)
		}

		postRunes := utf8.RuneCountInString(header) + utf8.RuneCountInString(body)
		postCompCount := 3 // separator + header + body
		if len(mediaURLs) > 0 && shownCount == 0 {
			// Reserve 1 slot for MediaGallery when space permits; omit extra galleries on overflow.
			if usedComponents+postCompCount+1 <= SafeReaderMaxComponents {
				postCompCount++
			} else {
				mediaURLs = nil
			}
		} else if len(mediaURLs) > 0 {
			if usedComponents+postCompCount+1 <= SafeReaderMaxComponents {
				postCompCount++
			} else {
				mediaURLs = nil
			}
		}

		if shownCount > 0 && (usedRunes+postRunes > SafeReaderTextDisplayRunes || usedComponents+postCompCount > SafeReaderMaxComponents) {
			break
		}

		if shownCount > 0 {
			subComps = append(subComps, discord.NewSmallSeparator())
			usedComponents++
		}
		subComps = append(subComps, discord.NewTextDisplay(header), discord.NewTextDisplay(body))
		usedRunes += postRunes
		usedComponents += 2

		if len(mediaURLs) > 0 {
			items := make([]discord.MediaGalleryItem, 0, len(mediaURLs))
			for _, u := range mediaURLs {
				items = append(items, discord.MediaGalleryItem{Media: discord.UnfurledMediaItem{URL: u}})
			}
			subComps = append(subComps, discord.NewMediaGallery(items...))
			usedComponents++
		}

		endRes = p.ResNumber
		shownCount++
	}

	prevRes := uint32(1)
	if startRes > uint32(max(shownCount, 1)) {
		prevRes = startRes - uint32(shownCount)
	}
	nextRes := endRes + 1
	if nextRes > thread.ResCount {
		nextRes = thread.ResCount
	}

	atStart := startRes <= 1
	atEnd := endRes >= thread.ResCount

	firstBtn := discord.NewSecondaryButton("先頭", fmt.Sprintf("/bbs/reader/%s/1", thread.ID))
	firstBtn.Disabled = atStart
	prevBtn := discord.NewSecondaryButton("前へ", fmt.Sprintf("/bbs/reader/%s/%d", thread.ID, prevRes))
	prevBtn.Disabled = atStart
	nextBtn := discord.NewSecondaryButton("次へ", fmt.Sprintf("/bbs/reader/%s/%d", thread.ID, nextRes))
	nextBtn.Disabled = atEnd
	latestBtn := discord.NewSecondaryButton("最新", fmt.Sprintf("/bbs/reader/%s/latest", thread.ID))
	latestBtn.Disabled = atEnd
	jumpBtn := discord.NewPrimaryButton("レス番指定", fmt.Sprintf("/bbs/reader/jump/%s", thread.ID))

	navRow := discord.NewActionRow(firstBtn, prevBtn, nextBtn, latestBtn, jumpBtn)
	container := discord.NewContainer(subComps...).WithAccentColor(ColorNotice)
	return []discord.LayoutComponent{container, navRow}, startRes, endRes
}

// ComputeLatestReaderStartIndex determines the starting post index whose packed page reaches the end of posts.
func ComputeLatestReaderStartIndex(thread bbs.Thread, posts []bbs.Post) int {
	if len(posts) <= 1 {
		return 0
	}
	// Walk backwards accumulating budget to find earliest start index that includes the final post.
	titleLine := fmt.Sprintf("### ■ %s (全%dレス)", TruncateRunes(bbs.ClearFormat(thread.Title), 80), thread.ResCount)
	usedRunes := utf8.RuneCountInString(titleLine)
	usedComponents := 3
	maxRes := thread.Rules.Normalize().MaxRes

	startIdx := len(posts) - 1
	for i := len(posts) - 1; i >= 0; i-- {
		p := posts[i]
		header := RenderPostHeader(p, thread.Rules)
		body := "あぼーん"
		if !p.Abone {
			body = bbs.RenderFormattedBody(p.Body, maxRes, nil)
		}
		postRunes := utf8.RuneCountInString(header) + utf8.RuneCountInString(body)
		postComps := 3
		if !p.Abone && len(bbs.ExtractMediaURLs(p.Body)) > 0 {
			postComps++
		}
		if i < len(posts)-1 && (usedRunes+postRunes > SafeReaderTextDisplayRunes || usedComponents+postComps > SafeReaderMaxComponents) {
			break
		}
		usedRunes += postRunes
		usedComponents += postComps
		startIdx = i
	}
	return startIdx
}

// RenderConsentGate builds the ephemeral policy consent UI (zero emojis).
func RenderConsentGate(pm *bbs.PolicyManager, locale discord.Locale, actionToken string) discord.MessageCreate {
	lang := string(locale)
	isEN := strings.HasPrefix(strings.ToLower(lang), "en")

	title := "### ■ 利用規約・ガイドライン・プライバシーポリシーへの同意"
	acceptLabel := "すべてに同意して続行"
	termsLabel := "利用規約"
	guideLabel := "ガイドライン"
	privacyLabel := "プライバシーポリシー"
	if isEN {
		title = "### ■ Policy Agreement Required"
		acceptLabel = "Agree and Continue"
		termsLabel = "Terms"
		guideLabel = "Guidelines"
		privacyLabel = "Privacy Policy"
	}

	if actionToken == "" {
		actionToken = "none"
	}

	docRow := discord.NewActionRow(
		discord.NewSecondaryButton(termsLabel, "/bbs/consent/view/terms"),
		discord.NewSecondaryButton(guideLabel, "/bbs/consent/view/guidelines"),
		discord.NewSecondaryButton(privacyLabel, "/bbs/consent/view/privacy"),
	)

	var urlButtons []discord.InteractiveComponent
	if pm.TermsURL != "" {
		urlButtons = append(urlButtons, discord.NewLinkButton(termsLabel+" (Web)", pm.TermsURL))
	}
	if pm.GuidelinesURL != "" {
		urlButtons = append(urlButtons, discord.NewLinkButton(guideLabel+" (Web)", pm.GuidelinesURL))
	}
	if pm.PrivacyPolicyURL != "" {
		urlButtons = append(urlButtons, discord.NewLinkButton(privacyLabel+" (Web)", pm.PrivacyPolicyURL))
	}

	subComps := []discord.ContainerSubComponent{
		discord.NewTextDisplay(title),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(pm.GetSummary(lang)),
		discord.NewTextDisplayf("-# policy_version: %s", pm.CurrentVersion),
		discord.NewLargeSeparator(),
		docRow,
	}
	if len(urlButtons) > 0 {
		subComps = append(subComps, discord.NewActionRow(urlButtons...))
	}
	subComps = append(subComps, discord.NewActionRow(
		discord.NewSuccessButton(acceptLabel, fmt.Sprintf("/bbs/consent/accept/%s", actionToken)),
	))

	return discord.NewMessageCreateV2(
		discord.NewContainer(subComps...).WithAccentColor(ColorLocalBoard),
	).WithEphemeral(true).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderRoleGatePanel builds the public guild role-gate panel (placed by admins in /bbs setup).
func RenderRoleGatePanel(guildID snowflake.ID, roleID snowflake.ID, version string) discord.MessageCreate {
	return discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay("### ■ 匿名掲示板（BBS）閲覧ロール取得"),
			discord.NewSmallSeparator(),
			discord.NewTextDisplayf("このサーバーの匿名掲示板チャンネルを閲覧・利用するには、利用規約・ガイドライン・プライバシーポリシー（version: %s）への同意が必要です。", version),
			discord.NewTextDisplayf("下のボタンから同意すると <@&%d> が付与されます。", roleID),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewPrimaryButton("同意して閲覧ロールを取得", fmt.Sprintf("/bbs/consent/role/%d", guildID)),
				discord.NewSecondaryButton("利用規約を確認", "/bbs/consent/view/terms"),
				discord.NewSecondaryButton("ガイドライン", "/bbs/consent/view/guidelines"),
				discord.NewSecondaryButton("プライバシーポリシー", "/bbs/consent/view/privacy"),
			),
		).WithAccentColor(ColorLocalBoard),
	).WithAllowedMentions(&discord.AllowedMentions{})
}
