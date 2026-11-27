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

package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

func createSQLiteTableForModel(t *testing.T, db *gorm.DB, model any) {
	t.Helper()
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		t.Fatalf("parse model failed: %v", err)
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
	sqlStr := fmt.Sprintf("CREATE TABLE `%s` (%s%s)", stmt.Schema.Table, strings.Join(cols, ", "), pkClause)
	if err := db.Exec(sqlStr).Error; err != nil {
		t.Fatalf("create table %s failed: %v", stmt.Schema.Table, err)
	}
}

func setupRealAdminEnv(t *testing.T) (serverURL string, hub *admin.LogHub) {
	t.Helper()
	return setupRealAdminEnvWithConfig(t, "http://127.0.0.1", "")
}

func setupRealAdminEnvWithConfig(t *testing.T, publicURL, basePath string) (serverURL string, hub *admin.LogHub) {
	t.Helper()
	t.Setenv("ADMIN_PUBLIC_URL", "")
	t.Setenv("ADMIN_BASE_PATH", "")
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

	createSQLiteTableForModel(t, gdb, &models.Guild{})
	createSQLiteTableForModel(t, gdb, &models.User{})
	if err := gdb.AutoMigrate(&models.AdminSession{}, &models.AdminAuditLog{}); err != nil {
		t.Fatalf("automigrate admin tables failed: %v", err)
	}

	oauthMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/oauth2/token":
			_ = r.ParseForm()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-" + r.Form.Get("code"),
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/users/@me":
			auth := r.Header.Get("Authorization")
			switch auth {
			case "Bearer tok-code-super":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":          "1001",
					"username":    "superop",
					"global_name": "Super Operator",
				})
			case "Bearer tok-code-viewer":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":          "1002",
					"username":    "viewerop",
					"global_name": "Viewer Operator",
				})
			default:
				http.Error(w, "invalid token", http.StatusUnauthorized)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(oauthMock.Close)

	hub = admin.NewLogHub(128)
	hub.Append(admin.LogEntry{
		Time:    time.Now(),
		Level:   "INFO",
		Message: "initial server log line",
	})

	comp := components.New(context.Background(), components.Config{}, &database.DB{DB: gdb})
	adminCfg := components.ConfigAdmin{
		Enabled:         true,
		PublicURL:       publicURL,
		BasePath:        basePath,
		SessionTTLHours: 24,
		LogBufferSize:   128,
		OAuth: components.ConfigAdminOAuth{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
		Users: []components.ConfigAdminUser{
			{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin},
			{UserID: snowflake.ID(1002), Role: admin.RoleViewer},
		},
	}

	srv, err := admin.NewServer(admin.ServerDeps{
		Config:             components.Config{Admin: adminCfg},
		AdminConfig:        adminCfg,
		DB:                 &database.DB{DB: gdb},
		Components:         comp,
		LogHub:             hub,
		Version:            "test-v2",
		StartedAt:          time.Now().Add(-10 * time.Minute),
		OAuthAPIBase:       oauthMock.URL,
		OAuthAuthorizeBase: oauthMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    oauthMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return strings.TrimRight(ts.URL+srv.BasePath(), "/"), hub
}

func collectAppMsgs(cmd tea.Cmd, wait time.Duration) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() {
		ch <- cmd()
	}()
	select {
	case msg := <-ch:
		if msg == nil {
			return nil
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, sub := range batch {
				out = append(out, collectAppMsgs(sub, wait)...)
			}
			return out
		}
		return []tea.Msg{msg}
	case <-time.After(wait):
		return nil
	}
}

func pumpAppCmd(t *testing.T, a *App, cmd tea.Cmd, wait time.Duration) (quitSeen bool) {
	t.Helper()
	queue := collectAppMsgs(cmd, wait)
	for len(queue) > 0 {
		msg := queue[0]
		queue = queue[1:]
		switch msg.(type) {
		case tea.QuitMsg:
			quitSeen = true
			continue
		case spinner.TickMsg, toastClearMsg, loginPollTickMsg:
			continue
		}
		_, next := a.Update(msg)
		queue = append(queue, collectAppMsgs(next, wait)...)
	}
	return quitSeen
}

type stubPanel struct {
	id     string
	title  string
	perm   admin.Permission
	closed bool
}

func (s *stubPanel) ID() string                           { return s.id }
func (s *stubPanel) Title() string                        { return s.title }
func (s *stubPanel) RequiredPermission() admin.Permission { return s.perm }
func (s *stubPanel) Init(ui.PanelContext) tea.Cmd         { return nil }
func (s *stubPanel) Update(_ tea.Msg, _ ui.PanelContext) (ui.Panel, tea.Cmd) {
	return s, nil
}
func (s *stubPanel) View(w, h int) string                    { return s.title }
func (s *stubPanel) Cursor() *tea.Cursor                     { return nil }
func (s *stubPanel) CapturingInput() bool                    { return false }
func (s *stubPanel) ShortHelp(ui.PanelContext) []key.Binding { return nil }
func (s *stubPanel) Close()                                  { s.closed = true }

func TestApp_RealServer_OAuthLogin_RBAC_CursorAndLogout(t *testing.T) {
	baseURL, _ := setupRealAdminEnv(t)
	sessPath := filepath.Join(t.TempDir(), "nested", "tui_session.json")
	store, err := adminclient.NewFileSessionStore(sessPath)
	if err != nil {
		t.Fatalf("NewFileSessionStore failed: %v", err)
	}

	var autoApprove atomic.Bool
	autoApprove.Store(true)

	origOpen := openBrowserFn
	t.Cleanup(func() { openBrowserFn = origOpen })
	openBrowserFn = func(rawURL string) error {
		if !autoApprove.Load() {
			return nil
		}
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		state := u.Query().Get("state")
		cbURL := fmt.Sprintf("%s/api/v1/auth/discord/callback?code=code-super&state=%s", baseURL, url.QueryEscape(state))
		go func() {
			resp, err := http.Get(cbURL)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}

	client := adminclient.New(baseURL, "")
	app := NewApp(AppOptions{
		BaseURL: baseURL,
		Client:  client,
		Store:   store,
	})
	defer app.Close()
	app.login.pollInterval = 20 * time.Millisecond

	_, _ = app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if pumpAppCmd(t, app, app.Init(), 250*time.Millisecond) {
		t.Fatal("unexpected quit during login")
	}

	// Poll explicitly until the async callback HTTP request completes and approves login.
	deadline := time.Now().Add(2 * time.Second)
	for app.mode != modeDashboard && time.Now().Before(deadline) {
		_, pollCmd := app.Update(loginPollTickMsg{Gen: app.login.gen})
		pumpAppCmd(t, app, pollCmd, 200*time.Millisecond)
		if app.mode != modeDashboard {
			time.Sleep(15 * time.Millisecond)
		}
	}
	if app.mode != modeDashboard {
		t.Fatalf("expected modeDashboard after OAuth login, got %v (err=%q)", app.mode, app.login.errMsg)
	}
	if app.principal.Role != admin.RoleSuperAdmin || app.principal.Username != "superop" {
		t.Fatalf("unexpected principal: %+v", app.principal)
	}

	fi, err := os.Stat(sessPath)
	if err != nil {
		t.Fatalf("expected session file created at %s: %v", sessPath, err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("expected session file permission 0600, got %o", fi.Mode().Perm())
	}

	// RBAC sidebar filtering: built-in BBS panel (PermBBSRead) + custom PermBBSAdmin plugin.
	// super_admin sees 5 built-in panels + 1 plugin (6 total); viewer without PermBBSRead sees 4 panels.
	bbsPlugin := &stubPanel{id: "custom_plugin", title: "Plugin", perm: admin.PermBBSAdmin}
	app.RegisterPanel(bbsPlugin)
	if len(app.visiblePanels) != 6 {
		t.Fatalf("expected 6 visible panels for super_admin, got %d", len(app.visiblePanels))
	}

	savedPrincipal := app.principal
	app.principal = admin.PrincipalInfo{
		UserID:      "1002",
		Username:    "viewerop",
		Role:        admin.RoleViewer,
		Permissions: []admin.Permission{admin.PermDashboardRead, admin.PermLogsRead, admin.PermGuildsRead, admin.PermAuditRead},
	}
	app.rebuildVisiblePanels()
	if len(app.visiblePanels) != 4 {
		t.Fatalf("expected BBS and PermBBSAdmin panels hidden for viewer (4 panels), got %d", len(app.visiblePanels))
	}
	app.principal = savedPrincipal
	app.rebuildVisiblePanels()

	// Switch to Logs panel via '2'.
	_, switchCmd := app.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	pumpAppCmd(t, app, switchCmd, 200*time.Millisecond)
	if app.ActivePanel() == nil || app.ActivePanel().ID() != "logs" {
		t.Fatalf("expected active panel 'logs', got %v", app.ActivePanel())
	}

	// Focus filter input, type 'q', verify App does NOT quit while CapturingInput() == true, and cursor translates accurately.
	_, focusCmd := app.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	pumpAppCmd(t, app, focusCmd, 50*time.Millisecond)
	if !app.ActivePanel().CapturingInput() {
		t.Fatal("expected ActivePanel().CapturingInput() == true")
	}
	_, qCmd := app.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if pumpAppCmd(t, app, qCmd, 50*time.Millisecond) {
		t.Fatal("global 'q' quit must be suppressed when panel CapturingInput() is true")
	}

	v := app.View()
	if !v.AltScreen {
		t.Fatal("expected View.AltScreen == true")
	}
	wantX := tuiSidebarWidth + 1 + app.theme.ContentBox.GetBorderLeftSize() + app.theme.ContentBox.GetPaddingLeft() + lipgloss.Width("Filter: ") + 1
	wantY := 1 + app.theme.ContentBox.GetBorderTopSize() + app.theme.ContentBox.GetPaddingTop() + 1
	if v.Cursor == nil || v.Cursor.X != wantX || v.Cursor.Y != wantY {
		t.Fatalf("expected translated cursor (X=%d, Y=%d), got %+v", wantX, wantY, v.Cursor)
	}

	// Help overlay hides cursor, '?' closes help.
	_, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	_, _ = app.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	helpView := app.View()
	if !app.showHelp || helpView.Cursor != nil || !strings.Contains(helpView.Content, "Keyboard Shortcuts") {
		t.Fatalf("expected help overlay without cursor, showHelp=%v cursor=%+v", app.showHelp, helpView.Cursor)
	}
	_, _ = app.Update(tea.KeyPressMsg{Code: '?', Text: "?"})

	// Logout ('L') calls server logout, clears local session, and transitions to modeLogin.
	autoApprove.Store(false)
	_, logoutCmd := app.Update(tea.KeyPressMsg{Code: 'L', Text: "L"})
	pumpAppCmd(t, app, logoutCmd, 250*time.Millisecond)
	if app.mode != modeLogin {
		t.Fatalf("expected modeLogin after logout, got %v", app.mode)
	}
	if _, ok, _ := store.Get(baseURL); ok {
		t.Fatal("expected stored session deleted after logout")
	}
	if !bbsPlugin.closed {
		t.Fatal("expected registered plugin panel Closed on logout")
	}
}

func TestApp_UnauthorizedMsg_ClearsTokenAndReturnsToLogin(t *testing.T) {
	baseURL, _ := setupRealAdminEnv(t)
	sessPath := filepath.Join(t.TempDir(), "tui_session.json")
	store, err := adminclient.NewFileSessionStore(sessPath)
	if err != nil {
		t.Fatalf("NewFileSessionStore failed: %v", err)
	}
	_ = store.Save(baseURL, adminclient.StoredSession{
		BaseURL:   baseURL,
		Token:     "stale-token",
		ExpiresAt: time.Now().Add(time.Hour),
	})

	origOpen := openBrowserFn
	t.Cleanup(func() { openBrowserFn = origOpen })
	openBrowserFn = func(string) error { return nil }

	p := &stubPanel{id: "overview", title: "Overview"}
	app := NewApp(AppOptions{
		BaseURL:       baseURL,
		Client:        adminclient.New(baseURL, "stale-token"),
		Store:         store,
		NoBrowser:     true,
		InitialPanels: []ui.Panel{p},
	})
	defer app.Close()

	app.mode = modeDashboard
	_, cmd := app.Update(ui.UnauthorizedMsg{Err: errors.New("unauthorized")})
	pumpAppCmd(t, app, cmd, 150*time.Millisecond)

	if app.mode != modeLogin {
		t.Fatalf("expected modeLogin after UnauthorizedMsg, got %v", app.mode)
	}
	if !p.closed {
		t.Fatal("expected active panels closed on UnauthorizedMsg")
	}
	if _, ok, _ := store.Get(baseURL); ok {
		t.Fatal("expected stale stored token removed on UnauthorizedMsg")
	}
	if app.client.Token != "" {
		t.Fatalf("expected client token cleared, got %q", app.client.Token)
	}
}

func TestCLI_LoginStatusLogout(t *testing.T) {
	baseURL, _ := setupRealAdminEnv(t)
	sessPath := filepath.Join(t.TempDir(), "tui_session.json")
	store, err := adminclient.NewFileSessionStore(sessPath)
	if err != nil {
		t.Fatalf("NewFileSessionStore failed: %v", err)
	}

	origOpen := openBrowserFn
	t.Cleanup(func() { openBrowserFn = origOpen })
	openBrowserFn = func(rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		state := u.Query().Get("state")
		cbURL := fmt.Sprintf("%s/api/v1/auth/discord/callback?code=code-viewer&state=%s", baseURL, url.QueryEscape(state))
		go func() {
			resp, err := http.Get(cbURL)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}

	client := adminclient.New(baseURL, "")
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runLogin(ctx, &out, client, store, baseURL, false, 15*time.Millisecond); err != nil {
		t.Fatalf("runLogin failed: %v", err)
	}
	if !strings.Contains(out.String(), "@viewerop [viewer]") {
		t.Fatalf("unexpected runLogin output:\n%s", out.String())
	}

	out.Reset()
	if err := runStatus(ctx, &out, client, store, baseURL); err != nil {
		t.Fatalf("runStatus failed: %v", err)
	}
	if !strings.Contains(out.String(), "Authenticated") || !strings.Contains(out.String(), "@viewerop") {
		t.Fatalf("unexpected runStatus output:\n%s", out.String())
	}

	out.Reset()
	if err := runLogout(ctx, &out, client, store, baseURL); err != nil {
		t.Fatalf("runLogout failed: %v", err)
	}
	if !strings.Contains(out.String(), "Logged out") {
		t.Fatalf("unexpected runLogout output:\n%s", out.String())
	}

	out.Reset()
	if err := runStatus(ctx, &out, client, store, baseURL); err != nil {
		t.Fatalf("runStatus after logout failed: %v", err)
	}
	if !strings.Contains(out.String(), "Not logged in") {
		t.Fatalf("expected 'Not logged in' after logout, got:\n%s", out.String())
	}
}

func TestCLI_LoginStatusLogout_WithBasePath(t *testing.T) {
	baseURL, _ := setupRealAdminEnvWithConfig(t, "https://dash.sabafly.net/gobot/beta", "")
	if !strings.HasSuffix(baseURL, "/gobot/beta") {
		t.Fatalf("expected server URL ending with /gobot/beta, got %q", baseURL)
	}

	sessPath := filepath.Join(t.TempDir(), "tui_session.json")
	store, err := adminclient.NewFileSessionStore(sessPath)
	if err != nil {
		t.Fatalf("NewFileSessionStore failed: %v", err)
	}

	origOpen := openBrowserFn
	t.Cleanup(func() { openBrowserFn = origOpen })
	openBrowserFn = func(rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		state := u.Query().Get("state")
		cbURL := fmt.Sprintf("%s/api/v1/auth/discord/callback?code=code-viewer&state=%s", baseURL, url.QueryEscape(state))
		go func() {
			resp, err := http.Get(cbURL)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}

	client := adminclient.New(baseURL, "")
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runLogin(ctx, &out, client, store, baseURL, false, 15*time.Millisecond); err != nil {
		t.Fatalf("runLogin with base_path failed: %v", err)
	}
	if !strings.Contains(out.String(), "@viewerop [viewer]") {
		t.Fatalf("unexpected runLogin output:\n%s", out.String())
	}

	out.Reset()
	if err := runStatus(ctx, &out, client, store, baseURL); err != nil {
		t.Fatalf("runStatus with base_path failed: %v", err)
	}
	if !strings.Contains(out.String(), "Authenticated") || !strings.Contains(out.String(), "/gobot/beta") {
		t.Fatalf("unexpected runStatus output:\n%s", out.String())
	}

	out.Reset()
	if err := runLogout(ctx, &out, client, store, baseURL); err != nil {
		t.Fatalf("runLogout with base_path failed: %v", err)
	}
	if !strings.Contains(out.String(), "Logged out") {
		t.Fatalf("unexpected runLogout output:\n%s", out.String())
	}
}

func TestApp_LogsView_NeverOverflowsTerminalHeight(t *testing.T) {
	baseURL, hub := setupRealAdminEnv(t)
	for i := range 40 {
		hub.Append(admin.LogEntry{
			Level:   "DEBUG",
			Message: fmt.Sprintf("ギルドオーナー情報 #%d\t長い全角ログメッセージ\r\n2行目候補", i),
			Source:  fmt.Sprintf("bot/components/guild.go:%d", 74+i),
			Attrs: map[string]any{
				"own_guilds": "[{ID:111111111111111111 Name:日本語テストギルドアルファ},{ID:222222222222222222 Name:日本語テストギルドベータ}]",
				"guilds":     "[全角文字多数ギルドA, 全角文字多数ギルドB, タブ\t混入ギルドC]",
				"ansi":       "\x1b[31minjected\x1b[0m",
			},
		})
	}

	sessPath := filepath.Join(t.TempDir(), "tui_session.json")
	store, err := adminclient.NewFileSessionStore(sessPath)
	if err != nil {
		t.Fatalf("NewFileSessionStore failed: %v", err)
	}

	origOpen := openBrowserFn
	t.Cleanup(func() { openBrowserFn = origOpen })
	openBrowserFn = func(rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		state := u.Query().Get("state")
		cbURL := fmt.Sprintf("%s/api/v1/auth/discord/callback?code=code-super&state=%s", baseURL, url.QueryEscape(state))
		go func() {
			resp, err := http.Get(cbURL)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}

	app := NewApp(AppOptions{
		BaseURL: baseURL,
		Client:  adminclient.New(baseURL, ""),
		Store:   store,
	})
	defer app.Close()
	app.login.pollInterval = 20 * time.Millisecond

	_, _ = app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pumpAppCmd(t, app, app.Init(), 250*time.Millisecond)
	deadline := time.Now().Add(2 * time.Second)
	for app.mode != modeDashboard && time.Now().Before(deadline) {
		_, pollCmd := app.Update(loginPollTickMsg{Gen: app.login.gen})
		pumpAppCmd(t, app, pollCmd, 200*time.Millisecond)
		if app.mode != modeDashboard {
			time.Sleep(15 * time.Millisecond)
		}
	}
	if app.mode != modeDashboard {
		t.Fatalf("expected modeDashboard, got %v", app.mode)
	}

	// Switch to Logs panel ('2').
	_, switchCmd := app.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	pumpAppCmd(t, app, switchCmd, 250*time.Millisecond)

	sizes := []tea.WindowSizeMsg{
		{Width: 100, Height: 30},
		{Width: 101, Height: 24},
		{Width: 99, Height: 20},
		{Width: 79, Height: 20},
	}
	checkAllSizes := func(label string) {
		t.Helper()
		for _, ws := range sizes {
			_, _ = app.Update(ws)
			v := app.View()
			lines := strings.Split(v.Content, "\n")
			if len(lines) != ws.Height {
				t.Fatalf("%s %dx%d: rendered height=%d, want %d", label, ws.Width, ws.Height, len(lines), ws.Height)
			}
			bottomBorder := lines[len(lines)-2]
			footerLine := lines[len(lines)-1]
			if !strings.Contains(bottomBorder, "╰") || !strings.Contains(bottomBorder, "╯") {
				t.Fatalf("%s %dx%d: expected bottom border corners on row %d, got %q", label, ws.Width, ws.Height, len(lines)-2, bottomBorder)
			}
			if !strings.Contains(footerLine, "l:level") {
				t.Fatalf("%s %dx%d: expected footer shortcut 'l:level' on bottom row, got %q", label, ws.Width, ws.Height, footerLine)
			}
		}
	}

	checkAllSizes("ALL")

	// Switch ALL -> DEBUG ('l') and verify across all terminal dimensions.
	_, lvlCmd := app.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	pumpAppCmd(t, app, lvlCmd, 250*time.Millisecond)
	checkAllSizes("DEBUG")
}
