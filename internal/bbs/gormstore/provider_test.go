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

	// Unconditional bump check: replying always updates BumpedAt even at MaxRes.
	updatedTh, err := svc.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}
	prevBump := updatedTh.BumpedAt
	time.Sleep(10 * time.Millisecond)

	// 12th post hits MaxRes=12 -> archives thread and updates BumpedAt.
	finalPost, archivedTh, err := store.CreatePost(ctx, bbs.CreatePostRecord{
		ThreadID:     thread.ID,
		Name:         "風吹けば名無し",
		AnonID:       "anonfinal",
		PosterHash:   "finalhash",
		Body:         "12到達で自動アーカイブ",
		AuthorUserID: userB,
	})
	if err != nil {
		t.Fatalf("12th CreatePost failed: %v", err)
	}
	if finalPost.ResNumber != 12 {
		t.Fatalf("expected res 12, got %d", finalPost.ResNumber)
	}
	if archivedTh.Status != bbs.ThreadStatusArchived {
		t.Fatalf("expected archived status at MaxRes=12, got %s", archivedTh.Status)
	}
	if !archivedTh.BumpedAt.After(prevBump) {
		t.Fatalf("reply must update BumpedAt: prev=%v, now=%v", prevBump, archivedTh.BumpedAt)
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

	// 10. CreateThread/CreatePost reject ClearFormat-expanded body exceeding MaxEscapedBodyRunes & preserve MaxNameRunes
	userEscaped := snowflake.ID(33003)
	userNameTest := snowflake.ID(33005)
	for _, uid := range []snowflake.ID{userEscaped, userNameTest} {
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
	_, _, longReply, _, err := svc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: disabledTh.ID,
		GuildID:  guildB,
		UserID:   userNameTest,
		RawName:  strings.Repeat("な", bbs.MaxNameRunes),
		RawBody:  "テスト>>1",
	})
	if err != nil || longReply.Name != strings.Repeat("な", bbs.MaxNameRunes) {
		t.Fatalf("expected 48-rune name reply to succeed, got %+v, err=%v", longReply, err)
	}

	// 11. ClearFormat-empty body/title (e.g. U+EFFD/U+EFFF only) rejected before DB write
	for i, emptyBody := range []string{"\uEFFD", "\uEFFF", "\uEFFD \uEFFF"} {
		uid := snowflake.ID(33010 + i)
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
		if _, _, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
			BoardID: disabledBoard.ID,
			GuildID: guildB,
			UserID:  uid,
			Title:   "有効なスレタイ",
			RawBody: emptyBody,
		}); !errors.Is(err, bbs.ErrInvalidBodyLength) {
			t.Fatalf("expected ErrInvalidBodyLength on CreateThread(%q), got %v", emptyBody, err)
		}
		if _, _, _, _, err := svc.CreatePost(ctx, bbs.CreatePostInput{
			ThreadID: disabledTh.ID,
			GuildID:  guildB,
			UserID:   uid,
			RawBody:  emptyBody,
		}); !errors.Is(err, bbs.ErrInvalidBodyLength) {
			t.Fatalf("expected ErrInvalidBodyLength on CreatePost(%q), got %v", emptyBody, err)
		}
	}
	if _, _, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: disabledBoard.ID,
		GuildID: guildB,
		UserID:  snowflake.ID(33010),
		Title:   "\uEFFD",
		RawBody: "本文あり",
	}); !errors.Is(err, bbs.ErrInvalidTitleLength) {
		t.Fatalf("expected ErrInvalidTitleLength on U+EFFD-only title, got %v", err)
	}

	// 12. GetBan and Report.BanID round-trip persistence
	banRow := &bbs.Ban{
		Scope:      bbs.ScopeGlobal,
		PosterHash: "rep_ban_hash",
		Reason:     "spam",
		CreatedBy:  adminID,
		CreatedAt:  time.Now(),
	}
	if err := store.CreateBan(ctx, banRow); err != nil {
		t.Fatalf("CreateBan for report failed: %v", err)
	}
	gotBan, err := store.GetBan(ctx, banRow.ID)
	if err != nil || gotBan.ID != banRow.ID {
		t.Fatalf("GetBan failed: %+v, err=%v", gotBan, err)
	}
	rep, err := svc.CreateReport(ctx, guildB, userNameTest, longReply.ID, "spam report")
	if err != nil {
		t.Fatalf("CreateReport failed: %v", err)
	}
	rep.BanID = &banRow.ID
	if err := store.UpdateReport(ctx, rep); err != nil {
		t.Fatalf("UpdateReport BanID failed: %v", err)
	}
	gotRep, err := store.GetReport(ctx, rep.ID)
	if err != nil || gotRep.BanID == nil || *gotRep.BanID != banRow.ID {
		t.Fatalf("expected persisted BanID %v on report, got %+v, err=%v", banRow.ID, gotRep, err)
	}
}

