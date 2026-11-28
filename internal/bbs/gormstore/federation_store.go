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

package gormstore

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/uuidv7"
)

type crossProviderFKTarget struct {
	Table           string
	Column          string
	ReferencedTable string
}

var crossProviderFKTargets = []crossProviderFKTarget{
	{Table: "bbs_board_mounts", Column: "board_id", ReferencedTable: "bbs_boards"},
	{Table: "bbs_thread_mirrors", Column: "thread_id", ReferencedTable: "bbs_threads"},
	{Table: "bbs_post_mirrors", Column: "post_id", ReferencedTable: "bbs_posts"},
	{Table: "bbs_post_references", Column: "thread_id", ReferencedTable: "bbs_threads"},
	{Table: "bbs_thread_watches", Column: "thread_id", ReferencedTable: "bbs_threads"},
	{Table: "bbs_reports", Column: "post_id", ReferencedTable: "bbs_posts"},
}

var safeSQLIdent = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// DropCrossProviderForeignKeys removes legacy MySQL foreign-key constraints that reference native
// BBS entity tables so remote/federated IDs can be mounted, mirrored, anchored, and watched locally,
// and backfills thread_id/res_number on existing bbs_post_mirrors rows.
func DropCrossProviderForeignKeys(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if db.Name() == "mysql" {
		for _, target := range crossProviderFKTargets {
			var constraintNames []string
			if err := db.Raw(`
				SELECT DISTINCT CONSTRAINT_NAME
				FROM information_schema.KEY_COLUMN_USAGE
				WHERE TABLE_SCHEMA = DATABASE()
				  AND TABLE_NAME = ?
				  AND COLUMN_NAME = ?
				  AND REFERENCED_TABLE_NAME = ?`,
				target.Table, target.Column, target.ReferencedTable,
			).Scan(&constraintNames).Error; err != nil {
				return fmt.Errorf("lookup foreign keys on %s.%s: %w", target.Table, target.Column, err)
			}
			for _, cname := range constraintNames {
				if !safeSQLIdent.MatchString(target.Table) || !safeSQLIdent.MatchString(cname) {
					return fmt.Errorf("unsafe sql identifier for fk drop: %s.%s", target.Table, cname)
				}
				stmt := fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", target.Table, cname)
				if err := db.Exec(stmt).Error; err != nil {
					return fmt.Errorf("drop foreign key %s on %s: %w", cname, target.Table, err)
				}
			}
		}
	}

	if err := db.Exec(`
		UPDATE bbs_post_mirrors
		SET thread_id = (SELECT thread_id FROM bbs_posts WHERE bbs_posts.id = bbs_post_mirrors.post_id),
		    res_number = (SELECT res_number FROM bbs_posts WHERE bbs_posts.id = bbs_post_mirrors.post_id)
		WHERE res_number = 0
		  AND EXISTS (SELECT 1 FROM bbs_posts WHERE bbs_posts.id = bbs_post_mirrors.post_id)`).Error; err != nil {
		return fmt.Errorf("backfill bbs_post_mirrors: %w", err)
	}
	return nil
}

// --- Origin Federation OAuth Grants & Refresh Tokens ---

