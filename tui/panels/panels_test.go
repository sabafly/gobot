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

package panels

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/tui/ui"
)

func collectFastMsgs(t *testing.T, cmd tea.Cmd, wait time.Duration) []tea.Msg {
	t.Helper()
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
				out = append(out, collectFastMsgs(t, sub, wait)...)
			}
			return out
		}
		return []tea.Msg{msg}
	case <-time.After(wait):
		return nil
	}
}

func pumpPanelCmd(t *testing.T, p ui.Panel, ctx ui.PanelContext, cmd tea.Cmd, wait time.Duration) (ui.Panel, []tea.Msg) {
	t.Helper()
	var emitted []tea.Msg
	queue := collectFastMsgs(t, cmd, wait)
	for len(queue) > 0 {
		msg := queue[0]
		queue = queue[1:]
		switch msg.(type) {
		case ui.UnauthorizedMsg, ui.ToastMsg, tea.QuitMsg:
			emitted = append(emitted, msg)
			continue
		}
		var next tea.Cmd
		p, next = p.Update(msg, ctx)
		queue = append(queue, collectFastMsgs(t, next, wait)...)
	}
	return p, emitted
}

func testSuperCtx(baseURL string) ui.PanelContext {
	return ui.PanelContext{
		Client:  adminclient.New(baseURL, "tok-super"),
		BaseURL: baseURL,
		Principal: admin.PrincipalInfo{
			UserID:      "1001",
			Username:    "superop",
			DisplayName: "Super Op",
			Role:        admin.RoleSuperAdmin,
			Permissions: admin.AllPermissions(),
		},
	}
}

func testViewerCtx(baseURL string) ui.PanelContext {
	return ui.PanelContext{
		Client:  adminclient.New(baseURL, "tok-viewer"),
		BaseURL: baseURL,
		Principal: admin.PrincipalInfo{
			UserID:      "1002",
			Username:    "viewerop",
			DisplayName: "Viewer Op",
			Role:        admin.RoleViewer,
			Permissions: []admin.Permission{
				admin.PermDashboardRead,
				admin.PermLogsRead,
				admin.PermGuildsRead,
				admin.PermAuditRead,
			},
		},
	}
}

func TestOverviewPanel_LoadRefreshAndUnauthorized(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/overview" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 1 {
			_ = json.NewEncoder(w).Encode(admin.OverviewResponse{
				Version:        "2.0-test",
				UptimeSeconds:  3665,
				GoVersion:      "go1.27",
				Goroutines:     42,
				MemAllocBytes:  16 << 20,
				MemSysBytes:    32 << 20,
				DBConnected:    true,
				DBLatencyMs:    3,
				BotConnected:   true,
				BotUsername:    "sabafly-bot",
				CachedGuilds:   12,
				DBGuilds:       14,
				DBUsers:        250,
				AvgShardPingMs: 18,
				Shards: []admin.ShardStatus{
					{ShardID: 0, Status: "Ready", LatencyMs: 18},
				},
				LogBufferLen:   120,
				LogBufferCap:   2000,
				ActiveSessions: 2,
				BBS: admin.BBSSummary{
					Available:    true,
					Categories:   3,
					Boards:       8,
					ActiveMounts: 10,
					Threads:      25,
					Posts:        340,
					OpenReports:  3,
					ActiveBans:   1,
				},
			})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(admin.ErrorResponse{
			Error: admin.APIError{Code: "unauthorized", Message: "session revoked"},
		})
	}))
	defer srv.Close()

	ctx := testSuperCtx(srv.URL)
	var p ui.Panel = NewOverviewPanel(ui.DefaultTheme())
	defer p.Close()

	cmd := p.Init(ctx)
	p, _ = pumpPanelCmd(t, p, ctx, cmd, 100*time.Millisecond)

	wideView := p.View(100, 30)
	for _, want := range []string{"2.0-test", "sabafly-bot", "ONLINE", "3 open!", "Shard #0"} {
		if !strings.Contains(wideView, want) {
			t.Fatalf("expected %q in wide overview view:\n%s", want, wideView)
		}
	}
	narrowView := p.View(70, 22)
	if !strings.Contains(narrowView, "sabafly-bot") {
		t.Fatalf("expected stacked overview view to render bot name:\n%s", narrowView)
	}

	var refreshCmd tea.Cmd
	p, refreshCmd = p.Update(tea.KeyPressMsg{Code: 'r', Text: "r"}, ctx)
	_, emitted := pumpPanelCmd(t, p, ctx, refreshCmd, 100*time.Millisecond)
	var gotUnauth bool
	for _, msg := range emitted {
		if _, ok := msg.(ui.UnauthorizedMsg); ok {
			gotUnauth = true
		}
	}
	if !gotUnauth {
		t.Fatalf("expected UnauthorizedMsg on 401 refresh, got %#v", emitted)
	}
}

