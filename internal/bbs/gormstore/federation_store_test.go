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
	"slices"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
)

func openFederationStoreTestDB(t *testing.T) (*gorm.DB, *gormstore.Provider) {
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
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable sqlite foreign keys failed: %v", err)
	}
	if err := db.AutoMigrate(gormstore.Models()...); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	if err := gormstore.DropCrossProviderForeignKeys(db); err != nil {
		t.Fatalf("DropCrossProviderForeignKeys failed: %v", err)
	}
	return db, gormstore.New(db)
}

func TestFederationStore_RemoteUUIDsWithoutNativeFKs(t *testing.T) {
	ctx := context.Background()
	db, store := openFederationStoreTestDB(t)

	// Re-running DropCrossProviderForeignKeys must be strictly idempotent.
	if err := gormstore.DropCrossProviderForeignKeys(db); err != nil {
		t.Fatalf("second DropCrossProviderForeignKeys failed: %v", err)
	}

	remoteBoardID := uuid.New()
	remoteThreadID := uuid.New()
	remotePost1ID := uuid.New()
	remotePost2ID := uuid.New()
	guildID := snowflake.ID(10001)
	channelID := snowflake.ID(20001)
	discordThreadID := snowflake.ID(30001)
	msg1ID := snowflake.ID(40001)
	msg2ID := snowflake.ID(40002)
	watcherID := snowflake.ID(50001)
	reporterID := snowflake.ID(50002)

	mount := &bbs.BoardMount{
		ID:          uuid.New(),
		BoardID:     remoteBoardID,
		ProviderKey: "remote:prod",
		GuildID:     guildID,
		ChannelID:   channelID,
		Locale:      "ja",
		IsForum:     true,
		Active:      true,
	}
	if err := store.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount with remote board UUID failed: %v", err)
	}
	gotMount, err := store.GetBoardMount(ctx, mount.ID)
	if err != nil {
		t.Fatalf("GetBoardMount failed: %v", err)
	}
	if gotMount.ProviderKey != "remote:prod" || gotMount.BoardID != remoteBoardID {
		t.Fatalf("unexpected persisted mount: %+v", gotMount)
	}

	tm := &bbs.ThreadMirror{
		ID:              uuid.New(),
		ThreadID:        remoteThreadID,
		BoardMountID:    mount.ID,
		GuildID:         guildID,
		DiscordThreadID: discordThreadID,
		SyncedUpToRes:   1,
	}
	pm1 := &bbs.PostMirror{
		ID:               uuid.New(),
		PostID:           remotePost1ID,
		ThreadID:         remoteThreadID,
		ResNumber:        1,
		ThreadMirrorID:   tm.ID,
		GuildID:          guildID,
		DiscordMessageID: msg1ID,
	}
	if err := store.CreateThreadAndPostMirror(ctx, tm, pm1); err != nil {
		t.Fatalf("CreateThreadAndPostMirror with remote UUIDs failed: %v", err)
	}

	pm2 := &bbs.PostMirror{
		ID:               uuid.New(),
		PostID:           remotePost2ID,
		ThreadID:         remoteThreadID,
		ResNumber:        2,
		ThreadMirrorID:   tm.ID,
		GuildID:          guildID,
		DiscordMessageID: msg2ID,
	}
	if err := store.CreatePostMirror(ctx, pm2); err != nil {
		t.Fatalf("CreatePostMirror with remote UUIDs failed: %v", err)
	}

	mirrorsByMount, err := store.ListThreadMirrorsByMount(ctx, mount.ID)
	if err != nil || len(mirrorsByMount) != 1 || mirrorsByMount[0].ID != tm.ID {
		t.Fatalf("ListThreadMirrorsByMount failed: %+v, err=%v", mirrorsByMount, err)
	}

	byRes, err := store.GetPostMirrorByResAndGuild(ctx, remoteThreadID, 2, guildID)
	if err != nil {
		t.Fatalf("GetPostMirrorByResAndGuild on remote post failed: %v", err)
	}
	if byRes.PostID != remotePost2ID || byRes.DiscordMessageID != msg2ID {
		t.Fatalf("unexpected GetPostMirrorByResAndGuild result: %+v", byRes)
	}

	if err := store.SavePostReferences(ctx, []bbs.PostReference{
		{ThreadID: remoteThreadID, FromRes: 2, ToRes: 1},
	}); err != nil {
		t.Fatalf("SavePostReferences on remote thread failed: %v", err)
	}
	refs, err := store.ListReferencingRes(ctx, remoteThreadID, 1)
	if err != nil || !slices.Equal(refs, []uint32{2}) {
		t.Fatalf("expected [2] referencing res, got %v (err=%v)", refs, err)
	}

	if err := store.SaveThreadWatch(ctx, &bbs.ThreadWatch{
		ID:       uuid.New(),
		ThreadID: remoteThreadID,
		UserID:   watcherID,
		Locale:   "ja",
	}); err != nil {
		t.Fatalf("SaveThreadWatch on remote thread failed: %v", err)
	}
	watches, err := store.ListThreadWatches(ctx, remoteThreadID)
	if err != nil || len(watches) != 1 || watches[0].UserID != watcherID {
		t.Fatalf("unexpected ListThreadWatches: %+v, err=%v", watches, err)
	}

	rep := &bbs.Report{
		ID:             uuid.New(),
		Scope:          bbs.ScopeGlobal,
		GuildID:        guildID,
		PostID:         remotePost2ID,
		ThreadID:       remoteThreadID,
		BoardID:        remoteBoardID,
		ReporterUserID: reporterID,
		Reason:         "spam",
		Status:         bbs.ReportStatusOpen,
	}
	if err := store.CreateReport(ctx, rep); err != nil {
		t.Fatalf("CreateReport on remote post failed: %v", err)
	}
	gotRep, err := store.GetReport(ctx, rep.ID)
	if err != nil || gotRep.PostID != remotePost2ID {
		t.Fatalf("GetReport failed: %+v, err=%v", gotRep, err)
	}

	transitioned, err := store.MarkThreadMirrorArchivedSynced(ctx, tm.ID)
	if err != nil || !transitioned {
		t.Fatalf("expected first MarkThreadMirrorArchivedSynced=true, got %v, err=%v", transitioned, err)
	}
	transitionedAgain, err := store.MarkThreadMirrorArchivedSynced(ctx, tm.ID)
	if err != nil || transitionedAgain {
		t.Fatalf("expected second MarkThreadMirrorArchivedSynced=false, got %v, err=%v", transitionedAgain, err)
	}
}

