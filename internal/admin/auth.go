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

package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/uuidv7"
)

const (
	defaultSessionTTL      = 7 * 24 * time.Hour
	lastUsedTouchThreshold = time.Minute
)

var ErrInvalidSession = errors.New("admin: invalid or expired session")

// SessionStore persists operator sessions in GORM using SHA-256 token hashes.
type SessionStore struct {
	db *gorm.DB
}

// NewSessionStore constructs a SessionStore backed by db.
func NewSessionStore(db *gorm.DB) *SessionStore {
	return &SessionStore{db: db}
}

// HashToken returns the lowercase SHA-256 hex digest of a raw bearer token or poll secret.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func generateRandomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func truncateRunes(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

// Create generates a new session token and stores its SHA-256 hash in admin_sessions.
func (s *SessionStore) Create(
	ctx context.Context,
	userID snowflake.ID,
	username string,
	displayName string,
	role string,
	clientInfo string,
	ttl time.Duration,
) (string, *models.AdminSession, error) {
	if s == nil || s.db == nil {
		return "", nil, errors.New("admin: session store database not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ttl <= 0 {
		ttl = defaultSessionTTL
	}
	rawToken, err := generateRandomToken(32)
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	if displayName == "" {
		displayName = username
	}
	row := &models.AdminSession{
		ID:          uuidv7.New(),
		TokenHash:   HashToken(rawToken),
		UserID:      userID,
		Username:    truncateRunes(username, 100),
		DisplayName: truncateRunes(displayName, 100),
		Role:        truncateRunes(role, 64),
		ClientInfo:  truncateRunes(clientInfo, 200),
		ExpiresAt:   now.Add(ttl),
		LastUsedAt:  now,
		CreatedAt:   now,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return "", nil, err
	}
	return rawToken, row, nil
}

// Authenticate validates rawToken against active non-revoked, non-expired sessions.
func (s *SessionStore) Authenticate(ctx context.Context, rawToken string) (*models.AdminSession, error) {
	if s == nil || s.db == nil || strings.TrimSpace(rawToken) == "" {
		return nil, ErrInvalidSession
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC()
	var row models.AdminSession
	err := s.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", HashToken(rawToken), now).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidSession
		}
		return nil, err
	}
	if now.Sub(row.LastUsedAt) >= lastUsedTouchThreshold {
		_ = s.db.WithContext(ctx).Model(&models.AdminSession{}).
			Where("id = ?", row.ID).
			Update("last_used_at", now).Error
		row.LastUsedAt = now
	}
	return &row, nil
}

// Get returns a session by ID.
func (s *SessionStore) Get(ctx context.Context, id uuid.UUID) (*models.AdminSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("admin: session store database not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var row models.AdminSession
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// List returns sessions ordered newest first, optionally filtered by userID and active state.
func (s *SessionStore) List(ctx context.Context, filterUserID *snowflake.ID, activeOnly bool) ([]models.AdminSession, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("admin: session store database not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	q := s.db.WithContext(ctx).Model(&models.AdminSession{})
	if filterUserID != nil {
		q = q.Where("user_id = ?", *filterUserID)
	}
	if activeOnly {
		q = q.Where("revoked_at IS NULL AND expires_at > ?", time.Now().UTC())
	}
	var rows []models.AdminSession
	if err := q.Order("created_at DESC, id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountActive returns the count of non-revoked, non-expired sessions.
func (s *SessionStore) CountActive(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.AdminSession{}).
		Where("revoked_at IS NULL AND expires_at > ?", time.Now().UTC()).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Revoke marks the session identified by id as revoked.
func (s *SessionStore) Revoke(ctx context.Context, id uuid.UUID) error {
	if s == nil || s.db == nil {
		return errors.New("admin: session store database not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	row, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if row.RevokedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&models.AdminSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}
