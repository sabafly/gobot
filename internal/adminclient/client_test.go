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

	srv, err := admin.NewServer(admin.ServerDeps{
		Config: components.Config{LocaleDir: "../../locales"},
		AdminConfig: components.ConfigAdmin{
			Enabled: true,
			OAuth: components.ConfigAdminOAuth{
				ClientSecret: "sec",
				RedirectURL:  "http://127.0.0.1:8080/api/v1/auth/discord/callback",
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
	if err := store.Save("https://bot.ts.net", adminclient.StoredSession{Token: "tok-ts"}); err != nil {
		t.Fatalf("Save second URL failed: %v", err)
	}
	if err := store.Delete("http://127.0.0.1:8080"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok, _ := store.Get("http://127.0.0.1:8080"); ok {
		t.Fatal("expected first URL session deleted")
	}
	if got2, ok2, _ := store.Get("https://bot.ts.net"); !ok2 || got2.Token != "tok-ts" {
		t.Fatalf("expected second URL session preserved, got=%+v ok=%v", got2, ok2)
	}
}
