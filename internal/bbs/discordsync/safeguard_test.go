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
		if r == '★' || r == '☆' || r == '☰' {
			// Monochrome 5ch-style cap marks and trigram menu icon are intentional text badges, not colored emoji.
			continue
		}
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
		case discord.SectionComponent:
			for _, sc := range c.Components {
				if td, ok := sc.(discord.TextDisplayComponent); ok {
					if containsPictographEmoji(td.Content) {
						t.Fatalf("unexpected emoji in Section TextDisplay: %q", td.Content)
					}
				}
			}
			if btn, ok := c.Accessory.(discord.ButtonComponent); ok {
				if btn.Emoji != nil || containsPictographEmoji(btn.Label) {
					t.Fatalf("unexpected emoji in Section Accessory Button: %+v", btn)
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

func TestPayloadValidator_StrictRejection(t *testing.T) {
	v := discordsync.PayloadValidator{}

	// 1. V2 rejects legacy fields instead of silently clearing them.
	mc := discord.MessageCreate{
		Flags:      discord.MessageFlagIsComponentsV2,
		Content:    "legacy",
		Embeds:     []discord.Embed{{Title: "e"}},
		StickerIDs: []snowflake.ID{1},
		Components: []discord.LayoutComponent{
			discord.NewTextDisplay("hello"),
		},
	}
	if err := v.ValidateMessageCreate(mc); err == nil {
		t.Fatal("expected error when Components V2 message sets legacy fields")
	}

	// Empty MediaGallery must be rejected instead of stripped.
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(
		discord.NewTextDisplay("hello"),
		discord.NewMediaGallery(),
	)); err == nil {
		t.Fatal("expected error for empty MediaGallery")
	}

	// Over-limit button label (>80) must be rejected instead of clamped.
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(
		discord.NewActionRow(discord.NewPrimaryButton(strings.Repeat("x", 81), "/test")),
	)); err == nil {
		t.Fatal("expected error when button label exceeds 80 chars")
	}

	// Duplicate custom_id within the same ActionRow, across Container/Section/Rows, Update, and Modal must be rejected (prevents 50035).
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(
		discord.NewActionRow(
			discord.NewPrimaryButton("A", "/dup"),
			discord.NewSecondaryButton("B", "/dup"),
		),
	)); err == nil {
		t.Fatal("expected error for duplicate button custom_id in same ActionRow")
	}
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(
		discord.NewContainer(
			discord.NewSection(discord.NewTextDisplay("hi")).WithAccessory(discord.NewPrimaryButton("A", "/dup2")),
			discord.NewActionRow(discord.NewSecondaryButton("B", "/dup2")),
		),
	)); err == nil {
		t.Fatal("expected error for duplicate button custom_id across Section and ActionRow")
	}
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(
		discord.NewActionRow(discord.NewStringSelectMenu("sel_dup", "pick", discord.NewStringSelectMenuOption("A", "a"))),
		discord.NewActionRow(discord.NewPrimaryButton("B", "sel_dup")),
	)); err == nil {
		t.Fatal("expected error for duplicate custom_id between StringSelect and Button")
	}
	dupComps := []discord.LayoutComponent{
		discord.NewActionRow(discord.NewPrimaryButton("A", "/upd_dup")),
		discord.NewActionRow(discord.NewSecondaryButton("B", "/upd_dup")),
	}
	updFlags := discord.MessageFlagIsComponentsV2
	if err := v.ValidateMessageUpdate(discord.MessageUpdate{Flags: &updFlags, Components: &dupComps}, true); err == nil {
		t.Fatal("expected error for duplicate custom_id in ValidateMessageUpdate")
	}
	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "dup_modal",
		Title:    "Dup",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Field 1", discord.NewShortTextInput("same_id")),
			discord.NewLabel("Field 2", discord.NewStringSelectMenu("same_id", "pick", discord.NewStringSelectMenuOption("A", "a"))),
		},
	}); err == nil {
		t.Fatal("expected error for duplicate custom_id across modal labels")
	}

	// Valid V2 passes.
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(discord.NewTextDisplay("hello"))); err != nil {
		t.Fatalf("unexpected error for valid V2 message: %v", err)
	}

	// 2. Reject empty TextDisplay (prevents 50006)
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(discord.NewTextDisplay("   "))); err == nil {
		t.Fatal("expected error for whitespace-only TextDisplay")
	}

	// 3. Reject >40 total components (prevents 50035)
	tooMany := make([]discord.LayoutComponent, 41)
	for i := range tooMany {
		tooMany[i] = discord.NewTextDisplay("a")
	}
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(tooMany...)); err == nil {
		t.Fatal("expected error when exceeding 40 components")
	}

	// 4. Reject >4000 total text display runes
	big := strings.Repeat("あ", 4001)
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(discord.NewTextDisplay(big))); err == nil {
		t.Fatal("expected error when exceeding 4000 runes")
	}

	// 5. Reject 0-option and >25-option StringSelect
	emptySel := discord.NewStringSelectMenu("sel", "pick")
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(discord.NewActionRow(emptySel))); err == nil {
		t.Fatal("expected error for 0-option StringSelect")
	}
	opts26 := make([]discord.StringSelectMenuOption, 26)
	for i := range opts26 {
		opts26[i] = discord.NewStringSelectMenuOption(fmt.Sprintf("O%d", i), fmt.Sprintf("v%d", i))
	}
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(discord.NewActionRow(
		discord.NewStringSelectMenu("sel", "pick", opts26...),
	))); err == nil {
		t.Fatal("expected error for 26-option StringSelect")
	}

	// 6. Autocomplete rejects >25 items and >100 char names without clamping
	ac26 := make([]discord.AutocompleteChoice, 26)
	for i := range ac26 {
		ac26[i] = discord.AutocompleteChoiceString{Name: "a", Value: "a"}
	}
	if err := discordsync.ValidateAutocompleteChoices(ac26); err == nil {
		t.Fatal("expected error when autocomplete choices exceed 25")
	}
	if err := discordsync.ValidateAutocompleteChoices([]discord.AutocompleteChoice{
		discord.AutocompleteChoiceString{Name: strings.Repeat("a", 101), Value: "1"},
	}); err == nil {
		t.Fatal("expected error when autocomplete choice name exceeds 100 chars")
	}

	// 7. Modal validation: disabled select, >45 title, >45 label, value > MaxLength all rejected strictly
	badModal := discord.ModalCreate{
		CustomID: "m1",
		Title:    "Title",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Field", discord.StringSelectMenuComponent{
				CustomID: "s1",
				Disabled: true,
				Options:  []discord.StringSelectMenuOption{{Label: "A", Value: "a"}},
			}),
		},
	}
	if err := v.ValidateModal(badModal); err == nil {
		t.Fatal("expected error for disabled select inside modal")
	}
	minTwo := 2
	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_range_default_max",
		Title:    "Select",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Pick", discord.StringSelectMenuComponent{
				CustomID:  "s_range",
				MinValues: &minTwo,
				Options: []discord.StringSelectMenuOption{
					{Label: "A", Value: "a"},
					{Label: "B", Value: "b"},
				},
			}),
		},
	}); err == nil {
		t.Fatal("expected ValidateModal to reject StringSelect with MinValues=2 and omitted MaxValues=0")
	}
	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_range_explicit_max",
		Title:    "Select",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Pick", discord.StringSelectMenuComponent{
				CustomID:  "s_range_ok",
				MinValues: &minTwo,
				MaxValues: 2,
				Options: []discord.StringSelectMenuOption{
					{Label: "A", Value: "a"},
					{Label: "B", Value: "b"},
				},
			}),
		},
	}); err != nil {
		t.Fatalf("expected explicit MaxValues=2 with MinValues=2 to pass ValidateModal, got %v", err)
	}

	overTitleModal := discord.ModalCreate{
		CustomID: "m_title",
		Title:    strings.Repeat("題", 46),
		Components: []discord.LayoutComponent{
			discord.NewLabel("項目", discord.NewShortTextInput("in").WithMaxLength(100)),
		},
	}
	if err := v.ValidateModal(overTitleModal); err == nil {
		t.Fatal("expected error when modal title exceeds 45 chars")
	}

	overValueModal := discord.ModalCreate{
		CustomID: "m_val",
		Title:    "Notice",
		Components: []discord.LayoutComponent{
			discord.NewLabel("項目", discord.TextInputComponent{
				CustomID:  "in",
				Style:     discord.TextInputStyleShort,
				MaxLength: 5,
				Value:     "123456",
			}),
		},
	}
	if err := v.ValidateModal(overValueModal); err == nil {
		t.Fatal("expected error when pre-filled modal value exceeds MaxLength")
	}

	overTextModal := discord.ModalCreate{
		CustomID: "m_over",
		Title:    "Notice",
		Components: []discord.LayoutComponent{
			discord.NewTextDisplay(strings.Repeat("a", 2001)),
			discord.NewTextDisplay(strings.Repeat("b", 2000)),
		},
	}
	if err := v.ValidateModal(overTextModal); err == nil {
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
	if err := v.ValidateModal(exactTextModal); err != nil {
		t.Fatalf("expected combined modal TextDisplay == 4000 to pass, got %v", err)
	}

	okModal := discord.ModalCreate{
		CustomID: "m2",
		Title:    strings.Repeat("題", 45),
		Components: []discord.LayoutComponent{
			discord.NewLabel(strings.Repeat("項", 45), discord.TextInputComponent{
				CustomID:  "in",
				Style:     discord.TextInputStyleShort,
				MaxLength: 100,
			}),
		},
	}
	if err := v.ValidateModal(okModal); err != nil {
		t.Fatalf("unexpected modal error: %v", err)
	}

	// 8. CheckboxGroup modal validation
	minZero := 0
	minThree := 3
	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_cb_empty",
		Title:    "Consent",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Agree", discord.CheckboxGroupComponent{
				CustomID: "cb",
				Required: true,
			}),
		},
	}); err == nil {
		t.Fatal("expected error for 0-option CheckboxGroup")
	}

	opts11 := make([]discord.CheckboxGroupOption, 11)
	for i := range opts11 {
		opts11[i] = discord.NewCheckboxGroupOption(fmt.Sprintf("v%d", i), fmt.Sprintf("L%d", i))
	}
	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_cb_11",
		Title:    "Consent",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Agree", discord.CheckboxGroupComponent{
				CustomID: "cb",
				Options:  opts11,
			}),
		},
	}); err == nil {
		t.Fatal("expected error for 11-option CheckboxGroup")
	}

	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_cb_dup",
		Title:    "Consent",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Agree", discord.CheckboxGroupComponent{
				CustomID: "cb",
				Options: []discord.CheckboxGroupOption{
					discord.NewCheckboxGroupOption("a", "Label A"),
					discord.NewCheckboxGroupOption("a", "Label B"),
				},
			}),
		},
	}); err == nil {
		t.Fatal("expected error for duplicate CheckboxGroup option values")
	}

	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_cb_emptylabel",
		Title:    "Consent",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Agree", discord.CheckboxGroupComponent{
				CustomID: "cb",
				Options: []discord.CheckboxGroupOption{
					discord.NewCheckboxGroupOption("a", "   "),
				},
			}),
		},
	}); err == nil {
		t.Fatal("expected error for empty CheckboxGroup option label")
	}

	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_cb_req_min0",
		Title:    "Consent",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Agree", discord.CheckboxGroupComponent{
				CustomID:  "cb",
				Required:  true,
				MinValues: &minZero,
				Options: []discord.CheckboxGroupOption{
					discord.NewCheckboxGroupOption("a", "A"),
				},
			}),
		},
	}); err == nil {
		t.Fatal("expected error for required CheckboxGroup with MinValues=0")
	}

	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_cb_minmax",
		Title:    "Consent",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Agree", discord.CheckboxGroupComponent{
				CustomID:  "cb",
				Required:  true,
				MinValues: &minThree,
				MaxValues: 2,
				Options: []discord.CheckboxGroupOption{
					discord.NewCheckboxGroupOption("a", "A"),
					discord.NewCheckboxGroupOption("b", "B"),
					discord.NewCheckboxGroupOption("c", "C"),
				},
			}),
		},
	}); err == nil {
		t.Fatal("expected error for CheckboxGroup MinValues > MaxValues")
	}

	if err := v.ValidateModal(discord.ModalCreate{
		CustomID: "m_cb_ok",
		Title:    "Consent",
		Components: []discord.LayoutComponent{
			discord.NewLabel("Agree", discord.CheckboxGroupComponent{
				CustomID:  "cb",
				Required:  true,
				MinValues: &minThree,
				MaxValues: 3,
				Options: []discord.CheckboxGroupOption{
					discord.NewCheckboxGroupOption("a", "A"),
					discord.NewCheckboxGroupOption("b", "B"),
					discord.NewCheckboxGroupOption("c", "C"),
				},
			}),
		},
	}); err != nil {
		t.Fatalf("expected valid 3-item required CheckboxGroup to pass, got %v", err)
	}
}

