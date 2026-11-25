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

package discordsync_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

func TestMain(m *testing.M) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load locales: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func containsPictographEmoji(s string) bool {
	for _, r := range s {
		if (r >= 0x1F300 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) {
			return true
		}
		if unicode.Is(unicode.So, r) && r > 0x2600 {
			return true
		}
	}
	return false
}

func assertNoEmojiInLayout(t *testing.T, comps []discord.LayoutComponent) {
	t.Helper()
	for _, comp := range comps {
		switch c := comp.(type) {
		case discord.TextDisplayComponent:
			if containsPictographEmoji(c.Content) {
				t.Fatalf("unexpected emoji in TextDisplay: %q", c.Content)
			}
		case discord.ActionRowComponent:
			for _, ic := range c.Components {
				if btn, ok := ic.(discord.ButtonComponent); ok {
					if btn.Emoji != nil || containsPictographEmoji(btn.Label) {
						t.Fatalf("unexpected emoji in Button: %+v", btn)
					}
				}
			}
		case discord.ContainerComponent:
			var sub []discord.LayoutComponent
			for _, sc := range c.Components {
				if lc, ok := sc.(discord.LayoutComponent); ok {
					sub = append(sub, lc)
				}
			}
			assertNoEmojiInLayout(t, sub)
		}
	}
}

