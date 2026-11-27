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
	"fmt"
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
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
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

	tok := sys.savePending(u1, pendingWatchThread, target)
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
	if act.Kind != pendingWatchThread || act.TargetID != target {
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

	// Also verify slip=vvv works in parseBoardRulesFromSlash and parseUpdateBoardInput.
	dataVVV := discord.SlashCommandInteractionData{
		Options: map[string]discord.SlashCommandOption{
			"slip": {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("vvv")},
		},
	}
	if rVVV, err := parseBoardRulesFromSlash(dataVVV, discord.LocaleJapanese); err != nil || rVVV.SLIP != bbscore.SLIPVVV {
		t.Fatalf("expected parseBoardRulesFromSlash slip=vvv -> SLIPVVV, got %+v, err=%v", rVVV, err)
	}
	if uVVV, err := parseUpdateBoardInput(dataVVV); err != nil || uVVV.SLIP == nil || *uVVV.SLIP != bbscore.SLIPVVV {
		t.Fatalf("expected parseUpdateBoardInput slip=vvv -> SLIPVVV, got %+v, err=%v", uVVV, err)
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
		{"cap_not_authorized", bbscore.ErrCapNotAuthorized, true},
		{"invalid_name_command", bbscore.ErrInvalidNameCommand, true},
		{"invalid_extend_command", bbscore.ErrInvalidExtendCommand, true},
		{"invalid_reader_jump", bbscore.ErrInvalidReaderJump, true},
		{"invalid_policy_version", bbscore.ErrInvalidPolicyVersion, true},
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

	// 1. Consent modal open/submit version and 3-checkbox enforcement.
	var lastRespBody string
	var lastRespType discord.InteractionResponseType
	mockClient := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			b, _ := io.ReadAll(req.Body)
			if len(b) > 0 {
				lastRespBody = string(b)
			}
		}
		return jsonResp(http.StatusOK, map[string]any{"id": "1"}), nil
	})
	captureRespond := func(rt discord.InteractionResponseType, data any) {
		lastRespType = rt
		if data != nil {
			raw, _ := json.Marshal(data)
			lastRespBody = string(raw)
		}
	}
	captureModalRespond := func(rt discord.InteractionResponseType, data discord.InteractionResponseData) {
		captureRespond(rt, data)
	}

	// Stale open button must reject with stale_version instead of opening modal.
	staleOpenEv := newInternalComponentEvent(t, mockClient, guildID, userID, "/bbs/consent/open/2025-01/none", map[string]string{
		"version": "2025-01",
		"token":   "none",
	}, captureRespond)
	if err := sys.handleConsentOpenModal(staleOpenEv); err != nil {
		t.Fatalf("handleConsentOpenModal stale returned err: %v", err)
	}
	if lastRespType != discord.InteractionResponseTypeCreateMessage || !strings.Contains(lastRespBody, i18n.TranslateText(discord.LocaleJapanese, "components.bbs.consent.stale_version")) {
		t.Fatalf("expected stale_version message response, got type=%d body=%s", lastRespType, lastRespBody)
	}

	// Current open button must open the consent modal.
	validTok := bbscore.PolicyVersionToken("2026-02")
	validOpenEv := newInternalComponentEvent(t, mockClient, guildID, userID, "/bbs/consent/open/"+validTok+"/none", map[string]string{
		"version": validTok,
		"token":   "none",
	}, captureRespond)
	if err := sys.handleConsentOpenModal(validOpenEv); err != nil {
		t.Fatalf("handleConsentOpenModal valid returned err: %v", err)
	}
	if lastRespType != discord.InteractionResponseTypeModal || !strings.Contains(lastRespBody, "/bbs/consent/modal/"+validTok+"/none") {
		t.Fatalf("expected consent modal response, got type=%d body=%s", lastRespType, lastRespBody)
	}

	// Modal submit with fewer than 3 checkboxes checked must reject without saving consent.
	partialModalEv := newInternalConsentModalEvent(t, mockClient, guildID, userID, "/bbs/consent/modal/"+validTok+"/none",
		[]string{discordsync.ConsentOptTerms, discordsync.ConsentOptGuidelines},
		nil,
		map[string]string{"version": validTok, "token": "none"},
		captureModalRespond,
	)
	if err := sys.handleConsentModal(partialModalEv); err != nil {
		t.Fatalf("handleConsentModal partial returned err: %v", err)
	}
	if _, err := svc.CheckConsent(ctx, userID); !errors.Is(err, bbscore.ErrConsentRequired) {
		t.Fatalf("partial checkboxes must not record consent, CheckConsent returned: %v", err)
	}
	if !strings.Contains(lastRespBody, i18n.TranslateText(discord.LocaleJapanese, "components.bbs.consent.modal.all_required")) {
		t.Fatalf("expected all_required error message, got %s", lastRespBody)
	}

	// Stale modal submit even with all 3 checkboxes checked must reject without saving consent.
	staleModalEv := newInternalConsentModalEvent(t, mockClient, guildID, userID, "/bbs/consent/modal/2025-01/none",
		[]string{discordsync.ConsentOptTerms, discordsync.ConsentOptGuidelines, discordsync.ConsentOptPrivacy},
		nil,
		map[string]string{"version": "2025-01", "token": "none"},
		captureModalRespond,
	)
	if err := sys.handleConsentModal(staleModalEv); err != nil {
		t.Fatalf("handleConsentModal stale returned err: %v", err)
	}
	if _, err := svc.CheckConsent(ctx, userID); !errors.Is(err, bbscore.ErrConsentRequired) {
		t.Fatalf("stale policy version must not record user consent, CheckConsent returned: %v", err)
	}
	if !strings.Contains(lastRespBody, i18n.TranslateText(discord.LocaleJapanese, "components.bbs.consent.stale_version")) {
		t.Fatalf("expected stale_version notice in response, got %s", lastRespBody)
	}

	// Valid modal submit with all 3 checkboxes checked records consent.
	validModalEv := newInternalConsentModalEvent(t, mockClient, guildID, userID, "/bbs/consent/modal/"+validTok+"/none",
		[]string{discordsync.ConsentOptTerms, discordsync.ConsentOptGuidelines, discordsync.ConsentOptPrivacy},
		nil,
		map[string]string{"version": validTok, "token": "none"},
		captureModalRespond,
	)
	if err := sys.handleConsentModal(validModalEv); err != nil {
		t.Fatalf("handleConsentModal valid returned err: %v", err)
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

func newInternalModalEvent(
	t *testing.T,
	client *bot.Client,
	guildID snowflake.ID,
	userID snowflake.ID,
	customID string,
	inputs map[string]string,
	vars map[string]string,
	onRespond func(discord.InteractionResponseType, discord.InteractionResponseData),
) *handler.ModalEvent {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"type":           discord.InteractionTypeModalSubmit,
		"id":             snowflake.New(time.Now()).String(),
		"application_id": "999000",
		"token":          "test-modal-token",
		"guild_id":       guildID.String(),
		"channel":        map[string]any{"id": "555001", "type": 0},
		"member": map[string]any{
			"user":        map[string]any{"id": userID.String(), "username": "mod"},
			"permissions": "8",
		},
		"locale": "ja",
		"data": map[string]any{
			"custom_id": customID,
		},
	})
	if err != nil {
		t.Fatalf("marshal modal interaction failed: %v", err)
	}
	var mi discord.ModalSubmitInteraction
	if err := json.Unmarshal(payload, &mi); err != nil {
		t.Fatalf("unmarshal ModalSubmitInteraction failed: %v", err)
	}
	comps := make([]discord.LayoutComponent, 0, len(inputs))
	for k, v := range inputs {
		comps = append(comps, discord.NewLabel(k, discord.TextInputComponent{
			CustomID: k,
			Style:    discord.TextInputStyleShort,
			Value:    v,
		}))
	}
	mi.Data.Components = comps
	return &handler.ModalEvent{
		ModalSubmitInteractionCreate: &events.ModalSubmitInteractionCreate{
			GenericEvent:           events.NewGenericEvent(client, 0, 0),
			ModalSubmitInteraction: mi,
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

func newInternalCommandEvent(
	t *testing.T,
	client *bot.Client,
	guildID snowflake.ID,
	userID snowflake.ID,
	opts map[string]discord.SlashCommandOption,
	onRespond func(discord.InteractionResponseType, discord.InteractionResponseData),
) (discord.SlashCommandInteractionData, *handler.CommandEvent) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"type":           discord.InteractionTypeApplicationCommand,
		"id":             snowflake.New(time.Now()).String(),
		"application_id": "999000",
		"token":          "test-cmd-token",
		"guild_id":       guildID.String(),
		"channel":        map[string]any{"id": "555001", "type": 0},
		"member": map[string]any{
			"user":        map[string]any{"id": userID.String(), "username": "admin"},
			"permissions": "8",
		},
		"locale": "ja",
		"data": map[string]any{
			"id":   "1",
			"type": discord.ApplicationCommandTypeSlash,
			"name": "bbs",
		},
	})
	if err != nil {
		t.Fatalf("marshal command interaction failed: %v", err)
	}
	var ai discord.ApplicationCommandInteraction
	if err := json.Unmarshal(payload, &ai); err != nil {
		t.Fatalf("unmarshal ApplicationCommandInteraction failed: %v", err)
	}
	data := discord.SlashCommandInteractionData{Options: opts}
	ev := &handler.CommandEvent{
		ApplicationCommandInteractionCreate: &events.ApplicationCommandInteractionCreate{
			GenericEvent:                  events.NewGenericEvent(client, 0, 0),
			ApplicationCommandInteraction: ai,
			Respond: func(respType discord.InteractionResponseType, d discord.InteractionResponseData, opts ...rest.RequestOpt) error {
				if onRespond != nil {
					onRespond(respType, d)
				}
				return nil
			},
		},
		Ctx: context.Background(),
	}
	return data, ev
}