func TestServiceCapPersistenceAndSanitization(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	store := gormstore.New(db)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	idGen := bbs.NewIdentityGenerator([]byte("cap-test-hmac-secret"))
	pm := bbs.NewPolicyManager(bbs.PolicyConfig{Version: "v1"})

	opUser := snowflake.ID(70001)
	capUser := snowflake.ID(70002)
	normalUser := snowflake.ID(70003)
	guildID := snowflake.ID(88801)

	svc := bbs.NewService(reg, store, idGen, pm, []snowflake.ID{opUser})
	if err := svc.SetAdminCaps([]bbs.CapDefinition{
		{ID: "admin", Name: "管理人", Description: "運営", GrantOperators: true, Users: []snowflake.ID{capUser}},
		{ID: "mod", Name: "監視員", GrantOperators: false, Users: []snowflake.ID{capUser}},
	}); err != nil {
		t.Fatalf("SetAdminCaps failed: %v", err)
	}

	if _, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: opUser}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	for _, uid := range []snowflake.ID{opUser, capUser, normalUser} {
		if _, err := svc.AcceptPolicy(ctx, uid); err != nil {
			t.Fatalf("AcceptPolicy %d failed: %v", uid, err)
		}
	}

	cat, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "capcat", Name: "Cap Category",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeLocal,
		GuildID:    &guildID,
		Slug:       "capboard",
		Name:       "Cap Board",
		Rules:      bbs.BoardRules{DefaultName: "偽◆名★子", HideID: false, SLIP: bbs.SLIPVVVVV},
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	if board.Rules.DefaultName != "偽◇名☆子" {
		t.Fatalf("expected CreateBoard to sanitize DefaultName, got %q", board.Rules.DefaultName)
	}
	updatedName := "更新◆板★名"
	board, err = svc.UpdateBoard(ctx, board.ID, bbs.UpdateBoardInput{DefaultName: &updatedName})
	if err != nil || board.Rules.DefaultName != "更新◇板☆名" {
		t.Fatalf("expected UpdateBoard to sanitize DefaultName, got %+v, err=%v", board, err)
	}

	// ListAvailableCaps authorization checks
	if got := svc.ListAvailableCaps(ctx, opUser, board.ID); len(got) != 1 || got[0].ID != "admin" {
		t.Fatalf("expected operator caps [admin], got %+v", got)
	}
	if got := svc.ListAvailableCaps(ctx, capUser, board.ID); len(got) != 2 || got[0].ID != "admin" || got[1].ID != "mod" {
		t.Fatalf("expected capUser caps [admin, mod], got %+v", got)
	}
	if got := svc.ListAvailableCaps(ctx, normalUser, board.ID); len(got) != 0 {
		t.Fatalf("expected normalUser to have 0 caps, got %+v", got)
	}

	// Unauthorized / unknown CapID rejected with ErrCapNotAuthorized
	for _, badCap := range []string{"admin", "unknown_cap"} {
		if _, _, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
			BoardID: board.ID, GuildID: guildID, UserID: normalUser,
			Title: "不正キャップ", CapID: badCap, RawBody: "テスト",
		}); !errors.Is(err, bbs.ErrCapNotAuthorized) {
			t.Fatalf("expected ErrCapNotAuthorized on CreateThread(cap=%q), got %v", badCap, err)
		}
	}

	// Authorized Cap: persists Post.Cap, hides AnonID & SlipText, and bypasses thread & post rate limits
	_, th1, p1, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID, GuildID: guildID, UserID: opUser,
		Title: "運営告知スレ1", RawName: "", CapID: "admin", RawBody: "第一告知",
	})
	if err != nil {
		t.Fatalf("CreateThread #1 with cap failed: %v", err)
	}
	if p1.Cap != "管理人" || p1.Name != "" || p1.AnonID != "" || p1.SlipText != "" {
		t.Fatalf("unexpected cap first post: %+v", p1)
	}
	// Immediate 2nd CreateThread succeeds (rate limit bypassed)
	_, _, p2, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID, GuildID: guildID, UserID: opUser,
		Title: "運営告知スレ2", RawName: "運営担当 !id:on", CapID: "admin", RawBody: "連続告知",
	})
	if err != nil {
		t.Fatalf("rapid CreateThread #2 with cap must bypass rate limit: %v", err)
	}
	if p2.Cap != "管理人" || p2.Name != "運営担当" || p2.AnonID != "" || p2.SlipText != "" {
		t.Fatalf("unexpected second cap post: %+v", p2)
	}

	// Rapid replies with cap also persist Cap, hide ID, and bypass post rate limit
	for i := range 3 {
		_, _, reply, _, err := svc.CreatePost(ctx, bbs.CreatePostInput{
			ThreadID: th1.ID, GuildID: guildID, UserID: capUser,
			RawName: "", CapID: "mod", RawBody: fmt.Sprintf("監視レス %d", i+1),
		})
		if err != nil {
			t.Fatalf("rapid CreatePost #%d with cap failed: %v", i, err)
		}
		if reply.Cap != "監視員" || reply.AnonID != "" || reply.SlipText != "" {
			t.Fatalf("unexpected cap reply: %+v", reply)
		}
	}

	// Unauthorized CapID on CreatePost rejected cleanly
	if _, _, _, _, err := svc.CreatePost(ctx, bbs.CreatePostInput{
		ThreadID: th1.ID, GuildID: guildID, UserID: normalUser,
		CapID: "mod", RawBody: "なりすまし",
	}); !errors.Is(err, bbs.ErrCapNotAuthorized) {
		t.Fatalf("expected ErrCapNotAuthorized on CreatePost, got %v", err)
	}

	// Reload from DB to verify Cap field round-trips through GORM
	persisted, err := svc.ListPosts(ctx, th1.ID, 0, 10)
	if err != nil || len(persisted) != 4 || persisted[0].Cap != "管理人" || persisted[1].Cap != "監視員" {
		t.Fatalf("expected persisted Cap fields from DB, got %+v, err=%v", persisted, err)
	}
}