func TestPayloadValidator_LimitsAndClamping(t *testing.T) {
	v := discordsync.PayloadValidator{}

	// 1. V2 clears legacy fields and drops empty MediaGallery
	mc := discord.MessageCreate{
		Flags:      discord.MessageFlagIsComponentsV2,
		Content:    "legacy",
		Embeds:     []discord.Embed{{Title: "e"}},
		StickerIDs: []snowflake.ID{1},
		Components: []discord.LayoutComponent{
			discord.NewTextDisplay("hello"),
			discord.NewMediaGallery(),
		},
	}
	validated, err := v.ValidateAndClampMessageCreate(mc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validated.Content != "" || len(validated.Embeds) != 0 || len(validated.StickerIDs) != 0 {
		t.Fatal("expected legacy fields cleared on Components V2 message")
	}
	if len(validated.Components) != 1 {
		t.Fatalf("expected empty MediaGallery stripped, got %d components", len(validated.Components))
	}

	// 2. Reject empty TextDisplay (prevents 50006)
	if _, err := v.ValidateAndClampMessageCreate(discord.NewMessageCreateV2(discord.NewTextDisplay("   "))); err == nil {
		t.Fatal("expected error for whitespace-only TextDisplay")
	}

	// 3. Reject >40 total components (prevents 50035)
	tooMany := make([]discord.LayoutComponent, 41)
	for i := range tooMany {
		tooMany[i] = discord.NewTextDisplay("a")
	}
	if _, err := v.ValidateAndClampMessageCreate(discord.NewMessageCreateV2(tooMany...)); err == nil {
		t.Fatal("expected error when exceeding 40 components")
	}

	// 4. Reject >4000 total text display runes
	big := strings.Repeat("あ", 4001)
	if _, err := v.ValidateAndClampMessageCreate(discord.NewMessageCreateV2(discord.NewTextDisplay(big))); err == nil {
		t.Fatal("expected error when exceeding 4000 runes")
	}

	// 5. Reject 0-option StringSelect
	emptySel := discord.NewStringSelectMenu("sel", "pick")
	if _, err := v.ValidateAndClampMessageCreate(discord.NewMessageCreateV2(discord.NewActionRow(emptySel))); err == nil {
		t.Fatal("expected error for 0-option StringSelect")
	}

	// 6. Modal validation: disabled select rejected, label clamped <= 45, TextDisplay sum <= 4000
	badModal := discord.ModalCreate{
		CustomID: "m1",
		Title:    strings.Repeat("題", 50),
		Components: []discord.LayoutComponent{
			discord.NewLabel(strings.Repeat("項目", 30), discord.StringSelectMenuComponent{
				CustomID: "s1",
				Disabled: true,
				Options:  []discord.StringSelectMenuOption{{Label: "A", Value: "a"}},
			}),
		},
	}
	if _, err := v.ValidateModal(badModal); err == nil {
		t.Fatal("expected error for disabled select inside modal")
	}

	overTextModal := discord.ModalCreate{
		CustomID: "m_over",
		Title:    "Notice",
		Components: []discord.LayoutComponent{
			discord.NewTextDisplay(strings.Repeat("a", 2001)),
			discord.NewTextDisplay(strings.Repeat("b", 2000)),
		},
	}
	if _, err := v.ValidateModal(overTextModal); err == nil {
		t.Fatal("expected error when combined modal TextDisplay exceeds 4000 runes")
	}
	exactTextModal := discord.ModalCreate{
		CustomID: "m_exact",
		Title:    "Notice",
		Components: []discord.LayoutComponent{
			discord.NewTextDisplay(strings.Repeat("a", 2000)),
			discord.NewTextDisplay(strings.Repeat("b", 2000)),
		},
	}
	if _, err := v.ValidateModal(exactTextModal); err != nil {
		t.Fatalf("expected combined modal TextDisplay == 4000 to pass, got %v", err)
	}

	okModal := discord.ModalCreate{
		CustomID: "m2",
		Title:    strings.Repeat("題", 50),
		Components: []discord.LayoutComponent{
			discord.NewLabel(strings.Repeat("項目", 30), discord.TextInputComponent{
				CustomID:  "in",
				Style:     discord.TextInputStyleShort,
				MaxLength: 100,
			}),
		},
	}
	vm, err := v.ValidateModal(okModal)
	if err != nil {
		t.Fatalf("unexpected modal error: %v", err)
	}
	if len([]rune(vm.Title)) > discordsync.MaxModalTitleRunes {
		t.Fatalf("expected title clamped to 45, got %d", len([]rune(vm.Title)))
	}
}

type fakeGuildState struct {
	features    []discord.GuildFeature
	channelCnt  int
	catChildren int
	botPerms    discord.Permissions
	botPos      int
	roles       map[snowflake.ID]discord.Role
}

func (f fakeGuildState) GuildFeatures(_ snowflake.ID) ([]discord.GuildFeature, bool) {
	return f.features, true
}
func (f fakeGuildState) GuildChannelCount(_ snowflake.ID) int { return f.channelCnt }
func (f fakeGuildState) CategoryChildCount(_, _ snowflake.ID) int {
	return f.catChildren
}
func (f fakeGuildState) BotGuildPermissions(_ snowflake.ID) (discord.Permissions, bool) {
	return f.botPerms, true
}
func (f fakeGuildState) BotHighestRolePosition(_ snowflake.ID) (int, bool) {
	return f.botPos, true
}
func (f fakeGuildState) Role(_, roleID snowflake.ID) (discord.Role, bool) {
	r, ok := f.roles[roleID]
	return r, ok
}

func TestPreflightCheckerAndMasking(t *testing.T) {
	guildID := snowflake.ID(100)
	roleHigh := snowflake.ID(200)
	roleOK := snowflake.ID(201)

	state := fakeGuildState{
		features: nil, // non-Community guild -> requires CreatePublicThreads
		botPerms: discord.PermissionManageChannels.Add(
			discord.PermissionViewChannel,
			discord.PermissionSendMessages,
			discord.PermissionSendMessagesInThreads,
			discord.PermissionManageThreads,
			discord.PermissionManageMessages,
			discord.PermissionReadMessageHistory,
			discord.PermissionManageRoles,
		),
		botPos: 10,
		roles: map[snowflake.ID]discord.Role{
			roleHigh: {ID: roleHigh, Name: "Admin", Position: 12},
			roleOK:   {ID: roleOK, Name: "BBSViewer", Position: 5},
		},
	}
	checker := discordsync.NewPreflightChecker(state)

	if _, err := checker.CheckChannelCreate(guildID, "テスト板", "説明", true, nil); err == nil {
		t.Fatal("expected missing CreatePublicThreads error on non-Community guild text fallback")
	}

	state.botPerms = state.botPerms.Add(discord.PermissionCreatePublicThreads)
	checker = discordsync.NewPreflightChecker(state)

	plan, err := checker.CheckChannelCreate(guildID, "テスト板", "説明", true, nil)
	if err != nil {
		t.Fatalf("CheckChannelCreate failed: %v", err)
	}
	if plan.UseForum || plan.FallbackReason != "community_required" {
		t.Fatalf("expected non-Community fallback to text channel, got %+v", plan)
	}

	// Community guild forum channel does not require CreatePublicThreads
	state.features = []discord.GuildFeature{discord.GuildFeatureCommunity}
	state.botPerms &^= discord.PermissionCreatePublicThreads
	checker = discordsync.NewPreflightChecker(state)
	forumPlan, err := checker.CheckChannelCreate(guildID, "テスト板", "説明", true, nil)
	if err != nil || !forumPlan.UseForum {
		t.Fatalf("expected Community forum check to pass without CreatePublicThreads, got %+v, err=%v", forumPlan, err)
	}

	// Overwrites masking prevents 403 from setting bits bot lacks (e.g. ManageWebhooks)
	masked := discordsync.MaskPermissionOverwrites(
		discord.PermissionViewChannel.Add(discord.PermissionSendMessages),
		[]discord.PermissionOverwrite{
			discord.RolePermissionOverwrite{
				RoleID: guildID,
				Allow:  discord.PermissionViewChannel.Add(discord.PermissionManageWebhooks),
				Deny:   discord.PermissionSendMessages.Add(discord.PermissionMentionEveryone),
			},
		},
	)
	ow := masked[0].(discord.RolePermissionOverwrite)
	if ow.Allow.Has(discord.PermissionManageWebhooks) || ow.Deny.Has(discord.PermissionMentionEveryone) {
		t.Fatalf("expected unheld permissions masked out, got allow=%v deny=%v", ow.Allow, ow.Deny)
	}

	if err := checker.CheckRoleAssign(guildID, roleHigh); err == nil {
		t.Fatal("expected role hierarchy error when target position >= bot position")
	}
	if err := checker.CheckRoleAssign(guildID, roleOK); err != nil {
		t.Fatalf("unexpected error for assignable role: %v", err)
	}
}

func TestRenderer_ZeroEmoji_MediaGallery_And_ReaderPagination(t *testing.T) {
	v := discordsync.PayloadValidator{}
	boardID := uuid.New()
	mountID := uuid.New()
	threadID := uuid.New()

	board := bbs.Board{
		ID:          boardID,
		Scope:       bbs.ScopeGlobal,
		Name:        "ニュース速報",
		Description: "テスト板",
		Rules:       bbs.BoardRules{}.Normalize(),
	}
	thread := bbs.Thread{
		ID:       threadID,
		BoardID:  boardID,
		Title:    "テストスレ",
		Status:   bbs.ThreadStatusActive,
		ResCount: 20,
		Rules:    bbs.BoardRules{}.Normalize(),
	}

	guide, err := v.ValidateAndClampMessageCreate(discordsync.RenderBoardGuide(board, mountID, discord.LocaleJapanese))
	if err != nil {
		t.Fatalf("guide validation failed: %v", err)
	}
	assertNoEmojiInLayout(t, guide.Components)

	pm := bbs.NewPolicyManager(bbs.PolicyConfig{})
	consentMsg, err := v.ValidateAndClampMessageCreate(discordsync.RenderConsentGate(pm, discord.LocaleJapanese, "tok"))
	if err != nil {
		t.Fatalf("consent gate validation failed: %v", err)
	}
	assertNoEmojiInLayout(t, consentMsg.Components)

	// MediaGallery added for normal post, removed on Abone
	post := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  threadID,
		ResNumber: 1,
		Name:      "風吹けば名無し",
		AnonID:    "abcd1234",
		IsOP:      true,
		Body:      "画像テスト https://example.com/pic.png",
		CreatedAt: time.Now(),
	}
	postMsg, err := v.ValidateAndClampMessageCreate(discordsync.RenderPostMessageCreate(post, thread, nil, discord.LocaleJapanese))
	if err != nil {
		t.Fatalf("post message failed: %v", err)
	}
	assertNoEmojiInLayout(t, postMsg.Components)

	hasGallery := false
	for _, c := range postMsg.Components {
		if c.Type() == discord.ComponentTypeMediaGallery {
			hasGallery = true
		}
	}
	if !hasGallery {
		t.Fatal("expected MediaGallery component for post with image URL")
	}

	post.Abone = true
	aboneMsg := discordsync.RenderPostMessageCreate(post, thread, nil, discord.LocaleJapanese)
	for _, c := range aboneMsg.Components {
		if c.Type() == discord.ComponentTypeMediaGallery {
			t.Fatal("MediaGallery must be removed when post is aboned")
		}
	}

	// Reader pagination & jump syntax
	var posts []bbs.Post
	for i := 1; i <= 20; i++ {
		posts = append(posts, bbs.Post{
			ID:        uuid.New(),
			ThreadID:  threadID,
			ResNumber: uint32(i),
			Name:      "名無し",
			AnonID:    "abcd1234",
			Body:      fmt.Sprintf("レス番号 %d の本文 %s", i, strings.Repeat("あ", 300)),
			CreatedAt: time.Now(),
		})
	}
	pageComps, startRes, endRes := discordsync.RenderReaderPage(thread, posts, 0, nil, discord.LocaleJapanese)
	if startRes != 1 || endRes >= 20 {
		t.Fatalf("expected multi-page split under 3500 runes, got %d..%d", startRes, endRes)
	}
	if _, err := v.ValidateAndClampMessageCreate(discord.NewMessageCreateV2(pageComps...).WithEphemeral(true)); err != nil {
		t.Fatalf("reader page failed validation: %v", err)
	}
	assertNoEmojiInLayout(t, pageComps)

	if got := discordsync.ParseReaderJumpInput("l5", 20); got != 16 {
		t.Fatalf("expected l5 -> 16, got %d", got)
	}
	if got := discordsync.ParseReaderJumpInput("99", 20); got != 20 {
		t.Fatalf("expected clamped jump 99 -> 20, got %d", got)
	}
}