func TestRound3_RoleGateBinding_BoardPreValidation_And_BanModalRetryReuse(t *testing.T) {
	ctx := context.Background()
	opID := snowflake.ID(910001)
	roleUserID := snowflake.ID(910002)
	authorID := snowflake.ID(910003)
	reporterID := snowflake.ID(910004)
	guildRole := snowflake.ID(910010)
	guildBoard := snowflake.ID(910011)

	store, svc, sys := newInternalTestEnv(t, bbscore.PolicyConfig{Version: "2026-03"}, opID)
	bbsCmd := &bbsCommand{sys: sys}
	adminCmd := &adminCommand{sys: sys}

	// 1. Legacy role gate button returns deprecation notice without granting roles or saving consent.
	roleB := snowflake.ID(987654322)
	if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{
		GuildID:     guildRole,
		Enabled:     true,
		ActorUserID: opID,
	}); err != nil {
		t.Fatalf("SetupGuild role failed: %v", err)
	}
	var roleGrantCalls atomic.Int32
	var deprecatedRespBody string
	roleClient := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		if req.Method == http.MethodPut && strings.Contains(p, "/roles/") {
			roleGrantCalls.Add(1)
			return jsonResp(http.StatusNoContent, nil), nil
		}
		return jsonResp(http.StatusOK, map[string]any{"id": "1"}), nil
	})
	verTok := bbscore.PolicyVersionToken("2026-03")

	evDeprecatedRole := newInternalComponentEvent(t, roleClient, guildRole, roleUserID,
		"/bbs/consent/role/"+guildRole.String()+"/"+roleB.String()+"/"+verTok,
		map[string]string{"guild_id": guildRole.String(), "role_id": roleB.String(), "version": verTok},
		func(_ discord.InteractionResponseType, data any) {
			raw, _ := json.Marshal(data)
			deprecatedRespBody = string(raw)
		},
	)
	if err := sys.handleConsentRoleDeprecated(evDeprecatedRole); err != nil {
		t.Fatalf("handleConsentRoleDeprecated returned err: %v", err)
	}
	if roleGrantCalls.Load() != 0 {
		t.Fatalf("expected 0 role grants on deprecated role gate panel, got %d", roleGrantCalls.Load())
	}
	if _, err := svc.CheckConsent(ctx, roleUserID); !errors.Is(err, bbscore.ErrConsentRequired) {
		t.Fatalf("expected consent NOT recorded on deprecated role gate panel, err=%v", err)
	}
	if !strings.Contains(deprecatedRespBody, i18n.TranslateText(discord.LocaleJapanese, "components.bbs.consent.role.deprecated_title")) {
		t.Fatalf("expected deprecated role gate notice, got %s", deprecatedRespBody)
	}

	// 2. /bbs board create validates rules and board name BEFORE resolveOrCreateDefaultLocalCategory (R3-m01)
	if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{GuildID: guildBoard, Enabled: true, ActorUserID: opID}); err != nil {
		t.Fatalf("SetupGuild board failed: %v", err)
	}
	rawJSON := func(v any) json.RawMessage {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal value failed: %v", err)
		}
		return b
	}
	badRulesData, badRulesCmdEv := newInternalCommandEvent(t, roleClient, guildBoard, opID, map[string]discord.SlashCommandOption{
		"name":         {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("valid-board")},
		"default_name": {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("   ")},
	}, nil)
	if err := bbsCmd.handleBoardCreate(badRulesData, badRulesCmdEv); err != nil {
		t.Fatalf("handleBoardCreate invalid rules returned err: %v", err)
	}
	badNameData, badNameCmdEv := newInternalCommandEvent(t, roleClient, guildBoard, opID, map[string]discord.SlashCommandOption{
		"name": {Type: discord.ApplicationCommandOptionTypeString, Value: rawJSON("   ")},
	}, nil)
	if err := bbsCmd.handleBoardCreate(badNameData, badNameCmdEv); err != nil {
		t.Fatalf("handleBoardCreate empty name returned err: %v", err)
	}
	localScope := bbscore.ScopeLocal
	catsAfterReject, err := svc.ListCategories(ctx, bbscore.CategoryFilter{Scope: &localScope, GuildID: &guildBoard})
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(catsAfterReject) != 0 {
		t.Fatalf("expected 0 default local categories created after rejected /bbs board create inputs, got %+v", catsAfterReject)
	}

	// 3. handleAdminBanModal persists BanID on report and reuses it on retry without duplicate Ban (R3-M01)
	gCat, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{Scope: bbscore.ScopeGlobal, Name: "ModGlobalCat"})
	if err != nil {
		t.Fatalf("CreateCategory global failed: %v", err)
	}
	gBoard, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{CategoryID: gCat.ID, Scope: bbscore.ScopeGlobal, Name: "ModGlobalBoard"})
	if err != nil {
		t.Fatalf("CreateBoard global failed: %v", err)
	}
	mountID := uuid.New()
	if err := store.CreateBoardMount(ctx, &bbscore.BoardMount{
		ID: mountID, BoardID: gBoard.ID, GuildID: guildRole, ChannelID: snowflake.ID(610001), Active: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	for _, uid := range []snowflake.ID{authorID, reporterID} {
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
	}
	_, th, opPost, err := svc.CreateThread(ctx, bbscore.CreateThreadInput{
		BoardID: gBoard.ID, GuildID: guildRole, UserID: authorID, Title: "BANテスト", RawBody: "違反投稿",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	discordTID := snowflake.ID(610010)
	mirrorID := uuid.New()
	if err := store.CreateThreadMirror(ctx, &bbscore.ThreadMirror{
		ID: mirrorID, ThreadID: th.ID, BoardMountID: mountID, GuildID: guildRole, DiscordThreadID: discordTID, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateThreadMirror failed: %v", err)
	}
	opMsgID := snowflake.ID(610101)
	if err := store.CreatePostMirror(ctx, &bbscore.PostMirror{
		ID: uuid.New(), PostID: opPost.ID, ThreadMirrorID: mirrorID, GuildID: guildRole, DiscordMessageID: opMsgID, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreatePostMirror failed: %v", err)
	}
	banReport, err := svc.CreateReport(ctx, guildRole, reporterID, opPost.ID, "global abuse")
	if err != nil {
		t.Fatalf("CreateReport failed: %v", err)
	}

	var failBanSync atomic.Bool
	failBanSync.Store(true)
	banSyncClient := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		if req.Method == http.MethodGet && p == "/channels/"+discordTID.String() {
			return jsonResp(http.StatusOK, map[string]any{"id": discordTID.String(), "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil
		}
		if req.Method == http.MethodPatch && strings.HasPrefix(p, "/channels/"+discordTID.String()+"/messages/") {
			if failBanSync.Load() {
				return jsonResp(http.StatusServiceUnavailable, map[string]any{"message": "503"}), nil
			}
			return jsonResp(http.StatusOK, map[string]any{"id": opMsgID.String()}), nil
		}
		return jsonResp(http.StatusOK, map[string]any{"id": "1"}), nil
	})

	modalEv1 := newInternalModalEvent(t, banSyncClient, guildRole, opID, "/bbs/admin/banmodal/"+banReport.ID.String(),
		map[string]string{"hours": "24", "reason": "global abuse"},
		map[string]string{"report_id": banReport.ID.String()},
		nil,
	)
	if err := adminCmd.handleAdminBanModal(modalEv1); err != nil {
		t.Fatalf("handleAdminBanModal fail attempt returned err: %v", err)
	}

	bansAfter1, err := store.ListBans(ctx, bbscore.BanFilter{Limit: 10, Now: time.Now()})
	if err != nil || len(bansAfter1) != 1 {
		t.Fatalf("expected exactly 1 ban after failed sync attempt, got %d, err=%v", len(bansAfter1), err)
	}
	repAfter1, err := store.GetReport(ctx, banReport.ID)
	if err != nil {
		t.Fatalf("GetReport 1 failed: %v", err)
	}
	if repAfter1.Status != bbscore.ReportStatusOpen || repAfter1.BanID == nil || *repAfter1.BanID != bansAfter1[0].ID {
		t.Fatalf("expected report open with saved BanID %v, got status=%s banID=%v", bansAfter1[0].ID, repAfter1.Status, repAfter1.BanID)
	}

	failBanSync.Store(false)
	modalEv2 := newInternalModalEvent(t, banSyncClient, guildRole, opID, "/bbs/admin/banmodal/"+banReport.ID.String(),
		map[string]string{"hours": "24", "reason": "global abuse"},
		map[string]string{"report_id": banReport.ID.String()},
		nil,
	)
	if err := adminCmd.handleAdminBanModal(modalEv2); err != nil {
		t.Fatalf("handleAdminBanModal retry attempt returned err: %v", err)
	}

	bansAfter2, err := store.ListBans(ctx, bbscore.BanFilter{Limit: 10, Now: time.Now()})
	if err != nil || len(bansAfter2) != 1 || bansAfter2[0].ID != bansAfter1[0].ID {
		t.Fatalf("expected retry to reuse initial Ban %v without duplicate rows, got %+v, err=%v", bansAfter1[0].ID, bansAfter2, err)
	}
	repAfter2, err := store.GetReport(ctx, banReport.ID)
	if err != nil || repAfter2.Status != bbscore.ReportStatusResolved || repAfter2.BanID == nil || *repAfter2.BanID != bansAfter1[0].ID {
		t.Fatalf("expected report resolved on retry preserving BanID %v, got %+v, err=%v", bansAfter1[0].ID, repAfter2, err)
	}
}

func newInternalConsentModalEvent(
	t *testing.T,
	client *bot.Client,
	guildID snowflake.ID,
	userID snowflake.ID,
	customID string,
	checkedValues []string,
	msgFlags *discord.MessageFlags,
	vars map[string]string,
	onRespond func(discord.InteractionResponseType, discord.InteractionResponseData),
) *handler.ModalEvent {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"type":           discord.InteractionTypeModalSubmit,
		"id":             snowflake.New(time.Now()).String(),
		"application_id": "999000",
		"token":          "test-consent-modal-token",
		"guild_id":       guildID.String(),
		"channel":        map[string]any{"id": "555001", "type": 0},
		"member": map[string]any{
			"user": map[string]any{"id": userID.String(), "username": "user"},
		},
		"locale": "ja",
		"data": map[string]any{
			"custom_id": customID,
		},
	})
	if err != nil {
		t.Fatalf("marshal consent modal interaction failed: %v", err)
	}
	var mi discord.ModalSubmitInteraction
	if err := json.Unmarshal(payload, &mi); err != nil {
		t.Fatalf("unmarshal ModalSubmitInteraction failed: %v", err)
	}
	mi.Data.Components = []discord.LayoutComponent{
		discord.NewLabel("checks", discord.CheckboxGroupComponent{
			CustomID: discordsync.ConsentCheckboxGroupCustomID,
			Values:   checkedValues,
		}),
	}
	if msgFlags != nil {
		mi.Message = &discord.Message{
			ID:    snowflake.ID(555099),
			Flags: *msgFlags,
		}
	}
	return &handler.ModalEvent{
		ModalSubmitInteractionCreate: &events.ModalSubmitInteractionCreate{
			GenericEvent:           events.NewGenericEvent(client, 0, 0),
			ModalSubmitInteraction: mi,
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

func newInternalAdminComponentEvent(
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
		"token":          "test-admin-interaction-token",
		"guild_id":       guildID.String(),
		"channel":        map[string]any{"id": "555001", "type": 0},
		"member": map[string]any{
			"user":        map[string]any{"id": userID.String(), "username": "guildadmin"},
			"permissions": "32",
		},
		"locale": "ja",
		"data": map[string]any{
			"component_type": discord.ComponentTypeButton,
			"custom_id":      customID,
		},
	})
	if err != nil {
		t.Fatalf("marshal admin component interaction failed: %v", err)
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

func TestConsentModalFlow_And_UnconsentedReaderAccess(t *testing.T) {
	ctx := context.Background()
	opID := snowflake.ID(920001)
	authorID := snowflake.ID(920002)
	guildID := snowflake.ID(920010)
	ver := "2026-04"
	verTok := bbscore.PolicyVersionToken(ver)

	store, svc, sys := newInternalTestEnv(t, bbscore.PolicyConfig{Version: ver}, opID)
	if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: opID}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	if _, err := svc.AcceptPolicy(ctx, authorID); err != nil {
		t.Fatalf("AcceptPolicy author failed: %v", err)
	}
	cat, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{Scope: bbscore.ScopeLocal, GuildID: &guildID, Name: "雑談"})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{CategoryID: cat.ID, Scope: bbscore.ScopeLocal, GuildID: &guildID, Name: "テスト板"})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	mountID := uuid.New()
	if err := store.CreateBoardMount(ctx, &bbscore.BoardMount{
		ID: mountID, BoardID: board.ID, GuildID: guildID, ChannelID: snowflake.ID(620001), Active: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	_, th, opPost, err := svc.CreateThread(ctx, bbscore.CreateThreadInput{
		BoardID: board.ID, GuildID: guildID, UserID: authorID, Title: "リーダーテストスレ", RawBody: "1レス目の本文",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	mirrorID := uuid.New()
	discordThreadID := snowflake.ID(620010)
	if err := store.CreateThreadMirror(ctx, &bbscore.ThreadMirror{
		ID: mirrorID, ThreadID: th.ID, BoardMountID: mountID, GuildID: guildID, DiscordThreadID: discordThreadID, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateThreadMirror failed: %v", err)
	}
	postMsgID := snowflake.ID(620101)
	if err := store.CreatePostMirror(ctx, &bbscore.PostMirror{
		ID: uuid.New(), PostID: opPost.ID, ThreadMirrorID: mirrorID, GuildID: guildID, DiscordMessageID: postMsgID, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreatePostMirror failed: %v", err)
	}

	mockClient := newInternalMockClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResp(http.StatusOK, map[string]any{"id": "1"}), nil
	})

	// 1. Unconsented user can read thread history and use jump modal without consent modal.
	uReader := snowflake.ID(920100)
	var respType discord.InteractionResponseType
	var respData any
	capComp := func(rt discord.InteractionResponseType, data any) {
		respType = rt
		respData = data
	}
	capModal := func(rt discord.InteractionResponseType, data discord.InteractionResponseData) {
		respType = rt
		respData = data
	}

	readerEv := newInternalComponentEvent(t, mockClient, guildID, uReader, "/bbs/reader/"+th.ID.String()+"/1",
		map[string]string{"thread_id": th.ID.String(), "page": "1"}, capComp)
	if err := sys.handleReaderPage(readerEv); err != nil {
		t.Fatalf("handleReaderPage failed: %v", err)
	}
	readerMC, ok := respData.(discord.MessageCreate)
	rawReader, _ := json.Marshal(respData)
	if respType != discord.InteractionResponseTypeCreateMessage || !ok || !strings.Contains(string(rawReader), "1レス目の本文") {
		t.Fatalf("expected unconsented reader page to render post body, type=%d body=%s", respType, rawReader)
	}
	pv := discordsync.PayloadValidator{}
	if err := pv.ValidateMessageCreate(readerMC); err != nil {
		t.Fatalf("reader page response failed ValidateMessageCreate: %v", err)
	}

	readerFirstEv := newInternalComponentEvent(t, mockClient, guildID, uReader, "/bbs/reader/"+th.ID.String()+"/first",
		map[string]string{"thread_id": th.ID.String(), "page": "first"}, capComp)
	if err := sys.handleReaderPage(readerFirstEv); err != nil {
		t.Fatalf("handleReaderPage(first) failed: %v", err)
	}
	firstMC, ok := respData.(discord.MessageCreate)
	rawFirst, _ := json.Marshal(respData)
	if respType != discord.InteractionResponseTypeCreateMessage || !ok || !strings.Contains(string(rawFirst), "1レス目の本文") {
		t.Fatalf("expected /first reader page to render post 1, type=%d body=%s", respType, rawFirst)
	}
	if err := pv.ValidateMessageCreate(firstMC); err != nil {
		t.Fatalf("/first reader page failed ValidateMessageCreate: %v", err)
	}

	readerUpdEv := newInternalComponentEvent(t, mockClient, guildID, uReader, "/bbs/reader/"+th.ID.String()+"/first",
		map[string]string{"thread_id": th.ID.String(), "page": "first"}, capComp)
	readerUpdEv.Message.Flags = discord.MessageFlagEphemeral | discord.MessageFlagIsComponentsV2
	if err := sys.handleReaderPage(readerUpdEv); err != nil {
		t.Fatalf("handleReaderPage(ephemeral update) failed: %v", err)
	}
	updMU, ok := respData.(discord.MessageUpdate)
	if respType != discord.InteractionResponseTypeUpdateMessage || !ok {
		t.Fatalf("expected UpdateMessage for ephemeral reader nav, got type=%d data=%#v", respType, respData)
	}
	if err := pv.ValidateMessageUpdate(updMU, true); err != nil {
		t.Fatalf("ephemeral reader page update failed ValidateMessageUpdate: %v", err)
	}

	jumpBtnEv := newInternalComponentEvent(t, mockClient, guildID, uReader, "/bbs/reader/jump/"+th.ID.String(),
		map[string]string{"thread_id": th.ID.String()}, capComp)
	if err := sys.handleReaderJumpButton(jumpBtnEv); err != nil {
		t.Fatalf("handleReaderJumpButton failed: %v", err)
	}
	jumpModal, ok := respData.(discord.ModalCreate)
	if respType != discord.InteractionResponseTypeModal || !ok || jumpModal.CustomID != "/bbs/reader/jumpmodal/"+th.ID.String() {
		t.Fatalf("expected unconsented jump button to open jump modal, type=%d data=%#v", respType, respData)
	}

	jumpModalEv := newInternalModalEvent(t, mockClient, guildID, uReader, "/bbs/reader/jumpmodal/"+th.ID.String(),
		map[string]string{"target": "1"}, map[string]string{"thread_id": th.ID.String()}, capModal)
	if err := sys.handleReaderJumpModal(jumpModalEv); err != nil {
		t.Fatalf("handleReaderJumpModal failed: %v", err)
	}
	rawJump, _ := json.Marshal(respData)
	if !strings.Contains(string(rawJump), "1レス目の本文") {
		t.Fatalf("expected unconsented jump modal submit to render post body, got %s", rawJump)
	}

	// 2. Unconsented user can open the Post Action Menu without consent modal, and unknown post_id fails without consent modal.
	uMenuViewer := snowflake.ID(920101)
	menuEv := newInternalComponentEvent(t, mockClient, guildID, uMenuViewer, "/bbs/post/menu/"+opPost.ID.String(),
		map[string]string{"post_id": opPost.ID.String()}, capComp)
	if err := sys.handlePostMenuButton(menuEv); err != nil {
		t.Fatalf("handlePostMenuButton unconsented failed: %v", err)
	}
	menuMC, ok := respData.(discord.MessageCreate)
	rawMenu, _ := json.Marshal(respData)
	if respType != discord.InteractionResponseTypeCreateMessage || !ok || !menuMC.Flags.Has(discord.MessageFlagEphemeral) {
		t.Fatalf("expected unconsented menu button to return ephemeral menu, type=%d data=%#v", respType, respData)
	}
	if err := pv.ValidateMessageCreate(menuMC); err != nil {
		t.Fatalf("post menu response failed ValidateMessageCreate: %v", err)
	}
	if !strings.Contains(string(rawMenu), "/bbs/post/reply/"+opPost.ID.String()) || !strings.Contains(string(rawMenu), "/bbs/report/post/"+opPost.ID.String()) {
		t.Fatalf("expected reply and report buttons in unconsented post menu, got %s", rawMenu)
	}

	unknownPostID := uuid.New()
	unknownMenuEv := newInternalComponentEvent(t, mockClient, guildID, uMenuViewer, "/bbs/post/menu/"+unknownPostID.String(),
		map[string]string{"post_id": unknownPostID.String()}, capComp)
	if err := sys.handlePostMenuButton(unknownMenuEv); err != nil {
		t.Fatalf("handlePostMenuButton unknown post failed: %v", err)
	}
	rawUnknown, _ := json.Marshal(respData)
	if respType != discord.InteractionResponseTypeCreateMessage || !strings.Contains(string(rawUnknown), i18n.TranslateText(discord.LocaleJapanese, "components.bbs.errors.title.fetch_failed")) {
		t.Fatalf("expected unknown post_id menu click to return fetch_failed error, type=%d body=%s", respType, rawUnknown)
	}

	// 3. First interactive action by unconsented users opens ConsentModal and stores pending action.
	allChecks := []string{discordsync.ConsentOptTerms, discordsync.ConsentOptGuidelines, discordsync.ConsentOptPrivacy}
	prefix := "/bbs/consent/modal/" + verTok + "/"
	publicFlags := discord.MessageFlags(0)

	// 3a. Watch toggle -> ConsentModal -> submit records consent AND Watch.
	uWatch := snowflake.ID(920102)
	watchEv := newInternalComponentEvent(t, mockClient, guildID, uWatch, "/bbs/watch/toggle/"+th.ID.String(),
		map[string]string{"thread_id": th.ID.String()}, capComp)
	if err := sys.handleWatchToggle(watchEv); err != nil {
		t.Fatalf("handleWatchToggle failed: %v", err)
	}
	watchConsentModal, ok := respData.(discord.ModalCreate)
	if respType != discord.InteractionResponseTypeModal || !ok || !strings.HasPrefix(watchConsentModal.CustomID, prefix) {
		t.Fatalf("expected watch toggle to open consent modal, type=%d data=%#v", respType, respData)
	}
	watchTok := strings.TrimPrefix(watchConsentModal.CustomID, prefix)
	watchModalSubmit := newInternalConsentModalEvent(t, mockClient, guildID, uWatch, watchConsentModal.CustomID,
		allChecks, &publicFlags, map[string]string{"version": verTok, "token": watchTok}, capModal)
	if err := sys.handleConsentModal(watchModalSubmit); err != nil {
		t.Fatalf("handleConsentModal watch failed: %v", err)
	}
	if respType != discord.InteractionResponseTypeCreateMessage {
		t.Fatalf("expected ephemeral message response after watch consent, got %d", respType)
	}
	if _, err := svc.CheckConsent(ctx, uWatch); err != nil {
		t.Fatalf("expected uWatch consent recorded, err=%v", err)
	}
	if w, wErr := store.GetThreadWatch(ctx, th.ID, uWatch); wErr != nil || w == nil {
		t.Fatalf("expected uWatch thread watch saved automatically, got %+v err=%v", w, wErr)
	}

	// 3b. NewThread / NewPost / ReplyPost / ReportThread / ReportPostButton -> ConsentModal -> continue button ephemeral response.
	actionCases := []struct {
		name           string
		userID         snowflake.ID
		trigger        func() error
		expectedBtnID  string
		expectedLabelK string
	}{
		{
			name:   "new_thread",
			userID: snowflake.ID(920103),
			trigger: func() error {
				ev := newInternalComponentEvent(t, mockClient, guildID, snowflake.ID(920103), "/bbs/thread/new/"+mountID.String(),
					map[string]string{"mount_id": mountID.String()}, capComp)
				return sys.handleNewThreadButton(ev)
			},
			expectedBtnID:  "/bbs/thread/new/" + mountID.String(),
			expectedLabelK: "components.bbs.consent.continue.new_thread",
		},
		{
			name:   "new_post",
			userID: snowflake.ID(920104),
			trigger: func() error {
				ev := newInternalComponentEvent(t, mockClient, guildID, snowflake.ID(920104), "/bbs/post/new/"+mirrorID.String(),
					map[string]string{"mirror_id": mirrorID.String()}, capComp)
				return sys.handleNewPostButton(ev)
			},
			expectedBtnID:  "/bbs/post/new/" + mirrorID.String(),
			expectedLabelK: "components.bbs.consent.continue.new_post",
		},
		{
			name:   "reply_post",
			userID: snowflake.ID(920110),
			trigger: func() error {
				ev := newInternalComponentEvent(t, mockClient, guildID, snowflake.ID(920110), "/bbs/post/reply/"+opPost.ID.String(),
					map[string]string{"post_id": opPost.ID.String()}, capComp)
				return sys.handleReplyPostButton(ev)
			},
			expectedBtnID:  "/bbs/post/reply/" + opPost.ID.String(),
			expectedLabelK: "components.bbs.consent.continue.reply_post",
		},
		{
			name:   "report_thread",
			userID: snowflake.ID(920105),
			trigger: func() error {
				ev := newInternalComponentEvent(t, mockClient, guildID, snowflake.ID(920105), "/bbs/report/thread/"+th.ID.String(),
					map[string]string{"thread_id": th.ID.String()}, capComp)
				return sys.handleReportThreadButton(ev)
			},
			expectedBtnID:  "/bbs/report/thread/" + th.ID.String(),
			expectedLabelK: "components.bbs.consent.continue.report_thread",
		},
		{
			name:   "report_post_btn",
			userID: snowflake.ID(920106),
			trigger: func() error {
				ev := newInternalComponentEvent(t, mockClient, guildID, snowflake.ID(920106), "/bbs/report/post/"+opPost.ID.String(),
					map[string]string{"post_id": opPost.ID.String()}, capComp)
				return sys.handleReportPostButton(ev)
			},
			expectedBtnID:  "/bbs/report/post/" + opPost.ID.String(),
			expectedLabelK: "components.bbs.consent.continue.report_post",
		},
	}

	for _, tc := range actionCases {
		if err := tc.trigger(); err != nil {
			t.Fatalf("%s trigger failed: %v", tc.name, err)
		}
		m, ok := respData.(discord.ModalCreate)
		if respType != discord.InteractionResponseTypeModal || !ok || !strings.HasPrefix(m.CustomID, prefix) {
			t.Fatalf("%s expected consent modal, got type=%d data=%#v", tc.name, respType, respData)
		}
		tok := strings.TrimPrefix(m.CustomID, prefix)
		subEv := newInternalConsentModalEvent(t, mockClient, guildID, tc.userID, m.CustomID,
			allChecks, &publicFlags, map[string]string{"version": verTok, "token": tok}, capModal)
		if err := sys.handleConsentModal(subEv); err != nil {
			t.Fatalf("%s handleConsentModal failed: %v", tc.name, err)
		}
		if respType != discord.InteractionResponseTypeCreateMessage {
			t.Fatalf("%s expected CreateMessage (ephemeral) rather than UpdateMessage, got type=%d", tc.name, respType)
		}
		mc, ok := respData.(discord.MessageCreate)
		if !ok || !mc.Flags.Has(discord.MessageFlagEphemeral) {
			t.Fatalf("%s expected ephemeral MessageCreate, got %#v", tc.name, respData)
		}
		rawMC, _ := json.Marshal(mc)
		if !strings.Contains(string(rawMC), tc.expectedBtnID) || !strings.Contains(string(rawMC), i18n.TranslateText(discord.LocaleJapanese, tc.expectedLabelK)) {
			t.Fatalf("%s expected continue button %q in %s", tc.name, tc.expectedBtnID, rawMC)
		}
	}

	// 3c. Consented user pressing continue button for reply opens prefilled reply modal, and report opens report modal.
	replyBtnEv := newInternalComponentEvent(t, mockClient, guildID, snowflake.ID(920110), "/bbs/post/reply/"+opPost.ID.String(),
		map[string]string{"post_id": opPost.ID.String()}, capComp)
	if err := sys.handleReplyPostButton(replyBtnEv); err != nil {
		t.Fatalf("handleReplyPostButton failed: %v", err)
	}
	replyModal, ok := respData.(discord.ModalCreate)
	if respType != discord.InteractionResponseTypeModal || !ok || replyModal.CustomID != "/bbs/post/modal/"+mirrorID.String() {
		t.Fatalf("expected prefilled post reply modal, got type=%d data=%#v", respType, respData)
	}
	var gotPrefill string
	for _, mc := range replyModal.Components {
		if lbl, ok := mc.(discord.LabelComponent); ok {
			if ti, ok := lbl.Component.(discord.TextInputComponent); ok && ti.CustomID == "body" {
				gotPrefill = ti.Value
			}
		}
	}
	if gotPrefill != ">>1\n" {
		t.Fatalf("expected reply modal body prefill %q, got %q", ">>1\n", gotPrefill)
	}

	repPostBtnEv := newInternalComponentEvent(t, mockClient, guildID, snowflake.ID(920106), "/bbs/report/post/"+opPost.ID.String(),
		map[string]string{"post_id": opPost.ID.String()}, capComp)
	if err := sys.handleReportPostButton(repPostBtnEv); err != nil {
		t.Fatalf("handleReportPostButton failed: %v", err)
	}
	repModal, ok := respData.(discord.ModalCreate)
	if respType != discord.InteractionResponseTypeModal || !ok || repModal.CustomID != "/bbs/report/postmodal/"+opPost.ID.String() {
		t.Fatalf("expected post report modal, got type=%d data=%#v", respType, respData)
	}

	// 4. Legacy /bbs/consent/accept/{version}/{token} redirects to ConsentModal without recording consent.
	uLegacy := snowflake.ID(920107)
	legacyEv := newInternalComponentEvent(t, mockClient, guildID, uLegacy, "/bbs/consent/accept/"+verTok+"/none",
		map[string]string{"version": verTok, "token": "none"}, capComp)
	if err := sys.handleConsentOpenModal(legacyEv); err != nil {
		t.Fatalf("legacy handleConsentOpenModal failed: %v", err)
	}
	if respType != discord.InteractionResponseTypeModal {
		t.Fatalf("expected legacy accept button to open modal, got type=%d", respType)
	}
	if _, err := svc.CheckConsent(ctx, uLegacy); !errors.Is(err, bbscore.ErrConsentRequired) {
		t.Fatalf("legacy button must not record consent before modal submit, err=%v", err)
	}

	// 5. ConsentModal without pending action updates ephemeral message, and creates ephemeral message when source is public.
	uEphemeral := snowflake.ID(920108)
	ephFlags := discord.MessageFlagEphemeral | discord.MessageFlagIsComponentsV2
	ephModalEv := newInternalConsentModalEvent(t, mockClient, guildID, uEphemeral, "/bbs/consent/modal/"+verTok+"/none",
		allChecks, &ephFlags, map[string]string{"version": verTok, "token": "none"}, capModal)
	if err := sys.handleConsentModal(ephModalEv); err != nil {
		t.Fatalf("handleConsentModal ephemeral failed: %v", err)
	}
	if respType != discord.InteractionResponseTypeUpdateMessage {
		t.Fatalf("expected UpdateMessage for ephemeral consent gate modal submit, got type=%d", respType)
	}

	uPublic := snowflake.ID(920109)
	pubModalEv := newInternalConsentModalEvent(t, mockClient, guildID, uPublic, "/bbs/consent/modal/"+verTok+"/none",
		allChecks, &publicFlags, map[string]string{"version": verTok, "token": "none"}, capModal)
	if err := sys.handleConsentModal(pubModalEv); err != nil {
		t.Fatalf("handleConsentModal public failed: %v", err)
	}
	if respType != discord.InteractionResponseTypeCreateMessage {
		t.Fatalf("expected CreateMessage (ephemeral) when source message is non-ephemeral, got type=%d", respType)
	}
}

func modalLabelCustomIDs(m discord.ModalCreate) []string {
	var ids []string
	for _, c := range m.Components {
		lbl, ok := c.(discord.LabelComponent)
		if !ok {
			continue
		}
		switch sub := lbl.Component.(type) {
		case discord.TextInputComponent:
			ids = append(ids, sub.CustomID)
		case discord.StringSelectMenuComponent:
			ids = append(ids, sub.CustomID)
		}
	}
	return ids
}

func findModalCapSelect(m discord.ModalCreate) (discord.StringSelectMenuComponent, bool) {
	for _, c := range m.Components {
		lbl, ok := c.(discord.LabelComponent)
		if !ok {
			continue
		}
		if sel, ok := lbl.Component.(discord.StringSelectMenuComponent); ok && sel.CustomID == capSelectCustomID {
			return sel, true
		}
	}
	return discord.StringSelectMenuComponent{}, false
}

func TestBuildPostingModals_ConditionalCapSelect(t *testing.T) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}
	v := discordsync.PayloadValidator{}
	targetID := uuid.New()

	// 1. Without caps: no mail field and no cap select menu.
	thNoCap := buildNewThreadModal(discord.LocaleJapanese, targetID, "テスト板", nil)
	if err := v.ValidateModal(thNoCap); err != nil {
		t.Fatalf("thNoCap failed ValidateModal: %v", err)
	}
	if ids := modalLabelCustomIDs(thNoCap); strings.Join(ids, ",") != "title,name,body" {
		t.Fatalf("expected thread modal labels [title,name,body] without caps, got %v", ids)
	}

	postNoCap := buildNewPostModal(discord.LocaleEnglishUS, targetID, nil, "")
	if err := v.ValidateModal(postNoCap); err != nil {
		t.Fatalf("postNoCap failed ValidateModal: %v", err)
	}
	if ids := modalLabelCustomIDs(postNoCap); strings.Join(ids, ",") != "name,body" {
		t.Fatalf("expected post modal labels [name,body] without caps, got %v", ids)
	}

	postPrefill := buildNewPostModal(discord.LocaleJapanese, targetID, nil, ">>5\n")
	if err := v.ValidateModal(postPrefill); err != nil {
		t.Fatalf("postPrefill failed ValidateModal: %v", err)
	}
	var gotBodyVal string
	for _, mc := range postPrefill.Components {
		if lbl, ok := mc.(discord.LabelComponent); ok {
			if ti, ok := lbl.Component.(discord.TextInputComponent); ok && ti.CustomID == "body" {
				gotBodyVal = ti.Value
			}
		}
	}
	if gotBodyVal != ">>5\n" {
		t.Fatalf("expected prefilled body value %q, got %q", ">>5\n", gotBodyVal)
	}

	// 2. With caps held: cap StringSelectMenu appears with default __none__ + caps.
	twoCaps := []bbscore.Cap{
		{ID: "admin", Name: "管理人", Description: "運営告知用"},
		{ID: "mod", Name: "モデレーター"},
	}
	thWithCap := buildNewThreadModal(discord.LocaleJapanese, targetID, "テスト板", twoCaps)
	if err := v.ValidateModal(thWithCap); err != nil {
		t.Fatalf("thWithCap failed ValidateModal: %v", err)
	}
	if ids := modalLabelCustomIDs(thWithCap); strings.Join(ids, ",") != "title,name,cap,body" {
		t.Fatalf("expected thread modal labels [title,name,cap,body] with caps, got %v", ids)
	}
	postWithCap := buildNewPostModal(discord.LocaleJapanese, targetID, twoCaps, "")
	if err := v.ValidateModal(postWithCap); err != nil {
		t.Fatalf("postWithCap failed ValidateModal: %v", err)
	}
	if ids := modalLabelCustomIDs(postWithCap); strings.Join(ids, ",") != "name,cap,body" {
		t.Fatalf("expected post modal labels [name,cap,body] with caps, got %v", ids)
	}

	for _, m := range []discord.ModalCreate{thWithCap, postWithCap} {
		sel, ok := findModalCapSelect(m)
		if !ok {
			t.Fatalf("expected cap select menu in modal %q", m.CustomID)
		}
		if sel.Required || sel.Disabled || sel.MaxValues != 1 || sel.MinValues == nil || *sel.MinValues != 0 {
			t.Fatalf("unexpected cap select attributes: %+v", sel)
		}
		if len(sel.Options) != 3 || sel.Options[0].Value != bbscore.CapSelectNoneValue || !sel.Options[0].Default {
			t.Fatalf("unexpected cap select options: %+v", sel.Options)
		}
		if sel.Options[1].Value != "admin" || sel.Options[1].Description != "運営告知用" || sel.Options[2].Value != "mod" {
			t.Fatalf("unexpected cap option values: %+v", sel.Options)
		}
	}

	// 3. >24 caps clamps cleanly to MaxSelectOptions (25) and passes ValidateModal.
	var manyCaps []bbscore.Cap
	for i := range 30 {
		manyCaps = append(manyCaps, bbscore.Cap{
			ID:   "cap_" + string(rune('a'+i%26)) + "_" + string(rune('0'+i/26)),
			Name: "Cap",
		})
	}
	thClamped := buildNewThreadModal(discord.LocaleJapanese, targetID, "板", manyCaps)
	if err := v.ValidateModal(thClamped); err != nil {
		t.Fatalf("thClamped failed ValidateModal: %v", err)
	}
	selClamped, _ := findModalCapSelect(thClamped)
	if len(selClamped.Options) != discordsync.MaxSelectOptions {
		t.Fatalf("expected %d clamped cap options, got %d", discordsync.MaxSelectOptions, len(selClamped.Options))
	}
}

func TestBuildSlipContext_ExtractsMemberVoiceAndGuildTraits(t *testing.T) {
	guildID := snowflake.ID(66001)
	userID := snowflake.ID(77001)
	chID := snowflake.ID(88001)
	joined := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	boosted := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	client := newInternalMockClient(t, nil)
	client.Caches = cache.New(cache.WithCaches(cache.FlagsAll))
	client.Caches.AddGuild(discord.CacheGuild{
		ID:          guildID,
		Name:        "SecretGuildName",
		PremiumTier: discord.PremiumTier3,
		Features:    []discord.GuildFeature{discord.GuildFeatureCommunity, discord.GuildFeatureVanityURL},
		MemberCount: 800,
	})
	client.Caches.AddVoiceState(discord.VoiceState{
		GuildID:    guildID,
		ChannelID:  &chID,
		UserID:     userID,
		SelfStream: true,
	})

	payload, err := json.Marshal(map[string]any{
		"type":           discord.InteractionTypeModalSubmit,
		"id":             snowflake.New(time.Now()).String(),
		"application_id": "999000",
		"token":          "test-modal-token",
		"guild_id":       guildID.String(),
		"locale":         "ja",
		"member": map[string]any{
			"nick":          "鯖ﾆｯｸ",
			"joined_at":     joined.Format(time.RFC3339),
			"premium_since": boosted.Format(time.RFC3339),
			"roles":         []string{"1", "2", "3"},
			"user": map[string]any{
				"id":           userID.String(),
				"username":     "user_raw",
				"global_name":  "GlobalDisplay",
				"public_flags": int(discord.UserFlagHouseBrilliance | discord.UserFlagEarlySupporter),
			},
		},
		"data": map[string]any{
			"custom_id":  "/bbs/post/modal/test",
			"components": []any{},
		},
	})
	if err != nil {
		t.Fatalf("marshal modal interaction failed: %v", err)
	}
	var mi discord.ModalSubmitInteraction
	if err := json.Unmarshal(payload, &mi); err != nil {
		t.Fatalf("unmarshal ModalSubmitInteraction failed: %v", err)
	}
	ev := &handler.ModalEvent{
		ModalSubmitInteractionCreate: &events.ModalSubmitInteractionCreate{
			GenericEvent:           events.NewGenericEvent(client, 0, 0),
			ModalSubmitInteraction: mi,
		},
		Ctx: context.Background(),
	}

	sc := buildSlipContext(ev, guildID)
	if sc.MemberNick != "鯖ﾆｯｸ" || sc.GlobalName != "GlobalDisplay" || sc.Username != "user_raw" {
		t.Fatalf("unexpected names in SlipContext: %+v", sc)
	}
	if sc.HypeSquadHouse != "brilliance" || !sc.HasSpecialBadge || !sc.IsDefaultAvatar || sc.UserLocale != "ja" {
		t.Fatalf("unexpected user flags/locale in SlipContext: %+v", sc)
	}
	if !sc.IsGuildBooster || sc.RoleCount != 3 || sc.HasZeroRoles || !sc.HasServerNick || sc.MemberJoinedAt == nil {
		t.Fatalf("unexpected member attrs in SlipContext: %+v", sc)
	}
	if !sc.InVoiceChannel || !sc.IsVoiceStreaming || sc.IsVoiceMutedOrDeaf {
		t.Fatalf("unexpected voice state in SlipContext: %+v", sc)
	}
	if sc.GuildMemberCount != 800 || sc.GuildBoostTier != 3 || !sc.GuildIsCommunity || !sc.GuildHasVanityURL {
		t.Fatalf("unexpected guild attrs in SlipContext: %+v", sc)
	}
}

func TestSystemGuildLocale_RespectsDBOverrideOverInteractionGuildLocale(t *testing.T) {
	ctx := context.Background()
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}

	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sql.DB failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := gdb.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	{
		stmt := &gorm.Statement{DB: gdb}
		if err := stmt.Parse(&models.Guild{}); err != nil {
			t.Fatalf("parse guild schema failed: %v", err)
		}
		var cols, pks []string
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" {
				continue
			}
			if f.PrimaryKey {
				pks = append(pks, fmt.Sprintf("`%s`", f.DBName))
			}
			cols = append(cols, fmt.Sprintf("`%s` TEXT", f.DBName))
		}
		pkClause := ""
		if len(pks) > 0 {
			pkClause = fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ", "))
		}
		if err := gdb.Exec(fmt.Sprintf("CREATE TABLE `%s` (%s%s)", stmt.Schema.Table, strings.Join(cols, ", "), pkClause)).Error; err != nil {
			t.Fatalf("create sqlite guilds table failed: %v", err)
		}
	}

	store := gormstore.New(gdb)
	reg := bbscore.NewProviderRegistry(bbscore.DefaultProviderKey)
	reg.Register(store)
	svc := bbscore.NewService(reg, store, bbscore.NewIdentityGenerator([]byte("guild-locale-test-secret-32bytes")), bbscore.NewPolicyManager(bbscore.PolicyConfig{}), nil)
	syncer := discordsync.NewSyncer(svc)
	comp := components.New(ctx, components.Config{}, &database.DB{DB: gdb})
	sys := NewSystem(comp, svc, syncer, nil)

	guildID := snowflake.ID(68001)
	if err := gdb.Create(&models.Guild{ID: guildID, Name: "NonCommunityGuild"}).Error; err != nil {
		t.Fatalf("create guild failed: %v", err)
	}
	if err := comp.SetGuildLocaleOverride(ctx, guildID, discord.LocaleJapanese); err != nil {
		t.Fatalf("SetGuildLocaleOverride failed: %v", err)
	}

	client := newInternalMockClient(t, nil)
	client.Caches = cache.New(cache.WithCaches(cache.FlagsAll))
	client.Caches.AddGuild(discord.CacheGuild{
		ID:              guildID,
		Name:            "NonCommunityGuild",
		PreferredLocale: "en-US",
	})

	payload, err := json.Marshal(map[string]any{
		"type":           discord.InteractionTypeApplicationCommand,
		"id":             snowflake.New(time.Now()).String(),
		"application_id": "999000",
		"token":          "test-cmd-token",
		"guild_id":       guildID.String(),
		"guild_locale":   "en-US",
		"locale":         "en-US",
		"channel":        map[string]any{"id": "555001", "type": 0},
		"member":         map[string]any{"user": map[string]any{"id": "123456", "username": "tester"}},
		"data": map[string]any{
			"id":   "111",
			"name": "bbs",
			"type": discord.ApplicationCommandTypeSlash,
		},
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var aci discord.ApplicationCommandInteraction
	if err := json.Unmarshal(payload, &aci); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	cmdEv := &handler.CommandEvent{
		ApplicationCommandInteractionCreate: &events.ApplicationCommandInteractionCreate{
			GenericEvent:                  events.NewGenericEvent(client, 0, 0),
			ApplicationCommandInteraction: aci,
		},
		Ctx: ctx,
	}

	// DB override ("ja") must win even when both cached PreferredLocale and event GuildLocale() are "en-US".
	if got := sys.guildLocale(ctx, cmdEv); got != discord.LocaleJapanese {
		t.Fatalf("expected sys.guildLocale ja via DB override, got %q", got)
	}

	// Clearing override falls back cleanly to Discord PreferredLocale ("en-US").
	if err := comp.SetGuildLocaleOverride(ctx, guildID, ""); err != nil {
		t.Fatalf("clear override failed: %v", err)
	}
	if got := sys.guildLocale(ctx, cmdEv); got != discord.LocaleEnglishUS {
		t.Fatalf("expected sys.guildLocale en-US after reset, got %q", got)
	}

	// When override, cached PreferredLocale, and event.GuildLocale() are all empty,
	// sys.guildLocale must fall back to i18n.DefaultLocale() ("ja"), NOT invoking user's e.Locale() ("en-US").
	emptyGuild := snowflake.ID(68002)
	emptyPayload, err := json.Marshal(map[string]any{
		"type":           discord.InteractionTypeApplicationCommand,
		"id":             snowflake.New(time.Now()).String(),
		"application_id": "999000",
		"token":          "test-cmd-token-empty",
		"guild_id":       emptyGuild.String(),
		"locale":         "en-US",
		"channel":        map[string]any{"id": "555002", "type": 0},
		"member":         map[string]any{"user": map[string]any{"id": "123456", "username": "tester"}},
		"data": map[string]any{
			"id":   "111",
			"name": "bbs",
			"type": discord.ApplicationCommandTypeSlash,
		},
	})
	if err != nil {
		t.Fatalf("marshal empty guild payload failed: %v", err)
	}
	var aciEmpty discord.ApplicationCommandInteraction
	if err := json.Unmarshal(emptyPayload, &aciEmpty); err != nil {
		t.Fatalf("unmarshal empty guild payload failed: %v", err)
	}
	emptyEv := &handler.CommandEvent{
		ApplicationCommandInteractionCreate: &events.ApplicationCommandInteractionCreate{
			GenericEvent:                  events.NewGenericEvent(client, 0, 0),
			ApplicationCommandInteraction: aciEmpty,
		},
		Ctx: ctx,
	}
	if got := sys.guildLocale(ctx, emptyEv); got != i18n.DefaultLocale() {
		t.Fatalf("expected sys.guildLocale to fall back to default %q rather than user locale en-US, got %q", i18n.DefaultLocale(), got)
	}
	sysNoComp := &System{Syncer: syncer}
	if got := sysNoComp.guildLocale(ctx, emptyEv); got != i18n.DefaultLocale() {
		t.Fatalf("expected sysNoComp.guildLocale fallback to %q, got %q", i18n.DefaultLocale(), got)
	}
}

func TestPostActionMenu_Permissions_Boundaries_And_States(t *testing.T) {
	ctx := context.Background()
	opID := snowflake.ID(940001)
	authorID := snowflake.ID(940002)
	globalAuthorID := snowflake.ID(940005)
	regularID := snowflake.ID(940003)
	adminID := snowflake.ID(940004)
	guildA := snowflake.ID(940010)
	guildB := snowflake.ID(940011)

	store, svc, sys := newInternalTestEnv(t, bbscore.PolicyConfig{Version: "2026-04"}, opID)
	for _, gid := range []snowflake.ID{guildA, guildB} {
		if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{GuildID: gid, Enabled: true, ActorUserID: opID}); err != nil {
			t.Fatalf("SetupGuild %d failed: %v", gid, err)
		}
	}
	for _, uid := range []snowflake.ID{authorID, globalAuthorID, regularID, adminID} {
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
	}

	localCat, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{Scope: bbscore.ScopeLocal, GuildID: &guildA, Name: "ローカル"})
	if err != nil {
		t.Fatalf("CreateCategory local failed: %v", err)
	}
	localBoard, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{CategoryID: localCat.ID, Scope: bbscore.ScopeLocal, GuildID: &guildA, Name: "ローカル板"})
	if err != nil {
		t.Fatalf("CreateBoard local failed: %v", err)
	}
	localMountID := uuid.New()
	if err := store.CreateBoardMount(ctx, &bbscore.BoardMount{
		ID: localMountID, BoardID: localBoard.ID, GuildID: guildA, ChannelID: snowflake.ID(740001), Active: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateBoardMount local failed: %v", err)
	}
	_, localThread, localPost, err := svc.CreateThread(ctx, bbscore.CreateThreadInput{
		BoardID: localBoard.ID, GuildID: guildA, UserID: authorID, Title: "ローカルスレッド", RawBody: "ローカル本文",
	})
	if err != nil {
		t.Fatalf("CreateThread local failed: %v", err)
	}
	localMirrorID := uuid.New()
	if err := store.CreateThreadMirror(ctx, &bbscore.ThreadMirror{
		ID: localMirrorID, ThreadID: localThread.ID, BoardMountID: localMountID, GuildID: guildA, DiscordThreadID: snowflake.ID(740010), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateThreadMirror local failed: %v", err)
	}

	globalCat, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{Scope: bbscore.ScopeGlobal, Name: "グローバル"})
	if err != nil {
		t.Fatalf("CreateCategory global failed: %v", err)
	}
	globalBoard, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{CategoryID: globalCat.ID, Scope: bbscore.ScopeGlobal, Name: "グローバル板"})
	if err != nil {
		t.Fatalf("CreateBoard global failed: %v", err)
	}
	globalMountID := uuid.New()
	if err := store.CreateBoardMount(ctx, &bbscore.BoardMount{
		ID: globalMountID, BoardID: globalBoard.ID, GuildID: guildA, ChannelID: snowflake.ID(740002), Active: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateBoardMount global failed: %v", err)
	}
	_, globalThread, globalPost, err := svc.CreateThread(ctx, bbscore.CreateThreadInput{
		BoardID: globalBoard.ID, GuildID: guildA, UserID: globalAuthorID, Title: "グローバルスレッド", RawBody: "グローバル本文",
	})
	if err != nil {
		t.Fatalf("CreateThread global failed: %v", err)
	}
	globalMirrorID := uuid.New()
	if err := store.CreateThreadMirror(ctx, &bbscore.ThreadMirror{
		ID: globalMirrorID, ThreadID: globalThread.ID, BoardMountID: globalMountID, GuildID: guildA, DiscordThreadID: snowflake.ID(740020), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateThreadMirror global failed: %v", err)
	}

	pv := discordsync.PayloadValidator{}
	mockClient := newInternalMockClient(t, nil)

	findButtons := func(mc discord.MessageCreate) map[string]discord.ButtonComponent {
		out := make(map[string]discord.ButtonComponent)
		for _, lc := range mc.Components {
			cont, ok := lc.(discord.ContainerComponent)
			if !ok {
				continue
			}
			for _, sc := range cont.Components {
				row, ok := sc.(discord.ActionRowComponent)
				if !ok {
					continue
				}
				for _, ic := range row.Components {
					if btn, ok := ic.(discord.ButtonComponent); ok && btn.CustomID != "" {
						out[btn.CustomID] = btn
					}
				}
			}
		}
		return out
	}

	var respType discord.InteractionResponseType
	var respData any
	capComp := func(rt discord.InteractionResponseType, d any) {
		respType = rt
		respData = d
	}

	// 1. Regular user on local board sees Reply + Report, never PosterHash / AuthorUserID / mod buttons.
	regEv := newInternalComponentEvent(t, mockClient, guildA, regularID, "/bbs/post/menu/"+localPost.ID.String(),
		map[string]string{"post_id": localPost.ID.String()}, capComp)
	if err := sys.handlePostMenuButton(regEv); err != nil {
		t.Fatalf("regular menu failed: %v", err)
	}
	regMC, ok := respData.(discord.MessageCreate)
	if respType != discord.InteractionResponseTypeCreateMessage || !ok || !regMC.Flags.Has(discord.MessageFlagEphemeral) {
		t.Fatalf("expected ephemeral menu for regular user, got type=%d data=%#v", respType, respData)
	}
	if err := pv.ValidateMessageCreate(regMC); err != nil {
		t.Fatalf("regular menu ValidateMessageCreate failed: %v", err)
	}
	regRaw, _ := json.Marshal(regMC)
	regStr := string(regRaw)
	if strings.Contains(regStr, localPost.PosterHash) || strings.Contains(regStr, authorID.String()) || strings.Contains(regStr, "/bbs/mod/") {
		t.Fatalf("regular user menu leaked sensitive/mod data: %s", regStr)
	}
	regBtns := findButtons(regMC)
	if _, ok := regBtns["/bbs/post/reply/"+localPost.ID.String()]; !ok {
		t.Fatalf("expected reply button for regular user, got %v", regBtns)
	}
	if _, ok := regBtns["/bbs/report/post/"+localPost.ID.String()]; !ok {
		t.Fatalf("expected report button for regular user, got %v", regBtns)
	}

	// 2. Guild admin on local board sees PosterHash + 3 mod buttons, never AuthorUserID.
	admEv := newInternalAdminComponentEvent(t, mockClient, guildA, adminID, "/bbs/post/menu/"+localPost.ID.String(),
		map[string]string{"post_id": localPost.ID.String()}, capComp)
	if err := sys.handlePostMenuButton(admEv); err != nil {
		t.Fatalf("admin menu failed: %v", err)
	}
	admMC := respData.(discord.MessageCreate)
	if err := pv.ValidateMessageCreate(admMC); err != nil {
		t.Fatalf("admin menu ValidateMessageCreate failed: %v", err)
	}
	admRaw, _ := json.Marshal(admMC)
	admStr := string(admRaw)
	if !strings.Contains(admStr, localPost.PosterHash) {
		t.Fatalf("expected PosterHash in local admin menu, got %s", admStr)
	}
	if strings.Contains(admStr, authorID.String()) {
		t.Fatalf("local admin menu leaked AuthorUserID: %s", admStr)
	}
	admBtns := findButtons(admMC)
	aboneBtn, okAb := admBtns["/bbs/mod/abone/"+localPost.ID.String()]
	archBtn, okAr := admBtns["/bbs/mod/archive/"+localThread.ID.String()]
	_, okBan := admBtns["/bbs/mod/ban/"+localPost.ID.String()]
	if !okAb || !okAr || !okBan || aboneBtn.Disabled || archBtn.Disabled {
		t.Fatalf("expected all 3 enabled mod buttons on active post, got %+v", admBtns)
	}

	// 3. Aboned post disables abone button & hides Reply; Archived thread disables archive button too.
	if _, err := svc.AbonePostLocal(ctx, guildA, localPost.ID, "spam"); err != nil {
		t.Fatalf("AbonePostLocal failed: %v", err)
	}
	admAbonedEv := newInternalAdminComponentEvent(t, mockClient, guildA, adminID, "/bbs/post/menu/"+localPost.ID.String(),
		map[string]string{"post_id": localPost.ID.String()}, capComp)
	if err := sys.handlePostMenuButton(admAbonedEv); err != nil {
		t.Fatalf("admin menu after abone failed: %v", err)
	}
	abonedBtns := findButtons(respData.(discord.MessageCreate))
	if _, hasReply := abonedBtns["/bbs/post/reply/"+localPost.ID.String()]; hasReply {
		t.Fatal("aboned post menu must hide Reply button")
	}
	if !abonedBtns["/bbs/mod/abone/"+localPost.ID.String()].Disabled {
		t.Fatal("aboned post menu must disable abone button")
	}
	if abonedBtns["/bbs/mod/archive/"+localThread.ID.String()].Disabled {
		t.Fatal("active thread archive button must remain enabled")
	}

	if _, err := svc.ArchiveThreadLocal(ctx, guildA, localThread.ID); err != nil {
		t.Fatalf("ArchiveThreadLocal failed: %v", err)
	}
	admArchEv := newInternalAdminComponentEvent(t, mockClient, guildA, adminID, "/bbs/post/menu/"+localPost.ID.String(),
		map[string]string{"post_id": localPost.ID.String()}, capComp)
	if err := sys.handlePostMenuButton(admArchEv); err != nil {
		t.Fatalf("admin menu after archive failed: %v", err)
	}
	archBtns := findButtons(respData.(discord.MessageCreate))
	if !archBtns["/bbs/mod/archive/"+localThread.ID.String()].Disabled {
		t.Fatal("archived thread menu must disable archive button")
	}

	// 4. Guild admin on global board sees global mod note + Report, never PosterHash or local /bbs/mod/ buttons.
	globAdmEv := newInternalAdminComponentEvent(t, mockClient, guildA, adminID, "/bbs/post/menu/"+globalPost.ID.String(),
		map[string]string{"post_id": globalPost.ID.String()}, capComp)
	if err := sys.handlePostMenuButton(globAdmEv); err != nil {
		t.Fatalf("global admin menu failed: %v", err)
	}
	globMC := respData.(discord.MessageCreate)
	if err := pv.ValidateMessageCreate(globMC); err != nil {
		t.Fatalf("global admin menu ValidateMessageCreate failed: %v", err)
	}
	globRaw, _ := json.Marshal(globMC)
	globStr := string(globRaw)
	if strings.Contains(globStr, globalPost.PosterHash) || strings.Contains(globStr, globalAuthorID.String()) || strings.Contains(globStr, "/bbs/mod/") {
		t.Fatalf("global admin menu leaked PosterHash/AuthorUserID/local mod buttons: %s", globStr)
	}
	if !strings.Contains(globStr, i18n.TranslateText(discord.LocaleJapanese, "components.bbs.post_menu.global_mod_note")) {
		t.Fatalf("expected global mod note in global admin menu, got %s", globStr)
	}

	// 5. Cross-guild boundary rejects foreign local post and unsubscribed global post across menu/reply/report.
	permTitle := i18n.TranslateText(discord.LocaleJapanese, "components.bbs.errors.title.permission")
	for _, target := range []struct {
		name   string
		postID uuid.UUID
	}{
		{"foreign_local", localPost.ID},
		{"unsubscribed_global", globalPost.ID},
	} {
		for _, act := range []struct {
			op string
			fn func(*handler.ComponentEvent) error
		}{
			{"menu", sys.handlePostMenuButton},
			{"reply", sys.handleReplyPostButton},
			{"report", sys.handleReportPostButton},
		} {
			ev := newInternalComponentEvent(t, mockClient, guildB, regularID, "/bbs/post/menu/"+target.postID.String(),
				map[string]string{"post_id": target.postID.String()}, capComp)
			if err := act.fn(ev); err != nil {
				t.Fatalf("%s %s returned error: %v", target.name, act.op, err)
			}
			rawErr, _ := json.Marshal(respData)
			if respType != discord.InteractionResponseTypeCreateMessage || !strings.Contains(string(rawErr), permTitle) {
				t.Fatalf("expected permission error for %s %s, got type=%d body=%s", target.name, act.op, respType, rawErr)
			}
		}
	}
}

func TestGlobalPostMenuAndReply_AvoidsDuplicateBoardMountQuery(t *testing.T) {
	ctx := context.Background()
	opID := snowflake.ID(950001)
	authorID := snowflake.ID(950002)
	regularID := snowflake.ID(950003)
	guildA := snowflake.ID(950010)

	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := gdb.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	for _, m := range []any{&models.User{}, &models.Guild{}} {
		stmt := &gorm.Statement{DB: gdb}
		if err := stmt.Parse(m); err != nil {
			t.Fatalf("parse schema %T failed: %v", m, err)
		}
		var cols, pks []string
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" {
				continue
			}
			if f.PrimaryKey {
				pks = append(pks, fmt.Sprintf("`%s`", f.DBName))
			}
			cols = append(cols, fmt.Sprintf("`%s` TEXT", f.DBName))
		}
		pkClause := ""
		if len(pks) > 0 {
			pkClause = fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ", "))
		}
		if err := gdb.Exec(fmt.Sprintf("CREATE TABLE `%s` (%s%s)", stmt.Schema.Table, strings.Join(cols, ", "), pkClause)).Error; err != nil {
			t.Fatalf("create table %s failed: %v", stmt.Schema.Table, err)
		}
	}
	prov := gormstore.New(gdb)
	reg := bbscore.NewProviderRegistry(bbscore.DefaultProviderKey)
	reg.Register(prov)
	svc := bbscore.NewService(reg, prov, bbscore.NewIdentityGenerator([]byte("dedup-mount-query-test-secret-32b")), bbscore.NewPolicyManager(bbscore.PolicyConfig{Version: "2026-04"}), []snowflake.ID{opID})
	syncer := discordsync.NewSyncer(svc)
	comp := components.New(ctx, components.Config{}, &database.DB{DB: gdb})
	sys := NewSystem(comp, svc, syncer, nil)

	if _, err := svc.SetupGuild(ctx, bbscore.GuildSetupInput{GuildID: guildA, Enabled: true, ActorUserID: opID}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	for _, uid := range []snowflake.ID{authorID, regularID} {
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
	}
	cat, err := svc.CreateCategory(ctx, bbscore.CreateCategoryInput{Scope: bbscore.ScopeGlobal, Name: "GlobalCat"})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbscore.CreateBoardInput{CategoryID: cat.ID, Scope: bbscore.ScopeGlobal, Name: "GlobalBoard"})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	mountID := uuid.New()
	if err := prov.CreateBoardMount(ctx, &bbscore.BoardMount{
		ID: mountID, BoardID: board.ID, GuildID: guildA, ChannelID: snowflake.ID(750001), Active: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	_, th, post, err := svc.CreateThread(ctx, bbscore.CreateThreadInput{
		BoardID: board.ID, GuildID: guildA, UserID: authorID, Title: "Global Thread", RawBody: "Global Body",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	if err := prov.CreateThreadMirror(ctx, &bbscore.ThreadMirror{
		ID: uuid.New(), ThreadID: th.ID, BoardMountID: mountID, GuildID: guildA, DiscordThreadID: snowflake.ID(750010), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateThreadMirror failed: %v", err)
	}

	var mountLookupQueries int
	if err := gdb.Callback().Query().Before("gorm:query").Register("test_count_mount_queries", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "bbs_board_mounts" {
			mountLookupQueries++
		}
	}); err != nil {
		t.Fatalf("register mount query callback failed: %v", err)
	}
	defer func() { _ = gdb.Callback().Query().Remove("test_count_mount_queries") }()

	mockClient := newInternalMockClient(t, nil)
	var respType discord.InteractionResponseType
	capComp := func(rt discord.InteractionResponseType, _ any) {
		respType = rt
	}

	mountLookupQueries = 0
	menuEv := newInternalComponentEvent(t, mockClient, guildA, regularID, "/bbs/post/menu/"+post.ID.String(),
		map[string]string{"post_id": post.ID.String()}, capComp)
	if err := sys.handlePostMenuButton(menuEv); err != nil {
		t.Fatalf("handlePostMenuButton failed: %v", err)
	}
	if respType != discord.InteractionResponseTypeCreateMessage {
		t.Fatalf("expected menu response CreateMessage, got %d", respType)
	}
	if mountLookupQueries != 1 {
		t.Fatalf("expected exactly 1 bbs_board_mounts query in handlePostMenuButton, got %d", mountLookupQueries)
	}

	mountLookupQueries = 0
	replyEv := newInternalComponentEvent(t, mockClient, guildA, regularID, "/bbs/post/reply/"+post.ID.String(),
		map[string]string{"post_id": post.ID.String()}, capComp)
	if err := sys.handleReplyPostButton(replyEv); err != nil {
		t.Fatalf("handleReplyPostButton failed: %v", err)
	}
	if respType != discord.InteractionResponseTypeModal {
		t.Fatalf("expected reply response Modal, got %d", respType)
	}
	if mountLookupQueries != 1 {
		t.Fatalf("expected exactly 1 bbs_board_mounts query in handleReplyPostButton, got %d", mountLookupQueries)
	}
}
