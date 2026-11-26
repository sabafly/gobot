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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	bbscore "github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
	"github.com/sabafly/gobot/internal/i18n"
)

func TestPendingActionTokensAndSlashRuleValidation(t *testing.T) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}
	sys := NewSystem(&components.Components{}, nil, nil, nil)
	u1 := snowflake.ID(101)
	u2 := snowflake.ID(202)
	target := uuid.New()

	tok := sys.savePending(u1, pendingOpenReader, target, "latest")
	if tok == "" {
		t.Fatal("expected non-empty token")
	}
	if _, ok := sys.popPending(tok, u2); ok {
		t.Fatal("another user must not consume u1's pending token")
	}
	act, ok := sys.popPending(tok, u1)
	if !ok {
		t.Fatal("expected u1 to consume pending token")
	}
	if act.Kind != pendingOpenReader || act.TargetID != target || act.Page != "latest" {
		t.Fatalf("unexpected pending action: %+v", act)
	}
	if _, ok := sys.popPending(tok, u1); ok {
		t.Fatal("consumed token must be single-use")
	}

	rawJSON := func(v any) json.RawMessage {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal option value failed: %v", err)
		}
		return b
	}

	// archive_days=0 must set DatOchiDisabled (-1) without Normalize overwriting it.
	dataDisabled := discord.SlashCommandInteractionData{
		Options: map[string]discord.SlashCommandOption{
			"archive_days": {Type: discord.ApplicationCommandOptionTypeInt, Value: rawJSON(0)},
			"max_res":      {Type: discord.ApplicationCommandOptionTypeInt, Value: rawJSON(500)},
			"slip":         {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("vvvvv")},
		},
	}
	rules, err := parseBoardRulesFromSlash(dataDisabled, discord.LocaleJapanese)
	if err != nil {
		t.Fatalf("parseBoardRulesFromSlash valid inputs failed: %v", err)
	}
	if rules.DatOchiIdleHours != bbscore.DatOchiDisabled || rules.DatOchiLowResHours != bbscore.DatOchiDisabled || rules.MaxRes != 500 || rules.SLIP != bbscore.SLIPVVVVV {
		t.Fatalf("unexpected parsed rules: %+v", rules)
	}

	// Out-of-range slash rule values must be rejected explicitly (no silent clamping/ignoring).
	for name, opts := range map[string]map[string]discord.SlashCommandOption{
		"max_res_low":   {"max_res": {Type: discord.ApplicationCommandOptionTypeInt, Value: rawJSON(10)}},
		"max_res_high":  {"max_res": {Type: discord.ApplicationCommandOptionTypeInt, Value: rawJSON(2000)}},
		"archive_neg":   {"archive_days": {Type: discord.ApplicationCommandOptionTypeInt, Value: rawJSON(-1)}},
		"archive_high":  {"archive_days": {Type: discord.ApplicationCommandOptionTypeInt, Value: rawJSON(999)}},
		"invalid_slip":  {"slip": {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("invalid")}},
		"long_def_name": {"default_name": {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON(strings.Repeat("あ", 60))}},
	} {
		if _, err := parseBoardRulesFromSlash(discord.SlashCommandInteractionData{Options: opts}, discord.LocaleJapanese); err == nil {
			t.Fatalf("expected parseBoardRulesFromSlash(%s) to reject invalid option", name)
		}
	}

	// parseUpdateBoardInput also rejects empty default_name or invalid slip.
	if _, err := parseUpdateBoardInput(discord.SlashCommandInteractionData{
		Options: map[string]discord.SlashCommandOption{
			"default_name": {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("   ")},
		},
	}); err == nil {
		t.Fatal("expected parseUpdateBoardInput to reject whitespace-only default_name")
	}
	if _, err := parseUpdateBoardInput(discord.SlashCommandInteractionData{
		Options: map[string]discord.SlashCommandOption{
			"slip": {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("bad")},
		},
	}); err == nil {
		t.Fatal("expected parseUpdateBoardInput to reject invalid slip")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResp(status int, body any) *http.Response {
	raw, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(raw)),
	}
}