func TestDebouncerAndInvalidRequestGuard(t *testing.T) {
	deb := discordsync.NewControlPanelDebouncer(20 * time.Millisecond)
	defer deb.Stop()

	id := uuid.New()
	var calls atomic.Int32
	done := make(chan int, 10)
	for i := range 5 {
		idx := i
		deb.Schedule(id, func() {
			calls.Add(1)
			done <- idx
		})
	}
	select {
	case gotIdx := <-done:
		if gotIdx != 4 {
			t.Fatalf("expected latest scheduled callback (4) to run, got %d", gotIdx)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for debounced callback")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 5 bursts coalesced into 1 call, got %d", got)
	}

	guard := discordsync.NewInvalidRequestGuard(10*time.Minute, 10, 2)
	guard.SetCircuitCooldown(1 * time.Minute)
	mountID := uuid.New()
	now := time.Now()

	err403 := &rest.Error{Response: &http.Response{StatusCode: 403, Header: http.Header{}}, Code: discordsync.JSONErrMissingPermissions}
	err404 := &rest.Error{Response: &http.Response{StatusCode: 404, Header: http.Header{}}, Code: discordsync.JSONErrUnknownChannel}
	errShared429 := &rest.Error{Response: &http.Response{StatusCode: 429, Header: http.Header{"X-Ratelimit-Scope": []string{"shared"}}}}

	// Shared 429s must not trip global guard
	for range 15 {
		guard.RecordError(uuid.Nil, errShared429, now)
	}
	if guard.IsThrottled(now) {
		t.Fatal("shared 429s must not count toward invalid-request throttle")
	}

	// 2 consecutive 403s trip mount circuit breaker
	if deact := guard.RecordError(mountID, err403, now); deact {
		t.Fatal("403 must not permanently deactivate mount")
	}
	if guard.IsMountOpen(mountID) {
		t.Fatal("1st 403 should not open circuit breaker yet")
	}
	guard.RecordError(mountID, err403, now)
	if !guard.IsMountOpenAt(mountID, now) {
		t.Fatal("2nd 403 should open mount circuit breaker")
	}

	// After cooldown, 1 half-open probe is permitted while concurrent check still sees open.
	afterCooldown := now.Add(2 * time.Minute)
	if guard.IsMountOpenAt(mountID, afterCooldown) {
		t.Fatal("expected first check after cooldown to allow half-open probe")
	}
	if !guard.IsMountOpenAt(mountID, afterCooldown) {
		t.Fatal("expected second concurrent check during half-open probe to stay open")
	}
	guard.RecordMountSuccess(mountID)
	if guard.IsMountOpenAt(mountID, afterCooldown) {
		t.Fatal("expected RecordMountSuccess to close circuit breaker")
	}

	// Global threshold test with separate mounts
	smallGuard := discordsync.NewInvalidRequestGuard(10*time.Minute, 3, 2)
	for range 3 {
		smallGuard.RecordError(uuid.New(), err403, now)
	}
	if !smallGuard.IsThrottled(now) {
		t.Fatal("expected global InvalidRequestGuard to throttle at threshold 3")
	}

	// 404 immediately signals mount deactivation
	if deact := guard.RecordError(uuid.New(), err404, now); !deact {
		t.Fatal("404 must request mount deactivation")
	}

	if !discordsync.IsJSONError(fmt.Errorf("wrap: %w", err404), discordsync.JSONErrUnknownChannel) {
		t.Fatal("expected IsJSONError to unwrap rest.Error")
	}
	if errors.Is(err403, errors.New("other")) {
		t.Fatal("unexpected errors.Is match")
	}

	if !discordsync.CanSendDMToUser(context.Background(), (*database.DB)(nil), 12345) {
		t.Fatal("expected nil DB to default CanSendDMToUser to true")
	}
	syncer := discordsync.NewSyncer(nil)
	syncer.CanSendDM = func(_ context.Context, uid snowflake.ID) bool { return uid != 999 }
	if syncer.CanSendDM(context.Background(), 999) {
		t.Fatal("expected CanSendDM hook to block user 999")
	}

	flaggedMsg := discord.Message{
		Components: []discord.LayoutComponent{
			discord.NewMediaGallery(
				discord.MediaGalleryItem{
					Media: discord.UnfurledMediaItem{
						URL: "https://example.com/a.png",
						ContentScanMetadata: &discord.UnfurledMediaItemContentScanMetadata{
							Flags: discord.UnfurledMediaItemContentScanMetadataFlagIsExplicitContent,
						},
					},
				},
			),
		},
	}
	if !discordsync.HasFlaggedMediaScan(flaggedMsg, false) {
		t.Fatal("expected explicit media in non-NSFW channel to flag")
	}
	if discordsync.HasFlaggedMediaScan(flaggedMsg, true) {
		t.Fatal("expected explicit media in NSFW channel to be allowed")
	}
}

