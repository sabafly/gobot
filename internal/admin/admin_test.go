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

package admin_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
	"github.com/sabafly/gobot/internal/i18n"
)

func createSQLiteModelTable(t *testing.T, db *gorm.DB, model any) {
	t.Helper()
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		t.Fatalf("parse schema for %T failed: %v", model, err)
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
	query := fmt.Sprintf("CREATE TABLE `%s` (%s%s)", stmt.Schema.Table, strings.Join(cols, ", "), pkClause)
	if err := db.Exec(query).Error; err != nil {
		t.Fatalf("create table %s failed: %v", stmt.Schema.Table, err)
	}
}

func newTestAdminDB(t *testing.T) *database.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sqlDB failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	createSQLiteModelTable(t, gdb, &models.User{})
	createSQLiteModelTable(t, gdb, &models.Guild{})
	if err := gdb.AutoMigrate(&models.AdminSession{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("automigrate admin models failed: %v", err)
	}
	if err := gdb.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate bbs models failed: %v", err)
	}
	return &database.DB{DB: gdb}
}

func TestRBAC_MatrixAndValidation(t *testing.T) {
	cfg := components.ConfigAdmin{
		Roles: map[string]components.ConfigAdminRole{
			"content_admin": {
				Permissions: []string{"dashboard.read", "bbs.read", "bbs.moderate", "bbs.admin"},
			},
		},
		Users: []components.ConfigAdminUser{
			{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin},
			{UserID: snowflake.ID(1002), Role: admin.RoleBBSModerator},
			{UserID: snowflake.ID(1003), Role: admin.RoleViewer},
			{UserID: snowflake.ID(1004), Role: "content_admin"},
		},
	}
	rbac, err := admin.NewRBAC(cfg)
	if err != nil {
		t.Fatalf("NewRBAC failed: %v", err)
	}

	role, perms, ok := rbac.ResolveUser(snowflake.ID(1001))
	if !ok || role != admin.RoleSuperAdmin || len(perms) != len(admin.AllPermissions()) {
		t.Fatalf("unexpected super_admin resolution: ok=%v role=%q perms=%v", ok, role, perms)
	}

	role, perms, ok = rbac.ResolveUser(snowflake.ID(1002))
	if !ok || role != admin.RoleBBSModerator {
		t.Fatalf("unexpected bbs_moderator: ok=%v role=%q", ok, role)
	}
	if !admin.HasPermission(perms, admin.PermBBSModerate) || admin.HasPermission(perms, admin.PermBBSAdmin) || admin.HasPermission(perms, admin.PermGuildsWrite) {
		t.Fatalf("unexpected bbs_moderator perms: %v", perms)
	}

	role, perms, ok = rbac.ResolveUser(snowflake.ID(1003))
	if !ok || role != admin.RoleViewer {
		t.Fatalf("unexpected viewer: ok=%v role=%q", ok, role)
	}
	if !admin.HasPermission(perms, admin.PermDashboardRead) || admin.HasPermission(perms, admin.PermBBSModerate) || admin.HasPermission(perms, admin.PermOpsExecute) {
		t.Fatalf("unexpected viewer perms: %v", perms)
	}

	role, perms, ok = rbac.ResolveUser(snowflake.ID(1004))
	if !ok || role != "content_admin" || !admin.HasPermission(perms, admin.PermBBSAdmin) {
		t.Fatalf("unexpected custom role: ok=%v role=%q perms=%v", ok, role, perms)
	}

	if _, _, ok := rbac.ResolveUser(snowflake.ID(9999)); ok {
		t.Fatal("expected unknown user to be rejected")
	}

	badConfigs := []components.ConfigAdmin{
		{Roles: map[string]components.ConfigAdminRole{"": {Permissions: []string{"dashboard.read"}}}},
		{Roles: map[string]components.ConfigAdminRole{"bad": {Permissions: []string{"nonexistent.perm"}}}},
		{Users: []components.ConfigAdminUser{{UserID: 0, Role: admin.RoleViewer}}},
		{Users: []components.ConfigAdminUser{{UserID: 123, Role: "missing_role"}}},
	}
	for i, bad := range badConfigs {
		if _, err := admin.NewRBAC(bad); err == nil {
			t.Fatalf("case %d: expected NewRBAC error for invalid config", i)
		}
	}
}

func TestLogHub_RingBufferTeeAndRedaction(t *testing.T) {
	hub := admin.NewLogHub(3)
	var outBuf bytes.Buffer
	base := slog.NewTextHandler(&outBuf, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
	logger := slog.New(admin.NewTeeHandler(base, hub)).With("component", "test")

	subCh, unsub := hub.Subscribe(1)
	defer unsub()

	// Fill slow subscriber buffer and exceed LogHub ring capacity (3).
	logger.Debug("msg one", "token", "super-secret-value")
	logger.Info("msg two", "gorm_dsn", "user:pass@tcp(db)/bot")
	logger.Warn("msg three", "count", 3)
	logger.Error("msg four", "err", "boom")

	// Wait a beat cleanly on synchronous Append -> Stats must be (3, 3).
	cnt, capVal := hub.Stats()
	if cnt != 3 || capVal != 3 {
		t.Fatalf("expected stats (3, 3), got (%d, %d)", cnt, capVal)
	}
	if !strings.Contains(outBuf.String(), "msg four") {
		t.Fatalf("expected downstream logger output preserved, got %q", outBuf.String())
	}

	// Slow subscriber must have received the first message without blocking emitters.
	first := <-subCh
	if first.Attrs["token"] != "[REDACTED]" {
		t.Fatalf("expected secret token redacted in LogHub entry, got %+v", first.Attrs)
	}

	entries, nextSeq := hub.List(admin.LogFilter{})
	if len(entries) != 3 || entries[0].Message != "msg two" || entries[2].Message != "msg four" || nextSeq != 5 {
		t.Fatalf("unexpected ring buffer entries=%+v nextSeq=%d", entries, nextSeq)
	}
	if entries[0].Attrs["gorm_dsn"] != "[REDACTED]" {
		t.Fatalf("expected dsn redacted, got %+v", entries[0].Attrs)
	}

	warnEntries, _ := hub.List(admin.LogFilter{MinLevel: "WARN"})
	if len(warnEntries) != 2 || warnEntries[0].Message != "msg three" || warnEntries[1].Message != "msg four" {
		t.Fatalf("unexpected WARN filtered entries: %+v", warnEntries)
	}

	qEntries, _ := hub.List(admin.LogFilter{Query: "four"})
	if len(qEntries) != 1 || qEntries[0].Message != "msg four" {
		t.Fatalf("unexpected query filtered entries: %+v", qEntries)
	}
}

func newMockDiscordOAuthServer(t *testing.T, expectedSecret string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/oauth2/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			if r.Form.Get("client_secret") != expectedSecret {
				http.Error(w, "invalid client_secret", http.StatusUnauthorized)
				return
			}
			code := r.Form.Get("code")
			if code == "" || code == "bad-code" {
				http.Error(w, "invalid code", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "acc-" + code,
				"token_type":   "Bearer",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/users/@me":
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			users := map[string]map[string]any{
				"acc-code-super":    {"id": "1001", "username": "superop", "global_name": "Super Op"},
				"acc-code-viewer":   {"id": "1003", "username": "viewop", "global_name": "View Op"},
				"acc-code-outsider": {"id": "9999", "username": "outsider", "global_name": "Outsider"},
			}
			u, ok := users[tok]
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(u)
		default:
			http.NotFound(w, r)
		}
	}))
}

func doJSONReq(t *testing.T, method, targetURL, token string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, targetURL, reader)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http %s %s failed: %v", method, targetURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	rawResp, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, rawResp
}

