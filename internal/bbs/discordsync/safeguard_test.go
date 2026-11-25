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
)

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

	// 6. Modal validation: disabled select rejected, label clamped <= 45
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
		features: nil, // non-Community guild
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

	plan, err := checker.CheckChannelCreate(guildID, "テスト板", "説明", true, nil)
	if err != nil {
		t.Fatalf("CheckChannelCreate failed: %v", err)
	}
	if plan.UseForum || plan.FallbackReason != "community_required" {
		t.Fatalf("expected non-Community fallback to text channel, got %+v", plan)
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
	postMsg, err := v.ValidateAndClampMessageCreate(discordsync.RenderPostMessageCreate(post, thread, nil))
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
	aboneMsg := discordsync.RenderPostMessageCreate(post, thread, nil)
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
	pageComps, startRes, endRes := discordsync.RenderReaderPage(thread, posts, 0, nil)
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
	deb := discordsync.NewControlPanelDebouncer(30 * time.Millisecond)
	defer deb.Stop()

	id := uuid.New()
	var calls atomic.Int32
	for range 5 {
		deb.Schedule(id, func() {
			calls.Add(1)
		})
	}
	time.Sleep(80 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 5 bursts coalesced into 1 call, got %d", got)
	}

	guard := discordsync.NewInvalidRequestGuard(10*time.Minute, 3, 2)
	mountID := uuid.New()
	now := time.Now()

	err403 := &rest.Error{Response: &http.Response{StatusCode: 403, Header: http.Header{}}, Code: discordsync.JSONErrMissingPermissions}
	err404 := &rest.Error{Response: &http.Response{StatusCode: 404, Header: http.Header{}}, Code: discordsync.JSONErrUnknownChannel}
	errShared429 := &rest.Error{Response: &http.Response{StatusCode: 429, Header: http.Header{"X-Ratelimit-Scope": []string{"shared"}}}}

	// Shared 429s must not trip global guard
	for range 5 {
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
	if !guard.IsMountOpen(mountID) {
		t.Fatal("2nd 403 should open mount circuit breaker")
	}

	// 3rd invalid request (err403) hits global threshold 3
	guard.RecordError(uuid.New(), err403, now)
	if !guard.IsThrottled(now) {
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