func containsJapaneseRune(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}

func assertLayoutENClean(t *testing.T, comps []discord.LayoutComponent) {
	t.Helper()
	assertNoEmojiInLayout(t, comps)
	for _, comp := range comps {
		switch c := comp.(type) {
		case discord.TextDisplayComponent:
			if strings.Contains(c.Content, "components.bbs.") || containsJapaneseRune(c.Content) {
				t.Fatalf("unexpected untranslated or Japanese text in en-US TextDisplay: %q", c.Content)
			}
		case discord.ActionRowComponent:
			for _, ic := range c.Components {
				if btn, ok := ic.(discord.ButtonComponent); ok {
					if strings.Contains(btn.Label, "components.bbs.") || containsJapaneseRune(btn.Label) {
						t.Fatalf("unexpected untranslated or Japanese text in en-US Button: %q", btn.Label)
					}
				}
			}
		case discord.ContainerComponent:
			var sub []discord.LayoutComponent
			for _, sc := range c.Components {
				if lc, ok := sc.(discord.LayoutComponent); ok {
					sub = append(sub, lc)
				}
			}
			assertLayoutENClean(t, sub)
		}
	}
}

func TestRenderer_I18n_JapaneseAndEnglish(t *testing.T) {
	v := discordsync.PayloadValidator{}
	boardID := uuid.New()
	mountID := uuid.New()
	mirrorID := uuid.New()
	threadID := uuid.New()

	board := bbs.Board{
		ID:          boardID,
		Scope:       bbs.ScopeGlobal,
		Name:        "TechTalk",
		Description: "General technology board",
		Rules: bbs.BoardRules{
			DefaultName: bbs.DefaultAnonymousName,
			SLIP:        bbs.SLIPVVVVV,
			MaxRes:      1000,
		},
	}
	thread := bbs.Thread{
		ID:        threadID,
		BoardID:   boardID,
		Title:     "Go 1.27 Discussion",
		Status:    bbs.ThreadStatusActive,
		ResCount:  5,
		Rules:     board.Rules,
		CreatedAt: time.Now().Add(-26 * time.Hour),
	}
	posts := []bbs.Post{
		{
			ID:        uuid.New(),
			ThreadID:  threadID,
			ResNumber: 1,
			Name:      bbs.DefaultAnonymousName,
			SlipText:  "abcd-efgh",
			AnonID:    "xyz12345",
			IsOP:      true,
			Body:      "Hello >>1",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			ThreadID:  threadID,
			ResNumber: 2,
			Abone:     true,
			CreatedAt: time.Now(),
		},
	}
	pm := bbs.NewPolicyManager(bbs.PolicyConfig{
		TermsURL: "https://example.com/terms",
	})

	for _, loc := range []discord.Locale{discord.LocaleJapanese, discord.LocaleEnglishUS} {
		msgs := []discord.MessageCreate{
			discordsync.RenderNotice(loc, "", "Notice body", discordsync.ColorNotice),
			discordsync.RenderError(loc, "Error Title", "Error body"),
			discordsync.RenderBoardGuide(board, mountID, loc),
			discordsync.RenderPostMessageCreate(posts[0], thread, nil, loc),
			discordsync.RenderPostMessageCreate(posts[1], thread, nil, loc),
			discordsync.RenderLazyCatchupBanner(threadID, 2, 5, loc),
			discordsync.RenderThreadControlPanel(mirrorID, thread, loc),
			discordsync.RenderOver1000Message(thread, loc),
			discordsync.RenderArchivedNotice(loc),
			discordsync.RenderConsentGate(pm, loc, "tok123"),
			discordsync.RenderRoleGatePanel(snowflake.ID(100), snowflake.ID(200), "2026-11-25", loc),
		}
		for i, m := range msgs {
			validated, err := v.ValidateAndClampMessageCreate(m)
			if err != nil {
				t.Fatalf("locale %s message %d failed validation: %v", loc, i, err)
			}
			assertNoEmojiInLayout(t, validated.Components)
			if loc == discord.LocaleEnglishUS {
				assertLayoutENClean(t, validated.Components)
			}
		}

		latestIdx := discordsync.ComputeLatestReaderStartIndex(thread, posts, nil, loc)
		readerComps, _, _ := discordsync.RenderReaderPage(thread, posts, latestIdx, nil, loc)
		if _, err := v.ValidateAndClampMessageCreate(discord.NewMessageCreateV2(readerComps...).WithEphemeral(true)); err != nil {
			t.Fatalf("locale %s reader failed validation: %v", loc, err)
		}
		if loc == discord.LocaleEnglishUS {
			assertLayoutENClean(t, readerComps)
		}
	}

	pe := &discordsync.PreflightError{
		Kind:     discordsync.PreflightErrRoleHierarchy,
		RoleName: "Member",
		RolePos:  8,
		BotPos:   5,
	}
	jaMsg, ok := discordsync.FormatPreflightError(discord.LocaleJapanese, pe)
	if !ok || !containsJapaneseRune(jaMsg) {
		t.Fatalf("expected Japanese preflight error message, got ok=%v %q", ok, jaMsg)
	}
	enMsg, ok := discordsync.FormatPreflightError(discord.LocaleEnglishUS, pe)
	if !ok || containsJapaneseRune(enMsg) || strings.Contains(enMsg, "components.bbs.") {
		t.Fatalf("expected clean English preflight error message, got ok=%v %q", ok, enMsg)
	}
}

