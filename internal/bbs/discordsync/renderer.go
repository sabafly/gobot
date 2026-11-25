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
	"github.com/sabafly/gobot/internal/i18n"
)

const (
	ColorLocalBoard  = 0x2F5496
	ColorGlobalBoard = 0x3B7A57
	ColorNotice      = 0x4A5568
	ColorError       = 0xB83227
)

// FormatTimestamp5ch formats time in classic 5ch JST style using localized weekday labels.
func FormatTimestamp5ch(t time.Time, locale discord.Locale) string {
	jst := t.In(bbs.JSTLocation)
	var wdKey string
	switch jst.Weekday() {
	case time.Sunday:
		wdKey = "components.bbs.post.weekday.0"
	case time.Monday:
		wdKey = "components.bbs.post.weekday.1"
	case time.Tuesday:
		wdKey = "components.bbs.post.weekday.2"
	case time.Wednesday:
		wdKey = "components.bbs.post.weekday.3"
	case time.Thursday:
		wdKey = "components.bbs.post.weekday.4"
	case time.Friday:
		wdKey = "components.bbs.post.weekday.5"
	case time.Saturday:
		wdKey = "components.bbs.post.weekday.6"
	default:
		wdKey = "components.bbs.post.weekday.0"
	}
	wd := i18n.TranslateText(locale, wdKey)
	return fmt.Sprintf("%s(%s) %s.%03d",
		jst.Format("2006/01/02"),
		wd,
		jst.Format("15:04:05"),
		jst.Nanosecond()/1e6,
	)
}

func resolveAnonymousDisplayName(raw string, locale discord.Locale) string {
	clean := bbs.ClearFormat(strings.TrimSpace(raw))
	if clean == "" || strings.TrimSpace(raw) == bbs.DefaultAnonymousName {
		return i18n.TranslateText(locale, "components.bbs.post.default_name")
	}
	return clean
}