func TestAdminServer_OAuthFlow_RBAC_CommonEndpoints(t *testing.T) {
	t.Setenv("ADMIN_DISCORD_CLIENT_SECRET", "")
	t.Setenv("DISCORD_CLIENT_SECRET", "")
	t.Setenv("ADMIN_PUBLIC_URL", "")
	t.Setenv("ADMIN_BASE_PATH", "")
	if err := i18n.LoadLocales("../../locales"); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	db := newTestAdminDB(t)
	ctx := context.Background()

	// Fail closed when client secret / public_url is missing.
	if _, err := admin.NewServer(admin.ServerDeps{
		DB: db,
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth:     components.ConfigAdminOAuth{ClientID: "999000"},
		},
	}); err == nil {
		t.Fatal("expected fail-closed error when client_secret is missing")
	}

	discordMock := newMockDiscordOAuthServer(t, "oauth-test-secret")
	defer discordMock.Close()

	guildID := snowflake.ID(888001)
	if err := db.DB.Create(&models.Guild{ID: guildID, Name: "NonCommunity"}).Error; err != nil {
		t.Fatalf("create guild failed: %v", err)
	}

	comp := components.New(ctx, components.Config{LocaleDir: "../../locales"}, db)
	var localeHookCalls int
	comp.RegisterGuildLocaleChangeHook(func(_ context.Context, _ *bot.Client, gid snowflake.ID, _ discord.Locale) {
		if gid == guildID {
			localeHookCalls++
		}
	})

	store := gormstore.New(db.DB)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	bbsSvc := bbs.NewService(reg, store, bbs.NewIdentityGenerator([]byte("admin-test-secret-32-bytes-min!!")), bbs.NewPolicyManager(bbs.PolicyConfig{Version: "2026-01"}), []snowflake.ID{snowflake.ID(1001)})
	bbsSync := discordsync.NewSyncer(bbsSvc)

	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	client.Caches.AddGuild(discord.CacheGuild{
		ID:              guildID,
		Name:            "NonCommunity",
		PreferredLocale: "en-US",
		MemberCount:     42,
	})

	logHub := admin.NewLogHub(100)
	logHub.Append(admin.LogEntry{Level: "INFO", Message: "boot complete"})

	srv, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth: components.ConfigAdminOAuth{
				ClientSecret: "oauth-test-secret",
			},
			Users: []components.ConfigAdminUser{
				{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin},
				{UserID: snowflake.ID(1003), Role: admin.RoleViewer},
			},
		},
		DB:                 db,
		Components:         comp,
		Client:             client,
		BBSService:         bbsSvc,
		BBSSyncer:          bbsSync,
		LogHub:             logHub,
		Version:            "vtest",
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Protected endpoints require bearer auth.
	for _, ep := range []string{"/api/v1/auth/me", "/api/v1/overview", "/api/v1/logs", "/api/v1/guilds", "/api/v1/audit"} {
		st, _ := doJSONReq(t, http.MethodGet, ts.URL+ep, "", nil)
		if st != http.StatusUnauthorized {
			t.Fatalf("expected 401 unauthenticated for %s, got %d", ep, st)
		}
	}

	performLogin := func(code string) (int, admin.AuthPollResponse) {
		st, startRaw := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/discord/start", "", admin.AuthStartRequest{ClientInfo: "test-tui"})
		if st != http.StatusOK {
			t.Fatalf("start failed status=%d body=%s", st, startRaw)
		}
		var startResp admin.AuthStartResponse
		if err := json.Unmarshal(startRaw, &startResp); err != nil {
			t.Fatalf("unmarshal start failed: %v", err)
		}
		u, _ := url.Parse(startResp.AuthorizeURL)
		state := u.Query().Get("state")

		// Bad poll_secret must return 401 and NOT destroy pending entry.
		badPollStatus, _ := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/discord/poll", "", admin.AuthPollRequest{
			LoginID:    startResp.LoginID,
			PollSecret: "wrong-secret",
		})
		if badPollStatus != http.StatusUnauthorized {
			t.Fatalf("expected 401 on bad poll secret, got %d", badPollStatus)
		}

		cbURL := fmt.Sprintf("%s/api/v1/auth/discord/callback?code=%s&state=%s", ts.URL, url.QueryEscape(code), url.QueryEscape(state))
		cbResp, err := http.Get(cbURL)
		if err != nil {
			t.Fatalf("callback failed: %v", err)
		}
		_ = cbResp.Body.Close()

		// Replaying same CSRF state must fail with 400.
		replayResp, _ := http.Get(cbURL)
		_ = replayResp.Body.Close()
		if replayResp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 on CSRF state replay, got %d", replayResp.StatusCode)
		}

		pollStatus, pollRaw := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/discord/poll", "", admin.AuthPollRequest{
			LoginID:    startResp.LoginID,
			PollSecret: startResp.PollSecret,
		})
		if pollStatus != http.StatusOK {
			t.Fatalf("expected 200 from poll, got %d (%s)", pollStatus, string(pollRaw))
		}
		var pollResp admin.AuthPollResponse
		_ = json.Unmarshal(pollRaw, &pollResp)

		// Single-use claim check when approved.
		if pollResp.Status == "approved" {
			secondStatus, _ := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/discord/poll", "", admin.AuthPollRequest{
				LoginID:    startResp.LoginID,
				PollSecret: startResp.PollSecret,
			})
			if secondStatus != http.StatusNotFound {
				t.Fatalf("expected 404 on second poll claim, got %d", secondStatus)
			}
		}
		return cbResp.StatusCode, pollResp
	}

	// 2. Unauthorized Discord user denied.
	cbOutsiderStatus, outsiderPoll := performLogin("code-outsider")
	if cbOutsiderStatus != http.StatusForbidden || outsiderPoll.Status != "denied" {
		t.Fatalf("expected outsider rejected (cb=%d poll=%+v)", cbOutsiderStatus, outsiderPoll)
	}

	// 3. Super admin login & operations.
	cbSuperStatus, superPoll := performLogin("code-super")
	if cbSuperStatus != http.StatusOK || superPoll.Status != "approved" || superPoll.SessionToken == "" || superPoll.Principal.Role != admin.RoleSuperAdmin {
		t.Fatalf("super_admin login failed: cb=%d poll=%+v", cbSuperStatus, superPoll)
	}
	superTok := superPoll.SessionToken

	// Overview
	st, ovRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/overview", superTok, nil)
	if st != http.StatusOK {
		t.Fatalf("overview failed: %d %s", st, ovRaw)
	}
	var ov admin.OverviewResponse
	_ = json.Unmarshal(ovRaw, &ov)
	if ov.Version != "vtest" || !ov.DBConnected || ov.CachedGuilds != 1 || !ov.BBS.Available {
		t.Fatalf("unexpected overview payload: %+v", ov)
	}

	// Update guild locale_override + enable BBS
	ja := "ja"
	bbsOn := true
	st, updRaw := doJSONReq(t, http.MethodPatch, ts.URL+"/api/v1/guilds/"+guildID.String(), superTok, admin.UpdateGuildRequest{
		LocaleOverride: &ja,
		BBSEnabled:     &bbsOn,
	})
	if st != http.StatusOK {
		t.Fatalf("patch guild failed: %d %s", st, updRaw)
	}
	var gd admin.GuildDetailResponse
	_ = json.Unmarshal(updRaw, &gd)
	if gd.Guild.LocaleOverride != "ja" || gd.Guild.EffectiveLocale != "ja" || !gd.Guild.BBSEnabled || localeHookCalls != 1 {
		t.Fatalf("unexpected guild update result: %+v hookCalls=%d", gd, localeHookCalls)
	}

	// Ops reload locales
	st, opsRaw := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/ops/reload-locales", superTok, nil)
	if st != http.StatusOK || !strings.Contains(string(opsRaw), `"ja"`) {
		t.Fatalf("reload-locales failed: %d %s", st, opsRaw)
	}

	// Audit logs contain auth.login_denied, auth.login, guild.update, ops.reload_locales
	st, auditRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/audit?limit=50", superTok, nil)
	if st != http.StatusOK {
		t.Fatalf("audit list failed: %d %s", st, auditRaw)
	}
	var auditList admin.AuditListResponse
	_ = json.Unmarshal(auditRaw, &auditList)
	var actions []string
	var sawGuildResName bool
	for _, e := range auditList.Entries {
		actions = append(actions, e.Action)
		if e.Action == "guild.update" && e.ResourceType == "guild" && e.ResourceID == guildID.String() && e.ResourceName == "NonCommunity" {
			sawGuildResName = true
		}
	}
	for _, want := range []string{"auth.login_denied", "auth.login", "guild.update", "ops.reload_locales"} {
		if !slices.Contains(actions, want) {
			t.Fatalf("expected audit action %q in %v", want, actions)
		}
	}
	if !sawGuildResName {
		t.Fatalf("expected guild.update audit entry to carry ResourceName='NonCommunity', got %+v", auditList.Entries)
	}

	// 4. Viewer RBAC enforcement & session visibility.
	_, viewerPoll := performLogin("code-viewer")
	if viewerPoll.Status != "approved" || viewerPoll.Principal.Role != admin.RoleViewer {
		t.Fatalf("viewer login failed: %+v", viewerPoll)
	}
	viewerTok := viewerPoll.SessionToken

	// Viewer cannot mutate guilds or run ops
	en := "en-US"
	if st, _ := doJSONReq(t, http.MethodPatch, ts.URL+"/api/v1/guilds/"+guildID.String(), viewerTok, admin.UpdateGuildRequest{LocaleOverride: &en}); st != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer guild patch, got %d", st)
	}
	if st, _ := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/ops/reload-locales", viewerTok, nil); st != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer reload-locales, got %d", st)
	}

	// Viewer sees only own sessions; super_admin sees both.
	_, vSessRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/auth/sessions", viewerTok, nil)
	var vSess admin.SessionsResponse
	_ = json.Unmarshal(vSessRaw, &vSess)
	if len(vSess.Sessions) != 1 {
		t.Fatalf("expected viewer to see only 1 own session, got %d", len(vSess.Sessions))
	}
	_, sSessRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/auth/sessions", superTok, nil)
	var sSess admin.SessionsResponse
	_ = json.Unmarshal(sSessRaw, &sSess)
	if len(sSess.Sessions) < 2 {
		t.Fatalf("expected super_admin to see >=2 sessions, got %d", len(sSess.Sessions))
	}
	// Viewer cannot revoke super_admin's session.
	for _, ss := range sSess.Sessions {
		if ss.UserID == "1001" {
			if st, _ := doJSONReq(t, http.MethodDelete, ts.URL+"/api/v1/auth/sessions/"+ss.ID, viewerTok, nil); st != http.StatusForbidden {
				t.Fatalf("expected 403 revoking other user's session, got %d", st)
			}
		}
	}

	// Logout invalidates token immediately.
	if st, _ := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/logout", viewerTok, nil); st != http.StatusOK {
		t.Fatalf("logout failed: %d", st)
	}
	if st, _ := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/auth/me", viewerTok, nil); st != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", st)
	}

	// 5. SSE Log Stream delivers live entry.
	sseCtx, sseCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer sseCancel()
	sseReq, _ := http.NewRequestWithContext(sseCtx, http.MethodGet, ts.URL+"/api/v1/logs/stream", nil)
	sseReq.Header.Set("Authorization", "Bearer "+superTok)
	sseResp, err := http.DefaultClient.Do(sseReq)
	if err != nil {
		t.Fatalf("open SSE failed: %v", err)
	}
	defer func() { _ = sseResp.Body.Close() }()
	if sseResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 SSE, got %d", sseResp.StatusCode)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		logHub.Append(admin.LogEntry{Level: "WARN", Message: "live sse check"})
	}()

	scanner := bufio.NewScanner(sseResp.Body)
	foundLive := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "live sse check") {
			foundLive = true
			sseCancel()
			break
		}
	}
	if !foundLive {
		t.Fatal("expected live SSE log entry before timeout")
	}
}