type mockInteractionEvent struct {
	deferErr    error
	createErr   error
	updateErr   error
	deferCalls  int
	createCalls int
	editCalls   int
	followCalls int
	updateCalls int
	modalCalls  int
}

func (m *mockInteractionEvent) Modal(_ discord.ModalCreate, _ ...rest.RequestOpt) error {
	m.modalCalls++
	return nil
}

func (m *mockInteractionEvent) DeferCreateMessage(_ bool, _ ...rest.RequestOpt) error {
	m.deferCalls++
	return m.deferErr
}

func (m *mockInteractionEvent) CreateMessage(_ discord.MessageCreate, _ ...rest.RequestOpt) error {
	m.createCalls++
	return m.createErr
}

func (m *mockInteractionEvent) UpdateInteractionResponse(_ discord.MessageUpdate, _ ...rest.RequestOpt) (*discord.Message, error) {
	m.editCalls++
	return &discord.Message{}, m.updateErr
}

func (m *mockInteractionEvent) CreateFollowupMessage(_ discord.MessageCreate, _ ...rest.RequestOpt) (*discord.Message, error) {
	m.followCalls++
	return &discord.Message{}, nil
}

func (m *mockInteractionEvent) UpdateMessage(_ discord.MessageUpdate, _ ...rest.RequestOpt) error {
	m.updateCalls++
	return m.updateErr
}