func newInternalMockClient(t *testing.T, rt roundTripFunc) *bot.Client {
	t.Helper()
	httpClient := &http.Client{Transport: rt}
	return &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Rest:          rest.New(rest.NewClient("Bot test-token", rest.WithHTTPClient(httpClient))),
		Caches:        cache.New(),
	}
}

func newInternalTestEnv(t *testing.T, pmCfg bbscore.PolicyConfig, operators ...snowflake.ID) (*gormstore.Provider, *bbscore.Service, *System) {
	t.Helper()
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	store := gormstore.New(db)
	reg := bbscore.NewProviderRegistry(bbscore.DefaultProviderKey)
	reg.Register(store)
	idGen := bbscore.NewIdentityGenerator([]byte("internal-test-secret-32-bytes!!!"))
	pm := bbscore.NewPolicyManager(pmCfg)
	svc := bbscore.NewService(reg, store, idGen, pm, operators)
	syncer := discordsync.NewSyncer(svc)
	sys := NewSystem(&components.Components{}, svc, syncer, nil)
	return store, svc, sys
}

func newInternalComponentEvent(
	t *testing.T,
	client *bot.Client,
	guildID snowflake.ID,
	userID snowflake.ID,
	customID string,
	vars map[string]string,
	onRespond func(discord.InteractionResponseType, any),
) *handler.ComponentEvent {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"type":           discord.InteractionTypeComponent,
		"id":             snowflake.New(time.Now()).String(),
		"application_id": "999000",
		"token":          "test-interaction-token",
		"guild_id":       guildID.String(),
		"channel":        map[string]any{"id": "555001", "type": 0},
		"member":         map[string]any{"user": map[string]any{"id": userID.String(), "username": "tester"}},
		"locale":         "ja",
		"data": map[string]any{
			"component_type": discord.ComponentTypeButton,
			"custom_id":      customID,
		},
	})
	if err != nil {
		t.Fatalf("marshal interaction failed: %v", err)
	}
	var ci discord.ComponentInteraction
	if err := json.Unmarshal(payload, &ci); err != nil {
		t.Fatalf("unmarshal ComponentInteraction failed: %v", err)
	}
	return &handler.ComponentEvent{
		ComponentInteractionCreate: &events.ComponentInteractionCreate{
			GenericEvent:         events.NewGenericEvent(client, 0, 0),
			ComponentInteraction: ci,
			Respond: func(respType discord.InteractionResponseType, data discord.InteractionResponseData, opts ...rest.RequestOpt) error {
				if onRespond != nil {
					onRespond(respType, data)
				}
				return nil
			},
		},
		Vars: vars,
		Ctx:  context.Background(),
	}
}

func hasJapaneseRuneInternal(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}