func TestNormalizeBasePathAndPublicURL(t *testing.T) {
	validBasePaths := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"/", ""},
		{" / ", ""},
		{"/gobot/beta", "/gobot/beta"},
		{"/gobot/beta/", "/gobot/beta"},
		{"gobot/beta", "/gobot/beta"},
		{" /gobot/v1-beta/ ", "/gobot/v1-beta"},
	}
	for _, tc := range validBasePaths {
		got, err := admin.NormalizeBasePath(tc.in)
		if err != nil {
			t.Fatalf("NormalizeBasePath(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeBasePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	invalidBasePaths := []string{
		"https://example.com/a",
		"/a//b",
		"//",
		"/a/./b",
		"/a/../b",
		"..",
		".",
		"/a?x=1",
		"/a#frag",
		"/a/{id}",
		"/a b",
		"/a%20b",
		"/a\\b",
	}
	for _, raw := range invalidBasePaths {
		if got, err := admin.NormalizeBasePath(raw); err == nil {
			t.Fatalf("NormalizeBasePath(%q) expected error, got %q", raw, got)
		}
	}

	validPublicURLs := []struct {
		in         string
		wantURL    string
		wantPath   string
		wantOrigin string
	}{
		{"https://dash.sabafly.net/gobot/beta", "https://dash.sabafly.net/gobot/beta", "/gobot/beta", "https://dash.sabafly.net"},
		{"https://DASH.SABAFLY.NET:443/gobot/beta/", "https://dash.sabafly.net/gobot/beta", "/gobot/beta", "https://dash.sabafly.net"},
		{"http://127.0.0.1:8080/", "http://127.0.0.1:8080", "", "http://127.0.0.1:8080"},
		{"http://127.0.0.1:80", "http://127.0.0.1", "", "http://127.0.0.1"},
		{"http://[::1]:8080/admin", "http://[::1]:8080/admin", "/admin", "http://[::1]:8080"},
	}
	for _, tc := range validPublicURLs {
		gotURL, gotPath, gotOrig, err := admin.NormalizePublicURL(tc.in)
		if err != nil {
			t.Fatalf("NormalizePublicURL(%q) unexpected error: %v", tc.in, err)
		}
		if gotURL != tc.wantURL || gotPath != tc.wantPath || gotOrig != tc.wantOrigin {
			t.Fatalf("NormalizePublicURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.in, gotURL, gotPath, gotOrig, tc.wantURL, tc.wantPath, tc.wantOrigin)
		}
	}

	invalidPublicURLs := []string{
		"",
		"   ",
		"/gobot/beta",
		"ftp://example.com",
		"https://user:pass@example.com",
		"https://example.com/a?x=1",
		"https://example.com/a#f",
		"https://example.com/a/../b",
		"https://example.com/a//b",
		"https://example.com:",
		"https://example.com/a%20b",
	}
	for _, raw := range invalidPublicURLs {
		if u, p, o, err := admin.NormalizePublicURL(raw); err == nil {
			t.Fatalf("NormalizePublicURL(%q) expected error, got (%q, %q, %q)", raw, u, p, o)
		}
	}
}

func TestAdminServer_PublicURL_BasePath_EnvAndCORS(t *testing.T) {
	t.Setenv("ADMIN_DISCORD_CLIENT_SECRET", "")
	t.Setenv("DISCORD_CLIENT_SECRET", "")
	t.Setenv("ADMIN_PUBLIC_URL", "")
	t.Setenv("ADMIN_BASE_PATH", "")

	db := newTestAdminDB(t)
	discordMock := newMockDiscordOAuthServer(t, "secret-bp")
	defer discordMock.Close()

	guildID := snowflake.ID(888001)
	if err := db.DB.Create(&models.Guild{ID: guildID, Name: "BetaGuild"}).Error; err != nil {
		t.Fatalf("create guild failed: %v", err)
	}
	comp := components.New(context.Background(), components.Config{}, db)
	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	client.Caches.AddGuild(discord.CacheGuild{
		ID: guildID, Name: "BetaGuild", PreferredLocale: "en-US",
	})

	newTestServer := func(publicURL, basePath string) (*admin.Server, error) {
		return admin.NewServer(admin.ServerDeps{
			DB:         db,
			Components: comp,
			Client:     client,
			AdminConfig: components.ConfigAdmin{
				Enabled:   true,
				PublicURL: publicURL,
				BasePath:  basePath,
				OAuth:     components.ConfigAdminOAuth{ClientSecret: "secret-bp"},
				Users:     []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
			},
			OAuthAPIBase:       discordMock.URL,
			OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
			OAuthHTTPClient:    discordMock.Client(),
		})
	}

	// Fail closed on missing/invalid public_url or invalid base_path.
	for _, tc := range []struct {
		pub  string
		base string
	}{
		{"", ""},
		{"/relative-only", ""},
		{"https://dash.sabafly.net/gobot/beta", "https://dash.sabafly.net/gobot/beta"},
		{"https://dash.sabafly.net/gobot/beta", "/a/../b"},
	} {
		if srv, err := newTestServer(tc.pub, tc.base); err == nil || srv != nil {
			t.Fatalf("expected fail-closed error for public_url=%q base_path=%q, got srv=%v", tc.pub, tc.base, srv)
		}
	}

	// 1. Inherited base_path from public_url + strict routing + CORS/Origin verification.
	srv, err := newTestServer("https://dash.sabafly.net/gobot/beta/", "")
	if err != nil {
		t.Fatalf("newTestServer failed: %v", err)
	}
	if srv.PublicURL() != "https://dash.sabafly.net/gobot/beta" || srv.BasePath() != "/gobot/beta" {
		t.Fatalf("unexpected resolved server config: publicURL=%q basePath=%q", srv.PublicURL(), srv.BasePath())
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Complete full OAuth flow and check redirect_uri + overview + {id} route under base_path.
	st, startRaw := doJSONReq(t, http.MethodPost, ts.URL+"/gobot/beta/api/v1/auth/discord/start", "", admin.AuthStartRequest{ClientInfo: "bp-test"})
	if st != http.StatusOK {
		t.Fatalf("start under base_path failed: %d %s", st, startRaw)
	}
	var startResp admin.AuthStartResponse
	_ = json.Unmarshal(startRaw, &startResp)
	authU, _ := url.Parse(startResp.AuthorizeURL)
	wantRedirect := "https://dash.sabafly.net/gobot/beta/api/v1/auth/discord/callback"
	if gotRedirect := authU.Query().Get("redirect_uri"); gotRedirect != wantRedirect {
		t.Fatalf("authorize redirect_uri = %q, want %q", gotRedirect, wantRedirect)
	}
	cbResp, err := http.Get(fmt.Sprintf("%s/gobot/beta/api/v1/auth/discord/callback?code=code-super&state=%s", ts.URL, url.QueryEscape(authU.Query().Get("state"))))
	if err != nil {
		t.Fatalf("callback under base_path failed: %v", err)
	}
	_ = cbResp.Body.Close()
	if cbResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 callback, got %d", cbResp.StatusCode)
	}
	pollStatus, pollRaw := doJSONReq(t, http.MethodPost, ts.URL+"/gobot/beta/api/v1/auth/discord/poll", "", admin.AuthPollRequest{
		LoginID:    startResp.LoginID,
		PollSecret: startResp.PollSecret,
	})
	if pollStatus != http.StatusOK {
		t.Fatalf("poll failed: %d %s", pollStatus, pollRaw)
	}
	var pollResp admin.AuthPollResponse
	_ = json.Unmarshal(pollRaw, &pollResp)

	if ovSt, _ := doJSONReq(t, http.MethodGet, ts.URL+"/gobot/beta/api/v1/overview", pollResp.SessionToken, nil); ovSt != http.StatusOK {
		t.Fatalf("expected 200 on /gobot/beta/api/v1/overview, got %d", ovSt)
	}
	if gSt, _ := doJSONReq(t, http.MethodGet, ts.URL+"/gobot/beta/api/v1/guilds/"+guildID.String(), pollResp.SessionToken, nil); gSt != http.StatusOK {
		t.Fatalf("expected 200 on /gobot/beta/api/v1/guilds/{id}, got %d", gSt)
	}

	// Strict prefix isolation: root, partial-segment overlap, and bare prefix must 404.
	for _, missPath := range []string{"/api/v1/overview", "/gobot/beta2/api/v1/overview", "/gobot/beta"} {
		missSt, _ := doJSONReq(t, http.MethodGet, ts.URL+missPath, pollResp.SessionToken, nil)
		if missSt != http.StatusNotFound {
			t.Fatalf("expected 404 for %s, got %d", missPath, missSt)
		}
	}

	// CORS / Origin enforcement.
	runHTTP := func(method, target, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	noOrig := runHTTP(http.MethodGet, "/gobot/beta/api/v1/overview", "")
	if noOrig.Code != http.StatusUnauthorized || !slices.Contains(noOrig.Header().Values("Vary"), "Origin") || noOrig.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected no-Origin response: code=%d headers=%v", noOrig.Code, noOrig.Header())
	}

	matchOrig := runHTTP(http.MethodGet, "/gobot/beta/api/v1/overview", "https://dash.sabafly.net")
	if matchOrig.Header().Get("Access-Control-Allow-Origin") != "https://dash.sabafly.net" {
		t.Fatalf("expected ACAO for matching origin, got %q", matchOrig.Header().Get("Access-Control-Allow-Origin"))
	}

	preflightOK := runHTTP(http.MethodOptions, "/gobot/beta/api/v1/overview", "https://dash.sabafly.net")
	if preflightOK.Code != http.StatusNoContent ||
		preflightOK.Header().Get("Access-Control-Allow-Methods") == "" ||
		preflightOK.Header().Get("Access-Control-Allow-Headers") == "" ||
		preflightOK.Header().Get("Access-Control-Max-Age") != "600" {
		t.Fatalf("unexpected matching OPTIONS preflight: code=%d headers=%v", preflightOK.Code, preflightOK.Header())
	}

	preflightOut := runHTTP(http.MethodOptions, "/api/v1/overview", "https://dash.sabafly.net")
	if preflightOut.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for OPTIONS outside base_path, got %d", preflightOut.Code)
	}

	for _, badCase := range []struct {
		method string
		path   string
		origin string
	}{
		{http.MethodGet, "/gobot/beta/api/v1/overview", "https://evil.example"},
		{http.MethodOptions, "/gobot/beta/api/v1/overview", "https://evil.example"},
		{http.MethodGet, "/gobot/beta/api/v1/overview", "null"},
	} {
		rec := runHTTP(badCase.method, badCase.path, badCase.origin)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s (Origin=%q): expected 403, got %d", badCase.method, badCase.path, badCase.origin, rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("%s %s: unexpected ACAO on rejected origin", badCase.method, badCase.path)
		}
		var errEnv admin.ErrorResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &errEnv)
		if errEnv.Error.Code != "forbidden_origin" {
			t.Fatalf("expected error code forbidden_origin, got %+v", errEnv)
		}
	}

	for _, cbOrigin := range []string{"https://evil.example", "null"} {
		cbRec := runHTTP(http.MethodGet, "/gobot/beta/api/v1/auth/discord/callback?state=bad", cbOrigin)
		if cbRec.Code != http.StatusBadRequest {
			t.Fatalf("OAuth callback with Origin=%q should bypass CORS 403 and reach handler (400), got %d", cbOrigin, cbRec.Code)
		}
		if cbRec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("OAuth callback with Origin=%q must not emit ACAO", cbOrigin)
		}
	}

	// 2. Explicit base_path: "/" strips subpath for listener while keeping full redirect_uri.
	srvRoot, err := newTestServer("https://dash.sabafly.net/gobot/beta", "/")
	if err != nil {
		t.Fatalf("newTestServer root override failed: %v", err)
	}
	if srvRoot.BasePath() != "" {
		t.Fatalf("expected root BasePath \"\", got %q", srvRoot.BasePath())
	}
	tsRoot := httptest.NewServer(srvRoot.Handler())
	defer tsRoot.Close()

	stRoot, rawRoot := doJSONReq(t, http.MethodPost, tsRoot.URL+"/api/v1/auth/discord/start", "", admin.AuthStartRequest{})
	if stRoot != http.StatusOK {
		t.Fatalf("expected 200 on root start, got %d", stRoot)
	}
	var rootStartResp admin.AuthStartResponse
	_ = json.Unmarshal(rawRoot, &rootStartResp)
	rootAuthU, _ := url.Parse(rootStartResp.AuthorizeURL)
	if got := rootAuthU.Query().Get("redirect_uri"); got != wantRedirect {
		t.Fatalf("expected redirect_uri %q when base_path=/, got %q", wantRedirect, got)
	}
	if subMissSt, _ := doJSONReq(t, http.MethodGet, tsRoot.URL+"/gobot/beta/api/v1/overview", "", nil); subMissSt != http.StatusNotFound {
		t.Fatalf("expected 404 on /gobot/beta/api/v1/overview when base_path=/, got %d", subMissSt)
	}

	// 3. Env overrides take precedence over YAML.
	t.Setenv("ADMIN_PUBLIC_URL", "https://env.example.com/sub")
	t.Setenv("ADMIN_BASE_PATH", "/custom")
	srvEnv, err := newTestServer("http://127.0.0.1:8080", "/old")
	if err != nil {
		t.Fatalf("newTestServer env override failed: %v", err)
	}
	if srvEnv.PublicURL() != "https://env.example.com/sub" || srvEnv.BasePath() != "/custom" {
		t.Fatalf("unexpected env override server: publicURL=%q basePath=%q", srvEnv.PublicURL(), srvEnv.BasePath())
	}

	t.Setenv("ADMIN_BASE_PATH", "/")
	srvEnvRoot, err := newTestServer("http://127.0.0.1:8080", "/old")
	if err != nil {
		t.Fatalf("newTestServer env root override failed: %v", err)
	}
	if srvEnvRoot.PublicURL() != "https://env.example.com/sub" || srvEnvRoot.BasePath() != "" {
		t.Fatalf("unexpected env root override server: publicURL=%q basePath=%q", srvEnvRoot.PublicURL(), srvEnvRoot.BasePath())
	}
}