func TestSafeResponder_WindowsAndAckStates(t *testing.T) {
	msg := discord.NewMessageCreateV2(discord.NewTextDisplay("ok"))
	upd := discord.NewMessageUpdateV2(discord.NewTextDisplay("updated"))

	// 1. Stale unacked interaction (>3s) must reject initial CreateMessage and UpdateComponentMessage
	stale := discordsync.NewSafeResponder(time.Now().Add(-5 * time.Second))
	evStale := &mockInteractionEvent{}
	if err := stale.RespondMessageCreate(evStale, msg); err == nil {
		t.Fatal("expected >3s unacked RespondMessageCreate to fail")
	}
	if err := stale.UpdateComponentMessage(evStale, upd); err == nil {
		t.Fatal("expected >3s unacked UpdateComponentMessage to fail")
	}
	if evStale.createCalls != 0 || evStale.updateCalls != 0 {
		t.Fatalf("stale interaction must not send REST calls: %+v", evStale)
	}

	// 2. Pre-ack 4xx failure on DeferCreate must keep initial CreateMessage available within 3s
	srPreAck := discordsync.NewSafeResponder(time.Now())
	ev400 := &mockInteractionEvent{
		deferErr: &rest.Error{Response: &http.Response{StatusCode: 400}},
	}
	if err := srPreAck.DeferCreate(ev400, true); err == nil {
		t.Fatal("expected DeferCreate 400 error to propagate")
	}
	if err := srPreAck.RespondMessageCreate(ev400, msg); err != nil {
		t.Fatalf("expected fallback CreateMessage after pre-ack 400 failure, got %v", err)
	}
	if ev400.createCalls != 1 || ev400.editCalls != 0 {
		t.Fatalf("expected CreateMessage fallback instead of UpdateInteractionResponse: %+v", ev400)
	}

	// 3. Successful DeferCreate allows UpdateInteractionResponse even 10s later, and blocks duplicate UpdateComponentMessage
	srComp := discordsync.NewSafeResponder(time.Now())
	evComp := &mockInteractionEvent{}
	if err := srComp.UpdateComponentMessage(evComp, upd); err != nil {
		t.Fatalf("first UpdateComponentMessage failed: %v", err)
	}
	if err := srComp.UpdateComponentMessage(evComp, upd); err == nil {
		t.Fatal("duplicate UpdateComponentMessage on same responder must fail")
	}
	if evComp.updateCalls != 1 {
		t.Fatalf("expected exactly 1 UpdateMessage call, got %d", evComp.updateCalls)
	}
}