func TestUpdateBoard_SLIPAndPartialFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	if !db.Migrator().HasColumn(&gormstore.BBSBoard{}, "slip_level") {
		t.Fatal("expected bbs_boards table to have slip_level column")
	}
	if !db.Migrator().HasColumn(&gormstore.BBSThread{}, "slip_level") {
		t.Fatal("expected bbs_threads table to have slip_level column")
	}

	store := gormstore.New(db)
	reg := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	reg.Register(store)
	idGen := bbs.NewIdentityGenerator([]byte("slip-test-secret"))
	pm := bbs.NewPolicyManager(bbs.PolicyConfig{Version: "v1"})
	opUser := snowflake.ID(77001)
	guildID := snowflake.ID(77100)
	svc := bbs.NewService(reg, store, idGen, pm, []snowflake.ID{opUser})

	if _, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: opUser}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	cat, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "slipcat", Name: "SLIP Cat",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeLocal,
		GuildID:    &guildID,
		Slug:       "slipboard",
		Name:       "SLIP Board",
		Rules:      bbs.BoardRules{DefaultName: "初期名無し", HideID: true, SLIP: bbs.SLIPNone},
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	lockedTrue := true
	if _, err := svc.UpdateBoard(ctx, board.ID, bbs.UpdateBoardInput{Locked: &lockedTrue}); err != nil {
		t.Fatalf("lock board failed: %v", err)
	}

	// Update SLIP from None -> VVV -> VVVVV -> None while preserving DefaultName, HideID=true, Locked=true.
	slipVVV := bbs.SLIPVVV
	updatedVVV, err := svc.UpdateBoard(ctx, board.ID, bbs.UpdateBoardInput{SLIP: &slipVVV})
	if err != nil || updatedVVV.Rules.SLIP != bbs.SLIPVVV {
		t.Fatalf("UpdateBoard SLIP=vvv failed: %+v, err=%v", updatedVVV, err)
	}
	if reloadedVVV, err := svc.GetBoard(ctx, board.ID); err != nil || reloadedVVV.Rules.SLIP != bbs.SLIPVVV {
		t.Fatalf("expected persisted SLIP=vvv after reload, got %+v, err=%v", reloadedVVV, err)
	}

	slipOn := bbs.SLIPVVVVV
	updatedOn, err := svc.UpdateBoard(ctx, board.ID, bbs.UpdateBoardInput{SLIP: &slipOn})
	if err != nil {
		t.Fatalf("UpdateBoard SLIP=vvvvv failed: %v", err)
	}
	if updatedOn.Rules.SLIP != bbs.SLIPVVVVV || !updatedOn.Locked || !updatedOn.Rules.HideID || updatedOn.Rules.DefaultName != "初期名無し" {
		t.Fatalf("unexpected board after SLIP=vvvvv: %+v", updatedOn)
	}
	reloadedOn, err := svc.GetBoard(ctx, board.ID)
	if err != nil || reloadedOn.Rules.SLIP != bbs.SLIPVVVVV || !reloadedOn.Locked || !reloadedOn.Rules.HideID || reloadedOn.Rules.DefaultName != "初期名無し" {
		t.Fatalf("expected persisted SLIP=vvvvv after reload, got %+v, err=%v", reloadedOn, err)
	}

	// Update SLIP back to None (zero value must persist) and keep untouched fields intact.
	slipOff := bbs.SLIPNone
	if _, err := svc.UpdateBoard(ctx, board.ID, bbs.UpdateBoardInput{SLIP: &slipOff}); err != nil {
		t.Fatalf("UpdateBoard SLIP=none failed: %v", err)
	}
	reloadedOff, err := svc.GetBoard(ctx, board.ID)
	if err != nil || reloadedOff.Rules.SLIP != bbs.SLIPNone || !reloadedOff.Locked || !reloadedOff.Rules.HideID || reloadedOff.Rules.DefaultName != "初期名無し" {
		t.Fatalf("expected persisted SLIP=none after reload, got %+v, err=%v", reloadedOff, err)
	}

	// Unlock board and verify long event + nickname + MMMM-XXXX SLIP text persists cleanly in BBSPost.
	lockedFalse := false
	if _, err := svc.UpdateBoard(ctx, board.ID, bbs.UpdateBoardInput{Locked: &lockedFalse, SLIP: &slipOn}); err != nil {
		t.Fatalf("unlock + SLIP=vvvvv failed: %v", err)
	}
	if _, err := svc.AcceptPolicy(ctx, opUser); err != nil {
		t.Fatalf("AcceptPolicy opUser failed: %v", err)
	}
	if err := svc.SetSlipEvents([]bbs.SlipEventDefinition{
		{ID: "long_evt", Label: strings.Repeat("ｲﾍﾞﾝﾄ", 4), StartDate: "01-01", Mode: bbs.SlipEventModeAppend},
	}); err != nil {
		t.Fatalf("SetSlipEvents failed: %v", err)
	}
	_, thSlip, firstSlipPost, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID,
		GuildID: guildID,
		UserID:  opUser,
		Title:   "SLIP永続化テスト",
		RawName: "!id:on",
		RawBody: "本文",
		SlipContext: bbs.SlipContext{
			MemberNick:     "テストニック",
			IsGuildBooster: true,
		},
		Now: time.Date(2026, 1, 1, 12, 0, 0, 0, bbs.JSTLocation),
	})
	if err != nil {
		t.Fatalf("CreateThread with event SLIP failed: %v", err)
	}
	listPosts, err := svc.ListPosts(ctx, thSlip.ID, 0, 10)
	if err != nil || len(listPosts) != 1 || listPosts[0].SlipText == "" || listPosts[0].SlipText != firstSlipPost.SlipText {
		t.Fatalf("expected persisted SlipText %q, got %+v, err=%v", firstSlipPost.SlipText, listPosts, err)
	}
}