func TestFederationStore_BackfillLegacyPostMirrors(t *testing.T) {
	ctx := context.Background()
	db, store := openFederationStoreTestDB(t)

	cat, err := store.CreateCategory(ctx, bbs.CreateCategoryInput{
		Scope: bbs.ScopeGlobal,
		Slug:  "cat",
		Name:  "Cat",
	})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	board, err := store.CreateBoard(ctx, bbs.CreateBoardInput{
		CategoryID: cat.ID,
		Scope:      bbs.ScopeGlobal,
		Slug:       "board",
		Name:       "Board",
		Rules:      bbs.BoardRules{MaxRes: 100}.Normalize(),
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	th, p1, err := store.CreateThread(ctx, bbs.CreateThreadRecord{
		BoardID: board.ID,
		Title:   "Backfill Thread",
		Rules:   board.Rules,
		FirstPost: bbs.CreatePostRecord{
			Name: "Name",
			Body: "First post",
		},
	})
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	guildID := snowflake.ID(60001)
	mount := &bbs.BoardMount{
		ID:        uuid.New(),
		BoardID:   board.ID,
		GuildID:   guildID,
		ChannelID: snowflake.ID(60002),
		Locale:    "ja",
		Active:    true,
	}
	if err := store.CreateBoardMount(ctx, mount); err != nil {
		t.Fatalf("CreateBoardMount failed: %v", err)
	}
	tm := &bbs.ThreadMirror{
		ID:              uuid.New(),
		ThreadID:        th.ID,
		BoardMountID:    mount.ID,
		GuildID:         guildID,
		DiscordThreadID: snowflake.ID(60003),
		SyncedUpToRes:   1,
	}
	if err := store.CreateThreadMirror(ctx, tm); err != nil {
		t.Fatalf("CreateThreadMirror failed: %v", err)
	}

	legacyRow := gormstore.BBSPostMirror{
		ID:               uuid.New(),
		PostID:           p1.ID,
		ThreadID:         uuid.Nil,
		ResNumber:        0,
		ThreadMirrorID:   tm.ID,
		GuildID:          guildID,
		DiscordMessageID: snowflake.ID(60004),
		CreatedAt:        time.Now().UTC(),
	}
	if err := db.Create(&legacyRow).Error; err != nil {
		t.Fatalf("insert legacy BBSPostMirror failed: %v", err)
	}

	// Run DropCrossProviderForeignKeys twice to verify backfill + idempotency.
	if err := gormstore.DropCrossProviderForeignKeys(db); err != nil {
		t.Fatalf("backfill run 1 failed: %v", err)
	}
	if err := gormstore.DropCrossProviderForeignKeys(db); err != nil {
		t.Fatalf("backfill run 2 failed: %v", err)
	}

	got, err := store.GetPostMirrorByResAndGuild(ctx, th.ID, p1.ResNumber, guildID)
	if err != nil {
		t.Fatalf("GetPostMirrorByResAndGuild after backfill failed: %v", err)
	}
	if got.PostID != p1.ID || got.ThreadID != th.ID || got.ResNumber != 1 {
		t.Fatalf("unexpected backfilled PostMirror: %+v", got)
	}
}

func TestFederationStore_OAuthGrantsRefreshRotationAndReuseRevocation(t *testing.T) {
	ctx := context.Background()
	_, store := openFederationStoreTestDB(t)

	now := time.Now().UTC()
	userID := snowflake.ID(70001)
	peerID := "dev-local"
	grant := &bbs.FederationGrant{
		PeerID:          peerID,
		UserID:          userID,
		Scopes:          []string{bbs.FederationScopeWrite, bbs.FederationScopeReport},
		PolicyVersion:   "2026-11-28",
		AccessTokenHash: "acc-hash-1",
		AccessExpiresAt: now.Add(time.Hour),
	}
	ref1 := &bbs.FederationRefreshToken{
		TokenHash: "ref-hash-1",
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := store.CreateFederationGrantWithRefresh(ctx, grant, ref1); err != nil {
		t.Fatalf("CreateFederationGrantWithRefresh failed: %v", err)
	}

	gotGrant, err := store.GetFederationGrantByAccessHash(ctx, "acc-hash-1")
	if err != nil || gotGrant.ID != grant.ID || !gotGrant.HasScope(bbs.FederationScopeWrite) {
		t.Fatalf("unexpected GetFederationGrantByAccessHash: %+v, err=%v", gotGrant, err)
	}

	active, err := store.ListActiveFederationGrantsByUser(ctx, userID, now)
	if err != nil || len(active) != 1 {
		t.Fatalf("expected 1 active grant, got %+v, err=%v", active, err)
	}

	// Normal single-use rotation.
	ref2 := &bbs.FederationRefreshToken{
		TokenHash: "ref-hash-2",
		ExpiresAt: now.Add(48 * time.Hour),
	}
	rotatedGrant, reused, err := store.RotateFederationRefreshToken(ctx, peerID, "ref-hash-1", "acc-hash-2", now.Add(2*time.Hour), ref2, now.Add(time.Minute))
	if err != nil || reused || rotatedGrant == nil || rotatedGrant.AccessTokenHash != "acc-hash-2" {
		t.Fatalf("expected clean refresh rotation, got grant=%+v reused=%v err=%v", rotatedGrant, reused, err)
	}

	if _, err := store.GetFederationGrantByAccessHash(ctx, "acc-hash-1"); !errors.Is(err, bbs.ErrNotFound) {
		t.Fatalf("expected old access hash to no longer resolve, got err=%v", err)
	}
	if _, err := store.GetFederationGrantByAccessHash(ctx, "acc-hash-2"); err != nil {
		t.Fatalf("expected new access hash to resolve, got err=%v", err)
	}

	// Refresh token reuse attack: replaying ref-hash-1 must revoke the entire family!
	ref3 := &bbs.FederationRefreshToken{
		TokenHash: "ref-hash-3",
		ExpiresAt: now.Add(48 * time.Hour),
	}
	_, reusedAttack, attackErr := store.RotateFederationRefreshToken(ctx, peerID, "ref-hash-1", "acc-hash-3", now.Add(2*time.Hour), ref3, now.Add(2*time.Minute))
	if !errors.Is(attackErr, bbs.ErrUnauthorized) || !reusedAttack {
		t.Fatalf("expected reuse detection with ErrUnauthorized, got reused=%v err=%v", reusedAttack, attackErr)
	}

	revokedGrant, err := store.GetFederationGrant(ctx, grant.ID)
	if err != nil || revokedGrant.RevokedAt == nil {
		t.Fatalf("expected grant to be revoked after reuse attack, got %+v, err=%v", revokedGrant, err)
	}

	// Previously minted ref-hash-2 in the same family must also now be unusable.
	ref4 := &bbs.FederationRefreshToken{
		TokenHash: "ref-hash-4",
		ExpiresAt: now.Add(48 * time.Hour),
	}
	if _, reused2, err2 := store.RotateFederationRefreshToken(ctx, peerID, "ref-hash-2", "acc-hash-4", now.Add(2*time.Hour), ref4, now.Add(3*time.Minute)); !errors.Is(err2, bbs.ErrUnauthorized) || !reused2 {
		t.Fatalf("expected family-revoked ref2 to fail with ErrUnauthorized/reused, got reused=%v err=%v", reused2, err2)
	}

	activeAfter, err := store.ListActiveFederationGrantsByUser(ctx, userID, now.Add(3*time.Minute))
	if err != nil || len(activeAfter) != 0 {
		t.Fatalf("expected 0 active grants after family revocation, got %+v, err=%v", activeAfter, err)
	}

	// Test bulk peer+user and all-user revocation helpers.
	gA := &bbs.FederationGrant{PeerID: "peer-a", UserID: userID, Scopes: []string{bbs.FederationScopeWrite}, PolicyVersion: "v1", AccessTokenHash: "a1", AccessExpiresAt: now.Add(time.Hour)}
	gB := &bbs.FederationGrant{PeerID: "peer-b", UserID: userID, Scopes: []string{bbs.FederationScopeWrite}, PolicyVersion: "v1", AccessTokenHash: "b1", AccessExpiresAt: now.Add(time.Hour)}
	_ = store.CreateFederationGrantWithRefresh(ctx, gA, &bbs.FederationRefreshToken{TokenHash: "ra1", ExpiresAt: now.Add(time.Hour)})
	_ = store.CreateFederationGrantWithRefresh(ctx, gB, &bbs.FederationRefreshToken{TokenHash: "rb1", ExpiresAt: now.Add(time.Hour)})

	if err := store.RevokeFederationGrantsByPeerAndUser(ctx, "peer-a", userID, now); err != nil {
		t.Fatalf("RevokeFederationGrantsByPeerAndUser failed: %v", err)
	}
	activeMid, _ := store.ListActiveFederationGrantsByUser(ctx, userID, now)
	if len(activeMid) != 1 || activeMid[0].PeerID != "peer-b" {
		t.Fatalf("expected only peer-b active, got %+v", activeMid)
	}

	if err := store.RevokeAllFederationGrantsByUser(ctx, userID, now); err != nil {
		t.Fatalf("RevokeAllFederationGrantsByUser failed: %v", err)
	}
	activeEnd, _ := store.ListActiveFederationGrantsByUser(ctx, userID, now)
	if len(activeEnd) != 0 {
		t.Fatalf("expected 0 active grants after RevokeAllFederationGrantsByUser, got %+v", activeEnd)
	}
}

func TestFederationStore_ListenerUserTokensCRUD(t *testing.T) {
	ctx := context.Background()
	_, store := openFederationStoreTestDB(t)

	userID := snowflake.ID(80001)
	now := time.Now().UTC()

	tok1 := &bbs.FederationUserToken{
		RemoteKey:        "prod",
		UserID:           userID,
		AccessTokenEnc:   "v1:acc1",
		AccessExpiresAt:  now.Add(time.Hour),
		RefreshTokenEnc:  "v1:ref1",
		RefreshExpiresAt: now.Add(24 * time.Hour),
		Scopes:           []string{bbs.FederationScopeWrite},
		PolicyVersion:    "v1",
	}
	if err := store.UpsertFederationUserToken(ctx, tok1); err != nil {
		t.Fatalf("UpsertFederationUserToken insert failed: %v", err)
	}
	firstID := tok1.ID

	tok1Updated := &bbs.FederationUserToken{
		RemoteKey:        "prod",
		UserID:           userID,
		AccessTokenEnc:   "v1:acc2",
		AccessExpiresAt:  now.Add(2 * time.Hour),
		RefreshTokenEnc:  "v1:ref2",
		RefreshExpiresAt: now.Add(48 * time.Hour),
		Scopes:           []string{bbs.FederationScopeWrite, bbs.FederationScopeCaps},
		PolicyVersion:    "v2",
	}
	if err := store.UpsertFederationUserToken(ctx, tok1Updated); err != nil {
		t.Fatalf("UpsertFederationUserToken update failed: %v", err)
	}
	if tok1Updated.ID != firstID {
		t.Fatalf("expected upsert to preserve ID %s, got %s", firstID, tok1Updated.ID)
	}

	tokOther := &bbs.FederationUserToken{
		RemoteKey:        "staging",
		UserID:           userID,
		AccessTokenEnc:   "v1:accs",
		AccessExpiresAt:  now.Add(time.Hour),
		RefreshTokenEnc:  "v1:refs",
		RefreshExpiresAt: now.Add(24 * time.Hour),
		Scopes:           []string{bbs.FederationScopeReport},
		PolicyVersion:    "v1",
	}
	if err := store.UpsertFederationUserToken(ctx, tokOther); err != nil {
		t.Fatalf("UpsertFederationUserToken staging failed: %v", err)
	}

	list, err := store.ListFederationUserTokensByUser(ctx, userID)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 tokens, got %+v, err=%v", list, err)
	}
	gotProd, err := store.GetFederationUserToken(ctx, "prod", userID)
	if err != nil || gotProd.AccessTokenEnc != "v1:acc2" || !gotProd.HasScope(bbs.FederationScopeCaps) || gotProd.PolicyVersion != "v2" {
		t.Fatalf("unexpected GetFederationUserToken: %+v, err=%v", gotProd, err)
	}

	if err := store.DeleteFederationUserToken(ctx, "prod", userID); err != nil {
		t.Fatalf("DeleteFederationUserToken failed: %v", err)
	}
	if _, err := store.GetFederationUserToken(ctx, "prod", userID); !errors.Is(err, bbs.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	if err := store.DeleteAllFederationUserTokensByUser(ctx, userID); err != nil {
		t.Fatalf("DeleteAllFederationUserTokensByUser failed: %v", err)
	}
	listEnd, _ := store.ListFederationUserTokensByUser(ctx, userID)
	if len(listEnd) != 0 {
		t.Fatalf("expected 0 tokens after DeleteAll, got %+v", listEnd)
	}
}
