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

package discordsync_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body any) *http.Response {
	raw, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(raw)),
	}
}

func newMockRestClient(t *testing.T, rt roundTripFunc) *bot.Client {
	t.Helper()
	httpClient := &http.Client{Transport: rt}
	return &bot.Client{
		ApplicationID: snowflake.ID(999000),
		Rest:          rest.New(rest.NewClient("Bot test-token", rest.WithHTTPClient(httpClient))),
		Caches:        cache.New(),
	}
}

func newSyncerTestEnv(t *testing.T) (*gormstore.Provider, *bbs.Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	store := gormstore.New(db)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	svc := bbs.NewService(
		reg,
		store,
		bbs.NewIdentityGenerator([]byte("syncer-test-secret-key-32-bytes!")),
		bbs.NewPolicyManager(bbs.PolicyConfig{}),
		[]snowflake.ID{snowflake.ID(99999)},
	)
	return store, svc
}

func TestSyncer_PublishFailSafeIsolation_ControlPanel_And_Guide(t *testing.T) {
	ctx := context.Background()
	store, svc := newSyncerTestEnv(t)

	guildSrc := snowflake.ID(10001)
	guildOther := snowflake.ID(10002)
	userID := snowflake.ID(30001)
	chSrc := snowflake.ID(50001)
	chOther := snowflake.ID(50002)

	for _, gid := range []snowflake.ID{guildSrc, guildOther} {
		if _, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: gid, Enabled: true, ActorUserID: 1}); err != nil {
			t.Fatalf("SetupGuild failed: %v", err)
		}
	}
	if _, err := svc.AcceptPolicy(ctx, userID); err != nil {
		t.Fatalf("AcceptPolicy failed: %v", err)
	}
	cat, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeGlobal, Name: "総合"})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeGlobal,
		Name:       "雑談",
		Rules:      bbs.BoardRules{MaxRes: 2}.Normalize(),
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}

	mountSrc := &bbs.BoardMount{ID: uuid.New(), BoardID: board.ID, GuildID: guildSrc, ChannelID: chSrc, Locale: "ja", Active: true}
	mountOther := &bbs.BoardMount{ID: uuid.New(), BoardID: board.ID, GuildID: guildOther, ChannelID: chOther, Locale: "ja", Active: true}
	if err := store.CreateBoardMount(ctx, mountSrc); err != nil {
		t.Fatalf("CreateBoardMount src failed: %v", err)
	}
	if err := store.CreateBoardMount(ctx, mountOther); err != nil {
		t.Fatalf("CreateBoardMount other failed: %v", err)
	}

	_, thread, firstPost, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: guildSrc,
		UserID:  userID,
		Title:   "フェイルセーフテスト",
		RawBody: "1レス目",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	var mu sync.Mutex
	var nextID int64 = 700000
	var otherMirrorThreadID snowflake.ID
	otherThreadMirrored := make(chan struct{}, 4)
	otherReplyMirrored := make(chan struct{}, 4)
	otherLockedAndArchived := make(chan struct{}, 4)
	failSrc := true

	client := newMockRestClient(t, func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		nextID++
		id := snowflake.ID(nextID)
		failSourceNow := failSrc
		otherTID := otherMirrorThreadID
		mu.Unlock()

		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		switch {
		case req.Method == http.MethodPost && p == fmt.Sprintf("/channels/%d/threads", chSrc) && failSourceNow:
			return jsonResponse(http.StatusInternalServerError, map[string]any{"message": "internal error"}), nil

		case req.Method == http.MethodPost && p == fmt.Sprintf("/channels/%d/threads", chOther):
			mu.Lock()
			otherMirrorThreadID = id
			mu.Unlock()
			return jsonResponse(http.StatusOK, map[string]any{"id": id.String(), "type": 11}), nil

		case req.Method == http.MethodPost && strings.HasPrefix(p, "/channels/") && strings.HasSuffix(p, "/messages"):
			rawBody, _ := io.ReadAll(req.Body)
			rawStr := string(rawBody)
			if otherTID != 0 && p == fmt.Sprintf("/channels/%d/messages", otherTID) && strings.Contains(rawStr, "2レス目 (max到達)") {
				select {
				case otherReplyMirrored <- struct{}{}:
				default:
				}
			} else if !strings.Contains(rawStr, "Over 1000 Thread") && !strings.Contains(rawStr, "Board Guide") && !strings.Contains(rawStr, "ローカル板") && !strings.Contains(rawStr, "グローバル板") && !strings.Contains(rawStr, "スレッド操作") {
				select {
				case otherThreadMirrored <- struct{}{}:
				default:
				}
			}
			return jsonResponse(http.StatusOK, map[string]any{"id": id.String(), "channel_id": "1"}), nil

		case req.Method == http.MethodPatch && strings.HasPrefix(p, "/channels/"):
			rawBody, _ := io.ReadAll(req.Body)
			var patchPayload map[string]any
			_ = json.Unmarshal(rawBody, &patchPayload)
			arch, _ := patchPayload["archived"].(bool)
			locked, _ := patchPayload["locked"].(bool)
			if otherTID != 0 && p == fmt.Sprintf("/channels/%d", otherTID) && arch && locked {
				select {
				case otherLockedAndArchived <- struct{}{}:
				default:
				}
			}
			return jsonResponse(http.StatusOK, map[string]any{"id": strings.TrimPrefix(p, "/channels/"), "type": 11}), nil

		case req.Method == http.MethodGet && strings.HasPrefix(p, "/channels/"):
			return jsonResponse(http.StatusOK, map[string]any{"id": strings.TrimPrefix(p, "/channels/"), "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil

		case req.Method == http.MethodDelete:
			return jsonResponse(http.StatusNoContent, nil), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{"id": id.String()}), nil
	})

	syncer := discordsync.NewSyncer(svc)
	syncer.Debouncer = discordsync.NewControlPanelDebouncerWithTimer(time.Second, func(_ time.Duration, _ func()) discordsync.DebounceTimer {
		return &fakeTimer{}
	})

	// 1. Source mount failure on PublishThread must return err, YET still fan out to mountOther.
	if _, _, pErr := syncer.PublishThread(ctx, client, mountSrc.ID, *board, *thread, *firstPost, "ja"); pErr == nil {
		t.Fatal("expected PublishThread error when source mount fails")
	}
	select {
	case <-otherThreadMirrored:
	case <-time.After(2 * time.Second):
		t.Fatal("expected fanout to create thread mirror on otherMount even when source mount failed")
	}

	// Verify mountOther now has a ThreadMirror in DB.
	var mirrors []bbs.ThreadMirror
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mirrors, _ = store.ListThreadMirrorsByThread(ctx, thread.ID)
		if len(mirrors) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(mirrors) != 1 || mirrors[0].BoardMountID != mountOther.ID {
		t.Fatalf("expected fanout mirror for mountOther, got %+v", mirrors)
	}

	// 2. Deactivate mountSrc mid-flight; PublishPost (res #2 reaching MaxRes=2 -> archived) must return ErrSourceMountInactive
	// YET still sync reply AND send SyncArchive PATCH{archived:true, locked:true} to mountOther!
	if err := store.DeactivateBoardMount(ctx, mountSrc.ID); err != nil {
		t.Fatalf("DeactivateBoardMount failed: %v", err)
	}
	_, archivedTh, replyPost, backRefs, err := svc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: thread.ID,
		GuildID:  guildSrc,
		UserID:   userID,
		RawBody:  "2レス目 (max到達)",
	})
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	_, _, postSyncErr := syncer.PublishPost(ctx, client, mountSrc.ID, *board, *archivedTh, *replyPost, backRefs, "ja")
	if !errors.Is(postSyncErr, discordsync.ErrSourceMountInactive) {
		t.Fatalf("expected ErrSourceMountInactive when source mount deactivated, got %v", postSyncErr)
	}
	select {
	case <-otherReplyMirrored:
	case <-time.After(2 * time.Second):
		t.Fatal("expected reply message #2 to sync to mountOther before continuing")
	}
	select {
	case <-otherLockedAndArchived:
	case <-time.After(2 * time.Second):
		t.Fatal("expected SyncArchive PATCH with archived=true and locked=true on otherMirrorThreadID")
	}

	// 3. SyncAbone must return REST error when UpdateMessage fails.
	errSyncer := discordsync.NewSyncer(svc)
	failClient := newMockRestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch && strings.Contains(req.URL.Path, "/messages/") {
			return jsonResponse(http.StatusInternalServerError, map[string]any{"message": "boom"}), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{"id": "1", "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil
	})
	if err := errSyncer.SyncAbone(ctx, failClient, *replyPost, *archivedTh); err == nil {
		t.Fatal("expected SyncAbone to return non-404 REST error")
	}

	// 4. SyncAbone must also return REST error when UpdateMessage succeeds (200) but ArchiveAndLockThread fails (503).
	archFailClient := newMockRestClient(t, func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		if req.Method == http.MethodPatch && !strings.Contains(p, "/messages/") {
			return jsonResponse(http.StatusServiceUnavailable, map[string]any{"message": "archive lock failed"}), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{"id": "1", "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil
	})
	if err := errSyncer.SyncAbone(ctx, archFailClient, *replyPost, *archivedTh); err == nil {
		t.Fatal("expected SyncAbone to propagate ArchiveAndLockThread 503 failure")
	}
}

