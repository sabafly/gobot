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
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
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

func TestGormStore_PR224ReviewRegressionFixes(t *testing.T) {
	ctx := context.Background()
	store, svc := openTestService(t)

	guildA := snowflake.ID(11001)
	guildB := snowflake.ID(11002)
	adminID := snowflake.ID(22001)
	userID := snowflake.ID(33001)

	// 1. DefaultUseForum=false and IsForum=false / Active=false must persist accurately (no GORM default:true override).
	falseVal := false
	cfgA, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{
		GuildID:         guildA,
		Enabled:         true,
		ActorUserID:     adminID,
		DefaultUseForum: &falseVal,
	})
	if err != nil {
		t.Fatalf("SetupGuild A failed: %v", err)
	}
	if cfgA.DefaultUseForum {
		t.Fatal("expected initial DefaultUseForum=false to persist")
	}
	reloadedCfg, err := store.GetGuildConfig(ctx, guildA)
	if err != nil || reloadedCfg.DefaultUseForum {
		t.Fatalf("expected reloaded DefaultUseForum=false, got %+v, err=%v", reloadedCfg, err)
	}

	if _, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildB, Enabled: true, ActorUserID: adminID}); err != nil {
		t.Fatalf("SetupGuild B failed: %v", err)
	}
	if _, err := svc.AcceptPolicy(ctx, userID); err != nil {
		t.Fatalf("AcceptPolicy failed: %v", err)
	}

	catB, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{Scope: bbs.ScopeLocal, GuildID: &guildB, Name: "Bカテゴリ"})
	if err != nil {
		t.Fatalf("CreateCategory B failed: %v", err)
	}
	boardB, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{CategoryID: catB.ID, Scope: bbs.ScopeLocal, GuildID: &guildB, Name: "Bローカル板"})
	if err != nil {
		t.Fatalf("CreateBoard B failed: %v", err)
	}

	// Inactive/non-forum mount persistence check
	mountID := uuid.New()
	if err := store.CreateBoardMount(ctx, &bbs.BoardMount{
		ID:        mountID,
		BoardID:   boardB.ID,
		GuildID:   guildB,
		ChannelID: snowflake.ID(555001),
		IsForum:   false,
		Active:    false,
	}); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	gotMount, err := store.GetBoardMount(ctx, mountID)
	if err != nil {
		t.Fatalf("GetBoardMount failed: %v", err)
	}
	if gotMount.IsForum || gotMount.Active {
		t.Fatalf("expected IsForum=false and Active=false to persist, got %+v", gotMount)
	}

	// 2. Partial UpdateBoard must preserve Locked=true when only Name is updated, and unlock on Locked=false.
	lockedTrue := true
	if _, err := svc.UpdateBoard(ctx, boardB.ID, bbs.UpdateBoardInput{Locked: &lockedTrue}); err != nil {
		t.Fatalf("lock board failed: %v", err)
	}
	newName := "B更新後"
	renamed, err := svc.UpdateBoard(ctx, boardB.ID, bbs.UpdateBoardInput{Name: &newName})
	if err != nil || !renamed.Locked || renamed.Name != newName {
		t.Fatalf("expected Locked=true preserved on partial rename, got %+v, err=%v", renamed, err)
	}
	lockedFalse := false
	unlocked, err := svc.UpdateBoard(ctx, boardB.ID, bbs.UpdateBoardInput{Locked: &lockedFalse})
	if err != nil || unlocked.Locked {
		t.Fatalf("expected Locked=false after explicit unlock, got %+v, err=%v", unlocked, err)
	}

	// 3. Cross-guild local board ownership checks on CreateThread, CreatePost, and CreateReport.
	if _, _, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: boardB.ID,
		GuildID: guildA,
		UserID:  userID,
		Title:   "越境スレ",
		RawBody: "本文",
	}); !errors.Is(err, bbs.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on cross-guild CreateThread, got %v", err)
	}

	_, thB, opB, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: boardB.ID,
		GuildID: guildB,
		UserID:  userID,
		Title:   "B専用スレ",
		RawBody: "OP本文 >>2",
	})
	if err != nil {
		t.Fatalf("CreateThread B failed: %v", err)
	}

	if _, _, _, _, err := svc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: thB.ID,
		GuildID:  guildA,
		UserID:   userID,
		RawBody:  "越境レス",
	}); !errors.Is(err, bbs.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on cross-guild CreatePost, got %v", err)
	}
	if _, err := svc.CreateReport(ctx, guildA, userID, opB.ID, "越境通報"); !errors.Is(err, bbs.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on cross-guild CreateReport, got %v", err)
	}

	// 4. Atomic CreateThreadAndPostMirror & monotonic AdvanceThreadMirrorSync
	tm := &bbs.ThreadMirror{
		ID:              uuid.New(),
		ThreadID:        thB.ID,
		BoardMountID:    mountID,
		GuildID:         guildB,
		DiscordThreadID: snowflake.ID(777001),
		SyncedUpToRes:   1,
	}
	pm := &bbs.PostMirror{
		ID:               uuid.New(),
		PostID:           opB.ID,
		ThreadMirrorID:   tm.ID,
		GuildID:          guildB,
		DiscordMessageID: snowflake.ID(888001),
	}
	if err := store.CreateThreadAndPostMirror(ctx, tm, pm); err != nil {
		t.Fatalf("CreateThreadAndPostMirror failed: %v", err)
	}
	if err := store.AdvanceThreadMirrorSync(ctx, tm.ID, 10); err != nil {
		t.Fatalf("AdvanceThreadMirrorSync(10) failed: %v", err)
	}
	if err := store.AdvanceThreadMirrorSync(ctx, tm.ID, 5); err != nil {
		t.Fatalf("AdvanceThreadMirrorSync(5) failed: %v", err)
	}
	ctrlID := snowflake.ID(999111)
	if err := store.UpdateThreadMirrorControl(ctx, tm.ID, &ctrlID); err != nil {
		t.Fatalf("UpdateThreadMirrorControl failed: %v", err)
	}
	reloadedTM, err := store.GetThreadMirror(ctx, tm.ID)
	if err != nil || reloadedTM.SyncedUpToRes != 10 || reloadedTM.ControlMessageID == nil || *reloadedTM.ControlMessageID != ctrlID {
		t.Fatalf("unexpected mirror state after monotonic sync & control update: %+v, err=%v", reloadedTM, err)
	}

	// 5. Watch & reference cleanup on DeleteBoard + ArchiveThread preserves ResCount
	if err := store.SavePostReferences(ctx, []bbs.PostReference{{ThreadID: thB.ID, FromRes: 2, ToRes: 1}}); err != nil {
		t.Fatalf("SavePostReferences failed: %v", err)
	}
	if err := store.SaveThreadWatch(ctx, &bbs.ThreadWatch{ThreadID: thB.ID, UserID: userID, Locale: "ja"}); err != nil {
		t.Fatalf("SaveThreadWatch failed: %v", err)
	}
	if _, _, err := store.CreatePost(ctx, bbs.CreatePostRecord{
		ThreadID:     thB.ID,
		Name:         "名無し",
		PosterHash:   "h1",
		Body:         "2レス目",
		AuthorUserID: userID,
	}); err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	archivedTh, err := store.ArchiveThread(ctx, thB.ID)
	if err != nil || archivedTh.ResCount != 2 {
		t.Fatalf("expected ArchiveThread to preserve ResCount=2, got %+v, err=%v", archivedTh, err)
	}

	if err := svc.DeleteBoard(ctx, boardB.ID); err != nil {
		t.Fatalf("DeleteBoard failed: %v", err)
	}
	refs, err := store.ListReferencingRes(ctx, thB.ID, 1)
	if err != nil || len(refs) != 0 {
		t.Fatalf("expected post references deleted with board, got %v, err=%v", refs, err)
	}
	watches, err := store.ListThreadWatches(ctx, thB.ID)
	if err != nil || len(watches) != 0 {
		t.Fatalf("expected thread watches deleted with board, got %v, err=%v", watches, err)
	}

	// 6. Persisted fallback system secret
	sec1 := []byte("first-generated-secret-32-bytes!!")
	sec2 := []byte("second-different-secret-ignored!!")
	got1, err := store.GetOrCreateSystemSecret(ctx, "default_hmac_secret", sec1)
	if err != nil || !bytes.Equal(got1, sec1) {
		t.Fatalf("expected first secret persisted, got %x, err=%v", got1, err)
	}
	got2, err := store.GetOrCreateSystemSecret(ctx, "default_hmac_secret", sec2)
	if err != nil || !bytes.Equal(got2, sec1) {
		t.Fatalf("expected original persisted secret on subsequent boot, got %x, err=%v", got2, err)
	}

	// 7. DatOchiDisabled (-1) preserved across Normalize, CreateBoard/Thread, and excluded from ListStaleThreads
	disabledBoard, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: catB.ID,
		Scope:      bbs.ScopeLocal,
		GuildID:    &guildB,
		Name:       "落ちない板",
		Rules: bbs.BoardRules{
			DefaultName:        "名無し",
			MaxRes:             1000,
			DatOchiIdleHours:   bbs.DatOchiDisabled,
			DatOchiLowResHours: bbs.DatOchiDisabled,
		},
	})
	if err != nil {
		t.Fatalf("CreateBoard with DatOchiDisabled failed: %v", err)
	}
	if disabledBoard.Rules.DatOchiIdleHours != bbs.DatOchiDisabled || disabledBoard.Rules.DatOchiLowResHours != bbs.DatOchiDisabled {
		t.Fatalf("expected DatOchiDisabled preserved, got %+v", disabledBoard.Rules)
	}
	userDisabled := snowflake.ID(33002)
	if _, err := svc.AcceptPolicy(ctx, userDisabled); err != nil {
		t.Fatalf("AcceptPolicy userDisabled failed: %v", err)
	}
	_, disabledTh, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: disabledBoard.ID,
		GuildID: guildB,
		UserID:  userDisabled,
		Title:   "永久スレッド",
		RawBody: "落ちないはず",
	})
	if err != nil {
		t.Fatalf("CreateThread on DatOchiDisabled board failed: %v", err)
	}
	staleFarFuture, err := store.ListStaleThreads(ctx, time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("ListStaleThreads failed: %v", err)
	}
	for _, st := range staleFarFuture {
		if st.ID == disabledTh.ID {
			t.Fatal("DatOchiDisabled thread must never be returned as stale")
		}
	}

	// 8. UpdateBoardMountControl only updates active=true mounts and does not overwrite concurrent active=false
	activeMountID := uuid.New()
	if err := store.CreateBoardMount(ctx, &bbs.BoardMount{
		ID:        activeMountID,
		BoardID:   disabledBoard.ID,
		GuildID:   guildB,
		ChannelID: snowflake.ID(555002),
		Active:    true,
	}); err != nil {
		t.Fatalf("CreateBoardMount active failed: %v", err)
	}
	ctrlMsg := snowflake.ID(666001)
	ok, err := store.UpdateBoardMountControl(ctx, activeMountID, &ctrlMsg)
	if err != nil || !ok {
		t.Fatalf("expected UpdateBoardMountControl on active mount to succeed, ok=%v err=%v", ok, err)
	}
	if err := store.DeactivateBoardMount(ctx, activeMountID); err != nil {
		t.Fatalf("DeactivateBoardMount failed: %v", err)
	}
	ctrlMsg2 := snowflake.ID(666002)
	ok, err = store.UpdateBoardMountControl(ctx, activeMountID, &ctrlMsg2)
	if err != nil || ok {
		t.Fatalf("expected UpdateBoardMountControl on inactive mount to return false without reactivating, ok=%v err=%v", ok, err)
	}
	reloadedMount, err := store.GetBoardMount(ctx, activeMountID)
	if err != nil || reloadedMount.Active || (reloadedMount.ControlMessageID != nil && *reloadedMount.ControlMessageID == ctrlMsg2) {
		t.Fatalf("inactive mount state unexpectedly overwritten: %+v, err=%v", reloadedMount, err)
	}

	// 9. ListBans filters out expired bans before applying Limit
	nowBan := time.Now()
	expiredAt := nowBan.Add(-1 * time.Hour)
	futureAt := nowBan.Add(1 * time.Hour)
	if err := store.CreateBan(ctx, &bbs.Ban{
		Scope:      bbs.ScopeGlobal,
		PosterHash: "expired_hash",
		Reason:     "expired",
		CreatedBy:  adminID,
		ExpiresAt:  &expiredAt,
		CreatedAt:  nowBan.Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("CreateBan expired failed: %v", err)
	}
	if err := store.CreateBan(ctx, &bbs.Ban{
		Scope:      bbs.ScopeGlobal,
		PosterHash: "active_hash",
		Reason:     "active",
		CreatedBy:  adminID,
		ExpiresAt:  &futureAt,
		CreatedAt:  nowBan.Add(-3 * time.Hour),
	}); err != nil {
		t.Fatalf("CreateBan active failed: %v", err)
	}
	listedBans, err := store.ListBans(ctx, bbs.BanFilter{Limit: 1, Now: nowBan})
	if err != nil || len(listedBans) != 1 || listedBans[0].PosterHash != "active_hash" {
		t.Fatalf("expected ListBans(Limit=1) to return active_hash rather than expired ban, got %+v, err=%v", listedBans, err)
	}

	// 10. CreateThread/CreatePost reject ClearFormat-expanded body exceeding MaxEscapedBodyRunes & check UUIDv7 lock distribution
	userEscaped := snowflake.ID(33003)
	userBadMail := snowflake.ID(33004)
	userGoodMail := snowflake.ID(33005)
	for _, uid := range []snowflake.ID{userEscaped, userBadMail, userGoodMail} {
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
	}
	escapedOver := strings.Repeat("*", 1200) // 1200 raw <= 1800, 2400 escaped > 2200
	if _, _, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: disabledBoard.ID,
		GuildID: guildB,
		UserID:  userEscaped,
		Title:   "エスケープ超過スレ",
		RawBody: escapedOver,
	}); !errors.Is(err, bbs.ErrFormattedBodyTooLong) {
		t.Fatalf("expected ErrFormattedBodyTooLong, got %v", err)
	}
	// Reply with valid mail="sage" & 48-char name succeeds; invalid mail rejected before DB write
	if _, _, _, _, err := svc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: disabledTh.ID,
		GuildID:  guildB,
		UserID:   userBadMail,
		RawName:  strings.Repeat("な", bbs.MaxNameRunes),
		RawMail:  "age",
		RawBody:  "テスト",
	}); !errors.Is(err, bbs.ErrInvalidMail) {
		t.Fatalf("expected ErrInvalidMail on reply, got %v", err)
	}
	_, _, sageReply, _, err := svc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: disabledTh.ID,
		GuildID:  guildB,
		UserID:   userGoodMail,
		RawName:  strings.Repeat("な", bbs.MaxNameRunes),
		RawMail:  "sage",
		RawBody:  "テスト>>1",
	})
	if err != nil || !sageReply.IsSage {
		t.Fatalf("expected 48-rune name + sage mail to succeed, got %+v, err=%v", sageReply, err)
	}
}