func TestRenderer_ExtDatBudget_And_LinkExpandedPagination(t *testing.T) {
	v := discordsync.PayloadValidator{}
	threadID := uuid.New()
	thread := bbs.Thread{
		ID:       threadID,
		Title:    "Test",
		ResCount: 4,
		ExtDat:   strings.Repeat("EXT", 55), // 165 runes > 120 old reserve
		Rules:    bbs.BoardRules{MaxRes: 1000}.Normalize(),
	}

	op := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  threadID,
		ResNumber: 1,
		Name:      "名無し",
		AnonID:    "abcd1234",
		IsOP:      true,
		Body:      strings.Repeat("あ", 1800),
		CreatedAt: time.Now(),
	}
	opMsg := discordsync.RenderPostMessageCreate(op, thread, nil, discord.LocaleJapanese)
	if _, err := v.ValidateAndClampMessageCreate(opMsg); err != nil {
		t.Fatalf("OP post with long ExtDat failed Components V2 validation: %v", err)
	}

	// Build 4 posts where without link expansion 3&4 look close, but link expansion pushes size over budget,
	// and res 2 + res 3 combined exceed budget so Prev from res 4 must land on res 3.
	posts := []bbs.Post{
		{ID: uuid.New(), ThreadID: threadID, ResNumber: 1, AnonID: "1", Body: strings.Repeat("a", 400), CreatedAt: time.Now()},
		{ID: uuid.New(), ThreadID: threadID, ResNumber: 2, AnonID: "2", Body: strings.Repeat("b", 2000), CreatedAt: time.Now()},
		{ID: uuid.New(), ThreadID: threadID, ResNumber: 3, AnonID: "3", Body: strings.Repeat("c", 2800), CreatedAt: time.Now()},
		{ID: uuid.New(), ThreadID: threadID, ResNumber: 4, AnonID: "4", Body: ">>1 >>1 >>1 >>1 " + strings.Repeat("d", 300), CreatedAt: time.Now()},
	}
	resolveURL := func(resNum uint32) (string, bool) {
		return fmt.Sprintf("https://discord.com/channels/100000000000000001/200000000000000002/%d", resNum), true
	}
	latestIdx := discordsync.ComputeLatestReaderStartIndex(thread, posts, resolveURL, discord.LocaleJapanese)
	_, startRes, endRes := discordsync.RenderReaderPage(thread, posts, latestIdx, resolveURL, discord.LocaleJapanese)
	if endRes != 4 {
		t.Fatalf("expected Latest page (starting idx=%d -> res=%d) to reach final res 4, stopped at %d", latestIdx, startRes, endRes)
	}

	// Verify Prev navigation from startIndex=3 (res 4) points to res 3, not res 2.
	comps, _, _ := discordsync.RenderReaderPage(thread, posts, 3, resolveURL, discord.LocaleJapanese)
	nav := comps[1].(discord.ActionRowComponent)
	prevBtn := nav.Components[1].(discord.ButtonComponent)
	if !strings.HasSuffix(prevBtn.CustomID, "/3") {
		t.Fatalf("expected Prev button to target res 3 right before page start 4, got customID=%q", prevBtn.CustomID)
	}
	_, prevStartRes, prevEndRes := discordsync.RenderReaderPage(thread, posts, 2, resolveURL, discord.LocaleJapanese)
	if prevStartRes != 3 || prevEndRes != 3 {
		t.Fatalf("expected Prev page at startIndex=2 to render res 3..3, got %d..%d", prevStartRes, prevEndRes)
	}
}
