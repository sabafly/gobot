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

package federation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

func newSubscriberTestStore(t *testing.T) *gormstore.Provider {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	return gormstore.New(db)
}

func TestSubscriber_ResyncRunsBeforeOpenEventsStream(t *testing.T) {
	store := newSubscriberTestStore(t)
	reg := bbs.NewProviderRegistry(store.Key())
	reg.Register(store)
	svc := bbs.NewService(reg, store, bbs.NewIdentityGenerator([]byte("test-secret-key-32-bytes-long!!!")), bbs.NewPolicyManager(bbs.PolicyConfig{}), nil)

	boardID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var callOrder []string

	fakeOrigin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callOrder = append(callOrder, r.URL.Path)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/bbs/federation/capabilities":
			_ = json.NewEncoder(w).Encode(CapabilitiesResponse{
				ProtocolVersion: ProtocolVersion,
				InstanceEpoch:   "epoch-1",
				LatestSeq:       10,
				PeerID:          "peer-a",
				PolicyVersion:   "2026-01-01",
			})
		case r.URL.Path == "/api/v1/bbs/federation/boards/"+boardID.String():
			_ = json.NewEncoder(w).Encode(bbs.Board{
				ID:    boardID,
				Scope: bbs.ScopeGlobal,
				Name:  "Board A",
			})
		case strings.HasSuffix(r.URL.Path, "/threads"):
			_ = json.NewEncoder(w).Encode(ThreadListResponse{Threads: []bbs.Thread{}})
		case r.URL.Path == "/api/v1/bbs/federation/events":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("event: ready\ndata: {\"epoch\":\"epoch-1\",\"latest_seq\":10}\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			cancel()
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer fakeOrigin.Close()

	rp, err := NewRemoteProvider(RemoteProviderDeps{
		Config: RemoteConfig{
			Key:     "prod",
			Label:   "Prod",
			BaseURL: fakeOrigin.URL,
			Token:   "tok",
		},
		Store: store,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider failed: %v", err)
	}
	reg.Register(rp)

	mount := &bbs.BoardMount{
		BoardID:     boardID,
		ProviderKey: rp.Key(),
		GuildID:     snowflake.ID(1001),
		ChannelID:   snowflake.ID(2001),
		Active:      true,
	}
	if err := store.CreateBoardMount(context.Background(), mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}

	sub := NewSubscriber(rp, svc, nil)
	sub.connectOnce(ctx, nil)

	mu.Lock()
	order := append([]string(nil), callOrder...)
	mu.Unlock()

	boardIdx, eventsIdx := -1, -1
	for i, p := range order {
		if p == "/api/v1/bbs/federation/boards/"+boardID.String() && boardIdx == -1 {
			boardIdx = i
		}
		if p == "/api/v1/bbs/federation/events" && eventsIdx == -1 {
			eventsIdx = i
		}
	}
	if boardIdx == -1 || eventsIdx == -1 {
		t.Fatalf("expected both board resync and events stream calls, got order=%v", order)
	}
	if boardIdx > eventsIdx {
		t.Fatalf("expected resync GetBoard (idx=%d) to run BEFORE OpenEventsStream (idx=%d); order=%v", boardIdx, eventsIdx, order)
	}
}

func TestSubscriber_ResyncPreservesMountExcludedByAllowedBoardIDs(t *testing.T) {
	store := newSubscriberTestStore(t)
	reg := bbs.NewProviderRegistry(store.Key())
	reg.Register(store)
	svc := bbs.NewService(reg, store, bbs.NewIdentityGenerator([]byte("test-secret-key-32-bytes-long!!!")), bbs.NewPolicyManager(bbs.PolicyConfig{}), nil)

	excludedBoardID := uuid.New()
	deletedBoardID := uuid.New()

	fakeOrigin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/bbs/federation/capabilities" {
			_ = json.NewEncoder(w).Encode(CapabilitiesResponse{
				ProtocolVersion: ProtocolVersion,
				InstanceEpoch:   "epoch-1",
				LatestSeq:       1,
				PeerID:          "peer-a",
				AllowedBoardIDs: []uuid.UUID{deletedBoardID},
				PolicyVersion:   "2026-01-01",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Code: CodeNotFound, Error: "not found"})
	}))
	defer fakeOrigin.Close()

	rp, err := NewRemoteProvider(RemoteProviderDeps{
		Config: RemoteConfig{
			Key:     "prod",
			Label:   "Prod",
			BaseURL: fakeOrigin.URL,
			Token:   "tok",
		},
		Store: store,
	})
	if err != nil {
		t.Fatalf("NewRemoteProvider failed: %v", err)
	}
	reg.Register(rp)

	ctx := context.Background()
	excludedMount := &bbs.BoardMount{
		BoardID:     excludedBoardID,
		ProviderKey: rp.Key(),
		GuildID:     snowflake.ID(1001),
		ChannelID:   snowflake.ID(2001),
		Active:      true,
	}
	if err := store.CreateBoardMount(ctx, excludedMount); err != nil {
		t.Fatalf("CreateBoardMount excluded failed: %v", err)
	}
	deletedMount := &bbs.BoardMount{
		BoardID:     deletedBoardID,
		ProviderKey: rp.Key(),
		GuildID:     snowflake.ID(1001),
		ChannelID:   snowflake.ID(2002),
		Active:      true,
	}
	if err := store.CreateBoardMount(ctx, deletedMount); err != nil {
		t.Fatalf("CreateBoardMount deleted failed: %v", err)
	}

	sub := NewSubscriber(rp, svc, nil)
	sub.resyncMountedBoards(ctx, nil, []uuid.UUID{deletedBoardID})

	gotExcluded, err := store.GetBoardMount(ctx, excludedMount.ID)
	if err != nil {
		t.Fatalf("GetBoardMount excluded failed: %v", err)
	}
	if !gotExcluded.Active {
		t.Fatal("expected mount excluded by allowed_board_ids to remain active")
	}

	gotDeleted, err := store.GetBoardMount(ctx, deletedMount.ID)
	if err != nil {
		t.Fatalf("GetBoardMount deleted failed: %v", err)
	}
	if gotDeleted.Active {
		t.Fatal("expected mount for actually deleted allowed board to be deactivated")
	}
}