type fakeGuildState struct {
	features    []discord.GuildFeature
	channelCnt  int
	catChildren int
	botPerms    discord.Permissions
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

func TestPreflightChecker_ForumWithoutCommunityAndStrictPerms(t *testing.T) {
	guildID := snowflake.ID(100)

	state := fakeGuildState{
		features: nil, // non-Community guild
		botPerms: discord.PermissionManageChannels.Add(
			discord.PermissionViewChannel,
			discord.PermissionSendMessages,
			discord.PermissionSendMessagesInThreads,
			discord.PermissionManageThreads,
			discord.PermissionManageMessages,
			discord.PermissionReadMessageHistory,
		),
	}
	checker := discordsync.NewPreflightChecker(state)

	// Missing CreatePublicThreads & CreatePrivateThreads must be rejected with PreflightErrMissingBotPerms.
	if _, err := checker.CheckChannelCreate(guildID, "テスト板", "説明", true, nil); err == nil {
		t.Fatal("expected missing thread permissions error")
	}

	// Only CreatePublicThreads is still insufficient without CreatePrivateThreads (since overwrites deny it).
	state.botPerms = state.botPerms.Add(discord.PermissionCreatePublicThreads)
	checker = discordsync.NewPreflightChecker(state)
	if _, err := checker.CheckChannelCreate(guildID, "テスト板", "説明", true, nil); err == nil {
		t.Fatal("expected missing CreatePrivateThreads error")
	}

	// Non-Community guild WITH required permissions creates Forum channel directly (no fallback to text).
	state.botPerms = state.botPerms.Add(discord.PermissionCreatePrivateThreads)
	checker = discordsync.NewPreflightChecker(state)

	plan, err := checker.CheckChannelCreate(guildID, "テスト板", strings.Repeat("あ", 2000), true, nil)
	if err != nil {
		t.Fatalf("CheckChannelCreate failed on non-Community guild: %v", err)
	}
	if !plan.UseForum {
		t.Fatalf("expected non-Community guild to support forum channel, got %+v", plan)
	}

	// Over-length name/topic must be rejected instead of clamped.
	if _, err := checker.CheckChannelCreate(guildID, strings.Repeat("板", 101), "説明", true, nil); err == nil {
		t.Fatal("expected error for channel name > 100 runes")
	}
	if _, err := checker.CheckChannelCreate(guildID, "板", strings.Repeat("あ", 1025), false, nil); err == nil {
		t.Fatal("expected error for text channel topic > 1024 runes")
	}
	if _, err := checker.CheckChannelCreate(guildID, "板", strings.Repeat("あ", 4097), true, nil); err == nil {
		t.Fatal("expected error for forum channel topic > 4096 runes")
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

	guide := discordsync.RenderBoardGuide(board, mountID, discord.LocaleJapanese)
	if err := v.ValidateMessageCreate(guide); err != nil {
		t.Fatalf("guide validation failed: %v", err)
	}
	assertNoEmojiInLayout(t, guide.Components)

	notice := discordsync.RenderNotice(discord.LocaleJapanese, "Title", "Body @everyone", discordsync.ColorNotice)
	if notice.AllowedMentions == nil || len(notice.AllowedMentions.Parse) != 0 {
		t.Fatalf("expected RenderNotice to suppress mentions cleanly, got %+v", notice.AllowedMentions)
	}

	pm := bbs.NewPolicyManager(bbs.PolicyConfig{Version: "v2-test"})
	consentMsg := discordsync.RenderConsentGate(pm, discord.LocaleJapanese, "tok")
	if err := v.ValidateMessageCreate(consentMsg); err != nil {
		t.Fatalf("consent gate validation failed: %v", err)
	}
	assertNoEmojiInLayout(t, consentMsg.Components)
	if !strings.Contains(fmt.Sprintf("%#v", consentMsg), "/bbs/consent/open/v2-test/tok") {
		t.Fatal("expected RenderConsentGate button CustomID to include policy version token")
	}

	consentModal := discordsync.RenderConsentModal(pm, discord.LocaleJapanese, "tok")
	if err := v.ValidateModal(consentModal); err != nil {
		t.Fatalf("consent modal validation failed: %v", err)
	}
	if consentModal.CustomID != "/bbs/consent/modal/v2-test/tok" {
		t.Fatalf("unexpected consent modal CustomID: %q", consentModal.CustomID)
	}

	docView := discordsync.RenderPolicyDocumentView(pm, bbs.PolicyDocTerms, discord.LocaleJapanese)
	if err := v.ValidateMessageCreate(docView); err != nil {
		t.Fatalf("policy doc view validation failed: %v", err)
	}
	assertNoEmojiInLayout(t, docView.Components)

	// MediaGallery added for normal post, removed on Abone
	post := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  threadID,
		ResNumber: 1,
		Name:      "風吹けば名無し",
		AnonID:    "abcd1234",
		IsOP:      true,
		Body:      "画像テスト https://example.com/a!pic.png",
		CreatedAt: time.Now(),
	}
	postMsg := discordsync.RenderPostMessageCreate(post, thread, nil, discord.LocaleJapanese)
	if err := v.ValidateMessageCreate(postMsg); err != nil {
		t.Fatalf("post message failed: %v", err)
	}
	assertNoEmojiInLayout(t, postMsg.Components)

	sec, ok := postMsg.Components[0].(discord.SectionComponent)
	if !ok || len(sec.Components) != 2 {
		t.Fatalf("expected SectionComponent with 2 text displays as post header+body, got %#v", postMsg.Components[0])
	}
	menuBtn, ok := sec.Accessory.(discord.ButtonComponent)
	if !ok || menuBtn.Label != discordsync.PostMenuButtonLabel || menuBtn.CustomID != "/bbs/post/menu/"+post.ID.String() {
		t.Fatalf("expected post section accessory button %q -> /bbs/post/menu/%s, got %#v", discordsync.PostMenuButtonLabel, post.ID, sec.Accessory)
	}

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
	aboneSec, ok := aboneMsg.Components[0].(discord.SectionComponent)
	if !ok {
		t.Fatalf("expected SectionComponent on aboned post, got %#v", aboneMsg.Components[0])
	}
	if aboneBtn, ok := aboneSec.Accessory.(discord.ButtonComponent); !ok || aboneBtn.CustomID != "/bbs/post/menu/"+post.ID.String() {
		t.Fatalf("expected menu accessory button preserved on aboned post, got %#v", aboneSec.Accessory)
	}
	for _, c := range aboneMsg.Components {
		if c.Type() == discord.ComponentTypeMediaGallery {
			t.Fatal("MediaGallery must be removed when post is aboned")
		}
	}

	// Reader pagination & strict jump syntax
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
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(pageComps...).WithEphemeral(true)); err != nil {
		t.Fatalf("reader page failed validation: %v", err)
	}
	assertNoEmojiInLayout(t, pageComps)

	if got, err := discordsync.ParseReaderJumpInput("l5", 20); err != nil || got != 16 {
		t.Fatalf("expected l5 -> 16, nil; got %d, %v", got, err)
	}
	for _, badJump := range []string{"", "0", "99", "abc", "l", "l0", "l99"} {
		if _, err := discordsync.ParseReaderJumpInput(badJump, 20); !errors.Is(err, bbs.ErrInvalidReaderJump) {
			t.Fatalf("expected ErrInvalidReaderJump for %q, got %v", badJump, err)
		}
	}
}