func TestLogHub_TeeHandlerCaptureLevel_And_AfterSeqPagination(t *testing.T) {
	hub := admin.NewLogHub(50)
	var outBuf bytes.Buffer
	base := slog.NewTextHandler(&outBuf, &slog.HandlerOptions{Level: slog.LevelInfo})
	tee := admin.NewTeeHandler(base, hub)
	logger := slog.New(tee)
	child := logger.With("mod", "bbs").WithGroup("sync")

	// 1. In default "auto" mode with INFO downstream, DEBUG logs must NOT be captured into LogHub.
	logger.Debug("hidden debug auto")
	child.Debug("child hidden debug auto")
	logger.Info("visible info auto")

	entries, _ := hub.List(admin.LogFilter{})
	if len(entries) != 1 || entries[0].Message != "visible info auto" {
		t.Fatalf("expected only INFO entry in auto mode, got %+v", entries)
	}
	if strings.Contains(outBuf.String(), "hidden debug auto") {
		t.Fatalf("expected DEBUG excluded from INFO stdout: %q", outBuf.String())
	}
	if lvl := hub.CaptureLevel(context.Background()); lvl.Mode != "auto" || lvl.EffectiveLevel != "INFO" {
		t.Fatalf("unexpected auto CaptureLevel: %+v", lvl)
	}

	// 2. Setting LogHub capture level to "debug" captures DEBUG into LogHub (and child loggers)
	// WITHOUT leaking DEBUG into INFO stdout.
	if resp, err := hub.SetCaptureLevel("debug"); err != nil || resp.Mode != "debug" || resp.EffectiveLevel != "DEBUG" {
		t.Fatalf("SetCaptureLevel(debug) unexpected: resp=%+v err=%v", resp, err)
	}
	logger.Debug("captured debug")
	child.Debug("child captured debug", "res", 42)
	if strings.Contains(outBuf.String(), "captured debug") {
		t.Fatalf("expected stdout INFO handler still not emitting DEBUG, got %q", outBuf.String())
	}
	entries, _ = hub.List(admin.LogFilter{})
	if len(entries) != 3 || entries[1].Message != "captured debug" || entries[2].Message != "child captured debug" {
		t.Fatalf("expected DEBUG entries captured for parent and child logger, got %+v", entries)
	}

	// 3. Setting capture level to "warn" filters LogHub while keeping stdout INFO intact.
	if resp, err := hub.SetCaptureLevel("warn"); err != nil || resp.Mode != "warn" || resp.EffectiveLevel != "WARN" {
		t.Fatalf("SetCaptureLevel(warn) unexpected: resp=%+v err=%v", resp, err)
	}
	logger.Info("stdout only info")
	logger.Warn("hub and stdout warn")
	if !strings.Contains(outBuf.String(), "stdout only info") {
		t.Fatalf("expected stdout to still receive INFO when hub is WARN")
	}
	entries, _ = hub.List(admin.LogFilter{})
	if entries[len(entries)-1].Message != "hub and stdout warn" || len(entries) != 4 {
		t.Fatalf("expected WARN capture only, got %+v", entries)
	}
	if _, err := hub.SetCaptureLevel("invalid-lvl"); err == nil {
		t.Fatal("expected SetCaptureLevel error for invalid level")
	}
	if _, err := hub.SetCaptureLevel("auto"); err != nil {
		t.Fatalf("SetCaptureLevel(auto) failed: %v", err)
	}

	// 4. AfterSeq pagination returns oldest unread slice (matched[:limit]) and accurate nextSeq,
	// while AfterSeq == 0 continues to return the latest slice.
	pgHub := admin.NewLogHub(50)
	for i := 1; i <= 10; i++ {
		pgHub.Append(admin.LogEntry{Level: "INFO", Message: fmt.Sprintf("msg-%02d", i)})
	}
	latest, latestNext := pgHub.List(admin.LogFilter{Limit: 3})
	if len(latest) != 3 || latest[0].Message != "msg-08" || latest[2].Message != "msg-10" || latestNext != 11 {
		t.Fatalf("unexpected latest slice (AfterSeq=0): entries=%+v next=%d", latest, latestNext)
	}

	page1, n1 := pgHub.List(admin.LogFilter{AfterSeq: 2, Limit: 3})
	if len(page1) != 3 || page1[0].Seq != 3 || page1[2].Seq != 5 || n1 != 6 {
		t.Fatalf("expected page1 seqs 3..5 next=6, got entries=%+v next=%d", page1, n1)
	}
	page2, n2 := pgHub.List(admin.LogFilter{AfterSeq: page1[len(page1)-1].Seq, Limit: 3})
	if len(page2) != 3 || page2[0].Seq != 6 || page2[2].Seq != 8 || n2 != 9 {
		t.Fatalf("expected page2 seqs 6..8 next=9, got entries=%+v next=%d", page2, n2)
	}
	page3, n3 := pgHub.List(admin.LogFilter{AfterSeq: page2[len(page2)-1].Seq, Limit: 3})
	if len(page3) != 2 || page3[0].Seq != 9 || page3[1].Seq != 10 || n3 != 11 {
		t.Fatalf("expected page3 seqs 9..10 next=11, got entries=%+v next=%d", page3, n3)
	}
}