// RenderNotice builds a pure text-based Components V2 notice container (no emojis).
func RenderNotice(locale discord.Locale, title string, body string, color int) discord.MessageCreate {
	if color == 0 {
		color = ColorNotice
	}
	titleText := TruncateRunes(strings.TrimSpace(title), 200)
	if titleText == "" {
		titleText = i18n.TranslateText(locale, "components.bbs.common.notice_title")
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
func RenderError(locale discord.Locale, title string, body string) discord.MessageCreate {
	mc := RenderNotice(locale, title, body, ColorError)
	return mc.WithEphemeral(true)
}

// RenderBoardGuide creates the pinned board panel / guide post in a board channel.
func RenderBoardGuide(board bbs.Board, mountID uuid.UUID, locale discord.Locale) discord.MessageCreate {
	scopeKey := "components.bbs.board_guide.scope.local"
	color := ColorLocalBoard
	if board.Scope == bbs.ScopeGlobal {
		scopeKey = "components.bbs.board_guide.scope.global"
		color = ColorGlobalBoard
	}
	scopeBadge := i18n.TranslateText(locale, scopeKey)

	idKey := "components.bbs.board_guide.id.shown"
	if board.Rules.HideID {
		idKey = "components.bbs.board_guide.id.hidden"
	}
	idMode := i18n.TranslateText(locale, idKey)

	slipKey := "components.bbs.board_guide.slip.off"
	if board.Rules.SLIP == bbs.SLIPVVVVV {
		slipKey = "components.bbs.board_guide.slip.on"
	}
	slipMode := i18n.TranslateText(locale, slipKey)

	desc := strings.TrimSpace(board.Description)
	if desc == "" {
		desc = i18n.TranslateText(locale, "components.bbs.board_guide.default_desc")
	} else {
		desc = bbs.ClearFormat(desc)
	}

	rules := board.Rules.Normalize()
	defName := resolveAnonymousDisplayName(rules.DefaultName, locale)
	createBtnLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.board_guide.btn.new_thread"), MaxButtonLabelRunes)
	listBtnLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.board_guide.btn.list"), MaxButtonLabelRunes)
	policyBtnLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.board_guide.btn.policy"), MaxButtonLabelRunes)
	ruleLine := i18n.TranslateText(locale, "components.bbs.board_guide.rules", map[string]any{
		"default_name": defName,
		"id_mode":      idMode,
		"slip_mode":    slipMode,
		"max_res":      rules.MaxRes,
	})

	container := discord.NewContainer(
		discord.NewTextDisplayf("## %s %s", scopeBadge, bbs.ClearFormat(board.Name)),
		discord.NewTextDisplay(TruncateRunes(desc, 1500)),
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
func RenderPostHeader(post bbs.Post, threadRules bbs.BoardRules, locale discord.Locale) string {
	if post.Abone {
		return i18n.TranslateText(locale, "components.bbs.post.abone_header", map[string]any{
			"res": post.ResNumber,
		})
	}
	name := resolveAnonymousDisplayName(post.Name, locale)
	if strings.TrimSpace(post.Name) == "" {
		name = resolveAnonymousDisplayName(threadRules.Normalize().DefaultName, locale)
	}
	var namePart strings.Builder
	namePart.WriteString(name)
	if post.Trip != "" {
		namePart.WriteByte(' ')
		namePart.WriteString(bbs.ClearFormat(post.Trip))
	}
	if post.SlipText != "" {
		slipPrefix := i18n.TranslateText(locale, "components.bbs.post.slip_prefix")
		namePart.WriteString(" (")
		namePart.WriteString(slipPrefix)
		namePart.WriteByte(' ')
		namePart.WriteString(bbs.ClearFormat(post.SlipText))
		namePart.WriteByte(')')
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d :%s:%s", post.ResNumber, namePart.String(), FormatTimestamp5ch(post.CreatedAt, locale))
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
	locale discord.Locale,
) []discord.LayoutComponent {
	header := RenderPostHeader(post, thread.Rules, locale)
	if post.Abone {
		return []discord.LayoutComponent{
			discord.NewTextDisplay(header),
			discord.NewTextDisplay(i18n.TranslateText(locale, "components.bbs.post.abone_body")),
			discord.NewLargeSeparator(),
		}
	}

	header = TruncateRunes(header, 400)
	var extDat string
	if includeExtDat && post.ResNumber == 1 && thread.ExtDat != "" {
		extDat = TruncateRunes(thread.ExtDat, 200)
	}

	maxRes := thread.Rules.Normalize().MaxRes
	formattedBody := bbs.RenderFormattedBody(post.Body, maxRes, resolveURL)
	maxBodyAllowed := SafeTextDisplayRunesTotal - utf8.RuneCountInString(header) - utf8.RuneCountInString(extDat) - 40
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
	if extDat != "" {
		comps = append(comps, discord.NewTextDisplay(extDat))
	}
	return comps
}

// RenderPostMessageCreate wraps post components in a validated MessageCreate.
func RenderPostMessageCreate(
	post bbs.Post,
	thread bbs.Thread,
	resolveURL func(resNum uint32) (string, bool),
	locale discord.Locale,
) discord.MessageCreate {
	comps := RenderPostComponents(post, thread, resolveURL, post.ResNumber == 1, locale)
	return discord.NewMessageCreateV2(comps...).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderPostMessageUpdate builds a MessageUpdate for anchor back-patching or abone updates.
func RenderPostMessageUpdate(
	post bbs.Post,
	thread bbs.Thread,
	resolveURL func(resNum uint32) (string, bool),
	locale discord.Locale,
) discord.MessageUpdate {
	comps := RenderPostComponents(post, thread, resolveURL, post.ResNumber == 1, locale)
	flags := discord.MessageFlagIsComponentsV2
	return discord.MessageUpdate{
		Flags:           &flags,
		Components:      &comps,
		AllowedMentions: &discord.AllowedMentions{},
	}
}

// RenderLazyCatchupBanner builds the banner inserted after >>1 when mirroring a pre-existing global thread.
func RenderLazyCatchupBanner(threadID uuid.UUID, omittedFrom, omittedTo uint32, locale discord.Locale) discord.MessageCreate {
	bannerText := i18n.TranslateText(locale, "components.bbs.catchup.banner", map[string]any{
		"from": omittedFrom,
		"to":   omittedTo,
	})
	btnLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.catchup.btn", map[string]any{
		"from": omittedFrom,
		"to":   omittedTo,
	}), MaxButtonLabelRunes)
	return discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay(bannerText),
			discord.NewActionRow(
				discord.NewSecondaryButton(btnLabel, fmt.Sprintf("/bbs/reader/%s/%d", threadID, omittedFrom)),
			),
		).WithAccentColor(ColorNotice),
	).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderThreadControlPanel creates the floating action panel placed at the bottom of active threads.
func RenderThreadControlPanel(mirrorID uuid.UUID, thread bbs.Thread, locale discord.Locale) discord.MessageCreate {
	titleText := i18n.TranslateText(locale, "components.bbs.control_panel.title", map[string]any{
		"title": TruncateRunes(bbs.ClearFormat(thread.Title), 60),
		"count": thread.ResCount,
	})
	postBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.control_panel.btn.post"), MaxButtonLabelRunes)
	watchBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.control_panel.btn.watch"), MaxButtonLabelRunes)
	readerBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.control_panel.btn.reader"), MaxButtonLabelRunes)
	reportBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.control_panel.btn.report"), MaxButtonLabelRunes)

	container := discord.NewContainer(
		discord.NewTextDisplay(titleText),
		discord.NewSmallSeparator(),
		discord.NewActionRow(
			discord.NewPrimaryButton(postBtn, fmt.Sprintf("/bbs/post/new/%s", mirrorID)),
			discord.NewSecondaryButton(watchBtn, fmt.Sprintf("/bbs/watch/toggle/%s", thread.ID)),
			discord.NewSecondaryButton(readerBtn, fmt.Sprintf("/bbs/reader/%s/1", thread.ID)),
			discord.NewDangerButton(reportBtn, fmt.Sprintf("/bbs/report/thread/%s", thread.ID)),
		),
	)
	return discord.NewMessageCreateV2(container).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderOver1000Message builds the classic 1001 Over 1000 Thread notice.
func RenderOver1000Message(thread bbs.Thread, locale discord.Locale) discord.MessageCreate {
	life := time.Since(thread.CreatedAt).Round(time.Second)
	lifeStr := i18n.TranslateText(locale, "components.bbs.over1000.lifetime", map[string]any{
		"days":    int(life.Hours()) / 24,
		"hours":   int(life.Hours()) % 24,
		"minutes": int(life.Minutes()) % 60,
	})
	body := i18n.TranslateText(locale, "components.bbs.over1000.body", map[string]any{
		"max":  thread.Rules.Normalize().MaxRes,
		"life": lifeStr,
	})
	return discord.NewMessageCreateV2(
		discord.NewTextDisplay("1001 :1001:Over 1000 Thread"),
		discord.NewTextDisplay(body),
	).WithAllowedMentions(&discord.AllowedMentions{})
}

// RenderArchivedNotice builds the archive warehouse message when a thread locks.
func RenderArchivedNotice(locale discord.Locale) discord.MessageCreate {
	return discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay(i18n.TranslateText(locale, "components.bbs.archived.notice")),
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
	locale discord.Locale,
) (components []discord.LayoutComponent, startRes uint32, endRes uint32) {
	titleLine := i18n.TranslateText(locale, "components.bbs.reader.title", map[string]any{
		"title": TruncateRunes(bbs.ClearFormat(thread.Title), 80),
		"count": thread.ResCount,
	})
	if len(posts) == 0 {
		c := discord.NewContainer(
			discord.NewTextDisplay(titleLine),
			discord.NewTextDisplay(i18n.TranslateText(locale, "components.bbs.reader.empty")),
		)
		return []discord.LayoutComponent{c}, 1, 1
	}
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex >= len(posts) {
		startIndex = len(posts) - 1
	}

	usedRunes := utf8.RuneCountInString(titleLine)
	usedComponents := 2 // outer container + titleLine

	subComps := []discord.ContainerSubComponent{
		discord.NewTextDisplay(titleLine),
		discord.NewSmallSeparator(),
	}
	usedComponents++ // separator

	aboneBody := i18n.TranslateText(locale, "components.bbs.post.abone_body")
	startRes = posts[startIndex].ResNumber
	endRes = startRes
	shownCount := 0
	maxRes := thread.Rules.Normalize().MaxRes

	for i := startIndex; i < len(posts); i++ {
		p := posts[i]
		header := RenderPostHeader(p, thread.Rules, locale)
		body := aboneBody
		var mediaURLs []string
		if !p.Abone {
			body = bbs.RenderFormattedBody(p.Body, maxRes, resolveURL)
			mediaURLs = bbs.ExtractMediaURLs(p.Body)
		}

		if shownCount == 0 {
			budget := SafeReaderTextDisplayRunes - usedRunes - utf8.RuneCountInString(header) - 40
			if budget < 200 {
				budget = 200
			}
			body = TruncateRunes(body, budget)
		}

		postRunes := utf8.RuneCountInString(header) + utf8.RuneCountInString(body)
		postCompCount := 3 // separator + header + body
		if len(mediaURLs) > 0 && shownCount == 0 {
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
	if startIndex > 0 {
		prevIdx := computeBackwardStartIndex(thread, posts, startIndex-1, resolveURL, locale)
		prevRes = posts[prevIdx].ResNumber
	}
	nextRes := endRes + 1
	if nextRes > thread.ResCount {
		nextRes = thread.ResCount
	}

	atStart := startRes <= 1
	atEnd := endRes >= thread.ResCount

	firstBtn := discord.NewSecondaryButton(TruncateRunes(i18n.TranslateText(locale, "components.bbs.reader.btn.first"), MaxButtonLabelRunes), fmt.Sprintf("/bbs/reader/%s/1", thread.ID))
	firstBtn.Disabled = atStart
	prevBtn := discord.NewSecondaryButton(TruncateRunes(i18n.TranslateText(locale, "components.bbs.reader.btn.prev"), MaxButtonLabelRunes), fmt.Sprintf("/bbs/reader/%s/%d", thread.ID, prevRes))
	prevBtn.Disabled = atStart
	nextBtn := discord.NewSecondaryButton(TruncateRunes(i18n.TranslateText(locale, "components.bbs.reader.btn.next"), MaxButtonLabelRunes), fmt.Sprintf("/bbs/reader/%s/%d", thread.ID, nextRes))
	nextBtn.Disabled = atEnd
	latestBtn := discord.NewSecondaryButton(TruncateRunes(i18n.TranslateText(locale, "components.bbs.reader.btn.latest"), MaxButtonLabelRunes), fmt.Sprintf("/bbs/reader/%s/latest", thread.ID))
	latestBtn.Disabled = atEnd
	jumpBtn := discord.NewPrimaryButton(TruncateRunes(i18n.TranslateText(locale, "components.bbs.reader.btn.jump"), MaxButtonLabelRunes), fmt.Sprintf("/bbs/reader/jump/%s", thread.ID))

	navRow := discord.NewActionRow(firstBtn, prevBtn, nextBtn, latestBtn, jumpBtn)
	container := discord.NewContainer(subComps...).WithAccentColor(ColorNotice)
	return []discord.LayoutComponent{container, navRow}, startRes, endRes
}

// ComputeLatestReaderStartIndex determines the starting post index whose packed page reaches the end of posts.
func ComputeLatestReaderStartIndex(
	thread bbs.Thread,
	posts []bbs.Post,
	resolveURL func(resNum uint32) (string, bool),
	locale discord.Locale,
) int {
	if len(posts) <= 1 {
		return 0
	}
	return computeBackwardStartIndex(thread, posts, len(posts)-1, resolveURL, locale)
}

func computeBackwardStartIndex(
	thread bbs.Thread,
	posts []bbs.Post,
	targetEndIdx int,
	resolveURL func(resNum uint32) (string, bool),
	locale discord.Locale,
) int {
	if len(posts) <= 1 || targetEndIdx <= 0 {
		return 0
	}
	if targetEndIdx >= len(posts) {
		targetEndIdx = len(posts) - 1
	}
	titleLine := i18n.TranslateText(locale, "components.bbs.reader.title", map[string]any{
		"title": TruncateRunes(bbs.ClearFormat(thread.Title), 80),
		"count": thread.ResCount,
	})
	aboneBody := i18n.TranslateText(locale, "components.bbs.post.abone_body")
	usedRunes := utf8.RuneCountInString(titleLine)
	usedComponents := 3
	maxRes := thread.Rules.Normalize().MaxRes

	startIdx := targetEndIdx
	for i := targetEndIdx; i >= 0; i-- {
		p := posts[i]
		header := RenderPostHeader(p, thread.Rules, locale)
		body := aboneBody
		var mediaURLs []string
		if !p.Abone {
			body = bbs.RenderFormattedBody(p.Body, maxRes, resolveURL)
			mediaURLs = bbs.ExtractMediaURLs(p.Body)
		}
		if i == targetEndIdx {
			budget := SafeReaderTextDisplayRunes - usedRunes - utf8.RuneCountInString(header) - 40
			if budget < 200 {
				budget = 200
			}
			body = TruncateRunes(body, budget)
		}
		postRunes := utf8.RuneCountInString(header) + utf8.RuneCountInString(body)
		postComps := 3
		if len(mediaURLs) > 0 && usedComponents+postComps+1 <= SafeReaderMaxComponents {
			postComps++
		}
		if i < targetEndIdx && (usedRunes+postRunes > SafeReaderTextDisplayRunes || usedComponents+postComps > SafeReaderMaxComponents) {
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
	title := i18n.TranslateText(locale, "components.bbs.consent.title")
	summary := i18n.TranslateText(locale, "components.bbs.consent.summary")
	acceptLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.consent.btn.accept"), MaxButtonLabelRunes)
	termsLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.consent.btn.terms"), MaxButtonLabelRunes)
	guideLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.consent.btn.guidelines"), MaxButtonLabelRunes)
	privacyLabel := TruncateRunes(i18n.TranslateText(locale, "components.bbs.consent.btn.privacy"), MaxButtonLabelRunes)
	webSuffix := i18n.TranslateText(locale, "components.bbs.consent.btn.web_suffix")

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
		urlButtons = append(urlButtons, discord.NewLinkButton(TruncateRunes(termsLabel+webSuffix, MaxButtonLabelRunes), pm.TermsURL))
	}
	if pm.GuidelinesURL != "" {
		urlButtons = append(urlButtons, discord.NewLinkButton(TruncateRunes(guideLabel+webSuffix, MaxButtonLabelRunes), pm.GuidelinesURL))
	}
	if pm.PrivacyPolicyURL != "" {
		urlButtons = append(urlButtons, discord.NewLinkButton(TruncateRunes(privacyLabel+webSuffix, MaxButtonLabelRunes), pm.PrivacyPolicyURL))
	}

	subComps := []discord.ContainerSubComponent{
		discord.NewTextDisplay(title),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(summary),
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
func RenderRoleGatePanel(guildID snowflake.ID, roleID snowflake.ID, version string, locale discord.Locale) discord.MessageCreate {
	title := i18n.TranslateText(locale, "components.bbs.role_gate.title")
	desc := i18n.TranslateText(locale, "components.bbs.role_gate.desc", map[string]any{"version": version})
	hint := i18n.TranslateText(locale, "components.bbs.role_gate.grant_hint", map[string]any{"role_id": roleID})
	grantBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.role_gate.btn.grant"), MaxButtonLabelRunes)
	termsBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.role_gate.btn.terms"), MaxButtonLabelRunes)
	guideBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.consent.btn.guidelines"), MaxButtonLabelRunes)
	privacyBtn := TruncateRunes(i18n.TranslateText(locale, "components.bbs.consent.btn.privacy"), MaxButtonLabelRunes)

	return discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewTextDisplay(title),
			discord.NewSmallSeparator(),
			discord.NewTextDisplay(desc),
			discord.NewTextDisplay(hint),
			discord.NewLargeSeparator(),
			discord.NewActionRow(
				discord.NewPrimaryButton(grantBtn, fmt.Sprintf("/bbs/consent/role/%d", guildID)),
				discord.NewSecondaryButton(termsBtn, "/bbs/consent/view/terms"),
				discord.NewSecondaryButton(guideBtn, "/bbs/consent/view/guidelines"),
				discord.NewSecondaryButton(privacyBtn, "/bbs/consent/view/privacy"),
			),
		).WithAccentColor(ColorLocalBoard),
	).WithAllowedMentions(&discord.AllowedMentions{})
}