func (p *Provider) CreateFederationGrantWithRefresh(ctx context.Context, grant *bbs.FederationGrant, refresh *bbs.FederationRefreshToken) error {
	if grant == nil || refresh == nil {
		return bbs.ErrInvalidInput
	}
	now := time.Now().UTC()
	if grant.ID == uuid.Nil {
		grant.ID = uuidv7.New()
	}
	if grant.FamilyID == uuid.Nil {
		grant.FamilyID = uuidv7.New()
	}
	if grant.CreatedAt.IsZero() {
		grant.CreatedAt = now
	}
	if grant.UpdatedAt.IsZero() {
		grant.UpdatedAt = now
	}
	if refresh.ID == uuid.Nil {
		refresh.ID = uuidv7.New()
	}
	refresh.GrantID = grant.ID
	refresh.FamilyID = grant.FamilyID
	refresh.PeerID = grant.PeerID
	refresh.UserID = grant.UserID
	if refresh.CreatedAt.IsZero() {
		refresh.CreatedAt = now
	}

	gRow := BBSFederationGrant{
		ID:              grant.ID,
		FamilyID:        grant.FamilyID,
		PeerID:          grant.PeerID,
		UserID:          grant.UserID,
		Scopes:          bbs.JoinFederationScopes(grant.Scopes),
		PolicyVersion:   grant.PolicyVersion,
		AccessTokenHash: grant.AccessTokenHash,
		AccessExpiresAt: grant.AccessExpiresAt,
		RevokedAt:       grant.RevokedAt,
		CreatedAt:       grant.CreatedAt,
		UpdatedAt:       grant.UpdatedAt,
	}
	rRow := BBSFederationRefreshToken{
		ID:        refresh.ID,
		GrantID:   refresh.GrantID,
		FamilyID:  refresh.FamilyID,
		PeerID:    refresh.PeerID,
		UserID:    refresh.UserID,
		TokenHash: refresh.TokenHash,
		UsedAt:    refresh.UsedAt,
		RevokedAt: refresh.RevokedAt,
		ExpiresAt: refresh.ExpiresAt,
		CreatedAt: refresh.CreatedAt,
	}
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&gRow).Error; err != nil {
			return err
		}
		return tx.Create(&rRow).Error
	})
}

func (p *Provider) GetFederationGrantByAccessHash(ctx context.Context, accessTokenHash string) (*bbs.FederationGrant, error) {
	var row BBSFederationGrant
	if err := p.session(ctx).Where("access_token_hash = ?", accessTokenHash).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainFederationGrant(&row), nil
}

func (p *Provider) GetFederationGrant(ctx context.Context, id uuid.UUID) (*bbs.FederationGrant, error) {
	var row BBSFederationGrant
	if err := p.session(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainFederationGrant(&row), nil
}

func (p *Provider) ListActiveFederationGrantsByUser(ctx context.Context, userID snowflake.ID, now time.Time) ([]bbs.FederationGrant, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	// List non-revoked grants that either have a valid access token or an unexpired unused/unrevoked refresh token.
	var rows []BBSFederationGrant
	err := p.session(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND (access_expires_at > ? OR EXISTS (SELECT 1 FROM bbs_federation_refresh_tokens rt WHERE rt.grant_id = bbs_federation_grants.id AND rt.revoked_at IS NULL AND rt.used_at IS NULL AND rt.expires_at > ?))",
			userID, now, now).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]bbs.FederationGrant, len(rows))
	for i := range rows {
		out[i] = *toDomainFederationGrant(&rows[i])
	}
	return out, nil
}

