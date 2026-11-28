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

package bbs_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	bbspanel "github.com/sabafly/gobot/tui/panels/bbs"
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

func superCtx(baseURL string) ui.PanelContext {
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

func viewerCtx(baseURL string) ui.PanelContext {
	return ui.PanelContext{
		Client:  adminclient.New(baseURL, "tok-viewer"),
		BaseURL: baseURL,
		Principal: admin.PrincipalInfo{
			UserID:      "1003",
			Username:    "viewop",
			DisplayName: "Viewer",
			Role:        admin.RoleViewer,
			Permissions: []admin.Permission{admin.PermBBSRead},
		},
	}
}

func TestBBSPanel_AllSubtabs_Forms_IME_RBAC_AndBounds(t *testing.T) {
	now := time.Now().UTC()
	guildStr := "888001"
	uidStr := "7001"

	var (
		mu              sync.Mutex
		resolvedCalls   []admin.ResolveBBSReportRequest
		abonedCalls     []string
		archivedCalls   []string
		createdBoards   []admin.CreateBBSBoardRequest
		updatedBoards   []admin.UpdateBBSBoardRequest
		createdCats     []admin.CreateBBSCategoryRequest
		createdBans     []admin.CreateBBSBanRequest
		deletedBanIDs   []string
		createdCaps     []admin.SaveBBSCapRequest
		createdSlips    []admin.SaveBBSSlipEventRequest
		lastThreadQuery string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/stats":
			_ = json.NewEncoder(w).Encode(admin.BBSStatsResponse{
				Available: true, Categories: 1, Boards: 1, ActiveThreads: 2, OpenReports: 1, ActiveBans: 1,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/reports":
			_ = json.NewEncoder(w).Encode(admin.BBSReportsResponse{
				Reports: []admin.BBSReportDTO{{
					ID: "11111111-1111-1111-1111-111111111111", Scope: "local", GuildID: guildStr, GuildName: "BBS Guild",
					PostID: "22222222-2222-2222-2222-222222222222", ThreadID: "33333333-3333-3333-3333-333333333333",
					ThreadTitle: "日本語スレッドタイトル",
					BoardID:     "44444444-4444-4444-4444-444444444444", BoardName: "VIP板テスト",
					ReporterUserID: "5555", ReporterName: "ReporterHanako",
					Reason: "日本語スパム通報", Status: "open", CreatedAt: now, UpdatedAt: now,
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/reports/11111111-1111-1111-1111-111111111111/inspect":
			_ = json.NewEncoder(w).Encode(admin.BBSReportInspectResponse{
				Report: admin.BBSReportDTO{
					ID: "11111111-1111-1111-1111-111111111111", Scope: "local", GuildID: guildStr, GuildName: "BBS Guild",
					PostID: "22222222-2222-2222-2222-222222222222", Reason: "日本語スパム通報", Status: "open", CreatedAt: now, UpdatedAt: now,
				},
				Post: &admin.BBSPostDTO{
					ID: "22222222-2222-2222-2222-222222222222", ResNumber: 3, AnonID: "AbCdEfGh",
					Body: "対象投稿の本文です", CreatedAt: now,
				},
				AuthorUserID:   "999888777",
				AuthorUserName: "AuthorTaro",
				PostGuildID:    guildStr,
				PostGuildName:  "BBS Guild",
				PosterHash:     "deadbeefcafe",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/reports/11111111-1111-1111-1111-111111111111/resolve":
			var req admin.ResolveBBSReportRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			resolvedCalls = append(resolvedCalls, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSReportDTO{
				ID: "11111111-1111-1111-1111-111111111111", Status: req.Status, ResolutionNote: req.ResolutionNote, UpdatedAt: now,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/categories":
			_ = json.NewEncoder(w).Encode(admin.BBSCategoriesResponse{
				Categories: []admin.BBSCategoryDTO{{
					ID: "55555555-5555-5555-5555-555555555555", Scope: "global", Slug: "news",
					Name: "ニュース", Description: "全般ニュース", SortOrder: 1, CreatedAt: now,
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/categories":
			var req admin.CreateBBSCategoryRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			createdCats = append(createdCats, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSCategoryDTO{ID: "55555555-5555-5555-5555-555555555556", Name: req.Name, Scope: req.Scope, Slug: req.Slug, CreatedAt: now})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/boards":
			_ = json.NewEncoder(w).Encode(admin.BBSBoardsResponse{
				Boards: []admin.BBSBoardDTO{{
					ID: "44444444-4444-4444-4444-444444444444", CategoryID: "55555555-5555-5555-5555-555555555555",
					CategoryName: "ニュース",
					Scope:        "local", GuildID: &guildStr, GuildName: "BBS Guild", Slug: "vip", Name: "VIP板テスト",
					Rules: admin.BBSBoardRulesDTO{DefaultName: "名無し", MaxRes: 1000, DatOchiIdleHours: 1200, DatOchiLowResHours: 240}, CreatedAt: now,
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/boards/44444444-4444-4444-4444-444444444444/mounts":
			_ = json.NewEncoder(w).Encode(admin.BBSBoardMountsResponse{
				Mounts: []admin.BBSBoardMountDTO{{
					ID: "66666666-6666-6666-6666-666666666666", BoardID: "44444444-4444-4444-4444-444444444444",
					GuildID: guildStr, GuildName: "BBS Guild", ChannelID: "777001", ChannelName: "bbs-lounge", Locale: "ja", Active: true, CreatedAt: now,
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/boards":
			var req admin.CreateBBSBoardRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			createdBoards = append(createdBoards, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSBoardDTO{ID: "44444444-4444-4444-4444-444444444445", Name: req.Name, Scope: "global", CreatedAt: now})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/bbs/boards/44444444-4444-4444-4444-444444444444":
			var req admin.UpdateBBSBoardRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			updatedBoards = append(updatedBoards, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSBoardDTO{
				ID: "44444444-4444-4444-4444-444444444444", Name: *req.Name,
				Rules:     admin.BBSBoardRulesDTO{MaxRes: *req.MaxRes, DatOchiIdleHours: *req.DatOchiIdleHours, DatOchiLowResHours: *req.DatOchiLowResHours},
				CreatedAt: now,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/threads":
			mu.Lock()
			lastThreadQuery = r.URL.Query().Get("q")
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSThreadsResponse{
				Threads: []admin.BBSThreadDTO{{
					ID: "33333333-3333-3333-3333-333333333333", BoardID: "44444444-4444-4444-4444-444444444444",
					Title: "日本語スレッドタイトル", Status: "active", ResCount: 2,
					Rules: admin.BBSBoardRulesDTO{MaxRes: 1000}, OwnerAnonID: "OPID1234", BumpedAt: now, CreatedAt: now, UpdatedAt: now,
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/threads/33333333-3333-3333-3333-333333333333":
			_ = json.NewEncoder(w).Encode(admin.BBSThreadDetailResponse{
				Thread: admin.BBSThreadDTO{
					ID: "33333333-3333-3333-3333-333333333333", Title: "日本語スレッドタイトル", Status: "active", ResCount: 2, CreatedAt: now, UpdatedAt: now,
				},
				Posts: []admin.BBSPostDTO{{
					ID: "22222222-2222-2222-2222-222222222222", ThreadID: "33333333-3333-3333-3333-333333333333",
					ResNumber: 1, Name: "名無しさん", AnonID: "AbCdEfGh", Body: "長い日本語の投稿テキストです\n二行目も綺麗に表示されます", IsOP: true, CreatedAt: now,
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/threads/33333333-3333-3333-3333-333333333333/archive":
			mu.Lock()
			archivedCalls = append(archivedCalls, "33333333-3333-3333-3333-333333333333")
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSArchiveThreadResponse{Thread: admin.BBSThreadDTO{ID: "33333333-3333-3333-3333-333333333333", Title: "日本語スレッドタイトル", Status: "archived"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/posts/22222222-2222-2222-2222-222222222222/abone":
			mu.Lock()
			abonedCalls = append(abonedCalls, "22222222-2222-2222-2222-222222222222")
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSAbonePostResponse{Post: admin.BBSPostDTO{ID: "22222222-2222-2222-2222-222222222222", ResNumber: 1, Abone: true}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/bans":
			_ = json.NewEncoder(w).Encode(admin.BBSBansResponse{
				Bans: []admin.BBSBanDTO{{
					ID: "77777777-7777-7777-7777-777777777777", Scope: "local", GuildID: &guildStr, GuildName: "BBS Guild",
					PosterHash: "deadbeef1234", TargetUserID: &uidStr, TargetUserName: "TargetJiro", Reason: "テストBAN",
					CreatedBy: "1001", CreatedByName: "Super Op", CreatedAt: now,
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/bans":
			var req admin.CreateBBSBanRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			createdBans = append(createdBans, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSBanDTO{ID: "77777777-7777-7777-7777-777777777778", Scope: req.Scope, PosterHash: "hash", CreatedBy: "1001", CreatedAt: now})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/bbs/bans/77777777-7777-7777-7777-777777777777":
			mu.Lock()
			deletedBanIDs = append(deletedBanIDs, "77777777-7777-7777-7777-777777777777")
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/config/caps":
			_ = json.NewEncoder(w).Encode(admin.BBSCapsResponse{
				Caps: []admin.BBSCapDTO{{ID: "admin", Name: "管理人 ★", GrantOperators: true, Users: []string{"1001"}, UserNames: map[string]string{"1001": "Super Op"}}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/config/caps":
			var req admin.SaveBBSCapRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			createdCaps = append(createdCaps, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSCapDTO{ID: req.ID, Name: req.Name, GrantOperators: req.GrantOperators, Users: req.Users})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/bbs/config/slip-events":
			_ = json.NewEncoder(w).Encode(admin.BBSSlipEventsResponse{
				Events: []admin.BBSSlipEventDTO{{ID: "sakura", Label: "ｻｸﾗ", StartDate: "04-01", EndDate: "04-05", Mode: "override"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/bbs/config/slip-events":
			var req admin.SaveBBSSlipEventRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			createdSlips = append(createdSlips, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(admin.BBSSlipEventDTO(req))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sCtx := superCtx(srv.URL)
	vCtx := viewerCtx(srv.URL)
	p := ui.Panel(bbspanel.NewPanel(ui.DefaultTheme()))

	// 1. Viewer cannot trigger moderation/admin actions and does not see forbidden shortcuts.
	p, _ = pumpPanelCmd(t, p, vCtx, p.Init(vCtx), 200*time.Millisecond)
	if len(p.ShortHelp(vCtx)) >= len(p.ShortHelp(sCtx)) {
		t.Fatalf("expected viewer ShortHelp to omit moderation/admin keys")
	}
	var toasts []tea.Msg
	p, toasts = pumpPanelCmd(t, p, vCtx, func() tea.Msg {
		var c tea.Cmd
		p, c = p.Update(tea.KeyPressMsg{Code: 'i', Text: "i"}, vCtx)
		return collectFastMsgs(t, c, 50*time.Millisecond)[0]
	}, 100*time.Millisecond)
	_ = toasts
	if p.CapturingInput() {
		t.Fatal("viewer should not open moderation modals")
	}

	// 2. Reports subtab: inspect + resolve via FormModal + IME cursor + bounds check.
	var cmd tea.Cmd
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'i', Text: "i"}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	vReport := p.View(120, 32)
	if !strings.Contains(vReport, "AuthorTaro (999888777)") || !strings.Contains(vReport, "deadbeefcafe") || !strings.Contains(vReport, "ReporterHanako (5555)") {
		t.Fatalf("expected report inspect & name details rendered, got:\n%s", vReport)
	}
	if !strings.Contains(vReport, "Caps") || !strings.Contains(vReport, "SLIP") {
		t.Fatalf("expected 6 flat subtabs including Caps and SLIP in subtab bar, got:\n%s", vReport)
	}

	p, cmd = p.Update(tea.KeyPressMsg{Code: 'v', Text: "v"}, sCtx)
	_ = cmd
	if !p.CapturingInput() {
		t.Fatal("expected CapturingInput() == true when resolve FormModal is active")
	}
	// Tab moves to Resolution Note FieldText -> Cursor() should align with rendered input cell!
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Text: "対処済み"}, sCtx)
	vForm := p.View(120, 32)
	formCur := p.Cursor()
	if formCur == nil {
		t.Fatal("expected non-nil Cursor() inside FormModal FieldText")
	}
	formLines := strings.Split(vForm, "\n")
	if formCur.Y < 0 || formCur.Y >= len(formLines) || !strings.Contains(ansi.Strip(formLines[formCur.Y]), "Resolution Note") {
		t.Fatalf("expected FormModal Cursor() on Resolution Note row, got cur=%+v\n%s", formCur, vForm)
	}
	if rowPlain := ansi.Strip(formLines[formCur.Y]); !strings.Contains(rowPlain, "対処済み") {
		t.Fatalf("expected typed text on cursor row %q", rowPlain)
	}
	// Verify cursor X aligns right after the typed value (accounting for padding inside the box).
	if wAtCur := ansi.StringWidth(ansi.Strip(ansi.Truncate(formLines[formCur.Y], formCur.X, ""))); wAtCur != formCur.X {
		t.Fatalf("expected clean cell column at cur.X=%d, got %d", formCur.X, wAtCur)
	}
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	mu.Lock()
	resCount := len(resolvedCalls)
	mu.Unlock()
	if resCount != 1 || resolvedCalls[0].ResolutionNote != "対処済み" {
		t.Fatalf("expected resolve call with Japanese note, got %+v", resolvedCalls)
	}

	// 3. Switch to Boards subtab (']'), view mounts, drill down to Threads -> IME search -> Posts -> Abone -> Back -> Create Board.
	p, cmd = p.Update(tea.KeyPressMsg{Code: ']', Text: "]"}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	vBoards := p.View(120, 32)
	if !strings.Contains(vBoards, "VIP板テスト") || !strings.Contains(vBoards, "#bbs-lounge (777001)") {
		t.Fatalf("expected Boards list + mount detail with channel name, got:\n%s", vBoards)
	}

	// Enter -> Threads level.
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	// Search '/' with Japanese keyword and verify separated search bar Cursor() on row 3.
	p, cmd = p.Update(tea.KeyPressMsg{Code: '/', Text: "/"}, sCtx)
	_ = cmd
	_ = p.View(120, 32)
	thCur := p.Cursor()
	if !p.CapturingInput() || thCur == nil || thCur.Y != 3 {
		t.Fatalf("expected CapturingInput() and Cursor().Y==3 during thread search, got %+v", thCur)
	}
	p, _ = p.Update(tea.KeyPressMsg{Text: "検索ワード"}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	mu.Lock()
	gotQ := lastThreadQuery
	mu.Unlock()
	if gotQ != "検索ワード" {
		t.Fatalf("expected thread search q='検索ワード', got %q", gotQ)
	}

	// Enter -> Posts level, view Japanese body, abalone & Confirm archive after Esc.
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	vPosts := p.View(120, 32)
	if !strings.Contains(vPosts, "長い日本語の投稿テキストです") {
		t.Fatalf("expected post body in detail viewport, got:\n%s", vPosts)
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: 'a', Text: "a"}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(abonedCalls) != 1 {
		t.Fatalf("expected 1 abone post call, got %v", abonedCalls)
	}

	// Back to Threads -> Archive ConfirmModal ('y' confirms danger modal).
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEsc}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: 'A', Text: "A"}, sCtx)
	if !p.CapturingInput() {
		t.Fatal("expected CapturingInput() during archive ConfirmModal")
	}
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(archivedCalls) != 1 {
		t.Fatalf("expected 1 archive call, got %v", archivedCalls)
	}

	// Back to Boards -> Create Board via FormModal.
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEsc}, sCtx)
	// Edit existing board and verify MaxRes / DatOchiIdleHours / DatOchiLowResHours prefill & PATCH payload.
	p, _ = p.Update(tea.KeyPressMsg{Code: 'e', Text: "e"}, sCtx)
	vEditForm := p.View(120, 32)
	if !strings.Contains(vEditForm, "Max Res") || !strings.Contains(vEditForm, "Idle Ochi") || !strings.Contains(vEditForm, "LowRes Ochi") {
		t.Fatalf("expected board.edit form to show MaxRes and DatOchi fields, got:\n%s", vEditForm)
	}
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(updatedBoards) != 1 || updatedBoards[0].MaxRes == nil || *updatedBoards[0].MaxRes != 1000 ||
		updatedBoards[0].DatOchiIdleHours == nil || *updatedBoards[0].DatOchiIdleHours != 1200 ||
		updatedBoards[0].DatOchiLowResHours == nil || *updatedBoards[0].DatOchiLowResHours != 240 {
		t.Fatalf("expected board.edit PATCH to send MaxRes/DatOchi fields, got %+v", updatedBoards)
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // slug
	p, _ = p.Update(tea.KeyPressMsg{Text: "chat"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // name
	p, _ = p.Update(tea.KeyPressMsg{Text: "新規雑談"}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(createdBoards) != 1 || createdBoards[0].Slug != "chat" || createdBoards[0].CategoryID != "55555555-5555-5555-5555-555555555555" || createdBoards[0].Rules.DatOchiLowResHours != 240 {
		t.Fatalf("expected created board payload with DatOchiLowResHours=240, got %+v", createdBoards)
	}

	// 4. Categories subtab (']'): create category.
	p, cmd = p.Update(tea.KeyPressMsg{Code: ']', Text: "]"}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // guild_id
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // slug
	p, _ = p.Update(tea.KeyPressMsg{Text: "tech"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // name
	p, _ = p.Update(tea.KeyPressMsg{Text: "技術"}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(createdCats) != 1 || createdCats[0].Slug != "tech" {
		t.Fatalf("expected created category, got %+v", createdCats)
	}

	// 5. Bans subtab ('l'): verify names + create + lift ban.
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'l', Text: "l"}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if vBans := p.View(120, 32); !strings.Contains(vBans, "TargetJiro (7001)") || !strings.Contains(vBans, "Super Op (1001)") {
		t.Fatalf("expected Bans view to render target/creator names, got:\n%s", vBans)
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // guild_id
	p, _ = p.Update(tea.KeyPressMsg{Text: guildStr}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // post_id
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx) // target_user_id
	p, _ = p.Update(tea.KeyPressMsg{Text: uidStr}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(createdBans) != 1 || createdBans[0].TargetUserID != uidStr {
		t.Fatalf("expected created ban, got %+v", createdBans)
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: 'D', Text: "D"}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(deletedBanIDs) != 1 {
		t.Fatalf("expected lift ban call, got %v", deletedBanIDs)
	}

	// 6. Caps subtab (']'): verify user name resolution + create Cap; then SLIP subtab ('right'): create SLIP Event.
	p, cmd = p.Update(tea.KeyPressMsg{Code: ']', Text: "]"}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if vCaps := p.View(120, 32); !strings.Contains(vCaps, "Super Op (1001)") {
		t.Fatalf("expected Caps view to render user name, got:\n%s", vCaps)
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Text: "mod"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Text: "モデレータ"}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(createdCaps) != 1 || createdCaps[0].ID != "mod" {
		t.Fatalf("expected created cap mod, got %+v", createdCaps)
	}
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyRight}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Text: "summer"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Text: "ﾅﾂﾏﾂﾘ"}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyTab}, sCtx)
	p, _ = p.Update(tea.KeyPressMsg{Text: "08-01"}, sCtx)
	p, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, sCtx)
	p, _ = pumpPanelCmd(t, p, sCtx, cmd, 200*time.Millisecond)
	if len(createdSlips) != 1 || createdSlips[0].ID != "summer" {
		t.Fatalf("expected created SLIP event summer, got %+v", createdSlips)
	}

	// 7. Strict width & height bounds verification in wide (split) & narrow (stacked) modes.
	for _, dims := range [][2]int{{120, 32}, {72, 22}} {
		w, h := dims[0], dims[1]
		out := p.View(w, h)
		lines := strings.Split(out, "\n")
		if len(lines) > h {
			t.Fatalf("View(%d,%d) exceeded height: got %d lines", w, h, len(lines))
		}
		for li, line := range lines {
			if lw := ansi.StringWidth(line); lw > w {
				t.Fatalf("View(%d,%d) line %d exceeded width: %d > %d", w, h, li, lw, w)
			}
			_ = lipgloss.Width(line)
		}
	}
}

func TestBBSPanel_HiddenReporterDisplay(t *testing.T) {
	now := time.Now().UTC()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/bbs/stats":
			_ = json.NewEncoder(w).Encode(admin.BBSStatsResponse{Available: true, OpenReports: 1})
		case "/api/v1/bbs/reports":
			_ = json.NewEncoder(w).Encode(admin.BBSReportsResponse{
				Reports: []admin.BBSReportDTO{{
					ID:        "11111111-1111-1111-1111-111111111111",
					Scope:     "global",
					GuildID:   "888001",
					GuildName: "BBS Guild",
					PostID:    "22222222-2222-2222-2222-222222222222",
					ThreadID:  "33333333-3333-3333-3333-333333333333",
					BoardID:   "44444444-4444-4444-4444-444444444444",
					Reason:    "隠匿テスト",
					Status:    "open",
					CreatedAt: now,
					UpdatedAt: now,
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	vCtx := viewerCtx(srv.URL)
	p := ui.Panel(bbspanel.NewPanel(ui.DefaultTheme()))
	p, _ = pumpPanelCmd(t, p, vCtx, p.Init(vCtx), 200*time.Millisecond)
	out := ansi.Strip(p.View(120, 32))
	if !strings.Contains(out, "(hidden)") {
		t.Fatalf("expected hidden reporter display '(hidden)', got:\n%s", out)
	}
}
