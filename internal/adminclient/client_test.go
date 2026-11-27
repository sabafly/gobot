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

package adminclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/internal/i18n"
)

func createSQLiteTable(t *testing.T, db *gorm.DB, model any) {
	t.Helper()
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var cols []string
	for _, f := range stmt.Schema.Fields {
		if f.DBName == "" {
			continue
		}
		if f.PrimaryKey {
			cols = append(cols, fmt.Sprintf("`%s` TEXT PRIMARY KEY", f.DBName))
		} else {
			cols = append(cols, fmt.Sprintf("`%s` TEXT", f.DBName))
		}
	}
	if err := db.Exec(fmt.Sprintf("CREATE TABLE `%s` (%s)", stmt.Schema.Table, joinComma(cols))).Error; err != nil {
		t.Fatalf("create table failed: %v", err)
	}
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

func TestClient_EndToEnd_REST_And_SSE(t *testing.T) {
	if err := i18n.LoadLocales("../../locales"); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	createSQLiteTable(t, gdb, &models.User{})
	createSQLiteTable(t, gdb, &models.Guild{})
	if err := gdb.AutoMigrate(&models.AdminSession{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	db := &database.DB{DB: gdb}
	guildID := snowflake.ID(700001)
	gdb.Create(&models.Guild{ID: guildID, Name: "ClientTestGuild"})

	discordMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-1"})
		case "/users/@me":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "1001", "username": "op1", "global_name": "Op One"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer discordMock.Close()

	comp := components.New(context.Background(), components.Config{LocaleDir: "../../locales"}, db)
	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	client.Caches.AddGuild(discord.CacheGuild{
		Guild: discord.Guild{ID: guildID, Name: "ClientTestGuild", PreferredLocale: "en-US"},
	})
	hub := admin.NewLogHub(100)

	t.Setenv("ADMIN_PUBLIC_URL", "")
	t.Setenv("ADMIN_BASE_PATH", "")

	srv, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth: components.ConfigAdminOAuth{
				ClientSecret: "sec",
			},
			Users: []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
		},
		DB:                 db,
		Components:         comp,
		Client:             client,
		LogHub:             hub,
		Version:            "v1-test",
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	c := adminclient.New(ts.URL+"/", "")

	// Unauthenticated Me returns 401 IsUnauthorized.
	if _, err := c.Me(ctx); !adminclient.IsUnauthorized(err) {
		t.Fatalf("expected IsUnauthorized, got %v", err)
	}

	start, err := c.StartDiscordLogin(ctx, "unit-test")
	if err != nil {
		t.Fatalf("StartDiscordLogin failed: %v", err)
	}
	parsedAuth, _ := url.Parse(start.AuthorizeURL)
	cbResp, err := http.Get(fmt.Sprintf("%s/api/v1/auth/discord/callback?code=c1&state=%s", ts.URL, url.QueryEscape(parsedAuth.Query().Get("state"))))
	if err != nil {
		t.Fatalf("callback failed: %v", err)
	}
	_ = cbResp.Body.Close()

	poll, err := c.PollDiscordLogin(ctx, start.LoginID, start.PollSecret)
	if err != nil || poll.Status != "approved" || poll.SessionToken == "" {
		t.Fatalf("PollDiscordLogin failed: poll=%+v err=%v", poll, err)
	}
	c.Token = poll.SessionToken

	me, err := c.Me(ctx)
	if err != nil || me.UserID != "1001" || me.Role != admin.RoleSuperAdmin {
		t.Fatalf("Me failed: me=%+v err=%v", me, err)
	}
	ov, err := c.Overview(ctx)
	if err != nil || ov.Version != "v1-test" {
		t.Fatalf("Overview failed: ov=%+v err=%v", ov, err)
	}

	ja := "ja"
	gd, err := c.UpdateGuild(ctx, guildID.String(), admin.UpdateGuildRequest{LocaleOverride: &ja})
	if err != nil || gd.Guild.LocaleOverride != "ja" {
		t.Fatalf("UpdateGuild failed: gd=%+v err=%v", gd, err)
	}
	listResp, err := c.ListGuilds(ctx, "ClientTest")
	if err != nil || len(listResp.Guilds) != 1 || listResp.Guilds[0].EffectiveLocale != "ja" {
		t.Fatalf("ListGuilds failed: list=%+v err=%v", listResp, err)
	}
	if _, err := c.ReloadLocales(ctx); err != nil {
		t.Fatalf("ReloadLocales failed: %v", err)
	}

	// StreamLogs receives live entries and honors ctx cancellation.
	streamCtx, streamCancel := context.WithTimeout(ctx, 3*time.Second)
	defer streamCancel()
	go func() {
		time.Sleep(50 * time.Millisecond)
		hub.Append(admin.LogEntry{Level: "INFO", Message: "client stream test"})
	}()
	var gotStreamed string
	err = c.StreamLogs(streamCtx, admin.LogFilter{MinLevel: "INFO"}, func(e admin.LogEntry) error {
		if e.Message == "client stream test" {
			gotStreamed = e.Message
			streamCancel()
		}
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("StreamLogs failed unexpectedly: %v", err)
	}
	if gotStreamed != "client stream test" {
		t.Fatalf("expected streamed log entry, got %q", gotStreamed)
	}

	aud, err := c.ListAuditLogs(ctx, admin.AuditFilter{Limit: 20})
	if err != nil || aud.Total < 2 {
		t.Fatalf("ListAuditLogs failed: aud=%+v err=%v", aud, err)
	}
	sessList, err := c.ListSessions(ctx)
	if err != nil || len(sessList.Sessions) != 1 {
		t.Fatalf("ListSessions failed: %+v err=%v", sessList, err)
	}
	if err := c.Logout(ctx); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	if _, err := c.Me(ctx); !adminclient.IsUnauthorized(err) {
		t.Fatalf("expected IsUnauthorized after Logout, got %v", err)
	}
}

func TestFileSessionStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gobot", "tui_session.json")
	store, err := adminclient.NewFileSessionStore(path)
	if err != nil {
		t.Fatalf("NewFileSessionStore failed: %v", err)
	}
	if _, ok, err := store.Get("http://127.0.0.1:8080"); err != nil || ok {
		t.Fatalf("expected missing initially, ok=%v err=%v", ok, err)
	}

	sess := adminclient.StoredSession{
		Token:       "tok-abc",
		UserID:      "1001",
		Username:    "admin",
		DisplayName: "Admin",
		Role:        admin.RoleSuperAdmin,
		ExpiresAt:   time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	}
	if err := store.Save("http://127.0.0.1:8080/", sess); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 file permissions, got %o", info.Mode().Perm())
	}

	got, ok, err := store.Get("http://127.0.0.1:8080")
	if err != nil || !ok || got.Token != "tok-abc" || got.Role != admin.RoleSuperAdmin {
		t.Fatalf("unexpected loaded session: got=%+v ok=%v err=%v", got, ok, err)
	}
	if err := store.Save("https://dash.sabafly.net/gobot/beta/", adminclient.StoredSession{Token: "tok-beta"}); err != nil {
		t.Fatalf("Save beta URL failed: %v", err)
	}
	if err := store.Save("https://dash.sabafly.net/gobot/prod", adminclient.StoredSession{Token: "tok-prod"}); err != nil {
		t.Fatalf("Save prod URL failed: %v", err)
	}
	if err := store.Delete("http://127.0.0.1:8080"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok, _ := store.Get("http://127.0.0.1:8080"); ok {
		t.Fatal("expected first URL session deleted")
	}
	if gotBeta, okBeta, _ := store.Get("https://dash.sabafly.net/gobot/beta"); !okBeta || gotBeta.Token != "tok-beta" {
		t.Fatalf("expected beta subpath session preserved, got=%+v ok=%v", gotBeta, okBeta)
	}
	if gotProd, okProd, _ := store.Get("https://dash.sabafly.net/gobot/prod/"); !okProd || gotProd.Token != "tok-prod" {
		t.Fatalf("expected prod subpath session preserved independently, got=%+v ok=%v", gotProd, okProd)
	}
	if err := store.Delete("https://dash.sabafly.net/gobot/beta"); err != nil {
		t.Fatalf("Delete beta failed: %v", err)
	}
	if _, ok, _ := store.Get("https://dash.sabafly.net/gobot/beta"); ok {
		t.Fatal("expected beta session deleted")
	}
	if gotProd, okProd, _ := store.Get("https://dash.sabafly.net/gobot/prod"); !okProd || gotProd.Token != "tok-prod" {
		t.Fatalf("expected prod session still intact after deleting beta, got=%+v ok=%v", gotProd, okProd)
	}
}

func TestClient_EndToEnd_WithBasePath(t *testing.T) {
	t.Setenv("ADMIN_PUBLIC_URL", "")
	t.Setenv("ADMIN_BASE_PATH", "")
	if err := i18n.LoadLocales("../../locales"); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	createSQLiteTable(t, gdb, &models.User{})
	createSQLiteTable(t, gdb, &models.Guild{})
	if err := gdb.AutoMigrate(&models.AdminSession{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	db := &database.DB{DB: gdb}
	guildID := snowflake.ID(700002)
	gdb.Create(&models.Guild{ID: guildID, Name: "BasePathGuild"})

	discordMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-bp"})
		case "/users/@me":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "1001", "username": "opbp", "global_name": "Op BasePath"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer discordMock.Close()

	comp := components.New(context.Background(), components.Config{LocaleDir: "../../locales"}, db)
	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	client.Caches.AddGuild(discord.CacheGuild{
		Guild: discord.Guild{ID: guildID, Name: "BasePathGuild", PreferredLocale: "en-US"},
	})
	hub := admin.NewLogHub(50)

	srv, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "https://dash.sabafly.net/gobot/beta",
			OAuth: components.ConfigAdminOAuth{
				ClientSecret: "sec",
			},
			Users: []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
		},
		DB:                 db,
		Components:         comp,
		Client:             client,
		LogHub:             hub,
		Version:            "v-bp",
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	if srv.BasePath() != "/gobot/beta" {
		t.Fatalf("expected inherited basePath /gobot/beta, got %q", srv.BasePath())
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	c := adminclient.New(ts.URL+"/gobot/beta/", "")
	start, err := c.StartDiscordLogin(ctx, "bp-tui")
	if err != nil {
		t.Fatalf("StartDiscordLogin failed: %v", err)
	}
	parsedAuth, _ := url.Parse(start.AuthorizeURL)
	cbResp, err := http.Get(fmt.Sprintf("%s/gobot/beta/api/v1/auth/discord/callback?code=c1&state=%s", ts.URL, url.QueryEscape(parsedAuth.Query().Get("state"))))
	if err != nil {
		t.Fatalf("callback failed: %v", err)
	}
	_ = cbResp.Body.Close()

	poll, err := c.PollDiscordLogin(ctx, start.LoginID, start.PollSecret)
	if err != nil || poll.Status != "approved" || poll.SessionToken == "" {
		t.Fatalf("PollDiscordLogin failed: poll=%+v err=%v", poll, err)
	}
	c.Token = poll.SessionToken

	if me, err := c.Me(ctx); err != nil || me.Username != "opbp" {
		t.Fatalf("Me failed: me=%+v err=%v", me, err)
	}
	if ov, err := c.Overview(ctx); err != nil || ov.Version != "v-bp" {
		t.Fatalf("Overview failed: ov=%+v err=%v", ov, err)
	}
	ja := "ja"
	if gd, err := c.UpdateGuild(ctx, guildID.String(), admin.UpdateGuildRequest{LocaleOverride: &ja}); err != nil || gd.Guild.LocaleOverride != "ja" {
		t.Fatalf("UpdateGuild failed: gd=%+v err=%v", gd, err)
	}

	streamCtx, streamCancel := context.WithTimeout(ctx, 3*time.Second)
	defer streamCancel()
	go func() {
		time.Sleep(40 * time.Millisecond)
		hub.Append(admin.LogEntry{Level: "INFO", Message: "subpath sse line"})
	}()
	var gotSSE string
	err = c.StreamLogs(streamCtx, admin.LogFilter{MinLevel: "INFO"}, func(e admin.LogEntry) error {
		if e.Message == "subpath sse line" {
			gotSSE = e.Message
			streamCancel()
		}
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("StreamLogs failed: %v", err)
	}
	if gotSSE != "subpath sse line" {
		t.Fatalf("expected SSE line under base_path, got %q", gotSSE)
	}
	if err := c.Logout(ctx); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
}