func TestAdminServer_SecurityAndConsistencyFixes(t *testing.T) {
	if err := i18n.LoadLocales("../../locales"); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	// Exposure warnings for #13
	if ws := admin.ExposureWarnings("127.0.0.1:8080", "https://dash.example.com"); len(ws) != 0 {
		t.Fatalf("expected no warnings for loopback bind + https, got %v", ws)
	}
	if ws := admin.ExposureWarnings("127.0.0.1:8080", "http://localhost:8080"); len(ws) != 0 {
		t.Fatalf("expected no warnings for loopback http://localhost, got %v", ws)
	}
	if ws := admin.ExposureWarnings("0.0.0.0:8080", "http://dash.example.com"); len(ws) != 2 {
		t.Fatalf("expected 2 exposure warnings for public bind + plaintext http, got %v", ws)
	}

	db := newTestAdminDB(t)
	discordMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/oauth2/token":
			_ = r.ParseForm()
			code := r.Form.Get("code")
			if code == "upstream-leak" {
				http.Error(w, "INTERNAL_SECRET_UPSTREAM_SQL_STATE", http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-" + code, "token_type": "Bearer"})
		case r.Method == http.MethodGet && r.URL.Path == "/users/@me":
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if tok == "tok-super" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": "1001", "username": "superop", "global_name": "Super"})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "9999", "username": "outsider"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer discordMock.Close()

	// Invalid log_level and invalid trusted_proxies fail fast.
	for _, badCfg := range []components.ConfigAdmin{
		{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			LogLevel:  "verbose",
			OAuth:     components.ConfigAdminOAuth{ClientID: "999", ClientSecret: "sec"},
		},
		{
			Enabled:        true,
			PublicURL:      "http://127.0.0.1:8080",
			TrustedProxies: []string{"not-an-ip"},
			OAuth:          components.ConfigAdminOAuth{ClientID: "999", ClientSecret: "sec"},
		},
	} {
		if _, err := admin.NewServer(admin.ServerDeps{
			AdminConfig: badCfg,
			DB:          db,
		}); err == nil {
			t.Fatalf("expected NewServer to fail on invalid admin config %+v", badCfg)
		}
	}

	comp := components.New(context.Background(), components.Config{LocaleDir: "../../locales"}, db)
	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	var hookCalls int
	comp.RegisterGuildLocaleChangeHook(func(_ context.Context, _ *bot.Client, _ snowflake.ID, _ discord.Locale) {
		hookCalls++
	})

	// Seed 6 guilds for N+1 check (#8) and guild update rollback check (#10)
	var guildIDs []snowflake.ID
	for i := range 6 {
		gid := snowflake.ID(700001 + i)
		guildIDs = append(guildIDs, gid)
		if err := db.DB.Create(&models.Guild{ID: gid, Name: fmt.Sprintf("Guild-%d", i), LocaleOverride: discord.LocaleJapanese}).Error; err != nil {
			t.Fatalf("seed guild %d failed: %v", i, err)
		}
		client.Caches.AddGuild(discord.CacheGuild{ID: gid, Name: fmt.Sprintf("Guild-%d", i), PreferredLocale: "en-US"})
	}
	targetGID := guildIDs[0]

	store := gormstore.New(db.DB)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	bbsSvc := bbs.NewService(reg, store, bbs.NewIdentityGenerator([]byte("admin-sec-test-secret-key-32byte")), bbs.NewPolicyManager(bbs.PolicyConfig{}), nil)
	bbsSync := discordsync.NewSyncer(bbsSvc)

	var outBuf bytes.Buffer
	base := slog.NewTextHandler(&outBuf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logHub := admin.NewLogHub(100)
	_ = admin.NewTeeHandler(base, logHub)

	// Server with trusted_proxies including test client loopback 127.0.0.1 and 10.0.0.0/8.
	srv, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:        true,
			PublicURL:      "http://127.0.0.1:8080",
			LogLevel:       "warn",
			TrustedProxies: []string{"10.0.0.0/8"},
			OAuth:          components.ConfigAdminOAuth{ClientSecret: "sec"},
			Users: []components.ConfigAdminUser{
				{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin},
				{UserID: snowflake.ID(1003), Role: admin.RoleViewer},
			},
		},
		DB:                 db,
		Components:         comp,
		Client:             client,
		BBSService:         bbsSvc,
		BBSSyncer:          bbsSync,
		LogHub:             logHub,
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Authenticate super_admin directly via SessionStore? Wait: use OAuth HTTP flow on httptest server!
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	st, rawStart := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/discord/start", "", admin.AuthStartRequest{})
	if st != http.StatusOK {
		t.Fatalf("start failed: %d %s", st, rawStart)
	}
	var startResp admin.AuthStartResponse
	_ = json.Unmarshal(rawStart, &startResp)
	au, _ := url.Parse(startResp.AuthorizeURL)
	cbResp, _ := http.Get(fmt.Sprintf("%s/api/v1/auth/discord/callback?code=super&state=%s", ts.URL, url.QueryEscape(au.Query().Get("state"))))
	_ = cbResp.Body.Close()
	_, rawPoll := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/discord/poll", "", admin.AuthPollRequest{
		LoginID:    startResp.LoginID,
		PollSecret: startResp.PollSecret,
	})
	var pollResp admin.AuthPollResponse
	_ = json.Unmarshal(rawPoll, &pollResp)
	superTok := pollResp.SessionToken

	// Check #7: runtime log level endpoints & RBAC
	st, rawLvl := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/logs/level", superTok, nil)
	if st != http.StatusOK || !strings.Contains(string(rawLvl), `"warn"`) {
		t.Fatalf("expected warn from GET /logs/level, got %d %s", st, rawLvl)
	}
	putLevel := func(lvl string) (int, []byte) {
		rawBody, _ := json.Marshal(admin.UpdateLogLevelRequest{Level: lvl})
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/logs/level", bytes.NewReader(rawBody))
		req.Header.Set("Authorization", "Bearer "+superTok)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("PUT /logs/level failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, b
	}
	if st, b := putLevel("debug"); st != http.StatusOK || !strings.Contains(string(b), `"DEBUG"`) {
		t.Fatalf("PUT /logs/level debug failed: %d %s", st, b)
	}
	if st, _ := putLevel("bogus"); st != http.StatusBadRequest {
		t.Fatalf("expected 400 on bogus log level, got %d", st)
	}

	// Check #8: collectGuildSummaries avoids N+1 queries against guilds table
	var guildTableQueries int
	if err := db.DB.Callback().Query().Before("gorm:query").Register("test_admin_n_plus_1", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "guilds" {
			guildTableQueries++
		}
	}); err != nil {
		t.Fatalf("register callback failed: %v", err)
	}
	st, guildsRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/guilds", superTok, nil)
	_ = db.DB.Callback().Query().Remove("test_admin_n_plus_1")
	if st != http.StatusOK {
		t.Fatalf("GET /guilds failed: %d %s", st, guildsRaw)
	}
	if guildTableQueries != 1 {
		t.Fatalf("expected exactly 1 SELECT on guilds across 6 guilds (no N+1), got %d", guildTableQueries)
	}

	// Check #10: handleUpdateGuild rolls back locale_override and skips hook when SetupGuild fails.
	en := "en-US"
	bbsOn := true
	if err := db.DB.Callback().Create().Before("gorm:create").Register("test_fail_bbs_setup", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "bbs_guild_configs" {
			_ = tx.AddError(fmt.Errorf("injected bbs setup failure"))
		}
	}); err != nil {
		t.Fatalf("register bbs failure callback failed: %v", err)
	}
	st, _ = doJSONReq(t, http.MethodPatch, ts.URL+"/api/v1/guilds/"+targetGID.String(), superTok, admin.UpdateGuildRequest{
		LocaleOverride: &en,
		BBSEnabled:     &bbsOn,
	})
	_ = db.DB.Callback().Create().Remove("test_fail_bbs_setup")
	if st != http.StatusInternalServerError {
		t.Fatalf("expected 500 when SetupGuild fails, got %d", st)
	}
	if got := comp.GuildLocaleOverride(context.Background(), targetGID); got != discord.LocaleJapanese {
		t.Fatalf("expected locale_override rolled back to ja, got %q", got)
	}
	if hookCalls != 0 {
		t.Fatalf("expected 0 locale change hook calls on partial failure, got %d", hookCalls)
	}

	// Check #10 also pre-validates missing bbsSvc BEFORE writing locale_override.
	srvNoBBS, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth:     components.ConfigAdminOAuth{ClientSecret: "sec"},
			Users:     []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
		},
		DB:                 db,
		Components:         comp,
		Client:             client,
		LogHub:             logHub,
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer(no BBS) failed: %v", err)
	}
	reqNoBBS := httptest.NewRequest(http.MethodPatch, "/api/v1/guilds/"+targetGID.String(), strings.NewReader(`{"locale_override":"en-US","bbs_enabled":true}`))
	reqNoBBS.Header.Set("Authorization", "Bearer "+superTok)
	reqNoBBS.Header.Set("Content-Type", "application/json")
	recNoBBS := httptest.NewRecorder()
	srvNoBBS.Handler().ServeHTTP(recNoBBS, reqNoBBS)
	if recNoBBS.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when bbsSvc missing, got %d", recNoBBS.Code)
	}
	if got := comp.GuildLocaleOverride(context.Background(), targetGID); got != discord.LocaleJapanese || hookCalls != 0 {
		t.Fatalf("expected no locale change when pre-validation failed, override=%q hooks=%d", got, hookCalls)
	}

	// Check #12: OAuth callback HTML must NOT leak upstream internal error text.
	_, sLeakRaw := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/auth/discord/start", "", admin.AuthStartRequest{})
	var leakStart admin.AuthStartResponse
	_ = json.Unmarshal(sLeakRaw, &leakStart)
	lu, _ := url.Parse(leakStart.AuthorizeURL)
	cbLeak, err := http.Get(fmt.Sprintf("%s/api/v1/auth/discord/callback?code=upstream-leak&state=%s", ts.URL, url.QueryEscape(lu.Query().Get("state"))))
	if err != nil {
		t.Fatalf("callback leak request failed: %v", err)
	}
	leakBody, _ := io.ReadAll(cbLeak.Body)
	_ = cbLeak.Body.Close()
	if strings.Contains(string(leakBody), "INTERNAL_SECRET_UPSTREAM_SQL_STATE") || strings.Contains(string(leakBody), "status 502") {
		t.Fatalf("callback HTML leaked internal error details: %s", leakBody)
	}
	if !strings.Contains(string(leakBody), "oauth_exchange_failed") {
		t.Fatalf("expected safe fixed error code in callback HTML, got: %s", leakBody)
	}

	// Check #11 & #13: OAuthManager global pending cap (256, no eviction), per-IP pending cap (8),
	// per-IP rate limit (429 + Retry-After), and trusted_proxies X-Forwarded-For resolution.
	sessStore := admin.NewSessionStore(db.DB)
	auditStore := admin.NewAuditStore(db.DB)
	rbac, _ := admin.NewRBAC(components.ConfigAdmin{
		Users: []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
	})
	mgr, err := admin.NewOAuthManager(admin.OAuthConfig{
		ClientID:     "999",
		ClientSecret: "sec",
		RedirectURL:  "http://127.0.0.1:8080/api/v1/auth/discord/callback",
		PendingTTL:   5 * time.Minute,
	}, rbac, sessStore, auditStore)
	if err != nil {
		t.Fatalf("NewOAuthManager failed: %v", err)
	}
	firstStart, err := mgr.Start("first", "198.51.100.1")
	if err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	// Exceed per-IP cap (8 total from 198.51.100.1)
	for i := 1; i < 8; i++ {
		if _, err := mgr.Start("c", "198.51.100.1"); err != nil {
			t.Fatalf("Start #%d for same IP unexpectedly failed: %v", i+1, err)
		}
	}
	if _, err := mgr.Start("c9", "198.51.100.1"); err != admin.ErrTooManyPendingLogins {
		t.Fatalf("expected ErrTooManyPendingLogins on 9th per-IP pending login, got %v", err)
	}
	// Fill remaining global capacity up to 256 across distinct IPs (8 pending per IP -> 31 other IPs * 8 + 7 = 255)
	count := 8
	for ipIdx := 2; count < 256; ipIdx++ {
		ip := fmt.Sprintf("198.51.%d.%d", ipIdx/250+1, ipIdx%250+1)
		for k := 0; k < 8 && count < 256; k++ {
			if _, err := mgr.Start("bulk", ip); err != nil {
				t.Fatalf("Start count=%d ip=%s failed: %v", count, ip, err)
			}
			count++
		}
	}
	if _, err := mgr.Start("overflow", "203.0.113.99"); err != admin.ErrTooManyPendingLogins {
		t.Fatalf("expected ErrTooManyPendingLogins at 256 global limit, got %v", err)
	}
	// Crucial: first entry MUST NOT have been evicted!
	if pr, pErr := mgr.Poll(firstStart.LoginID, firstStart.PollSecret); pErr != nil || pr.Status != "pending" {
		t.Fatalf("expected first login to survive 256 cap without eviction, got resp=%+v err=%v", pr, pErr)
	}

	// Test HTTP trusted_proxies + X-Forwarded-For (127.0.0.1 trusted proxy + XFF spoofing protection + rate limiting)
	srvProxy, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:        true,
			PublicURL:      "http://127.0.0.1:8080",
			TrustedProxies: []string{"127.0.0.1/32", "10.0.0.0/8"},
			OAuth:          components.ConfigAdminOAuth{ClientID: "999000", ClientSecret: "sec"},
			Users:          []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
		},
		DB:                 db,
		Client:             client,
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer(trusted_proxies) failed: %v", err)
	}
	callStart := func(remoteAddr, xff string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/start", strings.NewReader(`{}`))
		req.RemoteAddr = remoteAddr
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srvProxy.Handler().ServeHTTP(rec, req)
		return rec
	}
	// Same untrusted XFF spoof ("malformed, 203.0.113.55, 10.0.0.2") from trusted proxy 127.0.0.1 resolves right-to-left to 203.0.113.55!
	for i := range 8 {
		rec := callStart("127.0.0.1:1234", "malformed-spoof, 203.0.113.55, 10.0.0.2")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 on request %d for client 203.0.113.55, got %d %s", i+1, rec.Code, rec.Body.String())
		}
	}
	// 9th from 203.0.113.55 hits per-IP pending cap -> 429 with Retry-After.
	recCap := callStart("127.0.0.1:1234", "203.0.113.55, 10.0.0.3")
	if recCap.Code != http.StatusTooManyRequests || recCap.Header().Get("Retry-After") == "" {
		t.Fatalf("expected 429 with Retry-After on per-IP pending cap, got %d headers=%v", recCap.Code, recCap.Header())
	}
	// Another IP (203.0.113.77) behind trusted proxy still succeeds!
	recOther := callStart("127.0.0.1:1234", "203.0.113.77")
	if recOther.Code != http.StatusOK {
		t.Fatalf("expected 200 for separate forwarded IP 203.0.113.77, got %d", recOther.Code)
	}
	// Direct untrusted peer (198.51.100.5) trying to spoof XFF is ignored and rate-limited on its peer IP!
	for i := range 8 {
		rec := callStart("198.51.100.5:9000", fmt.Sprintf("198.51.100.%d", 100+i))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for untrusted peer req %d, got %d", i+1, rec.Code)
		}
	}
	recSpoofBlocked := callStart("198.51.100.5:9000", "198.51.100.250")
	if recSpoofBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected untrusted peer XFF spoof to still count against peer IP cap (429), got %d", recSpoofBlocked.Code)
	}

	// Verify HTTP per-IP rate limiters trigger 429 on /start (burst 10) and /poll (burst 60),
	// even from a distinct IP using completed/bad logins, plus viewer 403 on PUT /api/v1/logs/level.
	for i := range 10 {
		rec := callStart("198.51.200.10:5000", "")
		if i < 8 && rec.Code != http.StatusOK {
			t.Fatalf("expected 200 before cap on rate-limit test i=%d, got %d", i, rec.Code)
		}
		if i >= 8 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 on i=%d, got %d", i, rec.Code)
		}
	}
	recRateLimitedStart := callStart("198.51.200.10:5000", "")
	if recRateLimitedStart.Code != http.StatusTooManyRequests || recRateLimitedStart.Header().Get("Retry-After") == "" {
		t.Fatalf("expected rate-limited 429 with Retry-After on 11th /start, got %d", recRateLimitedStart.Code)
	}
	for i := range 60 {
		reqPoll := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/poll", strings.NewReader(`{"login_id":"x","poll_secret":"y"}`))
		reqPoll.RemoteAddr = "198.51.200.11:5000"
		reqPoll.Header.Set("Content-Type", "application/json")
		recPoll := httptest.NewRecorder()
		srvProxy.Handler().ServeHTTP(recPoll, reqPoll)
		if recPoll.Code == http.StatusTooManyRequests {
			t.Fatalf("did not expect 429 within burst 60 at i=%d", i)
		}
	}
	reqPollOver := httptest.NewRequest(http.MethodPost, "/api/v1/auth/discord/poll", strings.NewReader(`{"login_id":"x","poll_secret":"y"}`))
	reqPollOver.RemoteAddr = "198.51.200.11:5000"
	reqPollOver.Header.Set("Content-Type", "application/json")
	recPollOver := httptest.NewRecorder()
	srvProxy.Handler().ServeHTTP(recPollOver, reqPollOver)
	if recPollOver.Code != http.StatusTooManyRequests || recPollOver.Header().Get("Retry-After") == "" {
		t.Fatalf("expected 429 with Retry-After on 61st /poll, got %d", recPollOver.Code)
	}

	// Viewer cannot PUT /api/v1/logs/level (403)
	rawViewerSess, viewerSess, err := sessStore.Create(context.Background(), snowflake.ID(1003), "viewer", "Viewer", admin.RoleViewer, "test", time.Hour)
	if err != nil || viewerSess == nil {
		t.Fatalf("create viewer session failed: %v", err)
	}
	if stGet, _ := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/logs/level", rawViewerSess, nil); stGet != http.StatusOK {
		t.Fatalf("expected viewer GET /logs/level 200, got %d", stGet)
	}
	reqVPut, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/logs/level", strings.NewReader(`{"level":"info"}`))
	reqVPut.Header.Set("Authorization", "Bearer "+rawViewerSess)
	reqVPut.Header.Set("Content-Type", "application/json")
	respVPut, err := http.DefaultClient.Do(reqVPut)
	if err != nil {
		t.Fatalf("viewer PUT /logs/level request failed: %v", err)
	}
	_ = respVPut.Body.Close()
	if respVPut.StatusCode != http.StatusForbidden {
		t.Fatalf("expected viewer PUT /logs/level 403, got %d", respVPut.StatusCode)
	}

	// SSE stream with after_seq delivers >200 buffered unread entries without 200 truncation (#9)
	sseHub := admin.NewLogHub(500)
	srvSSE, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth:     components.ConfigAdminOAuth{ClientID: "999000", ClientSecret: "sec"},
			Users:     []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
		},
		DB:                 db,
		Client:             client,
		LogHub:             sseHub,
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer(SSE) failed: %v", err)
	}
	for i := 1; i <= 260; i++ {
		sseHub.Append(admin.LogEntry{Level: "INFO", Message: fmt.Sprintf("backlog-%03d", i)})
	}
	tsSSE := httptest.NewServer(srvSSE.Handler())
	defer tsSSE.Close()
	sseCtx, sseCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer sseCancel()
	sseReq, _ := http.NewRequestWithContext(sseCtx, http.MethodGet, tsSSE.URL+"/api/v1/logs/stream?after_seq=10", nil)
	sseReq.Header.Set("Authorization", "Bearer "+superTok)
	sseResp, err := http.DefaultClient.Do(sseReq)
	if err != nil {
		t.Fatalf("open backlog SSE failed: %v", err)
	}
	defer func() { _ = sseResp.Body.Close() }()
	sc := bufio.NewScanner(sseResp.Body)
	seenSeq11 := false
	seenSeq260 := false
	backlogCount := 0
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "data: ") {
			backlogCount++
			if strings.Contains(line, `"backlog-011"`) {
				seenSeq11 = true
			}
			if strings.Contains(line, `"backlog-260"`) {
				seenSeq260 = true
				sseCancel()
				break
			}
		}
	}
	if !seenSeq11 || !seenSeq260 || backlogCount != 250 {
		t.Fatalf("expected 250 SSE backlog entries (11..260), got count=%d seen11=%v seen260=%v", backlogCount, seenSeq11, seenSeq260)
	}
}

