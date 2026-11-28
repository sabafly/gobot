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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
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
	boardUpd, err := superCli.UpdateBBSBoard(ctx, board.ID, admin.UpdateBBSBoardRequest{Locked: &lockedTrue, SLIPLevel: &slip2})
	if err != nil || !boardUpd.Locked || boardUpd.Rules.SLIPLevel != 2 {
		t.Fatalf("UpdateBBSBoard failed: %+v err=%v", boardUpd, err)
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

	stats, err := viewCli.BBSStats(ctx)
	if err != nil || !stats.Available || stats.Categories != 0 || stats.Boards != 0 || stats.ActiveBans != 0 {
		t.Fatalf("unexpected final BBSStats: %+v err=%v", stats, err)
	}
}
