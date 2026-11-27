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
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
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
	if err := i18n.LoadLocales("../../locales"); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	db := newTestAdminDB(t)
	ctx := context.Background()

	// Fail closed when client secret / redirect is missing.
	if _, err := admin.NewServer(admin.ServerDeps{
		DB: db,
		AdminConfig: components.ConfigAdmin{
			Enabled: true,
			OAuth:   components.ConfigAdminOAuth{ClientID: "999000", RedirectURL: "http://127.0.0.1:8080/api/v1/auth/discord/callback"},
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
		Guild: discord.Guild{
			ID:              guildID,
			Name:            "NonCommunity",
			PreferredLocale: "en-US",
		},
		MemberCount: 42,
	})

	logHub := admin.NewLogHub(100)
	logHub.Append(admin.LogEntry{Level: "INFO", Message: "boot complete"})

	srv, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled: true,
			OAuth: components.ConfigAdminOAuth{
				ClientSecret: "oauth-test-secret",
				RedirectURL:  "http://127.0.0.1:8080/api/v1/auth/discord/callback",
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
	for _, e := range auditList.Entries {
		actions = append(actions, e.Action)
	}
	for _, want := range []string{"auth.login_denied", "auth.login", "guild.update", "ops.reload_locales"} {
		if !slices.Contains(actions, want) {
			t.Fatalf("expected audit action %q in %v", want, actions)
		}
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