func revokeGrantFamilyTx(tx *gorm.DB, familyID uuid.UUID, now time.Time) error {
	if err := tx.Model(&BBSFederationGrant{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
		return err
	}
	return tx.Model(&BBSFederationRefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error
}

func (p *Provider) RotateFederationRefreshToken(
	ctx context.Context,
	peerID string,
	oldRefreshHash string,
	newAccessHash string,
	newAccessExp time.Time,
	newRefresh *bbs.FederationRefreshToken,
	now time.Time,
) (*bbs.FederationGrant, bool, error) {
	if newRefresh == nil {
		return nil, false, bbs.ErrInvalidInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var updatedGrant *bbs.FederationGrant
	var reusedDetected bool
	var outcomeErr error

	err := p.session(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx
		if tx.Name() == "mysql" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var rt BBSFederationRefreshToken
		if findErr := q.Where("token_hash = ?", oldRefreshHash).First(&rt).Error; findErr != nil {
			return mapErr(findErr)
		}
		if rt.PeerID != peerID {
			return bbs.ErrNotFound
		}
		// Refresh token reuse or post-revocation replay compromises the entire token family.
		if rt.UsedAt != nil || rt.RevokedAt != nil {
			reusedDetected = true
			if rErr := revokeGrantFamilyTx(tx, rt.FamilyID, now); rErr != nil {
				return rErr
			}
			outcomeErr = bbs.ErrUnauthorized
			return nil
		}
		if !now.Before(rt.ExpiresAt) {
			if rErr := tx.Model(&BBSFederationRefreshToken{}).
				Where("id = ?", rt.ID).
				Update("revoked_at", now).Error; rErr != nil {
				return rErr
			}
			outcomeErr = bbs.ErrUnauthorized
			return nil
		}

		var gRow BBSFederationGrant
		if gErr := q.Where("id = ?", rt.GrantID).First(&gRow).Error; gErr != nil {
			if rErr := revokeGrantFamilyTx(tx, rt.FamilyID, now); rErr != nil {
				return rErr
			}
			outcomeErr = bbs.ErrUnauthorized
			return nil
		}
		if gRow.RevokedAt != nil || gRow.PeerID != peerID {
			if rErr := revokeGrantFamilyTx(tx, rt.FamilyID, now); rErr != nil {
				return rErr
			}
			outcomeErr = bbs.ErrUnauthorized
			return nil
		}

		res := tx.Model(&BBSFederationRefreshToken{}).
			Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", rt.ID).
			Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			reusedDetected = true
			if rErr := revokeGrantFamilyTx(tx, rt.FamilyID, now); rErr != nil {
				return rErr
			}
			outcomeErr = bbs.ErrUnauthorized
			return nil
		}

		if err := tx.Model(&BBSFederationGrant{}).
			Where("id = ? AND revoked_at IS NULL", gRow.ID).
			Updates(map[string]any{
				"access_token_hash": newAccessHash,
				"access_expires_at": newAccessExp,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}

		if newRefresh.ID == uuid.Nil {
			newRefresh.ID = uuidv7.New()
		}
		newRefresh.GrantID = gRow.ID
		newRefresh.FamilyID = gRow.FamilyID
		newRefresh.PeerID = gRow.PeerID
		newRefresh.UserID = gRow.UserID
		if newRefresh.CreatedAt.IsZero() {
			newRefresh.CreatedAt = now
		}
		nextRow := BBSFederationRefreshToken{
			ID:        newRefresh.ID,
			GrantID:   newRefresh.GrantID,
			FamilyID:  newRefresh.FamilyID,
			PeerID:    newRefresh.PeerID,
			UserID:    newRefresh.UserID,
			TokenHash: newRefresh.TokenHash,
			ExpiresAt: newRefresh.ExpiresAt,
			CreatedAt: newRefresh.CreatedAt,
		}
		if err := tx.Create(&nextRow).Error; err != nil {
			return err
		}

		gRow.AccessTokenHash = newAccessHash
		gRow.AccessExpiresAt = newAccessExp
		gRow.UpdatedAt = now
		updatedGrant = toDomainFederationGrant(&gRow)
		return nil
	})
	if err != nil {
		return nil, reusedDetected, err
	}
	if outcomeErr != nil {
		return nil, reusedDetected, outcomeErr
	}
	return updatedGrant, false, nil
}

func (p *Provider) RevokeFederationGrant(ctx context.Context, grantID uuid.UUID, userID snowflake.ID, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		var row BBSFederationGrant
		if err := tx.Where("id = ? AND user_id = ?", grantID, userID).First(&row).Error; err != nil {
			return mapErr(err)
		}
		return revokeGrantFamilyTx(tx, row.FamilyID, now)
	})
}

func (p *Provider) RevokeFederationGrantsByPeerAndUser(ctx context.Context, peerID string, userID snowflake.ID, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&BBSFederationGrant{}).
			Where("peer_id = ? AND user_id = ? AND revoked_at IS NULL", peerID, userID).
			Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&BBSFederationRefreshToken{}).
			Where("peer_id = ? AND user_id = ? AND revoked_at IS NULL", peerID, userID).
			Update("revoked_at", now).Error
	})
}

func (p *Provider) RevokeAllFederationGrantsByUser(ctx context.Context, userID snowflake.ID, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&BBSFederationGrant{}).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&BBSFederationRefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Update("revoked_at", now).Error
	})
}

// --- Listener Encrypted Federation User Tokens ---

func (p *Provider) GetFederationUserToken(ctx context.Context, remoteKey string, userID snowflake.ID) (*bbs.FederationUserToken, error) {
	var row BBSFederationUserToken
	if err := p.session(ctx).Where("remote_key = ? AND user_id = ?", remoteKey, userID).First(&row).Error; err != nil {
		return nil, mapErr(err)
	}
	return toDomainFederationUserToken(&row), nil
}