func TestAdminCapsAndSlipEvents_SeedPersistenceAndLiveReload(t *testing.T) {
	store, svc := openTestService(t)
	ctx := context.Background()

	opUser := snowflake.ID(9999)
	userA := snowflake.ID(7001)
	userB := snowflake.ID(7002)

	// 1. Empty DB seeds from YAML and reflects immediately in memory.
	initialCaps := []bbs.CapDefinition{
		{ID: "Admin", Name: "管理人", Description: "公式", GrantOperators: true, Users: []snowflake.ID{userA}},
	}
	initialSlips := []bbs.SlipEventDefinition{
		{ID: "sakura", Label: "ｻｸﾗ", StartDate: "04-01", EndDate: "04-05", Mode: bbs.SlipEventModeOverride},
	}
	if err := svc.LoadOrSeedAdminCaps(ctx, initialCaps); err != nil {
		t.Fatalf("LoadOrSeedAdminCaps initial failed: %v", err)
	}
	if err := svc.LoadOrSeedSlipEvents(ctx, initialSlips); err != nil {
		t.Fatalf("LoadOrSeedSlipEvents initial failed: %v", err)
	}
	if caps := svc.ListAvailableCaps(ctx, opUser, uuid.Nil); len(caps) != 1 || caps[0].ID != "Admin" {
		t.Fatalf("expected operator cap [Admin], got %+v", caps)
	}
	aprDay := time.Date(2026, 4, 2, 12, 0, 0, 0, bbs.JSTLocation)
	if slip := svc.Identity.GenerateSlip(bbs.SLIPVVV, uuid.New(), bbs.ScopeGlobal, 0, userA, bbs.SlipContext{}, aprDay); slip != "ｻｸﾗ" {
		t.Fatalf("expected seeded slip event ｻｸﾗ, got %q", slip)
	}

	// 2. Subsequent LoadOrSeed with different YAML ignores YAML because DB is already authoritative.
	if err := svc.LoadOrSeedAdminCaps(ctx, []bbs.CapDefinition{
		{ID: "other", Name: "別キャップ", Users: []snowflake.ID{userB}},
	}); err != nil {
		t.Fatalf("LoadOrSeedAdminCaps restart failed: %v", err)
	}
	if list, err := svc.ListAdminCaps(ctx); err != nil || len(list) != 1 || list[0].ID != "Admin" {
		t.Fatalf("expected DB cap [Admin] to win over YAML on restart, got %+v, err=%v", list, err)
	}

	// 3. Invalid YAML fails closed even when DB is seeded.
	if err := svc.LoadOrSeedAdminCaps(ctx, []bbs.CapDefinition{{ID: bbs.CapSelectNoneValue, Name: "bad"}}); !errors.Is(err, bbs.ErrInvalidCapConfig) {
		t.Fatalf("expected ErrInvalidCapConfig on invalid YAML, got %v", err)
	}
	if err := svc.LoadOrSeedSlipEvents(ctx, []bbs.SlipEventDefinition{{ID: "bad", Label: "x", StartDate: "99-99"}}); !errors.Is(err, bbs.ErrInvalidSlipConfig) {
		t.Fatalf("expected ErrInvalidSlipConfig on invalid YAML, got %v", err)
	}

	// 4. Create / Update (case-insensitive) / ErrAlreadyExists / ErrNotFound / Remove.
	if _, err := svc.CreateAdminCap(ctx, bbs.CapDefinition{ID: "admin", Name: "dup"}); !errors.Is(err, bbs.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists on duplicate cap ID, got %v", err)
	}
	createdCap, err := svc.CreateAdminCap(ctx, bbs.CapDefinition{
		ID:             "notice",
		Name:           "お知らせ",
		Description:    "告知",
		GrantOperators: false,
		Users:          []snowflake.ID{userB},
	})
	if err != nil || createdCap.ID != "notice" {
		t.Fatalf("CreateAdminCap failed: cap=%+v err=%v", createdCap, err)
	}
	if caps := svc.ListAvailableCaps(ctx, userB, uuid.Nil); len(caps) != 1 || caps[0].ID != "notice" {
		t.Fatalf("expected live userB cap [notice], got %+v", caps)
	}
	updatedCap, err := svc.UpdateAdminCap(ctx, "NOTICE", bbs.CapDefinition{
		Name:  "お知らせ改",
		Users: []snowflake.ID{userA, userB},
	})
	if err != nil || updatedCap.ID != "notice" || updatedCap.Name != "お知らせ改" {
		t.Fatalf("UpdateAdminCap case-insensitive failed: cap=%+v err=%v", updatedCap, err)
	}
	if _, err := svc.CreateSlipEvent(ctx, bbs.SlipEventDefinition{
		ID:        "exact_day",
		Label:     "ｷﾈﾝﾋﾞ",
		StartDate: "2026-08-15",
		EndDate:   "2026-08-16",
		Mode:      bbs.SlipEventModeAppend,
	}); err != nil {
		t.Fatalf("CreateSlipEvent YYYY-MM-DD failed: %v", err)
	}
	augDay := time.Date(2026, 8, 15, 12, 0, 0, 0, bbs.JSTLocation)
	if got := svc.Identity.GenerateSlip(bbs.SLIPVVV, uuid.New(), bbs.ScopeGlobal, 0, userA, bbs.SlipContext{}, augDay); !strings.HasPrefix(got, "ｷﾈﾝﾋﾞ ") {
		t.Fatalf("expected YYYY-MM-DD append slip event ｷﾈﾝﾋﾞ ..., got %q", got)
	}
	if _, err := svc.UpdateSlipEvent(ctx, "EXACT_DAY", bbs.SlipEventDefinition{
		Label:     "ﾊﾁｶﾞﾂ",
		StartDate: "2026-08-15",
		EndDate:   "2026-08-16",
		Mode:      bbs.SlipEventModeOverride,
	}); err != nil {
		t.Fatalf("UpdateSlipEvent failed: %v", err)
	}
	if got := svc.Identity.GenerateSlip(bbs.SLIPVVV, uuid.New(), bbs.ScopeGlobal, 0, userA, bbs.SlipContext{}, augDay); got != "ﾊﾁｶﾞﾂ" {
		t.Fatalf("expected updated override slip event ﾊﾁｶﾞﾂ, got %q", got)
	}

	// 5. Concurrent reads and writes under -race.
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for range 10 {
				_ = svc.ListAvailableCaps(ctx, userA, uuid.Nil)
				_ = svc.Identity.GenerateSlip(bbs.SLIPVVV, uuid.New(), bbs.ScopeGlobal, 0, userA, bbs.SlipContext{}, augDay)
				if n%2 == 0 {
					_, _ = store.GetBBSStats(ctx)
				}
			}
		}(i)
	}
	for i := range 4 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = svc.UpdateAdminCap(ctx, "notice", bbs.CapDefinition{
				Name:  fmt.Sprintf("お知らせ%d", n),
				Users: []snowflake.ID{userA, userB},
			})
		}(i)
	}
	wg.Wait()

	// 6. Deleting all caps/events marks seeded, so YAML will NOT resurrect them on restart.
	if err := svc.RemoveAdminCap(ctx, "ADMIN"); err != nil {
		t.Fatalf("RemoveAdminCap ADMIN failed: %v", err)
	}
	if err := svc.RemoveAdminCap(ctx, "notice"); err != nil {
		t.Fatalf("RemoveAdminCap notice failed: %v", err)
	}
	if err := svc.RemoveAdminCap(ctx, "missing"); !errors.Is(err, bbs.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on missing cap delete, got %v", err)
	}
	if err := svc.RemoveSlipEvent(ctx, "sakura"); err != nil {
		t.Fatalf("RemoveSlipEvent sakura failed: %v", err)
	}
	if err := svc.RemoveSlipEvent(ctx, "EXACT_DAY"); err != nil {
		t.Fatalf("RemoveSlipEvent exact_day failed: %v", err)
	}
	if err := svc.RemoveSlipEvent(ctx, "missing"); !errors.Is(err, bbs.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on missing slip delete, got %v", err)
	}

	if err := svc.LoadOrSeedAdminCaps(ctx, initialCaps); err != nil {
		t.Fatalf("LoadOrSeedAdminCaps after delete-all failed: %v", err)
	}
	if caps, err := svc.ListAdminCaps(ctx); err != nil || len(caps) != 0 {
		t.Fatalf("expected deleted caps to stay empty after restart, got %+v, err=%v", caps, err)
	}
	if err := svc.LoadOrSeedSlipEvents(ctx, initialSlips); err != nil {
		t.Fatalf("LoadOrSeedSlipEvents after delete-all failed: %v", err)
	}
	if evs, err := svc.ListSlipEvents(ctx); err != nil || len(evs) != 0 {
		t.Fatalf("expected deleted slip events to stay empty after restart, got %+v, err=%v", evs, err)
	}
}