func TestDescribeBBSError_LocalizationAndSpecificity(t *testing.T) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}

	sentinels := []struct {
		name              string
		err               error
		distinctFromInput bool
	}{
		{"guild_not_enabled", bbscore.ErrGuildNotEnabled, true},
		{"consent_required", bbscore.ErrConsentRequired, true},
		{"stale_policy_version", bbscore.ErrStalePolicyVersion, true},
		{"banned", bbscore.ErrBanned, true},
		{"rate_limited", bbscore.ErrRateLimited, true},
		{"thread_archived", bbscore.ErrThreadArchived, true},
		{"max_res_reached", bbscore.ErrMaxResReached, true},
		{"board_locked", bbscore.ErrBoardLocked, true},
		{"unauthorized", bbscore.ErrUnauthorized, true},
		{"already_exists", bbscore.ErrAlreadyExists, true},
		{"not_found", bbscore.ErrNotFound, true},
		{"source_mount_inactive", discordsync.ErrSourceMountInactive, true},
		{"invalid_category_name", bbscore.ErrInvalidCategoryName, true},
		{"invalid_board_name", bbscore.ErrInvalidBoardName, true},
		{"invalid_board_rules", bbscore.ErrInvalidBoardRules, true},
		{"invalid_title_length", bbscore.ErrInvalidTitleLength, true},
		{"invalid_name_length", bbscore.ErrInvalidNameLength, true},
		{"invalid_body_length", bbscore.ErrInvalidBodyLength, true},
		{"formatted_body_too_long", bbscore.ErrFormattedBodyTooLong, true},
		{"invalid_mail", bbscore.ErrInvalidMail, true},
		{"invalid_name_command", bbscore.ErrInvalidNameCommand, true},
		{"invalid_extend_command", bbscore.ErrInvalidExtendCommand, true},
		{"invalid_reader_jump", bbscore.ErrInvalidReaderJump, true},
		{"report_reason_required", bbscore.ErrReportReasonRequired, true},
		{"guild_id_required", bbscore.ErrGuildIDRequired, true},
		{"board_scope_mismatch", bbscore.ErrBoardScopeMismatch, true},
		{"board_provider_mismatch", bbscore.ErrBoardProviderMismatch, true},
		{"invalid_ban_duration", bbscore.ErrInvalidBanDuration, true},
		{"invalid_input", bbscore.ErrInvalidInput, false},
		{"preflight_channel_name_too_long", &discordsync.PreflightError{Kind: discordsync.PreflightErrChannelNameTooLong, Max: 100, Current: 120}, true},
		{"preflight_channel_topic_too_long", &discordsync.PreflightError{Kind: discordsync.PreflightErrChannelTopicTooLong, Max: 1024, Current: 1100}, true},
	}

	jaInternal := describeBBSError(discord.LocaleJapanese, errors.New("unexpected"))
	enInternal := describeBBSError(discord.LocaleEnglishUS, errors.New("unexpected"))
	jaInvalid := describeBBSError(discord.LocaleJapanese, bbscore.ErrInvalidInput)
	enInvalid := describeBBSError(discord.LocaleEnglishUS, bbscore.ErrInvalidInput)

	for _, tc := range sentinels {
		ja := describeBBSError(discord.LocaleJapanese, tc.err)
		en := describeBBSError(discord.LocaleEnglishUS, tc.err)
		if ja == "" || ja == jaInternal {
			t.Fatalf("%s ja message invalid or fell back to internal: %q", tc.name, ja)
		}
		if en == "" || en == enInternal {
			t.Fatalf("%s en message invalid or fell back to internal: %q", tc.name, en)
		}
		if hasJapaneseRuneInternal(en) {
			t.Fatalf("%s en message contains Japanese runes: %q", tc.name, en)
		}
		if tc.distinctFromInput {
			if ja == jaInvalid {
				t.Fatalf("%s ja message fell back to generic ErrInvalidInput: %q", tc.name, ja)
			}
			if en == enInvalid {
				t.Fatalf("%s en message fell back to generic ErrInvalidInput: %q", tc.name, en)
			}
		}
	}
}