type fakeTimer struct {
	fn      func()
	stopped atomic.Bool
}

func (f *fakeTimer) Stop() bool {
	return !f.stopped.Swap(true)
}

func TestDebouncerAndInvalidRequestGuard(t *testing.T) {
	var timers []*fakeTimer
	deb := discordsync.NewControlPanelDebouncerWithTimer(2500*time.Millisecond, func(_ time.Duration, f func()) discordsync.DebounceTimer {
		ft := &fakeTimer{fn: f}
		timers = append(timers, ft)
		return ft
	})
	defer deb.Stop()

	id := uuid.New()
	var calls atomic.Int32
	var gotIdx int
	for i := range 5 {
		idx := i
		deb.Schedule(id, func() {
			calls.Add(1)
			gotIdx = idx
		})
	}
	if len(timers) != 5 {
		t.Fatalf("expected 5 timer registrations, got %d", len(timers))
	}
	for i := range 4 {
		if !timers[i].stopped.Load() {
			t.Fatalf("expected superseded timer %d to be stopped", i)
		}
		timers[i].fn() // stale timer callback must be ignored even if fired after Stop
	}
	if calls.Load() != 0 {
		t.Fatalf("stale timers must not trigger callback, got %d calls", calls.Load())
	}
	timers[4].fn()
	if calls.Load() != 1 || gotIdx != 4 {
		t.Fatalf("expected latest scheduled callback (4) to run once, got calls=%d idx=%d", calls.Load(), gotIdx)
	}

	guard := discordsync.NewInvalidRequestGuard(10*time.Minute, 10, 2)
	guard.SetCircuitCooldown(1 * time.Minute)
	mountID := uuid.New()
	now := time.Now()

	err403 := &rest.Error{Response: &http.Response{StatusCode: 403, Header: http.Header{}}, Code: discordsync.JSONErrMissingPermissions}
	err503 := &rest.Error{Response: &http.Response{StatusCode: 503, Header: http.Header{}}}
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

	// After cooldown, 1 half-open probe is permitted; non-403 (503) + stale/non-probe calls must NOT clear another probe.
	afterCooldown := now.Add(2 * time.Minute)
	allowedA, seqA, releaseA := guard.AcquireMountWithProbe(mountID, afterCooldown)
	if !allowedA || seqA == 0 {
		t.Fatalf("expected first AcquireMountWithProbe after cooldown to permit half-open probe, allowed=%v seq=%d", allowedA, seqA)
	}
	// Pure IsMountOpenAt check shows mount still waiting/blocked while probe is active (since halfOpenProbe[mountID] != 0).
	if !guard.IsMountOpenAt(mountID, afterCooldown) {
		t.Fatal("expected IsMountOpenAt to report true while half-open probe is in flight")
	}
	if concurrent, _, _ := guard.AcquireMountWithProbe(mountID, afterCooldown); concurrent {
		t.Fatal("expected concurrent AcquireMountWithProbe during half-open probe to stay blocked")
	}
	// Non-probe RecordError (seq=0) during active probe A must NOT release probe A's marker!
	guard.RecordError(mountID, err503, afterCooldown)
	if stillBlocked, _, _ := guard.AcquireMountWithProbe(mountID, afterCooldown); stillBlocked {
		t.Fatal("non-probe 503 must not release another active half-open probe")
	}
	// When probe A records its own 503 error, its marker releases immediately so probe B can run.
	guard.RecordMountError(mountID, seqA, err503, afterCooldown)

	allowedB, seqB, releaseB := guard.AcquireMountWithProbe(mountID, afterCooldown)
	if !allowedB || seqB == 0 || seqB == seqA {
		t.Fatalf("expected probe B after probe A 503 failure, got allowed=%v seqB=%d", allowedB, seqB)
	}
	// Probe A's delayed defer releaseA() must NOT erase probe B's active marker (ABA check).
	releaseA()
	if blockedByB, _, _ := guard.AcquireMountWithProbe(mountID, afterCooldown); blockedByB {
		t.Fatal("stale probe A release() must not delete probe B's active marker")
	}
	// Standalone probe B failure with 503 clears B's own marker so retry C is permitted.
	guard.RecordMountError(mountID, seqB, err503, afterCooldown)
	releaseB()

	allowedC, seqC, releaseC := guard.AcquireMountWithProbe(mountID, afterCooldown)
	if !allowedC || seqC == 0 {
		t.Fatalf("expected probe C after probe B 503 failure, got allowed=%v seqC=%d", allowedC, seqC)
	}
	guard.RecordMountSuccess(mountID)
	releaseC()
	if guard.IsMountOpenAt(mountID, afterCooldown) {
		t.Fatal("expected RecordMountSuccess to close circuit breaker")
	}

	// Global threshold test with separate mounts: throttled AcquireMount must fail closed without leaking probe.
	smallGuard := discordsync.NewInvalidRequestGuard(10*time.Minute, 3, 2)
	for range 3 {
		smallGuard.RecordError(uuid.New(), err403, now)
	}
	if !smallGuard.IsThrottled(now) {
		t.Fatal("expected global InvalidRequestGuard to throttle at threshold 3")
	}
	if ok, rel := smallGuard.AcquireMount(uuid.New(), now); ok {
		rel()
		t.Fatal("expected throttled AcquireMount to deny execution")
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
		case discord.SectionComponent:
			for _, sc := range c.Components {
				if td, ok := sc.(discord.TextDisplayComponent); ok {
					if strings.Contains(td.Content, "components.bbs.") || containsJapaneseRune(td.Content) {
						t.Fatalf("unexpected untranslated or Japanese text in en-US Section TextDisplay: %q", td.Content)
					}
				}
			}
			if btn, ok := c.Accessory.(discord.ButtonComponent); ok {
				if strings.Contains(btn.Label, "components.bbs.") || containsJapaneseRune(btn.Label) {
					t.Fatalf("unexpected untranslated or Japanese text in en-US Section Button: %q", btn.Label)
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
	localBoard := bbs.Board{
		ID:    uuid.New(),
		Scope: bbs.ScopeLocal,
		Name:  "LocalTalk",
		Rules: board.Rules,
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
	archivedThread := thread
	archivedThread.Status = bbs.ThreadStatusArchived
	posts := []bbs.Post{
		{
			ID:         uuid.New(),
			ThreadID:   threadID,
			ResNumber:  1,
			Name:       bbs.DefaultAnonymousName,
			SlipText:   "abcd-efgh",
			AnonID:     "xyz12345",
			PosterHash: "hash001",
			IsOP:       true,
			Body:       "Hello >>1",
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			ThreadID:   threadID,
			ResNumber:  2,
			Abone:      true,
			PosterHash: "hash002",
			CreatedAt:  time.Now(),
		},
	}
	pm := bbs.NewPolicyManager(bbs.PolicyConfig{
		Version:  "2026-11-25",
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
			discordsync.RenderPostActionMenu(discordsync.PostActionMenuOptions{
				Board:    localBoard,
				Thread:   thread,
				Post:     posts[0],
				CanReply: true,
			}, loc),
			discordsync.RenderPostActionMenu(discordsync.PostActionMenuOptions{
				Board:       localBoard,
				Thread:      thread,
				Post:        posts[0],
				CanReply:    true,
				CanLocalMod: true,
			}, loc),
			discordsync.RenderPostActionMenu(discordsync.PostActionMenuOptions{
				Board:       localBoard,
				Thread:      archivedThread,
				Post:        posts[1],
				CanLocalMod: true,
			}, loc),
			discordsync.RenderPostActionMenu(discordsync.PostActionMenuOptions{
				Board:             board,
				Thread:            thread,
				Post:              posts[0],
				CanReply:          true,
				ShowGlobalModNote: true,
			}, loc),
			discordsync.RenderPolicyDocumentView(pm, bbs.PolicyDocTerms, loc),
		}
		for i, m := range msgs {
			if err := v.ValidateMessageCreate(m); err != nil {
				t.Fatalf("locale %s message %d failed validation: %v", loc, i, err)
			}
			assertNoEmojiInLayout(t, m.Components)
			if loc == discord.LocaleEnglishUS {
				assertLayoutENClean(t, m.Components)
			}
		}

		// Verify current doc switch button is disabled in RenderPolicyDocumentView
		docView := msgs[len(msgs)-1]
		docContainer := docView.Components[0].(discord.ContainerComponent)
		docSwitchRow := docContainer.Components[4].(discord.ActionRowComponent)
		if !docSwitchRow.Components[0].(discord.ButtonComponent).Disabled ||
			docSwitchRow.Components[1].(discord.ButtonComponent).Disabled ||
			docSwitchRow.Components[2].(discord.ButtonComponent).Disabled {
			t.Fatalf("expected only current policy doc button to be disabled in %s", loc)
		}

		modal := discordsync.RenderConsentModal(pm, loc, "tok123")
		if err := v.ValidateModal(modal); err != nil {
			t.Fatalf("locale %s consent modal failed validation: %v", loc, err)
		}
		if len(modal.Components) != 2 {
			t.Fatalf("expected 2 components in consent modal, got %d", len(modal.Components))
		}
		td := modal.Components[0].(discord.TextDisplayComponent)
		if !strings.Contains(td.Content, "https://example.com/terms") {
			t.Fatalf("expected consent modal TextDisplay to embed configured TermsURL, got %q", td.Content)
		}
		lbl := modal.Components[1].(discord.LabelComponent)
		cb := lbl.Component.(discord.CheckboxGroupComponent)
		if !cb.Required || cb.MinValues == nil || *cb.MinValues != 3 || cb.MaxValues != 3 || len(cb.Options) != 3 {
			t.Fatalf("unexpected CheckboxGroup configuration: %+v", cb)
		}
		for _, opt := range cb.Options {
			if opt.Default {
				t.Fatalf("consent checkbox option %q must default to false", opt.Value)
			}
			if containsPictographEmoji(opt.Label) || containsPictographEmoji(opt.Description) {
				t.Fatalf("unexpected emoji in option %+v", opt)
			}
			if loc == discord.LocaleEnglishUS && (containsJapaneseRune(opt.Label) || containsJapaneseRune(opt.Description)) {
				t.Fatalf("unexpected Japanese rune in en-US option %+v", opt)
			}
		}
		if containsPictographEmoji(modal.Title) || containsPictographEmoji(td.Content) || containsPictographEmoji(lbl.Label) || containsPictographEmoji(lbl.Description) {
			t.Fatalf("unexpected emoji in consent modal for %s", loc)
		}
		if loc == discord.LocaleEnglishUS {
			for _, txt := range []string{modal.Title, td.Content, lbl.Label, lbl.Description} {
				if strings.Contains(txt, "components.bbs.") || containsJapaneseRune(txt) {
					t.Fatalf("unexpected untranslated or Japanese text in en-US consent modal: %q", txt)
				}
			}
		}

		latestIdx := discordsync.ComputeLatestReaderStartIndex(thread, posts, nil, loc)
		readerComps, _, _ := discordsync.RenderReaderPage(thread, posts, latestIdx, nil, loc)
		if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(readerComps...).WithEphemeral(true)); err != nil {
			t.Fatalf("locale %s reader failed validation: %v", loc, err)
		}
		if loc == discord.LocaleEnglishUS {
			assertLayoutENClean(t, readerComps)
		}
	}

	pe := &discordsync.PreflightError{
		Kind:         discordsync.PreflightErrMissingBotPerms,
		MissingPerms: []string{"ManageChannels"},
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
	if err := v.ValidateMessageCreate(opMsg); err != nil {
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

	// Verify Prev navigation from startIndex=3 (res 4) targets res 3-3 (bounded), not res 2 or res 4.
	comps, _, _ := discordsync.RenderReaderPage(thread, posts, 3, resolveURL, discord.LocaleJapanese)
	nav := comps[1].(discord.ActionRowComponent)
	prevBtn := nav.Components[1].(discord.ButtonComponent)
	if !strings.HasSuffix(prevBtn.CustomID, "/3-3") {
		t.Fatalf("expected Prev button to target bounded range 3-3 right before page start 4, got customID=%q", prevBtn.CustomID)
	}
	_, prevStartRes, prevEndRes := discordsync.RenderReaderPageBounded(thread, posts, 2, 3, resolveURL, discord.LocaleJapanese)
	if prevStartRes != 3 || prevEndRes != 3 {
		t.Fatalf("expected Prev page at startIndex=2 bounded to 3 to render res 3..3, got %d..%d", prevStartRes, prevEndRes)
	}

	// Short 3-post thread: jumping to res 3 and pressing Prev yields bounded 1-2 so Next stays enabled to return to res 3.
	shortThread := bbs.Thread{
		ID:       uuid.New(),
		Title:    "Short Thread",
		ResCount: 3,
		Rules:    bbs.BoardRules{MaxRes: 1000}.Normalize(),
	}
	shortPosts := []bbs.Post{
		{ID: uuid.New(), ThreadID: shortThread.ID, ResNumber: 1, AnonID: "1", Body: "first", CreatedAt: time.Now()},
		{ID: uuid.New(), ThreadID: shortThread.ID, ResNumber: 2, AnonID: "2", Body: "second", CreatedAt: time.Now()},
		{ID: uuid.New(), ThreadID: shortThread.ID, ResNumber: 3, AnonID: "3", Body: "third", CreatedAt: time.Now()},
	}
	jump3Comps, s3, e3 := discordsync.RenderReaderPage(shortThread, shortPosts, 2, nil, discord.LocaleJapanese)
	if s3 != 3 || e3 != 3 {
		t.Fatalf("expected res 3 view to start/end at 3, got %d..%d", s3, e3)
	}
	prevShortBtn := jump3Comps[1].(discord.ActionRowComponent).Components[1].(discord.ButtonComponent)
	if !strings.HasSuffix(prevShortBtn.CustomID, "/1-2") {
		t.Fatalf("expected Prev from res 3 on 3-post thread to use bounded range /1-2, got %q", prevShortBtn.CustomID)
	}
	boundedComps, bStart, bEnd := discordsync.RenderReaderPageBounded(shortThread, shortPosts, 0, 2, nil, discord.LocaleJapanese)
	if bStart != 1 || bEnd != 2 {
		t.Fatalf("expected bounded Prev page to stop at res 2 (1..2), got %d..%d", bStart, bEnd)
	}
	nextShortBtn := boundedComps[1].(discord.ActionRowComponent).Components[2].(discord.ButtonComponent)
	if nextShortBtn.Disabled || !strings.HasSuffix(nextShortBtn.CustomID, "/3") {
		t.Fatalf("expected Next button on bounded 1..2 page to be enabled targeting /3, got disabled=%v customID=%q", nextShortBtn.Disabled, nextShortBtn.CustomID)
	}
}

func TestRenderPostHeader_TripAndCap(t *testing.T) {
	v := discordsync.PayloadValidator{}
	now := time.Date(2026, 11, 26, 22, 0, 0, 0, bbs.JSTLocation)
	rules := bbs.BoardRules{DefaultName: "風吹けば名無し"}.Normalize()
	thread := bbs.Thread{
		ID:       uuid.New(),
		BoardID:  uuid.New(),
		Title:    "キャップ・トリップテスト",
		Status:   bbs.ThreadStatusActive,
		ResCount: 1,
		Rules:    rules,
	}

	// 1. Genuine tripcode renders bold, while spoofed symbols & markdown in Name are neutralized.
	spoofPost := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  thread.ID,
		ResNumber: 1,
		Name:      "偽◆名★♦❖⬥✦⭑⭐**管理人**",
		Trip:      "◆AbcDefGhiJ",
		AnonID:    "12345678",
		Body:      "テスト本文",
		CreatedAt: now,
	}
	spoofHeader := discordsync.RenderPostHeader(spoofPost, rules, discord.LocaleJapanese)
	if strings.Contains(spoofHeader, "偽◆") || strings.Contains(spoofHeader, "★") {
		t.Fatalf("spoofed ◆/★ leaked into header: %q", spoofHeader)
	}
	if !strings.Contains(spoofHeader, "偽◇名☆◇◇◇☆☆☆\\*\\*管理人\\*\\* **◆AbcDefGhiJ**") {
		t.Fatalf("unexpected spoof-neutralized header: %q", spoofHeader)
	}

	// 2. Cap with empty name -> "**管理人 ★**" and no ID.
	capOnlyPost := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  thread.ID,
		ResNumber: 2,
		Cap:       "管理人",
		Body:      "公式告知",
		CreatedAt: now,
	}
	capOnlyHeader := discordsync.RenderPostHeader(capOnlyPost, rules, discord.LocaleJapanese)
	if !strings.Contains(capOnlyHeader, ":**管理人 ★**:") || strings.Contains(capOnlyHeader, "ID:") {
		t.Fatalf("unexpected cap-only header: %q", capOnlyHeader)
	}

	// 3. Cap with empty name + trip -> "**管理人 ★** **◆...**".
	capEmptyTripPost := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  thread.ID,
		ResNumber: 3,
		Trip:      "◆Trippppppp",
		Cap:       "管理人",
		Body:      "トリップ併用告知",
		CreatedAt: now,
	}
	capEmptyTripHeader := discordsync.RenderPostHeader(capEmptyTripPost, rules, discord.LocaleJapanese)
	if !strings.Contains(capEmptyTripHeader, ":**管理人 ★** **◆Trippppppp**:") {
		t.Fatalf("unexpected cap+trip header with empty name: %q", capEmptyTripHeader)
	}

	// 4. Cap with custom name + trip -> "さばふらい **◆...** **★管理人**".
	capNamedPost := bbs.Post{
		ID:        uuid.New(),
		ThreadID:  thread.ID,
		ResNumber: 4,
		Name:      "さばふらい",
		Trip:      "◆Trippppppp",
		Cap:       "管理人",
		Body:      "名前あり告知",
		CreatedAt: now,
	}
	capNamedHeader := discordsync.RenderPostHeader(capNamedPost, rules, discord.LocaleJapanese)
	if !strings.Contains(capNamedHeader, ":さばふらい **◆Trippppppp** **★管理人**:") || strings.Contains(capNamedHeader, "ID:") {
		t.Fatalf("unexpected named cap+trip header: %q", capNamedHeader)
	}

	// 5. All headers pass RenderPostMessageCreate validation and zero-emoji check.
	for _, p := range []bbs.Post{spoofPost, capOnlyPost, capEmptyTripPost, capNamedPost} {
		mc := discordsync.RenderPostMessageCreate(p, thread, nil, discord.LocaleJapanese)
		if err := v.ValidateMessageCreate(mc); err != nil {
			t.Fatalf("RenderPostMessageCreate validation failed for res %d: %v", p.ResNumber, err)
		}
		assertNoEmojiInLayout(t, mc.Components)
	}
}

func TestRenderReaderPage_UniqueCustomIDsAllStates(t *testing.T) {
	v := discordsync.PayloadValidator{}
	now := time.Date(2026, 11, 26, 23, 0, 0, 0, bbs.JSTLocation)
	rules := bbs.BoardRules{MaxRes: 1000}.Normalize()

	assertValidReaderComps := func(label string, comps []discord.LayoutComponent) {
		t.Helper()
		mc := discord.NewMessageCreateV2(comps...).WithEphemeral(true)
		if err := v.ValidateMessageCreate(mc); err != nil {
			t.Fatalf("%s failed ValidateMessageCreate: %v", label, err)
		}
		assertNoEmojiInLayout(t, comps)
		nav, ok := comps[1].(discord.ActionRowComponent)
		if !ok || len(nav.Components) != 5 {
			t.Fatalf("%s expected 5-button nav row, got %#v", label, comps)
		}
		seen := make(map[string]struct{}, len(nav.Components))
		for _, ic := range nav.Components {
			btn := ic.(discord.ButtonComponent)
			if btn.CustomID == "" {
				t.Fatalf("%s has empty button custom_id", label)
			}
			if _, dup := seen[btn.CustomID]; dup {
				t.Fatalf("%s has duplicate button custom_id %q", label, btn.CustomID)
			}
			seen[btn.CustomID] = struct{}{}
		}
	}

	// 1. 1-post thread (both atStart and atEnd on page 1).
	thSingle := bbs.Thread{ID: uuid.New(), Title: "Single Post Thread", ResCount: 1, Rules: rules}
	singlePosts := []bbs.Post{
		{ID: uuid.New(), ThreadID: thSingle.ID, ResNumber: 1, AnonID: "1", Body: "only post", CreatedAt: now},
	}
	cSingle, sRes, eRes := discordsync.RenderReaderPage(thSingle, singlePosts, 0, nil, discord.LocaleJapanese)
	if sRes != 1 || eRes != 1 {
		t.Fatalf("expected 1..1 for single post thread, got %d..%d", sRes, eRes)
	}
	assertValidReaderComps("single_post_page1", cSingle)

	// 2. Multi-page thread: verify page 1, middle page, and latest page all have unique custom_ids.
	thMulti := bbs.Thread{ID: uuid.New(), Title: "Multi Page Thread", ResCount: 10, Rules: rules}
	multiPosts := make([]bbs.Post, 10)
	for i := range multiPosts {
		multiPosts[i] = bbs.Post{
			ID:        uuid.New(),
			ThreadID:  thMulti.ID,
			ResNumber: uint32(i + 1),
			AnonID:    fmt.Sprintf("id%d", i),
			Body:      strings.Repeat("本文", 350),
			CreatedAt: now,
		}
	}
	cFirst, _, firstEnd := discordsync.RenderReaderPage(thMulti, multiPosts, 0, nil, discord.LocaleJapanese)
	assertValidReaderComps("multi_first_page", cFirst)

	if firstEnd < 10 {
		cMid, _, midEnd := discordsync.RenderReaderPage(thMulti, multiPosts, int(firstEnd), nil, discord.LocaleJapanese)
		if midEnd >= 10 {
			t.Fatalf("expected middle page before end, got midEnd=%d", midEnd)
		}
		assertValidReaderComps("multi_middle_page", cMid)
	}

	latestIdx := discordsync.ComputeLatestReaderStartIndex(thMulti, multiPosts, nil, discord.LocaleJapanese)
	cLatest, _, latestEnd := discordsync.RenderReaderPage(thMulti, multiPosts, latestIdx, nil, discord.LocaleJapanese)
	if latestEnd != 10 {
		t.Fatalf("expected latest page to end at 10, got %d", latestEnd)
	}
	assertValidReaderComps("multi_latest_page", cLatest)
}

func TestRenderSLIP_ModesAndHeader(t *testing.T) {
	v := discordsync.PayloadValidator{}
	boardID := uuid.New()
	mountID := uuid.New()
	threadID := uuid.New()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, bbs.JSTLocation)

	// 1. RenderBoardGuide handles none / vvv / vvvvv cleanly for JA and EN.
	for _, slip := range []bbs.SLIPLevel{bbs.SLIPNone, bbs.SLIPVVV, bbs.SLIPVVVVV} {
		board := bbs.Board{
			ID:    boardID,
			Scope: bbs.ScopeLocal,
			Name:  "SLIP Test Board",
			Rules: bbs.BoardRules{DefaultName: bbs.DefaultAnonymousName, SLIP: slip}.Normalize(),
		}
		for _, loc := range []discord.Locale{discord.LocaleJapanese, discord.LocaleEnglishUS} {
			guide := discordsync.RenderBoardGuide(board, mountID, loc)
			if err := v.ValidateMessageCreate(guide); err != nil {
				t.Fatalf("RenderBoardGuide(%v, %s) failed validation: %v", slip, loc, err)
			}
			assertNoEmojiInLayout(t, guide.Components)
			if loc == discord.LocaleEnglishUS {
				assertLayoutENClean(t, guide.Components)
			}
		}
	}

	// 2. RenderPostHeader formats SLIP without a redundant fixed prefix.
	rules := bbs.BoardRules{DefaultName: bbs.DefaultAnonymousName, SLIP: bbs.SLIPVVVVV}.Normalize()
	vvvHeader := discordsync.RenderPostHeader(bbs.Post{
		ID: uuid.New(), ThreadID: threadID, ResNumber: 1, SlipText: "ﾌﾞｰｽﾄｳｵ", AnonID: "id1", CreatedAt: now,
	}, rules, discord.LocaleJapanese)
	if !strings.Contains(vvvHeader, "風吹けば名無し (ﾌﾞｰｽﾄｳｵ):") {
		t.Fatalf("unexpected vvv header: %q", vvvHeader)
	}
	vvvvvHeader := discordsync.RenderPostHeader(bbs.Post{
		ID: uuid.New(), ThreadID: threadID, ResNumber: 2, SlipText: "ｵｼｮｳｶﾞﾂ ﾌﾞｰｽﾄｳｵ Ab12-Cd34", AnonID: "id1", CreatedAt: now,
	}, rules, discord.LocaleEnglishUS)
	if !strings.Contains(vvvvvHeader, "Anonymous (ｵｼｮｳｶﾞﾂ ﾌﾞｰｽﾄｳｵ Ab12\\-Cd34):") {
		t.Fatalf("unexpected vvvvv event header: %q", vvvvvHeader)
	}
}

func TestRenderReaderPage_SectionsAndSixPostPagination(t *testing.T) {
	v := discordsync.PayloadValidator{}
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, bbs.JSTLocation)
	rules := bbs.BoardRules{MaxRes: 1000}.Normalize()
	thread := bbs.Thread{ID: uuid.New(), Title: "Short Reader Pagination", ResCount: 10, Rules: rules}
	posts := make([]bbs.Post, 10)
	for i := range posts {
		posts[i] = bbs.Post{
			ID:        uuid.New(),
			ThreadID:  thread.ID,
			ResNumber: uint32(i + 1),
			AnonID:    fmt.Sprintf("id%d", i),
			Body:      fmt.Sprintf("short body %d", i+1),
			CreatedAt: now,
		}
	}

	page1, s1, e1 := discordsync.RenderReaderPage(thread, posts, 0, nil, discord.LocaleJapanese)
	if s1 != 1 || e1 != 6 {
		t.Fatalf("expected page 1 of short posts to pack 1..6 under component cap, got %d..%d", s1, e1)
	}
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(page1...).WithEphemeral(true)); err != nil {
		t.Fatalf("page 1 failed ValidateMessageCreate: %v", err)
	}
	assertNoEmojiInLayout(t, page1)

	container, ok := page1[0].(discord.ContainerComponent)
	if !ok {
		t.Fatalf("expected ContainerComponent at index 0, got %#v", page1[0])
	}
	seenMenuIDs := make(map[string]struct{})
	for _, sc := range container.Components {
		sec, ok := sc.(discord.SectionComponent)
		if !ok {
			continue
		}
		btn, ok := sec.Accessory.(discord.ButtonComponent)
		if !ok || btn.Label != discordsync.PostMenuButtonLabel || !strings.HasPrefix(btn.CustomID, "/bbs/post/menu/") {
			t.Fatalf("unexpected reader section accessory: %#v", sec.Accessory)
		}
		if _, dup := seenMenuIDs[btn.CustomID]; dup {
			t.Fatalf("duplicate menu button custom_id %q in reader page", btn.CustomID)
		}
		seenMenuIDs[btn.CustomID] = struct{}{}
	}
	if len(seenMenuIDs) != 6 {
		t.Fatalf("expected 6 Section menu buttons on page 1, got %d", len(seenMenuIDs))
	}

	page2, s2, e2 := discordsync.RenderReaderPage(thread, posts, 6, nil, discord.LocaleJapanese)
	if s2 != 7 || e2 != 10 {
		t.Fatalf("expected page 2 to cover 7..10, got %d..%d", s2, e2)
	}
	if err := v.ValidateMessageCreate(discord.NewMessageCreateV2(page2...).WithEphemeral(true)); err != nil {
		t.Fatalf("page 2 failed ValidateMessageCreate: %v", err)
	}

	prevBtn := page2[1].(discord.ActionRowComponent).Components[1].(discord.ButtonComponent)
	if !strings.HasSuffix(prevBtn.CustomID, "/1-6") {
		t.Fatalf("expected Prev button on page 2 (7..10) to target /1-6, got %q", prevBtn.CustomID)
	}
	latestIdx := discordsync.ComputeLatestReaderStartIndex(thread, posts, nil, discord.LocaleJapanese)
	if latestIdx != 4 {
		// Backward pack from post 10 fits 6 posts (5..10, index 4)
		t.Fatalf("expected ComputeLatestReaderStartIndex for 10 short posts to pack 5..10 (idx=4), got %d", latestIdx)
	}
	_, ls, le := discordsync.RenderReaderPage(thread, posts, latestIdx, nil, discord.LocaleJapanese)
	if ls != 5 || le != 10 {
		t.Fatalf("expected latest page 5..10, got %d..%d", ls, le)
	}
}