func TestAdminQueries_UpdateCategory_ThreadSearch_Mounts_Stats(t *testing.T) {
	store, svc := openTestService(t)
	ctx := context.Background()
	guildID := snowflake.ID(555001)
	userID := snowflake.ID(666001)
	if _, err := svc.SetupGuild(ctx, bbs.GuildSetupInput{GuildID: guildID, Enabled: true, ActorUserID: userID}); err != nil {
		t.Fatalf("SetupGuild failed: %v", err)
	}
	if _, err := svc.AcceptPolicy(ctx, userID); err != nil {
		t.Fatalf("AcceptPolicy failed: %v", err)
	}
	user2 := snowflake.ID(666002)
	if _, err := svc.AcceptPolicy(ctx, user2); err != nil {
		t.Fatalf("AcceptPolicy user2 failed: %v", err)
	}

	cat, err := svc.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "old", Name: "OldCat", SortOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	newName := "UpdatedCat"
	newSlug := "newcat"
	newDesc := "Updated Desc"
	newSort := 7
	updatedCat, err := svc.UpdateCategory(ctx, cat.ID, bbs.UpdateCategoryInput{
		Name: &newName, Slug: &newSlug, Description: &newDesc, SortOrder: &newSort,
	})
	if err != nil || updatedCat.Name != "UpdatedCat" || updatedCat.Slug != "newcat" || updatedCat.SortOrder != 7 {
		t.Fatalf("UpdateCategory failed: %+v, err=%v", updatedCat, err)
	}

	board, err := svc.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID, Scope: bbs.ScopeLocal, GuildID: &guildID, Slug: "misc", Name: "MiscBoard",
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}

	if err := store.CreateBoardMount(ctx, &bbs.BoardMount{
		BoardID: board.ID, GuildID: guildID, ChannelID: snowflake.ID(777001), Locale: "ja", Active: true,
	}); err != nil {
		t.Fatalf("CreateBoardMount 1 failed: %v", err)
	}
	inactive := &bbs.BoardMount{
		BoardID: board.ID, GuildID: guildID, ChannelID: snowflake.ID(777002), Locale: "en-US", Active: false,
	}
	if err := store.CreateBoardMount(ctx, inactive); err != nil {
		t.Fatalf("CreateBoardMount 2 failed: %v", err)
	}
	if err := store.DeactivateBoardMount(ctx, inactive.ID); err != nil {
		t.Fatalf("DeactivateBoardMount failed: %v", err)
	}
	onlyActive := true
	activeMounts, err := svc.ListBoardMounts(ctx, bbs.BoardMountFilter{BoardID: &board.ID, Active: &onlyActive})
	if err != nil || len(activeMounts) != 1 || activeMounts[0].ChannelID != snowflake.ID(777001) {
		t.Fatalf("expected 1 active mount, got %+v, err=%v", activeMounts, err)
	}

	_, thWild, p1, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID, GuildID: guildID, UserID: userID, Title: "100%テスト_検索スレ", RawBody: "OP body",
		Now: time.Now().Add(-5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateThread 1 failed: %v", err)
	}
	_, thOther, _, err := svc.CreateThread(ctx, bbs.CreateThreadInput{
		BoardID: board.ID, GuildID: guildID, UserID: user2, Title: "100XテストY他スレ", RawBody: "Other",
		Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateThread 2 failed: %v", err)
	}
	if _, _, err := svc.ArchiveThreadByAdmin(ctx, thOther.ID); err != nil {
		t.Fatalf("ArchiveThreadByAdmin failed: %v", err)
	}

	// Literal % and _ in Query must NOT match thOther ("100XテストY...").
	matched, err := svc.ListThreads(ctx, bbs.ThreadFilter{Query: "%テスト_"})
	if err != nil || len(matched) != 1 || matched[0].ID != thWild.ID {
		t.Fatalf("expected literal LIKE match on thWild only, got %+v, err=%v", matched, err)
	}

	reporterID := snowflake.ID(888001)
	if _, err := svc.AcceptPolicy(ctx, reporterID); err != nil {
		t.Fatalf("AcceptPolicy reporter failed: %v", err)
	}
	rep, err := svc.CreateReport(ctx, guildID, reporterID, p1.ID, "spam link")
	if err != nil {
		t.Fatalf("SubmitReport failed: %v", err)
	}
	gotRep, gotAudit, err := svc.InspectReportForAdmin(ctx, rep.ID)
	if err != nil || gotRep.ID != rep.ID || gotAudit == nil || gotAudit.AuthorUserID != userID {
		t.Fatalf("InspectReportForAdmin failed: rep=%+v audit=%+v err=%v", gotRep, gotAudit, err)
	}

	ban, err := svc.CreateBanByAdmin(ctx, bbs.AdminBanInput{
		Scope: bbs.ScopeLocal, PostID: &p1.ID, Reason: "test ban", DurationHours: 2, ActorID: snowflake.ID(9999),
	})
	if err != nil || ban.GuildID == nil || *ban.GuildID != guildID || ban.TargetUserID == nil || *ban.TargetUserID != userID {
		t.Fatalf("CreateBanByAdmin with PostID fallback failed: %+v, err=%v", ban, err)
	}
	past := time.Now().Add(-1 * time.Hour)
	if err := store.CreateBan(ctx, &bbs.Ban{
		Scope: bbs.ScopeGlobal, PosterHash: "expiredhash", CreatedBy: snowflake.ID(9999), ExpiresAt: &past,
	}); err != nil {
		t.Fatalf("CreateBan expired failed: %v", err)
	}

	st, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if st.Categories != 1 || st.Boards != 1 || st.ActiveThreads != 1 || st.OpenReports != 1 || st.ActiveBans != 1 {
		t.Fatalf("unexpected BBSStats: %+v", st)
	}

	if _, err := svc.ResolveReportForAdmin(ctx, rep.ID, bbs.ReportStatusResolved, snowflake.ID(9999), "handled"); err != nil {
		t.Fatalf("ResolveReportForAdmin failed: %v", err)
	}
	aboned, thAfter, err := svc.AbonePostByAdmin(ctx, p1.ID, "removed")
	if err != nil || !aboned.Abone || thAfter.ID != thWild.ID {
		t.Fatalf("AbonePostByAdmin failed: post=%+v err=%v", aboned, err)
	}
	if err := svc.DeleteBan(ctx, ban.ID); err != nil {
		t.Fatalf("DeleteBan failed: %v", err)
	}
}