func TestRefreshLocalMountGuide_ForumPinnedPostAndTextControl(t *testing.T) {
	ctx := context.Background()
	store, svc, sys := newInternalTestEnv(t, bbscore.PolicyConfig{})
	cmd := &bbsCommand{sys: sys}

	guildID := snowflake.ID(300001)
	if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: 1}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	cat, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{
		Scope:   bbscore.ScopeLocal,
		GuildID: &guildID,
		Name:    "雑談カテゴリ",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{
		CategoryID:  cat.ID,
		Scope:       bbscore.ScopeLocal,
		GuildID:     &guildID,
		Name:        "テスト板",
		Description: "説明文",
		Rules:       bbscore.BoardRules{MaxRes: 100}.Normalize(),
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}

	pinnedID := snowflake.ID(700001)
	forumChID := snowflake.ID(700000)
	mount := &bbscore.BoardMount{
		ID:           uuid.New(),
		BoardID:      board.ID,
		GuildID:      guildID,
		ChannelID:    forumChID,
		IsForum:      true,
		PinnedPostID: &pinnedID,
		Active:       true,
		Locale:       "ja",
		CreatedAt:    time.Now(),
	}
	if err := store.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}

	var unarchived, updatedStarter atomic.Bool
	var restCalls atomic.Int32
	client := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		restCalls.Add(1)
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		switch {
		case req.Method == http.MethodGet && p == "/channels/"+pinnedID.String():
			return jsonResp(http.StatusOK, map[string]any{
				"id":              pinnedID.String(),
				"type":            11,
				"thread_metadata": map[string]any{"archived": true, "locked": false},
			}), nil
		case req.Method == http.MethodPatch && p == "/channels/"+pinnedID.String():
			unarchived.Store(true)
			return jsonResp(http.StatusOK, map[string]any{
				"id":              pinnedID.String(),
				"type":            11,
				"thread_metadata": map[string]any{"archived": false, "locked": false},
			}), nil
		case req.Method == http.MethodPatch && p == "/channels/"+pinnedID.String()+"/messages/"+pinnedID.String():
			updatedStarter.Store(true)
			return jsonResp(http.StatusOK, map[string]any{"id": pinnedID.String()}), nil
		default:
			return jsonResp(http.StatusNotFound, map[string]any{"message": "unexpected path: " + req.Method + " " + p}), nil
		}
	})

	cmd.refreshLocalMountGuide(ctx, client, guildID, *board, discord.LocaleJapanese)
	if !unarchived.Load() || !updatedStarter.Load() {
		t.Fatalf("expected archived forum PinnedPostID to be unarchived (%v) and starter updated (%v)", unarchived.Load(), updatedStarter.Load())
	}

	// Inactive mount must not trigger any REST calls.
	if err := store.DeactivateBoardMount(ctx, mount.ID); err != nil {
		t.Fatalf("DeactivateBoardMount failed: %v", err)
	}
	beforeInactive := restCalls.Load()
	cmd.refreshLocalMountGuide(ctx, client, guildID, *board, discord.LocaleJapanese)
	if restCalls.Load() != beforeInactive {
		t.Fatalf("expected 0 REST calls for inactive mount, got %d", restCalls.Load()-beforeInactive)
	}

	// Text mount with ControlMessageID updates control message in ChannelID.
	guildTextID := snowflake.ID(300002)
	if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{GuildID: guildTextID, Enabled: true, ActorUserID: 1}); err != nil {
		t.Fatalf("SetupGuild text failed: %v", err)
	}
	cat2, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{
		Scope:   bbscore.ScopeLocal,
		GuildID: &guildTextID,
		Name:    "テキストカテゴリ",
	})
	if err != nil {
		t.Fatalf("CreateCategory 2 failed: %v", err)
	}
	board2, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{
		CategoryID: cat2.ID,
		Scope:      bbscore.ScopeLocal,
		GuildID:    &guildTextID,
		Name:       "テキスト板",
	})
	if err != nil {
		t.Fatalf("CreateBoard 2 failed: %v", err)
	}
	textChID := snowflake.ID(800000)
	ctrlMsgID := snowflake.ID(800001)
	if err := store.CreateBoardMount(ctx, &bbscore.BoardMount{
		ID:               uuid.New(),
		BoardID:          board2.ID,
		GuildID:          guildTextID,
		ChannelID:        textChID,
		IsForum:          false,
		ControlMessageID: &ctrlMsgID,
		Active:           true,
		Locale:           "en-US",
		CreatedAt:        time.Now(),
	}); err != nil {
		t.Fatalf("CreateBoardMount 2 failed: %v", err)
	}
	var updatedControl atomic.Bool
	textClient := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		if req.Method == http.MethodPatch && p == "/channels/"+textChID.String()+"/messages/"+ctrlMsgID.String() {
			updatedControl.Store(true)
			return jsonResp(http.StatusOK, map[string]any{"id": ctrlMsgID.String()}), nil
		}
		return jsonResp(http.StatusNotFound, map[string]any{"message": "unexpected: " + req.Method + " " + p}), nil
	})
	cmd.refreshLocalMountGuide(ctx, textClient, guildTextID, *board2, discord.LocaleEnglishUS)
	if !updatedControl.Load() {
		t.Fatal("expected text mount ControlMessageID to be updated")
	}
}