func TestAdminServer_NameResolution_CacheDBAndRestFallback(t *testing.T) {
	ctx := context.Background()
	db := newTestAdminDB(t)
	discordMock := newMockDiscordOAuthServer(t, "oauth-test-secret")
	defer discordMock.Close()

	var knownCalls, unknownCalls atomic.Int32
	restMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/users/555901"):
			knownCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "555901",
				"username":    "restuser",
				"global_name": "Rest Display",
			})
		case strings.Contains(r.URL.Path, "/users/555902"):
			unknownCalls.Add(1)
			http.Error(w, `{"message":"Unknown User","code":10013}`, http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer restMock.Close()

	guildID := snowflake.ID(888001)
	chID := snowflake.ID(555001)
	posterUID := snowflake.ID(555901)
	unknownUID := snowflake.ID(555902)
	reporterUID := snowflake.ID(444001)

	if err := db.DB.Create(&models.Guild{ID: guildID, Name: "NonCommunity"}).Error; err != nil {
		t.Fatalf("create guild failed: %v", err)
	}
	if err := db.DB.Create(&models.User{ID: reporterUID, Name: "ReporterDB"}).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	comp := components.New(ctx, components.Config{LocaleDir: "../../locales"}, db)
	store := gormstore.New(db.DB)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	bbsSvc := bbs.NewService(reg, store, bbs.NewIdentityGenerator([]byte("admin-name-test-secret-32bytes!")), bbs.NewPolicyManager(bbs.PolicyConfig{Version: "2026-01"}), []snowflake.ID{snowflake.ID(1001)})
	bbsSync := discordsync.NewSyncer(bbsSvc)

	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
		Rest:          rest.New(rest.NewClient("Bot test", rest.WithURL(restMock.URL), rest.WithHTTPClient(restMock.Client()))),
	}
	client.Caches.AddGuild(discord.CacheGuild{
		ID:              guildID,
		Name:            "NonCommunity",
		PreferredLocale: "en-US",
	})
	var modCh discord.GuildTextChannel
	if err := json.Unmarshal([]byte(`{"id":"555001","type":0,"guild_id":"888001","name":"mod-log"}`), &modCh); err != nil {
		t.Fatalf("unmarshal GuildTextChannel failed: %v", err)
	}
	client.Caches.AddChannel(modCh)

	if _, err := bbsSvc.SetupGuild(ctx, bbs.GuildSetupInput{
		GuildID:         guildID,
		Enabled:         true,
		ModLogChannelID: &chID,
		ActorUserID:     snowflake.ID(1001),
	}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	cat, err := bbsSvc.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "news", Name: "ニュース",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := bbsSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID, Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "vip", Name: "VIP板",
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	if _, err := bbsSvc.AcceptPolicy(ctx, posterUID); err != nil {
		t.Fatalf("AcceptPolicy poster failed: %v", err)
	}
	if _, err := bbsSvc.AcceptPolicy(ctx, reporterUID); err != nil {
		t.Fatalf("AcceptPolicy reporter failed: %v", err)
	}
	_, th1, post1, err := bbsSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID, GuildID: guildID, UserID: posterUID, Title: "Test Thread", RawBody: "Body",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	mount := &bbs.BoardMount{
		ID:        uuid.New(),
		BoardID:   board.ID,
		GuildID:   guildID,
		ChannelID: chID,
		Locale:    "ja",
		Active:    true,
		CreatedAt: time.Now(),
	}
	if err := bbsSvc.Store.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	tmID := uuid.New()
	if err := bbsSvc.Store.CreateThreadAndPostMirror(ctx, &bbs.ThreadMirror{
		ID:              tmID,
		ThreadID:        th1.ID,
		BoardMountID:    mount.ID,
		GuildID:         guildID,
		DiscordThreadID: snowflake.ID(666001),
		CreatedAt:       time.Now(),
	}, &bbs.PostMirror{
		ID:               uuid.New(),
		PostID:           post1.ID,
		ThreadMirrorID:   tmID,
		GuildID:          guildID,
		DiscordMessageID: snowflake.ID(666002),
		CreatedAt:        time.Now(),
	}); err != nil {
		t.Fatalf("CreateThreadAndPostMirror failed: %v", err)
	}
	rep, err := bbsSvc.CreateReport(ctx, guildID, reporterUID, post1.ID, "Check REST resolution")
	if err != nil {
		t.Fatalf("CreateReport failed: %v", err)
	}
	if _, err := bbsSvc.CreateAdminCap(ctx, bbs.CapDefinition{
		ID: "unkcap", Name: "UnknownCap", Users: []snowflake.ID{unknownUID},
	}); err != nil {
		t.Fatalf("CreateAdminCap failed: %v", err)
	}

	srv, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth:     components.ConfigAdminOAuth{ClientID: "999000", ClientSecret: "oauth-test-secret"},
			Users:     []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
		},
		DB:                 db,
		Components:         comp,
		Client:             client,
		BBSService:         bbsSvc,
		BBSSyncer:          bbsSync,
		LogHub:             admin.NewLogHub(32),
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	sessStore := admin.NewSessionStore(db.DB)
	tok, _, err := sessStore.Create(ctx, snowflake.ID(1001), "superop", "Super Op", admin.RoleSuperAdmin, "test", time.Hour)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	// 1. Guild detail resolves cached channel name.
	st, gdRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/guilds/"+guildID.String(), tok, nil)
	if st != http.StatusOK {
		t.Fatalf("get guild failed: %d %s", st, gdRaw)
	}
	var gd admin.GuildDetailResponse
	_ = json.Unmarshal(gdRaw, &gd)
	if gd.BBSModLogChannel != chID.String() || gd.BBSModLogChannelName != "mod-log" {
		t.Fatalf("expected BBSModLogChannelName='mod-log', got %+v", gd)
	}

	// 1b. Board mounts resolve cached channel & guild names.
	st, mntRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/bbs/boards/"+board.ID.String()+"/mounts", tok, nil)
	if st != http.StatusOK {
		t.Fatalf("list board mounts failed: %d %s", st, mntRaw)
	}
	var mntResp admin.BBSBoardMountsResponse
	_ = json.Unmarshal(mntRaw, &mntResp)
	if len(mntResp.Mounts) != 1 || mntResp.Mounts[0].GuildName != "NonCommunity" || mntResp.Mounts[0].ChannelName != "mod-log" {
		t.Fatalf("unexpected board mount name resolution: %+v", mntResp.Mounts)
	}

	// 2. ListBBSReports resolves DB reporter name without hitting REST.
	st, repsRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/bbs/reports", tok, nil)
	if st != http.StatusOK {
		t.Fatalf("list reports failed: %d %s", st, repsRaw)
	}
	var repsResp admin.BBSReportsResponse
	_ = json.Unmarshal(repsRaw, &repsResp)
	if len(repsResp.Reports) != 1 || repsResp.Reports[0].ReporterName != "ReporterDB" {
		t.Fatalf("unexpected reports name resolution: %+v", repsResp.Reports)
	}
	if knownCalls.Load() != 0 {
		t.Fatalf("expected 0 REST calls on ListBBSReports, got %d", knownCalls.Load())
	}

	// 3. InspectBBSReport uses REST once and caches positive lookup across subsequent Inspect & list endpoints.
	st, insRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/bbs/reports/"+rep.ID.String()+"/inspect", tok, nil)
	if st != http.StatusOK {
		t.Fatalf("inspect report failed: %d %s", st, insRaw)
	}
	var insResp admin.BBSReportInspectResponse
	_ = json.Unmarshal(insRaw, &insResp)
	if insResp.AuthorUserName != "Rest Display (@restuser)" || insResp.PostGuildID != guildID.String() || insResp.PostGuildName != "NonCommunity" {
		t.Fatalf("unexpected inspect name resolution: %+v", insResp)
	}
	if knownCalls.Load() != 1 {
		t.Fatalf("expected 1 REST call on first inspect, got %d", knownCalls.Load())
	}

	// Repeated inspect hits cache (no second REST call).
	st, _ = doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/bbs/reports/"+rep.ID.String()+"/inspect", tok, nil)
	if st != http.StatusOK || knownCalls.Load() != 1 {
		t.Fatalf("expected cached REST lookup (count=1), got status=%d calls=%d", st, knownCalls.Load())
	}

	// ListBBSBans (allowREST=false) reuses the cached REST name for target_user_name!
	st, banCreateRaw := doJSONReq(t, http.MethodPost, ts.URL+"/api/v1/bbs/bans", tok, admin.CreateBBSBanRequest{
		Scope: "local", GuildID: guildID.String(), TargetUserID: posterUID.String(), Reason: "Cache share check", DurationHours: 1,
	})
	if st != http.StatusCreated {
		t.Fatalf("create ban failed: %d %s", st, banCreateRaw)
	}
	st, bansRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/bbs/bans", tok, nil)
	if st != http.StatusOK {
		t.Fatalf("list bans failed: %d %s", st, bansRaw)
	}
	var bansResp admin.BBSBansResponse
	_ = json.Unmarshal(bansRaw, &bansResp)
	if len(bansResp.Bans) != 1 || bansResp.Bans[0].TargetUserName != "Rest Display (@restuser)" || knownCalls.Load() != 1 {
		t.Fatalf("expected cached TargetUserName without extra REST call, got %+v calls=%d", bansResp.Bans, knownCalls.Load())
	}

	// 4. ListBBSCaps performs REST once for unknown UID (404) and caches negative miss.
	st, capsRaw := doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/bbs/config/caps", tok, nil)
	if st != http.StatusOK || unknownCalls.Load() != 1 {
		t.Fatalf("expected 200 and 1 REST 404 call on ListBBSCaps, got status=%d calls=%d body=%s", st, unknownCalls.Load(), capsRaw)
	}
	var capsResp admin.BBSCapsResponse
	_ = json.Unmarshal(capsRaw, &capsResp)
	if len(capsResp.Caps) != 1 || len(capsResp.Caps[0].UserNames) != 0 {
		t.Fatalf("expected empty UserNames for 404 user, got %+v", capsResp.Caps)
	}
	st, _ = doJSONReq(t, http.MethodGet, ts.URL+"/api/v1/bbs/config/caps", tok, nil)
	if st != http.StatusOK || unknownCalls.Load() != 1 {
		t.Fatalf("expected cached negative lookup (count=1), got status=%d calls=%d", st, unknownCalls.Load())
	}
}
