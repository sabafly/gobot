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
	otherThreadMirrored := make(chan struct{}, 4)
	otherArchived := make(chan struct{}, 4)
	failSrc := true

	client := newMockRestClient(t, func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		nextID++
		id := snowflake.ID(nextID)
		failSourceNow := failSrc
		mu.Unlock()

		p := strings.TrimPrefix(req.URL.Path, "/api/v10")
		switch {
		case req.Method == http.MethodPost && p == fmt.Sprintf("/channels/%d/threads", chSrc) && failSourceNow:
			return jsonResponse(http.StatusInternalServerError, map[string]any{"message": "internal error"}), nil

		case req.Method == http.MethodPost && p == fmt.Sprintf("/channels/%d/threads", chOther):
			return jsonResponse(http.StatusOK, map[string]any{"id": id.String(), "type": 11}), nil

		case req.Method == http.MethodPost && strings.HasPrefix(p, "/channels/") && strings.HasSuffix(p, "/messages"):
			var payload map[string]any
			rawBody, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(rawBody, &payload)
			rawStr := string(rawBody)
			if strings.Contains(rawStr, "Over 1000 Thread") {
				select {
				case otherArchived <- struct{}{}:
				default:
				}
			} else if !strings.Contains(rawStr, "Board Guide") && !strings.Contains(rawStr, "ローカル板") && !strings.Contains(rawStr, "グローバル板") && !strings.Contains(rawStr, "スレッド操作") {
				select {
				case otherThreadMirrored <- struct{}{}:
				default:
				}
			}
			return jsonResponse(http.StatusOK, map[string]any{"id": id.String(), "channel_id": "1"}), nil

		case req.Method == http.MethodPatch && strings.HasPrefix(p, "/channels/"):
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
	// YET still sync reply + SyncArchive to mountOther!
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
	case <-otherArchived:
	case <-time.After(2 * time.Second):
		t.Fatal("expected SyncArchive fanout even when source mount was inactive")
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
}
