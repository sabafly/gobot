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

package gormstore_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

func openTestService(t *testing.T) (*gormstore.Provider, *bbs.Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	store := gormstore.New(db)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	svc := bbs.NewService(
		reg,
		store,
		bbs.NewIdentityGenerator([]byte("sqlite-test-secret")),
		bbs.NewPolicyManager(bbs.PolicyConfig{Version: "2026-11-25"}),
		[]snowflake.ID{snowflake.ID(9999)},
	)
	return store, svc
}

func TestGormStore_CRUD_Concurrency_Archive_Abone_Ban(t *testing.T) {
	ctx := context.Background()
	store, svc := openTestService(t)

	guildID := snowflake.ID(10001)
	adminID := snowflake.ID(20001)
	userA := snowflake.ID(30001)
	userB := snowflake.ID(30002)

	// Opt-in enforcement
	if _, err := svc.EnsureGuildEnabled(ctx, guildID); !errors.Is(err, bbs.ErrGuildNotEnabled) {
		t.Fatalf("expected ErrGuildNotEnabled, got %v", err)
	}
	if _, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: adminID}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}

	// Category & Board (MaxRes = 12 for archive test)
	cat, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope:   bbs.ScopeLocal,
		GuildID: &guildID,
		Slug:    "zatsudan",
		Name:    "雑談カテゴリ",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeLocal,
		GuildID:    &guildID,
		Slug:       "nanJ",
		Name:       "なんでも実況",
		Rules: bbs.BoardRules{
			DefaultName: "風吹けば名無し",
			MaxRes:      12,
		},
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}

	// Consent enforcement
	if _, _, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: guildID,
		UserID:  userA,
		Title:   "テストスレ",
		RawBody: "1レス目",
	}); !errors.Is(err, bbs.ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired, got %v", err)
	}

	if _, err := svc.AcceptPolicy(ctx, userA); err != nil {
		t.Fatalf("AcceptPolicy A failed: %v", err)
	}
	if _, err := svc.AcceptPolicy(ctx, userB); err != nil {
		t.Fatalf("AcceptPolicy B failed: %v", err)
	}

	_, thread, firstPost, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: guildID,
		UserID:  userA,
		Title:   "テストスレ",
		RawBody: ">>1 テスト",
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}
	if firstPost.ResNumber != 1 || !firstPost.IsOP {
		t.Fatalf("expected OP res 1, got %+v", firstPost)
	}

	// Concurrent CreatePost directly via provider (bypassing per-user cooldown)
	const workers = 10
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			uid := userB
			if idx%2 == 0 {
				uid = snowflake.ID(40000 + idx)
			}
			_, _, err := store.CreatePost(ctx, bbs.CreatePostRecord{
				ThreadID:     thread.ID,
				Name:         "風吹けば名無し",
				AnonID:       "anon1234",
				PosterHash:   fmt.Sprintf("hash%d", idx),
				IsSage:       idx == 0,
				Body:         fmt.Sprintf("並行レス %d >>1", idx),
				AuthorUserID: uid,
			})
			if err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent CreatePost failed: %v", err)
	}

	posts, err := svc.ListPosts(ctx, thread.ID, 0, 50)
	if err != nil {
		t.Fatalf("ListPosts failed: %v", err)
	}
	if len(posts) != 11 {
		t.Fatalf("expected 11 posts, got %d", len(posts))
	}
	for i, p := range posts {
		if p.ResNumber != uint32(i+1) {
			t.Fatalf("expected contiguous ResNumber %d, got %d", i+1, p.ResNumber)
		}
	}

	// Sage check: post with sage does not bump bumped_at relative to non-sage post
	updatedTh, err := svc.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}
	prevBump := updatedTh.BumpedAt
	time.Sleep(10 * time.Millisecond)

	// 12th post (sage) hits MaxRes=12 -> archives thread, does not update BumpedAt
	sagePost, archivedTh, err := store.CreatePost(ctx, bbs.CreatePostRecord{
		ThreadID:     thread.ID,
		Name:         "風吹けば名無し",
		AnonID:       "sageanon",
		PosterHash:   "sagehash",
		IsSage:       true,
		Body:         "sage進行で12到達",
		AuthorUserID: userB,
	})
	if err != nil {
		t.Fatalf("12th CreatePost failed: %v", err)
	}
	if sagePost.ResNumber != 12 {
		t.Fatalf("expected res 12, got %d", sagePost.ResNumber)
	}
	if archivedTh.Status != bbs.ThreadStatusArchived {
		t.Fatalf("expected archived status at MaxRes=12, got %s", archivedTh.Status)
	}
	if !archivedTh.BumpedAt.Equal(prevBump) {
		t.Fatalf("sage post must not update BumpedAt: prev=%v, now=%v", prevBump, archivedTh.BumpedAt)
	}

	// Abone preserves res number
	aboned, err := svc.AbonePostLocal(ctx, guildID, posts[2].ID, "荒らし")
	if err != nil {
		t.Fatalf("AbonePostLocal failed: %v", err)
	}
	if !aboned.Abone || aboned.ResNumber != 3 {
		t.Fatalf("expected abone preserving res 3, got %+v", aboned)
	}

	// Local ban blocks userA while preserving anonymity in DTOs
	if _, err := svc.BanLocal(ctx, guildID, adminID, firstPost.PosterHash, "連投", nil); err != nil {
		t.Fatalf("BanLocal failed: %v", err)
	}
	if _, _, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: guildID,
		UserID:  userA,
		Title:   "BAN後スレ",
		RawBody: "書けないはず",
	}); !errors.Is(err, bbs.ErrBanned) {
		t.Fatalf("expected ErrBanned, got %v", err)
	}

	// Report & operator audit
	rep, err := svc.CreateReport(ctx, guildID, userB, firstPost.ID, "テスト通報")
	if err != nil {
		t.Fatalf("CreateReport failed: %v", err)
	}
	if _, _, err := svc.InspectReportForOperator(ctx, userA, rep.ID); !errors.Is(err, bbs.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for non-operator, got %v", err)
	}
	_, audit, err := svc.InspectReportForOperator(ctx, snowflake.ID(9999), rep.ID)
	if err != nil {
		t.Fatalf("InspectReportForOperator failed: %v", err)
	}
	if audit.AuthorUserID != userA {
		t.Fatalf("expected audit AuthorUserID %d, got %d", userA, audit.AuthorUserID)
	}
}
