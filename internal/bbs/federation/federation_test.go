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

package federation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/federation"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
	"github.com/sabafly/gobot/internal/i18n"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResp(status int, body any) *http.Response {
	raw, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(raw)),
	}
}

type noopTimer struct{}

func (noopTimer) Stop() bool               { return false }
func (noopTimer) Reset(time.Duration) bool { return false }

func newMockBotClient(rt roundTripFunc) *bot.Client {
	return &bot.Client{
		ApplicationID: snowflake.ID(999111),
		Rest:          rest.New(rest.NewClient("Bot test-token", rest.WithHTTPClient(&http.Client{Transport: rt}))),
		Caches:        cache.New(),
	}
}

func seedUserDelegation(
	t *testing.T,
	originStore *gormstore.Provider,
	originSvc *bbs.Service,
	listenerStore *gormstore.Provider,
	cipher *federation.TokenCipher,
	peerID string,
	remoteKey string,
	userID snowflake.ID,
	accessTTL time.Duration,
	scopes []string,
) {
	t.Helper()
	ctx := context.Background()
	if _, err := originSvc.AcceptPolicy(ctx, userID); err != nil {
		t.Fatalf("origin AcceptPolicy failed: %v", err)
	}
	now := time.Now().UTC()
	rawAccess, _ := federation.GenerateRandomToken(24)
	rawRefresh, _ := federation.GenerateRandomToken(24)

	grant := &bbs.FederationGrant{
		PeerID:          peerID,
		UserID:          userID,
		Scopes:          scopes,
		PolicyVersion:   originSvc.Policy.CurrentVersion,
		AccessTokenHash: federation.HashToken(rawAccess),
		AccessExpiresAt: now.Add(time.Hour),
	}
	rt := &bbs.FederationRefreshToken{
		TokenHash: federation.HashToken(rawRefresh),
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := originStore.CreateFederationGrantWithRefresh(ctx, grant, rt); err != nil {
		t.Fatalf("CreateFederationGrantWithRefresh failed: %v", err)
	}

	accEnc, err := cipher.Encrypt(rawAccess)
	if err != nil {
		t.Fatalf("encrypt access failed: %v", err)
	}
	refEnc, err := cipher.Encrypt(rawRefresh)
	if err != nil {
		t.Fatalf("encrypt refresh failed: %v", err)
	}
	if err := listenerStore.UpsertFederationUserToken(ctx, &bbs.FederationUserToken{
		RemoteKey:        remoteKey,
		UserID:           userID,
		AccessTokenEnc:   accEnc,
		AccessExpiresAt:  now.Add(accessTTL),
		RefreshTokenEnc:  refEnc,
		RefreshExpiresAt: now.Add(24 * time.Hour),
		Scopes:           scopes,
		PolicyVersion:    originSvc.Policy.CurrentVersion,
	}); err != nil {
		t.Fatalf("UpsertFederationUserToken failed: %v", err)
	}
}

func TestFederation_TwoInstanceLifecycle_Sync_Resync_And_FaultIsolation(t *testing.T) {
	if err := i18n.LoadLocales("../../../locales"); err != nil {
		t.Fatalf("load locales failed: %v", err)
	}
	ctx := context.Background()
	capUser := snowflake.ID(11001)
	userB := snowflake.ID(11002)

	_, originStore, originSvc := openTestServiceDB(t, "2026-11-28", []bbs.CapDefinition{
		{ID: "staff", Name: "★スタッフ", Users: []snowflake.ID{capUser}},
	})
	_, listenerStore, listenerSvc := openTestServiceDB(t, "2026-11-28", nil)

	// Native boards on both Origin and Listener.
	origCat, _ := originSvc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeGlobal, Slug: "ocat", Name: "Origin Cat"})
	origBoard, _ := originSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: origCat.ID, Scope: bbs.ScopeGlobal, Slug: "oboard", Name: "本番掲示板",
		Rules: bbs.BoardRules{MaxRes: 100}.Normalize(),
	})

	listCat, _ := listenerSvc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeGlobal, Slug: "lcat", Name: "Listener Cat"})
	listNativeBoard, _ := listenerSvc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: listCat.ID, Scope: bbs.ScopeGlobal, Slug: "lboard", Name: "ローカルグローバル板",
		Rules: bbs.BoardRules{MaxRes: 100}.Normalize(),
	})

	// Setup Origin federation server with token refresh counter wrapper.
	var tokenRefreshCount atomic.Int32
	hub := federation.NewEventHub(2) // deliberately small ring buffer (size 2) to test overflow resync too!
	originSvc.SetEventSink(hub)

	originSrv, err := federation.NewServer(federation.ServerDeps{
		Config: federation.ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  "http://127.0.0.1:18090",
			OAuth:      federation.OAuthConfig{ClientID: "disc", ClientSecret: "sec"},
			Peers: []federation.PeerConfig{
				{ID: "peer-dev", Name: "Dev", Token: "tok-dev", AllowWrite: true, AllowCaps: true, RedirectURIs: []string{"http://127.0.0.1:18091/api/v1/bbs/federation/client/callback"}},
				{ID: "peer-ro", Name: "ReadOnly", Token: "tok-ro", AllowWrite: false},
			},
		},
		Service: originSvc,
		Hub:     hub,
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	var currentOriginHandler atomic.Value
	currentOriginHandler.Store(originSrv.Handler())

	originHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/bbs/federation/oauth/token" {
			tokenRefreshCount.Add(1)
		}
		h := currentOriginHandler.Load().(http.Handler)
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(func() {
		originHTTP.CloseClientConnections()
		originHTTP.Close()
	})

	cipher, err := federation.NewTokenCipher([]byte("two-instance-cipher-key-32bytes!"))
	if err != nil {
		t.Fatalf("NewTokenCipher failed: %v", err)
	}

	// Register writable remote provider and read-only remote provider.
	rp, err := federation.NewRemoteProvider(federation.RemoteProviderDeps{
		Config: federation.RemoteConfig{
			Key:     "prod",
			Label:   "prod",
			BaseURL: originHTTP.URL,
			PeerID:  "peer-dev",
			Token:   "tok-dev",
		},
		Store:           listenerStore,
		Cipher:          cipher,
		CallbackEnabled: true,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider failed: %v", err)
	}
	listenerSvc.Registry.Register(rp)

	rpRO, err := federation.NewRemoteProvider(federation.RemoteProviderDeps{
		Config: federation.RemoteConfig{
			Key:      "prodro",
			Label:    "prod-ro",
			BaseURL:  originHTTP.URL,
			PeerID:   "peer-ro",
			Token:    "tok-ro",
			ReadOnly: true,
		},
		Store:           listenerStore,
		Cipher:          cipher,
		CallbackEnabled: true,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider ro failed: %v", err)
	}

	// 1. Coexistence of native and remote global boards.
	scopeG := bbs.ScopeGlobal
	allGlobal, err := listenerSvc.ListBoards(ctx, bbs.BoardFilter{Scope: &scopeG})
	if err != nil {
		t.Fatalf("ListBoards failed: %v", err)
	}
	if len(allGlobal) != 2 {
		t.Fatalf("expected 2 global boards (native + remote), got %d: %+v", len(allGlobal), allGlobal)
	}
	if allGlobal[0].ID != listNativeBoard.ID || allGlobal[0].ProviderKey != bbs.DefaultProviderKey {
		t.Fatalf("expected default/native board first, got %+v", allGlobal[0])
	}
	if allGlobal[1].ID != origBoard.ID || allGlobal[1].ProviderKey != "remote:prod" {
		t.Fatalf("expected remote:prod board second, got %+v", allGlobal[1])
	}

	// 2. Client read_only fails closed on write & report.
	if _, _, _, err := rpRO.CreateRemoteThread(ctx, bbs.CreateThreadInput{
		BoardID: origBoard.ID, GuildID: 1, UserID: capUser, Title: "RO", RawBody: "Body",
	}); !errors.Is(err, bbs.ErrFederationReadOnly) {
		t.Fatalf("expected ErrFederationReadOnly, got %v", err)
	}
	if _, err := rpRO.CreateRemoteReport(ctx, 1, capUser, uuid.New(), "spam"); !errors.Is(err, bbs.ErrFederationReadOnly) {
		t.Fatalf("expected ErrFederationReadOnly on report, got %v", err)
	}

	// 3. Setup Listener guild, mount remote board, verify Listener CheckConsent gate + Origin delegation.
	originGuild := snowflake.ID(60001)
	_, _ = originSvc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: originGuild, Enabled: true, ActorUserID: 1})
	_, _ = originSvc.AcceptPolicy(ctx, userB)
	listenerGuild := snowflake.ID(70001)
	listenerCh := snowflake.ID(70002)
	_, _ = listenerSvc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: listenerGuild, Enabled: true, ActorUserID: 1})

	// Seed Origin delegation for capUser with expired-soon AccessExpiresAt (30s < 2m skew) BEFORE granting local consent.
	seedUserDelegation(t, originStore, originSvc, listenerStore, cipher, "peer-dev", "prod", capUser, 30*time.Second,
		[]string{bbs.FederationScopeWrite, bbs.FederationScopeReport, bbs.FederationScopeCaps})

	// Missing local Listener consent must return ErrConsentRequired even when Origin delegation exists!
	if _, _, _, err := listenerSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: origBoard.ID, GuildID: listenerGuild, UserID: capUser, Title: "NoLocalConsent", RawBody: "Body",
	}); !errors.Is(err, bbs.ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired when Listener consent missing, got %v", err)
	}
	_, _ = listenerSvc.AcceptPolicy(ctx, capUser)
	_, _ = listenerSvc.AcceptPolicy(ctx, userB)

	// 4. Create initial remote thread via Listener Service (triggers transparent auto-refresh!).
	_, remoteTh1, remoteP1, err := listenerSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: origBoard.ID,
		GuildID: listenerGuild,
		UserID:  capUser,
		Title:   "連携初期スレ",
		CapID:   "staff",
		RawBody: "1レス目本文",
	})
	if err != nil {
		t.Fatalf("CreateThread on remote board failed: %v", err)
	}
	if tokenRefreshCount.Load() != 1 || remoteP1.Cap == "" {
		t.Fatalf("expected 1 auto-refresh and staff cap applied, refreshes=%d cap=%q", tokenRefreshCount.Load(), remoteP1.Cap)
	}

	// 5. Force access token near-expiry again & run 8 concurrent capped writes -> ONLY 1 refresh rotation must happen!
	tokNow, _ := listenerStore.GetFederationUserToken(ctx, "prod", capUser)
	tokNow.AccessExpiresAt = time.Now().UTC().Add(30 * time.Second)
	_ = listenerStore.UpsertFederationUserToken(ctx, tokNow)
	beforeConcurrent := tokenRefreshCount.Load()

	var concWG sync.WaitGroup
	concErrs := make(chan error, 8)
	for i := range 8 {
		concWG.Add(1)
		go func(idx int) {
			defer concWG.Done()
			_, _, _, _, cErr := listenerSvc.CreatePost(ctx, bbs.CreatePostInput{
				ThreadID: remoteTh1.ID,
				GuildID:  listenerGuild,
				UserID:   capUser,
				CapID:    "staff",
				RawBody:  fmt.Sprintf("並列書き込み %d", idx),
			})
			if cErr != nil {
				concErrs <- cErr
			}
		}(i)
	}
	concWG.Wait()
	close(concErrs)
	for cErr := range concErrs {
		t.Fatalf("concurrent CreatePost failed: %v", cErr)
	}
	if delta := tokenRefreshCount.Load() - beforeConcurrent; delta != 1 {
		t.Fatalf("expected per-user mutex to serialize concurrent refresh to 1 call, got %d", delta)
	}

	// Remote report succeeds via delegated user grant + report scope.
	rep, err := listenerSvc.CreateReport(ctx, listenerGuild, capUser, remoteP1.ID, "通報テスト")
	if err != nil || rep == nil {
		t.Fatalf("CreateReport on remote post failed: %v", err)
	}

	// 6. Start SSE Subscriber with mock Discord client and test LIVE updates + no duplicate mirrors.
	ctrlMsgID := snowflake.ID(70009)
	mount := &bbs.BoardMount{
		ID:               uuid.New(),
		BoardID:          origBoard.ID,
		ProviderKey:      "remote:prod",
		GuildID:          listenerGuild,
		ChannelID:        listenerCh,
		ControlMessageID: &ctrlMsgID,
		Locale:           "ja",
		IsForum:          false,
		Active:           true,
	}
	if err := listenerStore.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}

	var discordMu sync.Mutex
	var nextDiscID int64 = 900000
	var createdThreads int
	var postedBodies []string
	var editedBodies []string
	var archiveNoticeCount int
	var guideUpdatedCount int

	livePostCh := make(chan string, 16)
	archiveCh := make(chan struct{}, 4)
	guideCh := make(chan struct{}, 8)

	mockClient := newMockBotClient(func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		raw, _ := io.ReadAll(req.Body)
		bodyStr := string(raw)

		discordMu.Lock()
		defer discordMu.Unlock()
		nextDiscID++
		idStr := fmt.Sprintf("%d", nextDiscID)

		switch {
		case req.Method == http.MethodPost && p == fmt.Sprintf("/channels/%d/threads", listenerCh):
			createdThreads++
			return jsonHTTPResp(http.StatusOK, map[string]any{"id": idStr, "type": 11}), nil
		case req.Method == http.MethodPost && strings.HasSuffix(p, "/messages"):
			postedBodies = append(postedBodies, bodyStr)
			if strings.Contains(bodyStr, "このスレッドはアーカイブされました") || strings.Contains(bodyStr, "アーカイブ") || strings.Contains(bodyStr, "過去ログ") {
				archiveNoticeCount++
				select {
				case archiveCh <- struct{}{}:
				default:
				}
			}
			select {
			case livePostCh <- bodyStr:
			default:
			}
			if p == fmt.Sprintf("/channels/%d/messages", listenerCh) {
				guideUpdatedCount++
				select {
				case guideCh <- struct{}{}:
				default:
				}
			}
			return jsonHTTPResp(http.StatusOK, map[string]any{"id": idStr}), nil
		case req.Method == http.MethodPatch && strings.Contains(p, "/messages/"):
			editedBodies = append(editedBodies, bodyStr)
			if p == fmt.Sprintf("/channels/%d/messages/%d", listenerCh, ctrlMsgID) {
				guideUpdatedCount++
				select {
				case guideCh <- struct{}{}:
				default:
				}
			}
			select {
			case livePostCh <- bodyStr:
			default:
			}
			return jsonHTTPResp(http.StatusOK, map[string]any{"id": "1"}), nil
		case req.Method == http.MethodPatch && strings.HasPrefix(p, "/channels/"):
			return jsonHTTPResp(http.StatusOK, map[string]any{"id": strings.TrimPrefix(p, "/channels/"), "type": 11}), nil
		case req.Method == http.MethodGet && strings.HasPrefix(p, "/channels/"):
			return jsonHTTPResp(http.StatusOK, map[string]any{"id": strings.TrimPrefix(p, "/channels/"), "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil
		}
		return jsonHTTPResp(http.StatusOK, map[string]any{"id": idStr}), nil
	})

	syncer := discordsync.NewSyncer(listenerSvc)
	syncer.Debouncer = discordsync.NewControlPanelDebouncerWithTimer(time.Hour, func(_ time.Duration, _ func()) discordsync.DebounceTimer {
		return noopTimer{}
	})

	sub := federation.NewSubscriber(rp, listenerSvc, syncer)
	subCtx, subCancel := context.WithCancel(ctx)
	subDone := make(chan struct{})
	t.Cleanup(func() {
		subCancel()
		<-subDone
	})
	go func() {
		defer close(subDone)
		sub.Run(subCtx, mockClient)
	}()

	// Wait for initial guide refresh signal on initial SSE sync.
	select {
	case <-guideCh:
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for initial SSE resync guide refresh")
	}

	// Live events from Origin: thread.created -> post.created -> post.aboned -> board.updated -> thread.archived.
	_, liveTh, _, err := originSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: origBoard.ID, GuildID: originGuild, UserID: userB, Title: "ライブ通知テストスレ", RawBody: "ライブ1レス目",
	})
	if err != nil {
		t.Fatalf("origin live CreateThread failed: %v", err)
	}
	waitBody := func(substr string) {
		t.Helper()
		timeout := time.After(4 * time.Second)
		for {
			select {
			case s := <-livePostCh:
				if strings.Contains(s, substr) {
					return
				}
			case <-timeout:
				t.Fatalf("timed out waiting for Discord sync payload containing %q", substr)
			}
		}
	}
	waitBody("ライブ1レス目")

	_, _, liveP2, _, err := originSvc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: liveTh.ID, GuildID: originGuild, UserID: userB, RawBody: "ライブ2レス目",
	})
	if err != nil {
		t.Fatalf("origin live CreatePost failed: %v", err)
	}
	waitBody("ライブ2レス目")

	if _, _, err := originSvc.AbonePostByAdmin(ctx, liveP2.ID, "不適切"); err != nil {
		t.Fatalf("origin AbonePostByAdmin failed: %v", err)
	}
	waitBody("あぼーん")

	newBoardName := "本番掲示板改"
	if _, err := originSvc.UpdateBoard(ctx, origBoard.ID, bbs.UpdateBoardInput{Name: &newBoardName}); err != nil {
		t.Fatalf("origin UpdateBoard failed: %v", err)
	}
	select {
	case <-guideCh:
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for guide refresh on board.updated")
	}

	if _, _, err := originSvc.ArchiveThreadByAdmin(ctx, liveTh.ID); err != nil {
		t.Fatalf("origin ArchiveThreadByAdmin failed: %v", err)
	}
	select {
	case <-archiveCh:
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for thread.archived notice")
	}

	// Verify exactly 1 Discord thread mirror exists for liveTh (no duplicates!).
	mirrorsNow, _ := listenerStore.ListThreadMirrorsByMount(ctx, mount.ID)
	liveMirrors := 0
	for _, m := range mirrorsNow {
		if m.ThreadID == liveTh.ID {
			liveMirrors++
			if !m.ArchivedSynced {
				t.Fatalf("expected ArchivedSynced=true on archived mirror")
			}
		}
	}
	if liveMirrors != 1 {
		t.Fatalf("expected exactly 1 ThreadMirror for liveTh, got %d", liveMirrors)
	}

	// 7. Stop Subscriber, perform Origin Epoch swap & offline updates, verify resync catchup + NO duplicate archive!
	subCancel()
	<-subDone

	// Ensure mount.CreatedAt <= offlineTh.CreatedAt cleanly by advancing a moment if needed.
	time.Sleep(15 * time.Millisecond)

	// Offline changes on Origin:
	// (a) new thread created after mount while Listener offline (plus a 2nd reply).
	_, offlineTh, _, err := originSvc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: origBoard.ID, GuildID: originGuild, UserID: capUser, Title: "オフライン新スレ", CapID: "staff", RawBody: "オフラインOP",
	})
	if err != nil {
		t.Fatalf("origin offline CreateThread failed: %v", err)
	}
	if _, _, _, _, pErr := originSvc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: offlineTh.ID, GuildID: originGuild, UserID: capUser, CapID: "staff", RawBody: "オフライン返信2",
	}); pErr != nil {
		t.Fatalf("origin offline CreatePost failed: %v", pErr)
	}

	// Swap Origin to a fresh EventHub (simulates Origin restart with new InstanceEpoch).
	hub2 := federation.NewEventHub(64)
	originSvc.SetEventSink(hub2)
	originSrv2, _ := federation.NewServer(federation.ServerDeps{
		Config: federation.ServerConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:0",
			PublicURL:  originHTTP.URL,
			OAuth:      federation.OAuthConfig{ClientID: "disc", ClientSecret: "sec"},
			Peers:      []federation.PeerConfig{{ID: "peer-dev", Name: "Dev", Token: "tok-dev", AllowWrite: true, AllowCaps: true, RedirectURIs: []string{"http://127.0.0.1:18091/cb"}}},
		},
		Service: originSvc,
		Hub:     hub2,
	})
	currentOriginHandler.Store(originSrv2.Handler())

	discordMu.Lock()
	archiveCountBeforeResync := archiveNoticeCount
	discordMu.Unlock()

	sub2 := federation.NewSubscriber(rp, listenerSvc, syncer)
	sub2Ctx, sub2Cancel := context.WithCancel(ctx)
	sub2Done := make(chan struct{})
	t.Cleanup(func() {
		sub2Cancel()
		<-sub2Done
	})
	go func() {
		defer close(sub2Done)
		sub2.Run(sub2Ctx, mockClient)
	}()

	waitBody("オフラインOP")
	waitBody("オフライン返信2")
	sub2Cancel()
	<-sub2Done

	discordMu.Lock()
	archiveCountAfterResync := archiveNoticeCount
	discordMu.Unlock()
	if archiveCountAfterResync != archiveCountBeforeResync {
		t.Fatalf("expected no duplicate archive notice on resync (ArchivedSynced=true), before=%d after=%d", archiveCountBeforeResync, archiveCountAfterResync)
	}

	// Live board.deleted deactivates mount cleanly.
	sub3 := federation.NewSubscriber(rp, listenerSvc, syncer)
	sub3Ctx, sub3Cancel := context.WithCancel(ctx)
	sub3Done := make(chan struct{})
	t.Cleanup(func() {
		sub3Cancel()
		<-sub3Done
	})
	go func() {
		defer close(sub3Done)
		sub3.Run(sub3Ctx, mockClient)
	}()
	select {
	case <-guideCh:
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for sub3 initial sync")
	}
	if err := originSvc.DeleteBoard(ctx, origBoard.ID); err != nil {
		t.Fatalf("origin DeleteBoard failed: %v", err)
	}
	deadlineDeact := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadlineDeact) {
		mNow, _ := listenerStore.GetBoardMount(ctx, mount.ID)
		if mNow != nil && !mNow.Active {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	mFinal, _ := listenerStore.GetBoardMount(ctx, mount.ID)
	if mFinal == nil || mFinal.Active {
		t.Fatalf("expected board.deleted SSE event to deactivate Listener mount, got %+v", mFinal)
	}
	sub3Cancel()
	<-sub3Done

	// 8. Offline remote Fault Isolation: shutdown originHTTP completely!
	originHTTP.CloseClientConnections()
	originHTTP.Close()
	rp.InvalidateMetadataCache()

	// Listener ListBoards still returns native board without error!
	boardsOffline, err := listenerSvc.ListBoards(ctx, bbs.BoardFilter{Scope: &scopeG})
	if err != nil {
		t.Fatalf("expected native ListBoards to succeed when remote offline, got err=%v", err)
	}
	if len(boardsOffline) != 1 || boardsOffline[0].ID != listNativeBoard.ID {
		t.Fatalf("expected native board returned while remote offline, got %+v", boardsOffline)
	}
	// Native GetBoard succeeds; missing remote board returns ErrFederationUnavailable!
	if gotLocal, gErr := listenerSvc.GetBoard(ctx, listNativeBoard.ID); gErr != nil || gotLocal.ID != listNativeBoard.ID {
		t.Fatalf("expected native GetBoard to succeed while remote offline, got %+v, err=%v", gotLocal, gErr)
	}
	if _, gErr := listenerSvc.GetBoard(ctx, uuid.New()); !errors.Is(gErr, bbs.ErrFederationUnavailable) {
		t.Fatalf("expected ErrFederationUnavailable for missing ID when remote offline, got %v", gErr)
	}
	_ = editedBodies
	_ = postedBodies
	_ = createdThreads
	_ = guideUpdatedCount
}