func TestLogsPanel_InitialSSE_JapaneseSearchCursor_PauseAndClose(t *testing.T) {
	var mu sync.Mutex
	var recordedQueries []string
	var streamClosed atomic.Bool
	releaseSecond := make(chan struct{})
	var sseCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/logs":
			mu.Lock()
			recordedQueries = append(recordedQueries, r.URL.RawQuery)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(admin.LogsResponse{
				Entries: []admin.LogEntry{
					{Seq: 1, Time: time.Now(), Level: "INFO", Message: "booted", Attrs: map[string]any{"ok": true}},
					{Seq: 2, Time: time.Now(), Level: "DEBUG", Message: "debug line"},
				},
				NextSeq: 2,
			})
		case "/api/v1/logs/stream":
			n := sseCount.Add(1)
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "no flusher", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			b, _ := json.Marshal(admin.LogEntry{
				Seq:     uint64(2 + n),
				Time:    time.Now(),
				Level:   "WARN",
				Message: fmt.Sprintf("live-msg-%d", n),
			})
			_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", b)
			flusher.Flush()

			if n == 1 {
				select {
				case <-releaseSecond:
					b2, _ := json.Marshal(admin.LogEntry{
						Seq:     99,
						Time:    time.Now(),
						Level:   "ERROR",
						Message: "buffered-while-paused",
					})
					_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", b2)
					flusher.Flush()
				case <-r.Context().Done():
					streamClosed.Store(true)
					return
				}
			}
			<-r.Context().Done()
			streamClosed.Store(true)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := testSuperCtx(srv.URL)
	lp := NewLogsPanel(ui.DefaultTheme())
	var p ui.Panel = lp
	defer p.Close()

	_ = p.View(100, 26)
	initCmd := p.Init(ctx)
	initMsgs := collectFastMsgs(t, initCmd, 250*time.Millisecond)
	if len(initMsgs) != 1 {
		t.Fatalf("expected 1 initial logs msg, got %d", len(initMsgs))
	}
	var waitCmd1 tea.Cmd
	p, waitCmd1 = p.Update(initMsgs[0], ctx)

	liveMsgs1 := collectFastMsgs(t, waitCmd1, 250*time.Millisecond)
	if len(liveMsgs1) != 1 {
		t.Fatalf("expected 1 SSE live log msg, got %d", len(liveMsgs1))
	}
	var waitCmd2 tea.Cmd
	p, waitCmd2 = p.Update(liveMsgs1[0], ctx)

	view := p.View(100, 26)
	if !strings.Contains(view, "booted") || !strings.Contains(view, "live-msg-1") {
		t.Fatalf("expected initial and live SSE log lines in view:\n%s", view)
	}

	// Pause with space, send live entry from server, verify it waits in pausedBuf, then resume.
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, ctx)
	if !lp.paused {
		t.Fatal("expected space to pause live logs")
	}
	close(releaseSecond)
	liveMsgs2 := collectFastMsgs(t, waitCmd2, 300*time.Millisecond)
	if len(liveMsgs2) != 1 {
		t.Fatalf("expected buffered SSE live log msg, got %d", len(liveMsgs2))
	}
	p, _ = p.Update(liveMsgs2[0], ctx)
	pausedView := p.View(100, 26)
	if strings.Contains(pausedView, "buffered-while-paused") || !strings.Contains(pausedView, "PAUSED (+1)") {
		t.Fatalf("expected entry buffered while paused, got view:\n%s", pausedView)
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, ctx)
	if !strings.Contains(p.View(100, 26), "buffered-while-paused") {
		t.Fatal("expected buffered entry flushed on resume")
	}

	// Activate keyword search & test Japanese IME display-width real cursor calculation.
	var cmd tea.Cmd
	p, cmd = p.Update(tea.KeyPressMsg{Code: '/', Text: "/"}, ctx)
	_, _ = pumpPanelCmd(t, p, ctx, cmd, 50*time.Millisecond)
	if !p.CapturingInput() {
		t.Fatal("expected CapturingInput() == true during log search")
	}
	p, _ = p.Update(tea.KeyPressMsg{Text: "日本語テスト"}, ctx)
	_ = p.View(100, 26)
	cur := p.Cursor()
	if cur == nil {
		t.Fatal("expected non-nil hardware cursor during log search")
	}
	wantX := lipgloss.Width(logsFilterPromptText) + lipgloss.Width("日本語テスト")
	if cur.Y != 1 || cur.X != wantX {
		t.Fatalf("expected cursor (X=%d, Y=1) for 12-col Japanese text, got %+v", wantX, *cur)
	}

	// Submit search query.
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, ctx)
	if p.CapturingInput() {
		t.Fatal("expected CapturingInput() == false after Enter")
	}
	p, _ = pumpPanelCmd(t, p, ctx, cmd, 250*time.Millisecond)

	mu.Lock()
	queries := strings.Join(recordedQueries, "|")
	mu.Unlock()
	if !strings.Contains(queries, "q=%E6%97%A5%E6%9C%AC%E8%AA%9E%E3%83%86%E3%82%B9%E3%83%88") {
		t.Fatalf("expected Japanese query sent to /api/v1/logs, got queries=%q", queries)
	}

	p.Close()
	deadline := time.Now().Add(2 * time.Second)
	for !streamClosed.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !streamClosed.Load() {
		t.Fatal("expected SSE HTTP request context to close cleanly after Panel.Close()")
	}
}