func (p *Provider) ListFederationUserTokensByUser(ctx context.Context, userID snowflake.ID) ([]bbs.FederationUserToken, error) {
	var rows []BBSFederationUserToken
	if err := p.session(ctx).Where("user_id = ?", userID).Order("remote_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]bbs.FederationUserToken, len(rows))
	for i := range rows {
		out[i] = *toDomainFederationUserToken(&rows[i])
	}
	return out, nil
}

func (p *Provider) UpsertFederationUserToken(ctx context.Context, tok *bbs.FederationUserToken) error {
	if tok == nil || strings.TrimSpace(tok.RemoteKey) == "" || tok.UserID == 0 {
		return bbs.ErrInvalidInput
	}
	now := time.Now().UTC()
	if tok.ID == uuid.Nil {
		tok.ID = uuidv7.New()
	}
	if tok.CreatedAt.IsZero() {
		tok.CreatedAt = now
	}
	tok.UpdatedAt = now
	scopesStr := bbs.JoinFederationScopes(tok.Scopes)

	return p.session(ctx).Transaction(func(tx *gorm.DB) error {
		var existing BBSFederationUserToken
		findErr := tx.Where("remote_key = ? AND user_id = ?", tok.RemoteKey, tok.UserID).First(&existing).Error
		switch {
		case findErr == nil:
			if err := tx.Model(&BBSFederationUserToken{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"access_token_enc":   tok.AccessTokenEnc,
				"access_expires_at":  tok.AccessExpiresAt,
				"refresh_token_enc":  tok.RefreshTokenEnc,
				"refresh_expires_at": tok.RefreshExpiresAt,
				"scopes":             scopesStr,
				"policy_version":     tok.PolicyVersion,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}
			tok.ID = existing.ID
			tok.CreatedAt = existing.CreatedAt
			return nil
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			row := BBSFederationUserToken{
				ID:               tok.ID,
				RemoteKey:        tok.RemoteKey,
				UserID:           tok.UserID,
				AccessTokenEnc:   tok.AccessTokenEnc,
				AccessExpiresAt:  tok.AccessExpiresAt,
				RefreshTokenEnc:  tok.RefreshTokenEnc,
				RefreshExpiresAt: tok.RefreshExpiresAt,
				Scopes:           scopesStr,
				PolicyVersion:    tok.PolicyVersion,
				CreatedAt:        tok.CreatedAt,
				UpdatedAt:        now,
			}
			return tx.Create(&row).Error
		default:
			return findErr
		}
	})
}

func (p *Provider) DeleteFederationUserToken(ctx context.Context, remoteKey string, userID snowflake.ID) error {
	return p.session(ctx).Where("remote_key = ? AND user_id = ?", remoteKey, userID).Delete(&BBSFederationUserToken{}).Error
}

func (p *Provider) DeleteAllFederationUserTokensByUser(ctx context.Context, userID snowflake.ID) error {
	return p.session(ctx).Where("user_id = ?", userID).Delete(&BBSFederationUserToken{}).Error
}

func toDomainFederationGrant(r *BBSFederationGrant) *bbs.FederationGrant {
	return &bbs.FederationGrant{
		ID:              r.ID,
		FamilyID:        r.FamilyID,
		PeerID:          r.PeerID,
		UserID:          r.UserID,
		Scopes:          bbs.ParseFederationScopes(r.Scopes),
		PolicyVersion:   r.PolicyVersion,
		AccessTokenHash: r.AccessTokenHash,
		AccessExpiresAt: r.AccessExpiresAt,
		RevokedAt:       r.RevokedAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func toDomainFederationUserToken(r *BBSFederationUserToken) *bbs.FederationUserToken {
	return &bbs.FederationUserToken{
		ID:               r.ID,
		RemoteKey:        r.RemoteKey,
		UserID:           r.UserID,
		AccessTokenEnc:   r.AccessTokenEnc,
		AccessExpiresAt:  r.AccessExpiresAt,
		RefreshTokenEnc:  r.RefreshTokenEnc,
		RefreshExpiresAt: r.RefreshExpiresAt,
		Scopes:           bbs.ParseFederationScopes(r.Scopes),
		PolicyVersion:    r.PolicyVersion,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}