func TestSyncer_Round3ControlPreservation_LateReplyLock_And_RefreshGuide(t *testing.T) {
	ctx := context.Background()
	store, svc := newSyncerTestEnv(t)

	guildID := snowflake.ID(20001)
	userAuthor := snowflake.ID(40001)
	userLate := snowflake.ID(40002)
	if _, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: 1}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	for _, uid := range []snowflake.ID{userAuthor, userLate} {
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
	}
	cat, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeLocal, GuildID: &guildID, Name: "カテゴリ"})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeLocal,
		GuildID:    &guildID,
		Name:       "ローカル板",
		Rules:      bbs.BoardRules{DefaultName: "初版名無し", MaxRes: 100}.Normalize(),
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	chID := snowflake.ID(51001)
	ctrlGuideID := snowflake.ID(51002)
	mount := &bbs.BoardMount{
		ID:               uuid.New(),
		BoardID:          board.ID,
		GuildID:          guildID,
		ChannelID:        chID,
		Locale:           "ja",
		IsForum:          false,
		ControlMessageID: &ctrlGuideID,
		Active:           true,
	}
	if err := store.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}

	_, th, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: guildID,
		UserID:  userAuthor,
		Title:   "遅延同期スレッド",
		RawBody: "1レス目",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	discordTID := snowflake.ID(51010)
	mirrorCtrlID := snowflake.ID(51011)
	tm := &bbs.ThreadMirror{
		ID:               uuid.New(),
		ThreadID:         th.ID,
		BoardMountID:     mount.ID,
		GuildID:          guildID,
		DiscordThreadID:  discordTID,
		ControlMessageID: &mirrorCtrlID,
		SyncedUpToRes:    1,
	}
	if err := store.CreateThreadMirror(ctx, tm); err != nil {
		t.Fatalf("CreateThreadMirror failed: %v", err)
	}

	// 1. SyncArchive: when DELETE control message returns 503, ControlMessageID must be preserved in DB;
	// when DELETE returns 200 later, ControlMessageID is cleared to nil.
	var mu sync.Mutex
	deleteFail := true
	rearchivedAfterLateReply := false
	var guideEditBody string

	client := newMockRestClient(t, func(req *http.Request) (*http.Response, error) {
		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		mu.Lock()
		shouldFailDel := deleteFail
		mu.Unlock()

		switch {
		case req.Method == http.MethodDelete && p == fmt.Sprintf("/channels/%d/messages/%d", discordTID, mirrorCtrlID):
			if shouldFailDel {
				return jsonResponse(http.StatusServiceUnavailable, map[string]any{"message": "503"}), nil
			}
			return jsonResponse(http.StatusNoContent, nil), nil
		case req.Method == http.MethodPatch && p == fmt.Sprintf("/channels/%d", discordTID):
			raw, _ := io.ReadAll(req.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			if arch, _ := body["archived"].(bool); arch {
				if lck, _ := body["locked"].(bool); lck {
					mu.Lock()
					rearchivedAfterLateReply = true
					mu.Unlock()
				}
			}
			return jsonResponse(http.StatusOK, map[string]any{"id": discordTID.String(), "type": 11}), nil
		case req.Method == http.MethodPatch && p == fmt.Sprintf("/channels/%d/messages/%d", chID, ctrlGuideID):
			raw, _ := io.ReadAll(req.Body)
			mu.Lock()
			guideEditBody = string(raw)
			mu.Unlock()
			return jsonResponse(http.StatusOK, map[string]any{"id": ctrlGuideID.String()}), nil
		case req.Method == http.MethodGet && strings.HasPrefix(p, "/channels/"):
			return jsonResponse(http.StatusOK, map[string]any{"id": strings.TrimPrefix(p, "/channels/"), "type": 11, "thread_metadata": map[string]any{"archived": false}}), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{"id": "880001"}), nil
	})

	syncer := discordsync.NewSyncer(svc)
	debouncerScheduled := false
	syncer.Debouncer = discordsync.NewControlPanelDebouncerWithTimer(time.Second, func(_ time.Duration, _ func()) discordsync.DebounceTimer {
		debouncerScheduled = true
		return &fakeTimer{}
	})

	// Create late reply BEFORE archiving DB thread, then archive thread before late reply mirrors!
	_, _, latePost, backRefs, err := svc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: th.ID,
		GuildID:  guildID,
		UserID:   userLate,
		RawBody:  "閉鎖直前の遅延レス",
	})
	if err != nil {
		t.Fatalf("CreatePost late failed: %v", err)
	}
	archivedTh, err := store.ArchiveThread(ctx, th.ID)
	if err != nil {
		t.Fatalf("ArchiveThread failed: %v", err)
	}

	syncer.SyncArchive(ctx, client, *board, *archivedTh, false)
	reloadedTM, err := store.GetThreadMirror(ctx, tm.ID)
	if err != nil || reloadedTM.ControlMessageID == nil || *reloadedTM.ControlMessageID != mirrorCtrlID {
		t.Fatalf("expected ControlMessageID to be preserved on DELETE 503, got %+v, err=%v", reloadedTM, err)
	}

	mu.Lock()
	deleteFail = false
	rearchivedAfterLateReply = false
	mu.Unlock()
	syncer.SyncArchive(ctx, client, *board, *archivedTh, false)
	reloadedTM, err = store.GetThreadMirror(ctx, tm.ID)
	if err != nil || reloadedTM.ControlMessageID != nil {
		t.Fatalf("expected ControlMessageID to be cleared once DELETE succeeds, got %+v, err=%v", reloadedTM, err)
	}

	// 2. Late PublishPost carrying stale Active snapshot for an already-archived thread must re-lock and NOT schedule control panel.
	staleActiveTh := *th
	staleActiveTh.Status = bbs.ThreadStatusActive
	mu.Lock()
	rearchivedAfterLateReply = false
	mu.Unlock()
	debouncerScheduled = false
	if _, _, pErr := syncer.PublishPost(ctx, client, mount.ID, *board, staleActiveTh, *latePost, backRefs, "ja"); pErr != nil {
		t.Fatalf("PublishPost late failed: %v", pErr)
	}
	mu.Lock()
	wasRelocked := rearchivedAfterLateReply
	mu.Unlock()
	if !wasRelocked {
		t.Fatal("expected late reply on DB-archived thread to re-apply ArchiveAndLockThread")
	}
	if debouncerScheduled {
		t.Fatal("expected late reply on DB-archived thread NOT to schedule control panel")
	}

	// 3. RefreshBoardGuide must reload latest board rules from DB under channel lock.
	newDefName := "更新後の名無し"
	if _, err := svc.UpdateBoard(ctx, board.ID, bbs.UpdateBoardInput{DefaultName: &newDefName}); err != nil {
		t.Fatalf("UpdateBoard failed: %v", err)
	}
	syncer.RefreshBoardGuide(ctx, client, guildID, board.ID, "ja")
	mu.Lock()
	bodyAfterRefresh := guideEditBody
	mu.Unlock()
	if !strings.Contains(bodyAfterRefresh, newDefName) {
		t.Fatalf("expected RefreshBoardGuide to send updated rule %q, got %s", newDefName, bodyAfterRefresh)
	}
}