func TestGuildsPanel_SearchCursor_ViewerVsAdminWrite(t *testing.T) {
	var mu sync.Mutex
	var patchBodies []admin.UpdateGuildRequest
	curOverride := ""
	curBBSEnabled := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/guilds":
			eff := "en-US"
			if curOverride != "" {
				eff = curOverride
			}
			_ = json.NewEncoder(w).Encode(admin.GuildsResponse{
				AvailableLocales: []string{"en-US", "ja"},
				Guilds: []admin.GuildSummary{
					{
						ID:              "777001",
						Name:            "テスト鯖",
						Cached:          true,
						MemberCount:     42,
						PreferredLocale: "en-US",
						LocaleOverride:  curOverride,
						EffectiveLocale: eff,
						BBSEnabled:      curBBSEnabled,
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/guilds/777001":
			eff := "en-US"
			if curOverride != "" {
				eff = curOverride
			}
			_ = json.NewEncoder(w).Encode(admin.GuildDetailResponse{
				AvailableLocales: []string{"en-US", "ja"},
				Guild: admin.GuildSummary{
					ID:              "777001",
					Name:            "テスト鯖",
					Cached:          true,
					MemberCount:     42,
					PreferredLocale: "en-US",
					LocaleOverride:  curOverride,
					EffectiveLocale: eff,
					BBSEnabled:      curBBSEnabled,
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/guilds/777001":
			var req admin.UpdateGuildRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			patchBodies = append(patchBodies, req)
			if req.LocaleOverride != nil {
				curOverride = *req.LocaleOverride
			}
			if req.BBSEnabled != nil {
				curBBSEnabled = *req.BBSEnabled
			}
			eff := "en-US"
			if curOverride != "" {
				eff = curOverride
			}
			_ = json.NewEncoder(w).Encode(admin.GuildDetailResponse{
				AvailableLocales: []string{"en-US", "ja"},
				Guild: admin.GuildSummary{
					ID:              "777001",
					Name:            "テスト鯖",
					Cached:          true,
					MemberCount:     42,
					PreferredLocale: "en-US",
					LocaleOverride:  curOverride,
					EffectiveLocale: eff,
					BBSEnabled:      curBBSEnabled,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// 1. Viewer cannot write guilds & ShortHelp hides write shortcuts.
	viewerCtx := testViewerCtx(srv.URL)
	gp := NewGuildsPanel(ui.DefaultTheme())
	var p ui.Panel = gp
	_ = p.View(100, 26)
	p, _ = pumpPanelCmd(t, p, viewerCtx, p.Init(viewerCtx), 150*time.Millisecond)

	for _, b := range p.ShortHelp(viewerCtx) {
		k := b.Help().Key
		if k == "l" || k == "b" {
			t.Fatalf("viewer must not see write key %q in ShortHelp", k)
		}
	}
	var cmd tea.Cmd
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'l', Text: "l"}, viewerCtx)
	_, emitted := pumpPanelCmd(t, p, viewerCtx, cmd, 50*time.Millisecond)
	if gp.selectModal.Active || len(emitted) == 0 {
		t.Fatal("expected viewer 'l' press to warn instead of opening locale select modal")
	}

	// 2. Japanese search cursor calculation.
	p, _ = p.Update(tea.KeyPressMsg{Code: '/', Text: "/"}, viewerCtx)
	p, _ = p.Update(tea.KeyPressMsg{Text: "テスト"}, viewerCtx)
	cur := p.Cursor()
	wantX := lipgloss.Width(guildsSearchPrompt) + lipgloss.Width("テスト")
	if cur == nil || cur.Y != 1 || cur.X != wantX {
		t.Fatalf("expected guilds search cursor (X=%d, Y=1), got %+v", wantX, cur)
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEsc}, viewerCtx)

	// 3. Super admin edits locale override via SelectModal and toggles BBS via ConfirmModal.
	superCtx := testSuperCtx(srv.URL)
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'l', Text: "l"}, superCtx)
	if !gp.selectModal.Active || !p.CapturingInput() {
		t.Fatal("expected locale SelectModal to activate for super_admin")
	}
	// Options: [auto, en-US, ja] -> press 'j' twice -> Enter
	p, _ = p.Update(tea.KeyPressMsg{Code: 'j', Text: "j"}, superCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: 'j', Text: "j"}, superCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, superCtx)
	p, _ = pumpPanelCmd(t, p, superCtx, cmd, 150*time.Millisecond)

	p, cmd = p.Update(tea.KeyPressMsg{Code: 'b', Text: "b"}, superCtx)
	if !gp.confirmModal.Active || !p.CapturingInput() {
		t.Fatal("expected BBS toggle ConfirmModal to activate")
	}
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}, superCtx)
	p, _ = pumpPanelCmd(t, p, superCtx, cmd, 150*time.Millisecond)

	mu.Lock()
	patches := len(patchBodies)
	var firstLoc string
	var secondBBS bool
	if patches >= 2 && patchBodies[0].LocaleOverride != nil && patchBodies[1].BBSEnabled != nil {
		firstLoc = *patchBodies[0].LocaleOverride
		secondBBS = *patchBodies[1].BBSEnabled
	}
	mu.Unlock()
	if patches != 2 || firstLoc != "ja" || !secondBBS {
		t.Fatalf("expected PATCH locale='ja' and bbs_enabled=true, got %+v", patchBodies)
	}
}

func TestOpsAuditPanel_Permissions_AuditFilterCursor_And_SessionRevoke(t *testing.T) {
	var mu sync.Mutex
	var reloaded atomic.Bool
	var revokedID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/ops/reload-locales":
			reloaded.Store(true)
			_ = json.NewEncoder(w).Encode(admin.OpsReloadLocalesResponse{Locales: []string{"ja", "en-US"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/audit":
			_ = json.NewEncoder(w).Encode(admin.AuditListResponse{
				Total: 1,
				Limit: 25,
				Entries: []admin.AuditLogEntry{
					{
						ID:           "aud-1",
						ActorUserID:  "1001",
						ActorName:    "superop",
						ActorRole:    admin.RoleSuperAdmin,
						Action:       "guild.update",
						ResourceType: "guild",
						ResourceID:   "777001",
						Summary:      "Updated locale",
						DetailJSON:   `{"locale_override":"ja"}`,
						Status:       "success",
						RemoteAddr:   "127.0.0.1",
						CreatedAt:    time.Now(),
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/auth/sessions":
			_ = json.NewEncoder(w).Encode(admin.SessionsResponse{
				Sessions: []admin.SessionInfo{
					{
						ID:         "sess-cur",
						UserID:     "1001",
						Username:   "superop",
						Role:       admin.RoleSuperAdmin,
						Current:    true,
						LastUsedAt: time.Now(),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/auth/sessions/sess-cur":
			revokedID = "sess-cur"
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// 1. Viewer defaults to Audit Logs tab, sees permission notice on Ops, and has working action filter cursor.
	viewerCtx := testViewerCtx(srv.URL)
	opViewer := NewOpsAuditPanel(ui.DefaultTheme())
	var pv ui.Panel = opViewer
	_ = pv.View(100, 26)
	pv, _ = pumpPanelCmd(t, pv, viewerCtx, pv.Init(viewerCtx), 150*time.Millisecond)
	if opViewer.subTab != subTabAudit {
		t.Fatalf("expected viewer to default to subTabAudit, got %v", opViewer.subTab)
	}
	auditRendered := pv.View(100, 26)
	if !strings.Contains(auditRendered, "guild.update") || !strings.Contains(auditRendered, `"locale_override": "ja"`) {
		t.Fatalf("expected audit list and formatted DetailJSON, got:\n%s", auditRendered)
	}
	pv, _ = pv.Update(tea.KeyPressMsg{Code: '/', Text: "/"}, viewerCtx)
	pv, _ = pv.Update(tea.KeyPressMsg{Text: "ops."}, viewerCtx)
	if cur := pv.Cursor(); cur == nil || cur.Y != 1 || cur.X != lipgloss.Width(auditFilterPrompt)+4 {
		t.Fatalf("expected audit filter cursor at X=%d Y=1, got %+v", lipgloss.Width(auditFilterPrompt)+4, cur)
	}
	pv, _ = pv.Update(tea.KeyPressMsg{Code: tea.KeyEsc}, viewerCtx)
	pv, _ = pv.Update(tea.KeyPressMsg{Code: '[', Text: "["}, viewerCtx)
	if !strings.Contains(pv.View(100, 26), "Permission required: ops.execute") {
		t.Fatal("expected Ops tab to show ops.execute permission notice for viewer")
	}

	// 2. Super admin confirms Reload Locales and revokes own session (triggering UnauthorizedMsg).
	superCtx := testSuperCtx(srv.URL)
	opAdmin := NewOpsAuditPanel(ui.DefaultTheme())
	var pa ui.Panel = opAdmin
	_ = pa.View(100, 26)
	pa, _ = pumpPanelCmd(t, pa, superCtx, pa.Init(superCtx), 150*time.Millisecond)
	if opAdmin.subTab != subTabOps {
		t.Fatalf("expected super_admin to default to subTabOps, got %v", opAdmin.subTab)
	}

	var cmd tea.Cmd
	pa, _ = pa.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, superCtx)
	if !opAdmin.confirmModal.Active {
		t.Fatal("expected confirm modal before executing reload-locales")
	}
	pa, cmd = pa.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}, superCtx)
	pa, _ = pumpPanelCmd(t, pa, superCtx, cmd, 150*time.Millisecond)
	if !reloaded.Load() {
		t.Fatal("expected reload-locales POST endpoint called")
	}

	// Switch to Sessions tab (Ops -> Audit -> Sessions via ']' twice).
	pa, _ = pa.Update(tea.KeyPressMsg{Code: ']', Text: "]"}, superCtx)
	pa, _ = pa.Update(tea.KeyPressMsg{Code: ']', Text: "]"}, superCtx)
	if opAdmin.subTab != subTabSessions {
		t.Fatalf("expected subTabSessions, got %v", opAdmin.subTab)
	}
	pa, _ = pa.Update(tea.KeyPressMsg{Code: 'x', Text: "x"}, superCtx)
	pa, cmd = pa.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}, superCtx)
	_, emitted := pumpPanelCmd(t, pa, superCtx, cmd, 150*time.Millisecond)

	mu.Lock()
	gotRevoked := revokedID
	mu.Unlock()
	if gotRevoked != "sess-cur" {
		t.Fatalf("expected DELETE sess-cur called, got %q", gotRevoked)
	}
	var gotUnauth bool
	for _, m := range emitted {
		if _, ok := m.(ui.UnauthorizedMsg); ok {
			gotUnauth = true
		}
	}
	if !gotUnauth {
		t.Fatalf("expected UnauthorizedMsg after revoking current session, got %#v", emitted)
	}
}

func TestLogsPanel_NoVerticalOverflow_AllAndDebugJapaneseTabsANSI(t *testing.T) {
	allEntries := make([]admin.LogEntry, 0, 30)
	now := time.Now()
	for i := range 30 {
		lvl := "DEBUG"
		msg := "ギルドオーナー情報\tタブ入り確認-長い連結データ-boundary"
		if i%5 == 0 {
			lvl = "INFO"
			msg = "定常処理完了\r\n改行テスト"
		} else if i%7 == 0 {
			lvl = "WARN"
			msg = "警告\rキャリッジリターン\x1b[31mred-attack\x1b[0m"
		}
		allEntries = append(allEntries, admin.LogEntry{
			Seq:     uint64(i + 1),
			Time:    now.Add(time.Duration(i) * time.Millisecond),
			Level:   lvl,
			Message: msg,
			Source:  fmt.Sprintf("bot/components/guild.go:%d", 74+i),
			Attrs: map[string]any{
				"own_guilds": "[{ID:123456789012345678 Name:日本語テストギルド壱番館},{ID:234567890123456789 Name:日本語テストギルド弐番館}]",
				"guilds":     "[日本語サーバAlpha-ベータ運営コミュニティ, テスト用チャンネル管理サーバーその参, 長文属性チェック用ギルド\tタブ含む]",
				"evil_ansi":  "\x1b[31mraw-escape\x1b[0m",
			},
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/logs":
			lvl := strings.ToUpper(r.URL.Query().Get("level"))
			var filtered []admin.LogEntry
			for _, e := range allEntries {
				if lvl == "DEBUG" && e.Level != "DEBUG" && e.Level != "INFO" && e.Level != "WARN" && e.Level != "ERROR" {
					continue
				}
				filtered = append(filtered, e)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(admin.LogsResponse{
				Entries: filtered,
				NextSeq: 30,
			})
		case "/api/v1/logs/stream":
			flusher, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, ": connected\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := testSuperCtx(srv.URL)
	lp := NewLogsPanel(ui.DefaultTheme())
	defer lp.Close()

	var p ui.Panel = lp
	_ = p.View(75, 24)
	p, _ = pumpPanelCmd(t, p, ctx, p.Init(ctx), 150*time.Millisecond)

	assertBounds := func(label string, w, h int) {
		t.Helper()
		rendered := p.View(w, h)
		if gotH := lipgloss.Height(rendered); gotH != h {
			t.Fatalf("%s (%dx%d): rendered height=%d, want %d\n---VIEW---\n%s", label, w, h, gotH, h, rendered)
		}
		for idx, line := range strings.Split(rendered, "\n") {
			if lw := lipgloss.Width(line); lw > w {
				t.Fatalf("%s (%dx%d): line %d width=%d exceeds %d: %q", label, w, h, idx, lw, w, line)
			}
		}
		if strings.Contains(rendered, "\x1b[31m") {
			t.Fatalf("%s (%dx%d): expected raw injected ANSI stripped", label, w, h)
		}
	}

	sizes := []struct{ w, h int }{{73, 24}, {75, 18}, {50, 16}, {100, 30}}
	for _, sz := range sizes {
		assertBounds("ALL", sz.w, sz.h)
	}

	// Switch ALL -> DEBUG ('l') and verify again across odd/narrow/standard sizes.
	var cmd tea.Cmd
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'l', Text: "l"}, ctx)
	p, _ = pumpPanelCmd(t, p, ctx, cmd, 150*time.Millisecond)
	if logLevelCycle[lp.levelIdx] != "DEBUG" {
		t.Fatalf("expected DEBUG level active, got %q", logLevelCycle[lp.levelIdx])
	}
	for _, sz := range sizes {
		assertBounds("DEBUG", sz.w, sz.h)
	}

	// Scrolling up ('k') and jumping bottom ('G') preserve height invariant.
	p, _ = p.Update(tea.KeyPressMsg{Code: 'k', Text: "k"}, ctx)
	assertBounds("DEBUG-scroll-up", 73, 18)
	p, _ = p.Update(tea.KeyPressMsg{Code: 'G', Text: "G"}, ctx)
	assertBounds("DEBUG-scroll-bottom", 73, 18)
}

func TestLogsPanel_NextSeqCatchup_And_CaptureLevelToggle(t *testing.T) {
	var mu sync.Mutex
	curMode := "auto"
	curEffective := "INFO"
	var recordedStreamQueries []string
	var putBodies []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/logs":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(admin.LogsResponse{
				Entries: []admin.LogEntry{
					{Seq: 1, Time: time.Now(), Level: "INFO", Message: "booted"},
					{Seq: 2, Time: time.Now(), Level: "INFO", Message: "ready"},
				},
				NextSeq: 5, // Entries 3..4 filtered out; next live entry will be Seq=5!
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/logs/level":
			mu.Lock()
			resp := admin.LogLevelResponse{Mode: curMode, EffectiveLevel: curEffective}
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/logs/level":
			var req admin.UpdateLogLevelRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			putBodies = append(putBodies, req.Level)
			curMode = req.Level
			curEffective = strings.ToUpper(req.Level)
			if curEffective == "AUTO" {
				curEffective = "INFO"
			}
			resp := admin.LogLevelResponse{Mode: curMode, EffectiveLevel: curEffective}
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/logs/stream":
			mu.Lock()
			recordedStreamQueries = append(recordedStreamQueries, r.URL.RawQuery)
			mu.Unlock()
			flusher, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			// Emit Seq=5 (= initial NextSeq); must NOT be dropped by off-by-one!
			b, _ := json.Marshal(admin.LogEntry{Seq: 5, Time: time.Now(), Level: "WARN", Message: "seq5-not-dropped"})
			_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Viewer without PermOpsExecute sees capture badge but pressing 'L' does not PUT.
	viewerCtx := testViewerCtx(srv.URL)
	lpViewer := NewLogsPanel(ui.DefaultTheme())
	defer lpViewer.Close()
	var pv ui.Panel = lpViewer
	_ = pv.View(110, 20)
	pv, _ = pumpPanelCmd(t, pv, viewerCtx, pv.Init(viewerCtx), 150*time.Millisecond)
	pv, _ = pv.Update(tea.KeyPressMsg{Code: 'C', Text: "C"}, viewerCtx)
	mu.Lock()
	if len(putBodies) != 0 {
		t.Fatalf("expected viewer without ops.execute not to trigger PUT /logs/level, got %v", putBodies)
	}
	mu.Unlock()

	// SuperAdmin receives Seq=5 without off-by-one skip and toggles capture level with 'L'.
	superCtx := testSuperCtx(srv.URL)
	lp := NewLogsPanel(ui.DefaultTheme())
	defer lp.Close()
	var p ui.Panel = lp
	_ = p.View(110, 20)
	initMsgs := collectFastMsgs(t, p.Init(superCtx), 150*time.Millisecond)
	if len(initMsgs) != 1 {
		t.Fatalf("expected 1 initial msg, got %d", len(initMsgs))
	}
	var streamCmd tea.Cmd
	p, streamCmd = p.Update(initMsgs[0], superCtx)
	if lp.lastSeq != 4 {
		t.Fatalf("expected lastSeq=NextSeq-1=4 to preserve Seq=5, got %d", lp.lastSeq)
	}
	liveMsgs := collectFastMsgs(t, streamCmd, 150*time.Millisecond)
	if len(liveMsgs) != 1 {
		t.Fatalf("expected 1 live stream msg, got %d", len(liveMsgs))
	}
	p, _ = p.Update(liveMsgs[0], superCtx)
	view1 := p.View(110, 20)
	if !strings.Contains(view1, "seq5-not-dropped") {
		t.Fatalf("expected Seq=5 entry rendered without off-by-one drop:\n%s", view1)
	}
	if !strings.Contains(view1, "Cap:AUTO(INFO)") {
		t.Fatalf("expected Cap:AUTO(INFO) header badge, got:\n%s", view1)
	}

	var capCmd tea.Cmd
	p, capCmd = p.Update(tea.KeyPressMsg{Code: 'C', Text: "C"}, superCtx)
	p, _ = pumpPanelCmd(t, p, superCtx, capCmd, 150*time.Millisecond)
	mu.Lock()
	gotPuts := append([]string(nil), putBodies...)
	gotStreamQs := append([]string(nil), recordedStreamQueries...)
	mu.Unlock()
	if len(gotPuts) != 1 || gotPuts[0] != "debug" {
		t.Fatalf("expected Shift+C from auto to cycle to debug via PUT, got %v", gotPuts)
	}
	if len(gotStreamQs) == 0 || !strings.Contains(gotStreamQs[0], "after_seq=4") {
		t.Fatalf("expected stream query with after_seq=4, got %v", gotStreamQs)
	}
	view2 := p.View(110, 20)
	if !strings.Contains(view2, "Cap:DEBUG") {
		t.Fatalf("expected Cap:DEBUG header badge after toggle, got:\n%s", view2)
	}
}
