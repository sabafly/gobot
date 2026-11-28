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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/internal/adminclient"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

func isAPIStatus(err error, code int) bool {
	var apiErr *adminclient.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == code
}

func mustParseUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("invalid uuid %q: %v", raw, err)
	}
	return id
}

func newBBSRoleMockOAuth(t *testing.T, secret string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			_ = r.ParseForm()
			if r.FormValue("client_secret") != secret {
				http.Error(w, "bad secret", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "acc-" + r.FormValue("code"),
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "/users/@me":
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			users := map[string]map[string]any{
				"acc-super": {"id": "1001", "username": "superop", "global_name": "Super Op"},
				"acc-mod":   {"id": "1002", "username": "bbsmod", "global_name": "BBS Mod"},
				"acc-view":  {"id": "1003", "username": "viewop", "global_name": "View Op"},
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

func loginClient(t *testing.T, baseURL, code string) *adminclient.Client {
	t.Helper()
	ctx := context.Background()
	anon := adminclient.New(baseURL, "")
	start, err := anon.StartDiscordLogin(ctx, "bbs-test")
	if err != nil {
		t.Fatalf("StartDiscordLogin failed: %v", err)
	}
	u, err := url.Parse(start.AuthorizeURL)
	if err != nil {
		t.Fatalf("parse authorize url failed: %v", err)
	}
	cbURL := baseURL + "/api/v1/auth/discord/callback?code=" + url.QueryEscape(code) + "&state=" + url.QueryEscape(u.Query().Get("state"))
	resp, err := http.Get(cbURL)
	if err != nil {
		t.Fatalf("callback get failed: %v", err)
	}
	_ = resp.Body.Close()

	poll, err := anon.PollDiscordLogin(ctx, start.LoginID, start.PollSecret)
	if err != nil || poll.Status != "approved" {
		t.Fatalf("PollDiscordLogin failed: status=%s err=%v", poll.Status, err)
	}
	return adminclient.New(baseURL, poll.SessionToken)
}

func TestAdminServer_BBSManagement_EndToEnd_RBAC_And_ConfigReload(t *testing.T) {
	ctx := context.Background()
	db := newTestAdminDB(t)
	discordMock := newBBSRoleMockOAuth(t, "bbs-oauth-secret")
	defer discordMock.Close()

	guildID := snowflake.ID(777001)
	posterUID := snowflake.ID(333001)
	reporterUID := snowflake.ID(444001)
	if err := db.DB.Create(&models.Guild{ID: guildID, Name: "BBSGuild"}).Error; err != nil {
		t.Fatalf("create guild failed: %v", err)
	}
	if err := db.DB.Create(&models.User{ID: posterUID, Name: "PosterUser"}).Error; err != nil {
		t.Fatalf("create poster user failed: %v", err)
	}
	if err := db.DB.Create(&models.User{ID: reporterUID, Name: "ReporterUser"}).Error; err != nil {
		t.Fatalf("create reporter user failed: %v", err)
	}

	comp := components.New(ctx, components.Config{LocaleDir: "../../locales"}, db)
	store := gormstore.New(db.DB)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	bbsSvc := bbs.NewService(
		reg,
		store,
		bbs.NewIdentityGenerator([]byte("admin-bbs-test-secret-32-bytes!!")),
		bbs.NewPolicyManager(bbs.PolicyConfig{Version: "2026-01"}),
		[]snowflake.ID{snowflake.ID(1001)},
	)
	bbsSync := discordsync.NewSyncer(bbsSvc)

	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	client.Caches.AddGuild(discord.CacheGuild{
		ID: guildID, Name: "BBSGuild", PreferredLocale: "ja",
	})

	// Test unavailable behavior when BBSService is nil.
	srvNoBBS, err := admin.NewServer(admin.ServerDeps{
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth:     components.ConfigAdminOAuth{ClientID: "999000", ClientSecret: "bbs-oauth-secret"},
			Users:     []components.ConfigAdminUser{{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin}},
		},
		DB:                 db,
		Components:         comp,
		LogHub:             admin.NewLogHub(16),
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer noBBS failed: %v", err)
	}
	tsNoBBS := httptest.NewServer(srvNoBBS.Handler())
	defer tsNoBBS.Close()
	noBBSCli := loginClient(t, tsNoBBS.URL, "super")
	stNoBBS, err := noBBSCli.BBSStats(ctx)
	if err != nil || stNoBBS.Available {
		t.Fatalf("expected BBSStats Available=false when nil bbsSvc, got %+v err=%v", stNoBBS, err)
	}
	if _, err := noBBSCli.ListBBSCategories(ctx, admin.BBSCategoryFilter{}); !isAPIStatus(err, http.StatusServiceUnavailable) {
		t.Fatalf("expected 503 bbs_unavailable, got %v", err)
	}

	// Main server with BBS enabled + all 3 RBAC roles.
	srv, err := admin.NewServer(admin.ServerDeps{
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth:     components.ConfigAdminOAuth{ClientSecret: "bbs-oauth-secret"},
			Users: []components.ConfigAdminUser{
				{UserID: snowflake.ID(1001), Role: admin.RoleSuperAdmin},
				{UserID: snowflake.ID(1002), Role: admin.RoleBBSModerator},
				{UserID: snowflake.ID(1003), Role: admin.RoleViewer},
			},
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
	t.Cleanup(func() {
		waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.WaitBackgroundSync(waitCtx)
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	superCli := loginClient(t, ts.URL, "super")
	modCli := loginClient(t, ts.URL, "mod")
	viewCli := loginClient(t, ts.URL, "view")

	// Seed a guild thread/post/report directly so moderation endpoints operate on real data.
	if _, err := bbsSvc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: snowflake.ID(1001)}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	if _, err := bbsSvc.AcceptPolicy(ctx, posterUID); err != nil {
		t.Fatalf("AcceptPolicy failed: %v", err)
	}
	if _, err := bbsSvc.AcceptPolicy(ctx, reporterUID); err != nil {
		t.Fatalf("AcceptPolicy reporter failed: %v", err)
	}

	// 1. RBAC guards: viewer & moderator cannot hit PermBBSAdmin endpoints.
	if _, err := modCli.CreateBBSCategory(ctx, admin.CreateBBSCategoryRequest{Scope: "global", Slug: "forbidden", Name: "Forbidden"}); !adminclient.IsForbidden(err) {
		t.Fatalf("expected 403 for moderator creating category, got %v", err)
	}
	if _, err := viewCli.InspectBBSReport(ctx, "00000000-0000-0000-0000-000000000001"); !adminclient.IsForbidden(err) {
		t.Fatalf("expected 403 for viewer inspecting report, got %v", err)
	}

	// 2. SuperAdmin Categories + Boards CRUD + Mounts.
	cat, err := superCli.CreateBBSCategory(ctx, admin.CreateBBSCategoryRequest{
		Scope: "local", GuildID: guildID.String(), Slug: "chat", Name: "雑談", SortOrder: 2,
	})
	if err != nil || cat.Name != "雑談" || cat.GuildName != "BBSGuild" {
		t.Fatalf("CreateBBSCategory failed: %+v err=%v", cat, err)
	}
	updName := "総合雑談"
	catUpd, err := superCli.UpdateBBSCategory(ctx, cat.ID, admin.UpdateBBSCategoryRequest{Name: &updName})
	if err != nil || catUpd.Name != "総合雑談" || catUpd.GuildName != "BBSGuild" {
		t.Fatalf("UpdateBBSCategory failed: %+v err=%v", catUpd, err)
	}

	board, err := superCli.CreateBBSBoard(ctx, admin.CreateBBSBoardRequest{
		CategoryID: cat.ID, Slug: "lounge", Name: "ラウンジ", Description: "まったり",
		Rules: admin.BBSBoardRulesDTO{DefaultName: "通りすがり", HideID: false, SLIPLevel: 1, MaxRes: 500, DatOchiIdleHours: 24},
	})
	if err != nil || board.Scope != "local" || board.Name != "ラウンジ" || board.CategoryName != "総合雑談" || board.GuildName != "BBSGuild" {
		t.Fatalf("CreateBBSBoard failed: %+v err=%v", board, err)
	}
	lockedTrue := true
	slip2 := 2
	updMaxRes := uint32(500)
	updIdleH := -1
	updLowResH := 48
	boardUpd, err := superCli.UpdateBBSBoard(ctx, board.ID, admin.UpdateBBSBoardRequest{
		Locked:             &lockedTrue,
		SLIPLevel:          &slip2,
		MaxRes:             &updMaxRes,
		DatOchiIdleHours:   &updIdleH,
		DatOchiLowResHours: &updLowResH,
	})
	if err != nil || !boardUpd.Locked || boardUpd.Rules.SLIPLevel != 2 ||
		boardUpd.Rules.MaxRes != 500 || boardUpd.Rules.DatOchiIdleHours != -1 || boardUpd.Rules.DatOchiLowResHours != 48 {
		t.Fatalf("UpdateBBSBoard failed: %+v err=%v", boardUpd, err)
	}
	zeroMaxRes := uint32(0)
	if _, err := superCli.UpdateBBSBoard(ctx, board.ID, admin.UpdateBBSBoardRequest{MaxRes: &zeroMaxRes}); !isAPIStatus(err, http.StatusBadRequest) {
		t.Fatalf("expected 400 on max_res=0 update, got %v", err)
	}
	zeroIdleH := 0
	if _, err := superCli.UpdateBBSBoard(ctx, board.ID, admin.UpdateBBSBoardRequest{DatOchiIdleHours: &zeroIdleH}); !isAPIStatus(err, http.StatusBadRequest) {
		t.Fatalf("expected 400 on dat_ochi_idle_hours=0 update, got %v", err)
	}
	lockedFalse := false
	if _, err := superCli.UpdateBBSBoard(ctx, board.ID, admin.UpdateBBSBoardRequest{Locked: &lockedFalse}); err != nil {
		t.Fatalf("unlock board failed: %v", err)
	}
	if mounts, err := viewCli.ListBBSBoardMounts(ctx, board.ID); err != nil || len(mounts.Mounts) != 0 {
		t.Fatalf("ListBBSBoardMounts failed: %+v err=%v", mounts, err)
	}

	// Create thread + post + report in the new board.
	_, th, p1, err := bbsSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: mustParseUUID(t, board.ID), GuildID: guildID, UserID: posterUID, Title: "管理画面テストスレ", RawBody: "テスト投稿",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	rep, err := bbsSvc.CreateReport(ctx, guildID, snowflake.ID(444001), p1.ID, "要確認")
	if err != nil {
		t.Fatalf("CreateReport failed: %v", err)
	}

	// 3. Moderator flows: Threads search/detail, Report Inspect (audited), Ban, Bans redaction for viewer, Abone, Archive, Resolve.
	if thList, err := viewCli.ListBBSThreads(ctx, admin.BBSThreadFilter{BoardID: board.ID, Query: "テストスレ"}); err != nil || len(thList.Threads) != 1 {
		t.Fatalf("ListBBSThreads search failed: %+v err=%v", thList, err)
	}
	if thDetail, err := viewCli.GetBBSThreadDetail(ctx, th.ID.String(), 0, 50); err != nil || len(thDetail.Posts) != 1 || thDetail.Posts[0].Body != "テスト投稿" {
		t.Fatalf("GetBBSThreadDetail failed: %+v err=%v", thDetail, err)
	}

	// Viewer MUST NOT see ReporterUserID or ReporterName on ListBBSReports; Moderator sees both.
	viewReps, err := viewCli.ListBBSReports(ctx, admin.BBSReportFilter{})
	if err != nil || len(viewReps.Reports) != 1 || viewReps.Reports[0].ReporterUserID != "" || viewReps.Reports[0].ReporterName != "" ||
		viewReps.Reports[0].GuildName != "BBSGuild" || viewReps.Reports[0].BoardName != "ラウンジ" {
		t.Fatalf("expected ReporterUserID and ReporterName redacted for viewer, got %+v err=%v", viewReps, err)
	}
	modReps, err := modCli.ListBBSReports(ctx, admin.BBSReportFilter{})
	if err != nil || len(modReps.Reports) != 1 || modReps.Reports[0].ReporterUserID != reporterUID.String() ||
		modReps.Reports[0].ReporterName != "ReporterUser" {
		t.Fatalf("expected ReporterUserID and ReporterName visible for moderator, got %+v err=%v", modReps, err)
	}

	ins, err := modCli.InspectBBSReport(ctx, rep.ID.String())
	if err != nil || ins.AuthorUserID != posterUID.String() || ins.Post == nil ||
		ins.Report.GuildName != "BBSGuild" || ins.Report.BoardName != "ラウンジ" ||
		ins.Report.ThreadTitle != "管理画面テストスレ" || ins.Report.ReporterName != "ReporterUser" ||
		ins.AuthorUserName != "PosterUser" {
		t.Fatalf("InspectBBSReport failed: %+v err=%v", ins, err)
	}
	auditList, err := superCli.ListAuditLogs(ctx, admin.AuditFilter{Action: "bbs.report.inspect"})
	if err != nil || len(auditList.Entries) == 0 {
		t.Fatalf("expected audit log entry for bbs.report.inspect, got %+v err=%v", auditList, err)
	}

	ban, err := modCli.CreateBBSBan(ctx, admin.CreateBBSBanRequest{
		Scope: "local", GuildID: guildID.String(), PostID: p1.ID.String(), Reason: "ルール違反", DurationHours: 12,
	})
	if err != nil || ban.TargetUserID == nil || *ban.TargetUserID != posterUID.String() ||
		ban.TargetUserName != "PosterUser" || ban.GuildName != "BBSGuild" || ban.CreatedByName != "BBS Mod (@bbsmod)" {
		t.Fatalf("CreateBBSBan failed: %+v err=%v", ban, err)
	}

	// Viewer MUST NOT see TargetUserID or TargetUserName on ListBBSBans; Moderator sees both.
	viewBans, err := viewCli.ListBBSBans(ctx, admin.BBSBanFilter{})
	if err != nil || len(viewBans.Bans) != 1 || viewBans.Bans[0].TargetUserID != nil || viewBans.Bans[0].TargetUserName != "" ||
		viewBans.Bans[0].GuildName != "BBSGuild" || viewBans.Bans[0].CreatedByName != "BBS Mod (@bbsmod)" {
		t.Fatalf("expected TargetUserID and TargetUserName redacted for viewer, got %+v err=%v", viewBans, err)
	}
	modBans, err := modCli.ListBBSBans(ctx, admin.BBSBanFilter{})
	if err != nil || len(modBans.Bans) != 1 || modBans.Bans[0].TargetUserID == nil || *modBans.Bans[0].TargetUserID != posterUID.String() ||
		modBans.Bans[0].TargetUserName != "PosterUser" || modBans.Bans[0].CreatedByName != "BBS Mod (@bbsmod)" {
		t.Fatalf("expected TargetUserID and TargetUserName visible for moderator, got %+v err=%v", modBans, err)
	}

	aboned, err := modCli.AboneBBSPost(ctx, p1.ID.String(), "モデレーション削除")
	if err != nil || !aboned.Post.Abone {
		t.Fatalf("AboneBBSPost failed: %+v err=%v", aboned, err)
	}
	archResp, err := modCli.ArchiveBBSThread(ctx, th.ID.String())
	if err != nil || archResp.Thread.Status != "archived" {
		t.Fatalf("ArchiveBBSThread failed: %+v err=%v", archResp, err)
	}
	resolved, err := modCli.ResolveBBSReport(ctx, rep.ID.String(), admin.ResolveBBSReportRequest{
		Status: "resolved", ResolutionNote: "対応完了",
	})
	if err != nil || resolved.Status != "resolved" || resolved.ResolutionNote != "対応完了" || resolved.HandledByName != "BBS Mod (@bbsmod)" {
		t.Fatalf("ResolveBBSReport failed: %+v err=%v", resolved, err)
	}

	if err := modCli.DeleteBBSBan(ctx, ban.ID); err != nil {
		t.Fatalf("DeleteBBSBan failed: %v", err)
	}
	if err := modCli.DeleteBBSBan(ctx, ban.ID); !isAPIStatus(err, http.StatusNotFound) {
		t.Fatalf("expected 404 deleting already-deleted ban, got %v", err)
	}

	// 4. Caps & SLIP Events live CRUD via API + conflict/not-found/validation errors + live memory reflection.
	createdCap, err := superCli.CreateBBSCap(ctx, admin.SaveBBSCapRequest{
		ID: "staff", Name: "スタッフ ★", Description: "公式", Users: []string{posterUID.String()},
	})
	if err != nil || createdCap.UserNames[posterUID.String()] != "PosterUser" {
		t.Fatalf("CreateBBSCap failed or missing user name: %+v err=%v", createdCap, err)
	}
	capsList, err := superCli.ListBBSCaps(ctx)
	if err != nil || len(capsList.Caps) != 1 || capsList.Caps[0].UserNames[posterUID.String()] != "PosterUser" {
		t.Fatalf("ListBBSCaps missing user name: %+v err=%v", capsList, err)
	}
	if _, err := superCli.CreateBBSCap(ctx, admin.SaveBBSCapRequest{
		ID: "STAFF", Name: "重複",
	}); !isAPIStatus(err, http.StatusConflict) {
		t.Fatalf("expected 409 conflict on duplicate cap id, got %v", err)
	}
	if avail := bbsSvc.ListAvailableCaps(ctx, posterUID, mustParseUUID(t, board.ID)); len(avail) != 1 || avail[0].ID != "staff" {
		t.Fatalf("expected created cap immediately active in Service memory, got %+v", avail)
	}
	if _, err := superCli.UpdateBBSCap(ctx, "staff", admin.SaveBBSCapRequest{
		Name: "スタッフ改", Users: []string{posterUID.String()},
	}); err != nil {
		t.Fatalf("UpdateBBSCap failed: %v", err)
	}

	if _, err := superCli.CreateBBSSlipEvent(ctx, admin.SaveBBSSlipEventRequest{
		ID: "oct", Label: "ｱｷﾉﾍﾞﾝ", StartDate: "2026-10-01", EndDate: "2026-10-05", Mode: "override",
	}); err != nil {
		t.Fatalf("CreateBBSSlipEvent failed: %v", err)
	}
	octDay := time.Date(2026, 10, 3, 12, 0, 0, 0, bbs.JSTLocation)
	if got := bbsSvc.Identity.GenerateSlip(bbs.SLIPVVV, mustParseUUID(t, board.ID), bbs.ScopeLocal, guildID, posterUID, bbs.SlipContext{}, octDay); got != "ｱｷﾉﾍﾞﾝ" {
		t.Fatalf("expected created SLIP event live in IdentityGenerator, got %q", got)
	}
	if _, err := superCli.UpdateBBSSlipEvent(ctx, "missing", admin.SaveBBSSlipEventRequest{
		Label: "X", StartDate: "01-01", Mode: "override",
	}); !isAPIStatus(err, http.StatusNotFound) {
		t.Fatalf("expected 404 updating missing SLIP event, got %v", err)
	}

	if err := superCli.DeleteBBSCap(ctx, "staff"); err != nil {
		t.Fatalf("DeleteBBSCap failed: %v", err)
	}
	if err := superCli.DeleteBBSSlipEvent(ctx, "oct"); err != nil {
		t.Fatalf("DeleteBBSSlipEvent failed: %v", err)
	}
	if err := superCli.DeleteBBSBoard(ctx, board.ID); err != nil {
		t.Fatalf("DeleteBBSBoard failed: %v", err)
	}
	if err := superCli.DeleteBBSCategory(ctx, cat.ID); err != nil {
		t.Fatalf("DeleteBBSCategory failed: %v", err)
	}

	// 5. Consistent "error" audit logging across all mutation/inspect handlers (#229).
	missingUUID := uuid.NewString()
	_, _ = superCli.CreateBBSCategory(ctx, admin.CreateBBSCategoryRequest{Scope: "invalid", Slug: "bad", Name: "Bad"})
	_, _ = superCli.UpdateBBSCategory(ctx, missingUUID, admin.UpdateBBSCategoryRequest{Name: &updName})
	_ = superCli.DeleteBBSCategory(ctx, missingUUID)
	_, _ = superCli.CreateBBSBoard(ctx, admin.CreateBBSBoardRequest{CategoryID: missingUUID, Slug: "x", Name: "X"})
	_, _ = superCli.UpdateBBSBoard(ctx, missingUUID, admin.UpdateBBSBoardRequest{Locked: &lockedTrue})
	_ = superCli.DeleteBBSBoard(ctx, missingUUID)
	_, _ = modCli.ArchiveBBSThread(ctx, missingUUID)
	_, _ = modCli.AboneBBSPost(ctx, missingUUID, "x")
	_, _ = modCli.InspectBBSReport(ctx, missingUUID)
	_, _ = modCli.ResolveBBSReport(ctx, missingUUID, admin.ResolveBBSReportRequest{Status: "resolved"})
	_, _ = modCli.CreateBBSBan(ctx, admin.CreateBBSBanRequest{Scope: "local", Reason: "no guild"})
	_, _ = superCli.UpdateBBSCap(ctx, "missing_cap", admin.SaveBBSCapRequest{Name: "X"})
	_ = superCli.DeleteBBSCap(ctx, "missing_cap")
	_, _ = superCli.CreateBBSSlipEvent(ctx, admin.SaveBBSSlipEventRequest{ID: "bad_slip", Label: "x", StartDate: "99-99", Mode: "override"})
	_ = superCli.DeleteBBSSlipEvent(ctx, "missing_slip")

	errAudits, err := superCli.ListAuditLogs(ctx, admin.AuditFilter{Status: "error", Limit: 100})
	if err != nil {
		t.Fatalf("ListAuditLogs error status failed: %v", err)
	}
	seenActions := make(map[string]bool, len(errAudits.Entries))
	for _, entry := range errAudits.Entries {
		seenActions[entry.Action] = true
	}
	for _, wantAction := range []string{
		"bbs.category.create",
		"bbs.category.update",
		"bbs.category.delete",
		"bbs.board.create",
		"bbs.board.update",
		"bbs.board.delete",
		"bbs.thread.archive",
		"bbs.post.abone",
		"bbs.report.inspect",
		"bbs.report.resolve",
		"bbs.ban.create",
		"bbs.ban.delete",
		"bbs.cap.create",
		"bbs.cap.update",
		"bbs.cap.delete",
		"bbs.slip_event.create",
		"bbs.slip_event.update",
		"bbs.slip_event.delete",
	} {
		if !seenActions[wantAction] {
			t.Fatalf("expected error audit log entry for %q, seen=%v", wantAction, seenActions)
		}
	}

	stats, err := viewCli.BBSStats(ctx)
	if err != nil || !stats.Available || stats.Categories != 0 || stats.Boards != 0 || stats.ActiveBans != 0 {
		t.Fatalf("unexpected final BBSStats: %+v err=%v", stats, err)
	}
}

type asyncTestRoundTrip func(*http.Request) (*http.Response, error)

func (f asyncTestRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestAdminServer_BBSAsyncDiscordSync(t *testing.T) {
	ctx := context.Background()
	db := newTestAdminDB(t)
	discordMock := newBBSRoleMockOAuth(t, "bbs-async-secret")
	defer discordMock.Close()

	guildID := snowflake.ID(888001)
	channelID := snowflake.ID(888101)
	ctrlMsgID := snowflake.ID(888201)
	discordThreadID := snowflake.ID(888301)
	discordPostMsgID := snowflake.ID(888401)
	discordCtrlThreadMsgID := snowflake.ID(888501)
	userID := snowflake.ID(1001)

	if err := db.DB.Create(&models.Guild{ID: guildID, Name: "AsyncGuild"}).Error; err != nil {
		t.Fatalf("create guild failed: %v", err)
	}
	if err := db.DB.Create(&models.User{ID: userID, Name: "AsyncUser"}).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	comp := components.New(ctx, components.Config{LocaleDir: "../../locales"}, db)
	store := gormstore.New(db.DB)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	bbsSvc := bbs.NewService(
		reg,
		store,
		bbs.NewIdentityGenerator([]byte("admin-async-test-secret-32-byte!")),
		bbs.NewPolicyManager(bbs.PolicyConfig{Version: "2026-01"}),
		[]snowflake.ID{userID},
	)
	bbsSync := discordsync.NewSyncer(bbsSvc)

	if _, err := bbsSvc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: userID}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	if _, err := bbsSvc.AcceptPolicy(ctx, userID); err != nil {
		t.Fatalf("AcceptPolicy failed: %v", err)
	}
	cat, err := bbsSvc.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "async-cat", Name: "AsyncCat",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := bbsSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID, Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "async-board", Name: "AsyncBoard",
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	mount := &bbs.BoardMount{
		ID: uuid.New(), BoardID: board.ID, GuildID: guildID, ChannelID: channelID,
		ControlMessageID: &ctrlMsgID, Locale: "ja", Active: true,
	}
	if err := store.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	_, th, p1, err := bbsSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID, GuildID: guildID, UserID: userID, Title: "Async Thread", RawBody: "Body",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	if err := store.CreateThreadAndPostMirror(ctx,
		&bbs.ThreadMirror{
			ThreadID: th.ID, BoardMountID: mount.ID, GuildID: guildID,
			DiscordThreadID: discordThreadID, ControlMessageID: &discordCtrlThreadMsgID,
		},
		&bbs.PostMirror{
			PostID: p1.ID, ThreadID: th.ID, ResNumber: 1,
			GuildID: guildID, DiscordMessageID: discordPostMsgID,
		},
	); err != nil {
		t.Fatalf("CreateThreadAndPostMirror failed: %v", err)
	}

	var (
		mu             sync.Mutex
		guideUpdated   bool
		aboneAttempted bool
		archivePatched bool
		ctxActiveCount int
	)
	startedCh := make(chan string, 16)
	releaseCh := make(chan struct{})

	httpClient := &http.Client{
		Transport: asyncTestRoundTrip(func(req *http.Request) (*http.Response, error) {
			p := strings.TrimPrefix(req.URL.Path, "/api/v10")
			select {
			case startedCh <- req.Method + " " + p:
			default:
			}
			<-releaseCh

			mu.Lock()
			if req.Context().Err() == nil {
				ctxActiveCount++
			}
			switch {
			case req.Method == http.MethodPatch && p == "/channels/"+channelID.String()+"/messages/"+ctrlMsgID.String():
				guideUpdated = true
			case req.Method == http.MethodPatch && p == "/channels/"+discordThreadID.String()+"/messages/"+discordPostMsgID.String():
				aboneAttempted = true
				mu.Unlock()
				raw, _ := json.Marshal(map[string]any{"message": "upstream unavailable"})
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewReader(raw)),
				}, nil
			case req.Method == http.MethodPatch && p == "/channels/"+discordThreadID.String():
				archivePatched = true
			}
			mu.Unlock()

			raw, _ := json.Marshal(map[string]any{"id": "1"})
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader(raw)),
			}, nil
		}),
	}
	client := &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Rest:          rest.New(rest.NewClient("Bot async-test-token", rest.WithHTTPClient(httpClient))),
		Caches:        cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	client.Caches.AddGuild(discord.CacheGuild{ID: guildID, Name: "AsyncGuild", PreferredLocale: "ja"})

	srv, err := admin.NewServer(admin.ServerDeps{
		AdminConfig: components.ConfigAdmin{
			Enabled:   true,
			PublicURL: "http://127.0.0.1:8080",
			OAuth:     components.ConfigAdminOAuth{ClientSecret: "bbs-async-secret"},
			Users: []components.ConfigAdminUser{
				{UserID: userID, Role: admin.RoleSuperAdmin},
			},
		},
		DB:                 db,
		Components:         comp,
		Client:             client,
		BBSService:         bbsSvc,
		BBSSyncer:          bbsSync,
		LogHub:             admin.NewLogHub(16),
		OAuthAPIBase:       discordMock.URL,
		OAuthAuthorizeBase: discordMock.URL + "/oauth2/authorize",
		OAuthHTTPClient:    discordMock.Client(),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	t.Cleanup(func() {
		waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.WaitBackgroundSync(waitCtx)
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	superCli := loginClient(t, ts.URL, "super")

	// 1. All three handlers must return immediately while Discord REST is blocked on releaseCh!
	newDesc := "async updated"
	if _, err := superCli.UpdateBBSBoard(ctx, board.ID.String(), admin.UpdateBBSBoardRequest{Description: &newDesc}); err != nil {
		t.Fatalf("UpdateBBSBoard failed while Discord REST blocked: %v", err)
	}
	if _, err := superCli.AboneBBSPost(ctx, p1.ID.String(), "async abone"); err != nil {
		t.Fatalf("AboneBBSPost failed while Discord REST blocked: %v", err)
	}
	if _, err := superCli.ArchiveBBSThread(ctx, th.ID.String()); err != nil {
		t.Fatalf("ArchiveBBSThread failed while Discord REST blocked: %v", err)
	}

	// Wait until the first blocked REST call has started before releasing, proving it runs async post-response.
	select {
	case <-startedCh:
	case <-time.After(3 * time.Second):
		t.Fatal("expected background Discord REST sync to start")
	}
	close(releaseCh)

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := srv.WaitBackgroundSync(waitCtx); err != nil {
		t.Fatalf("WaitBackgroundSync failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !guideUpdated || !aboneAttempted || !archivePatched || ctxActiveCount == 0 {
		t.Fatalf("expected all async sync operations to complete with active bgCtx: guide=%v abone=%v archive=%v activeCtx=%d",
			guideUpdated, aboneAttempted, archivePatched, ctxActiveCount)
	}
}