func TestConsentVersionMismatch_And_AdminAboneSyncFailure(t *testing.T) {
	ctx := context.Background()
	opID := snowflake.ID(900001)
	userID := snowflake.ID(900002)
	guildID := snowflake.ID(900003)

	store, svc, sys := newInternalTestEnv(t, bbscore.PolicyConfig{Version: "2026-02"}, opID)
	adminCmd := &adminCommand{sys: sys}

	// 1. Stale version in /bbs/consent/accept/{version}/{token} must reject without recording consent.
	var lastRespBody string
	mockClient := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			b, _ := io.ReadAll(req.Body)
			if len(b) > 0 {
				lastRespBody = string(b)
			}
		}
		return jsonResp(http.StatusOK, map[string]any{"id": "1"}), nil
	})
	captureRespond := func(_ discord.InteractionResponseType, data any) {
		if data != nil {
			raw, _ := json.Marshal(data)
			lastRespBody = string(raw)
		}
	}

	staleEv := newInternalComponentEvent(t, mockClient, guildID, userID, "/bbs/consent/accept/2025-01/none", map[string]string{
		"version": "2025-01",
		"token":   "none",
	}, captureRespond)
	if err := sys.handleConsentAccept(staleEv); err != nil {
		t.Fatalf("handleConsentAccept stale returned err: %v", err)
	}
	if _, err := svc.CheckConsent(ctx, userID); !errors.Is(err, bbscore.ErrConsentRequired) {
		t.Fatalf("stale policy version must not record user consent, CheckConsent returned: %v", err)
	}
	if !strings.Contains(lastRespBody, i18n.TranslateText(discord.LocaleJapanese, "components.bbs.consent.stale_version")) {
		t.Fatalf("expected stale_version notice in response, got %s", lastRespBody)
	}

	validTok := bbscore.PolicyVersionToken("2026-02")
	validEv := newInternalComponentEvent(t, mockClient, guildID, userID, "/bbs/consent/accept/"+validTok+"/none", map[string]string{
		"version": validTok,
		"token":   "none",
	}, captureRespond)
	if err := sys.handleConsentAccept(validEv); err != nil {
		t.Fatalf("handleConsentAccept current returned err: %v", err)
	}
	if _, err := svc.CheckConsent(ctx, userID); err != nil {
		t.Fatalf("expected current policy version %q to record consent, err=%v", validTok, err)
	}

	// 2. Admin abone button must keep Report Open if Discord SyncAbone fails, and Resolve when it succeeds.
	if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: opID}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	gCat, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{Scope: bbscore.ScopeGlobal, Name: "GlobalCat"})
	if err != nil {
		t.Fatalf("CreateCategory global failed: %v", err)
	}
	gBoard, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{CategoryID: gCat.ID, Scope: bbscore.ScopeGlobal, Name: "GlobalBoard"})
	if err != nil {
		t.Fatalf("CreateBoard global failed: %v", err)
	}
	mountID := uuid.New()
	if err := store.CreateBoardMount(ctx, &bbscore.BoardMount{
		ID:        mountID,
		BoardID:   gBoard.ID,
		GuildID:   guildID,
		ChannelID: snowflake.ID(600001),
		Active:    true,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}

	authorThread := snowflake.ID(900010)
	authorPost1 := snowflake.ID(900011)
	authorPost2 := snowflake.ID(900012)
	for _, uid := range []snowflake.ID{authorThread, authorPost1, authorPost2} {
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
	}

	_, th, opPost, err := svc.CreateThread(ctx, bbscore.CreateThreadInput{
		BoardID: gBoard.ID,
		GuildID: guildID,
		UserID:  authorThread,
		Title:   "スレッド",
		RawBody: "1レス目",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	discordThreadID := snowflake.ID(600010)
	mirrorID := uuid.New()
	if err := store.CreateThreadMirror(ctx, &bbscore.ThreadMirror{
		ID:              mirrorID,
		ThreadID:        th.ID,
		BoardMountID:    mountID,
		GuildID:         guildID,
		DiscordThreadID: discordThreadID,
		CreatedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("CreateThreadMirror failed: %v", err)
	}

	// Create two posts + two open reports: one tested under REST 500 failure, one under REST 200 success.
	_, _, post1, _, err := svc.CreatePost(ctx, bbscore.CreatePostInput{ThreadID: th.ID, GuildID: guildID, UserID: authorPost1, RawBody: "bad 1"})
	if err != nil {
		t.Fatalf("CreatePost 1 failed: %v", err)
	}
	msg1ID := snowflake.ID(600101)
	if err := store.CreatePostMirror(ctx, &bbscore.PostMirror{
		ID: uuid.New(), PostID: post1.ID, ThreadMirrorID: mirrorID, GuildID: guildID,
		DiscordMessageID: msg1ID, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreatePostMirror 1 failed: %v", err)
	}
	rep1, err := svc.CreateReport(ctx, guildID, authorThread, post1.ID, "spam 1")
	if err != nil {
		t.Fatalf("CreateReport 1 failed: %v", err)
	}

	_, _, post2, _, err := svc.CreatePost(ctx, bbscore.CreatePostInput{ThreadID: th.ID, GuildID: guildID, UserID: authorPost2, RawBody: "bad 2"})
	if err != nil {
		t.Fatalf("CreatePost 2 failed: %v", err)
	}
	msg2ID := snowflake.ID(600102)
	if err := store.CreatePostMirror(ctx, &bbscore.PostMirror{
		ID: uuid.New(), PostID: post2.ID, ThreadMirrorID: mirrorID, GuildID: guildID,
		DiscordMessageID: msg2ID, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreatePostMirror 2 failed: %v", err)
	}
	rep2, err := svc.CreateReport(ctx, guildID, authorPost1, post2.ID, "spam 2")
	if err != nil {
		t.Fatalf("CreateReport 2 failed: %v", err)
	}
	_ = opPost

	var failMessageSync atomic.Bool
	failMessageSync.Store(true)
	aboneClient := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		if strings.HasPrefix(p, "/webhooks/") {
			if req.Body != nil {
				b, _ := io.ReadAll(req.Body)
				if len(b) > 0 {
					lastRespBody = string(b)
				}
			}
			return jsonResp(http.StatusOK, map[string]any{"id": "1"}), nil
		}
		if req.Method == http.MethodGet && p == "/channels/"+discordThreadID.String() {
			return jsonResp(http.StatusOK, map[string]any{"id": discordThreadID.String(), "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil
		}
		if req.Method == http.MethodPatch && strings.HasPrefix(p, "/channels/"+discordThreadID.String()+"/messages/") {
			if failMessageSync.Load() {
				return jsonResp(http.StatusInternalServerError, map[string]any{"message": "sync fail"}), nil
			}
			return jsonResp(http.StatusOK, map[string]any{"id": msg2ID.String()}), nil
		}
		return jsonResp(http.StatusOK, map[string]any{"id": "1"}), nil
	})

	evFail := newInternalComponentEvent(t, aboneClient, guildID, opID, "/bbs/admin/abone/"+rep1.ID.String(), map[string]string{"report_id": rep1.ID.String()}, captureRespond)
	if err := adminCmd.handleAdminAboneButton(evFail); err != nil {
		t.Fatalf("handleAdminAboneButton (fail path) returned err: %v", err)
	}
	storedRep1, err := store.GetReport(ctx, rep1.ID)
	if err != nil {
		t.Fatalf("GetReport 1 failed: %v", err)
	}
	if storedRep1.Status != bbscore.ReportStatusOpen {
		t.Fatalf("expected report 1 to remain open on SyncAbone failure, got %s", storedRep1.Status)
	}

	failMessageSync.Store(false)
	evOK := newInternalComponentEvent(t, aboneClient, guildID, opID, "/bbs/admin/abone/"+rep2.ID.String(), map[string]string{"report_id": rep2.ID.String()}, captureRespond)
	if err := adminCmd.handleAdminAboneButton(evOK); err != nil {
		t.Fatalf("handleAdminAboneButton (ok path) returned err: %v", err)
	}
	storedRep2, err := store.GetReport(ctx, rep2.ID)
	if err != nil {
		t.Fatalf("GetReport 2 failed: %v", err)
	}
	if storedRep2.Status != bbscore.ReportStatusResolved {
		t.Fatalf("expected report 2 to become resolved after SyncAbone success, got %s", storedRep2.Status)
	}
}
