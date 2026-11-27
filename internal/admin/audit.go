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
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/uuidv7"
)

const (
	defaultAuditListLimit = 50
	maxAuditListLimit     = 200
)

// AuditStore persists and queries operator management audit entries in admin_audit_logs.
type AuditStore struct {
	db *gorm.DB
}

// NewAuditStore constructs an AuditStore backed by db.
func NewAuditStore(db *gorm.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Record stores an audit entry into admin_audit_logs.
func (s *AuditStore) Record(ctx context.Context, entry models.AdminAuditLog) {
	if s == nil || s.db == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if entry.ID == [16]byte{} {
		entry.ID = uuidv7.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if entry.DetailJSON == "" {
		entry.DetailJSON = "{}"
	}
	if entry.Status == "" {
		entry.Status = "success"
	}
	entry.ActorName = truncateRunes(entry.ActorName, 100)
	entry.ActorRole = truncateRunes(entry.ActorRole, 64)
	entry.Action = truncateRunes(entry.Action, 100)
	entry.ResourceType = truncateRunes(entry.ResourceType, 64)
	entry.ResourceID = truncateRunes(entry.ResourceID, 128)
	entry.Summary = truncateRunes(entry.Summary, 255)
	entry.Status = truncateRunes(entry.Status, 32)
	entry.RemoteAddr = truncateRunes(entry.RemoteAddr, 64)

	if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
		slog.Warn("admin audit record failed", "action", entry.Action, "err", err)
	}
}

// RecordAction marshals detail as JSON and persists an audit log entry.
func (s *AuditStore) RecordAction(
	ctx context.Context,
	actorID snowflake.ID,
	actorName string,
	actorRole string,
	action string,
	resourceType string,
	resourceID string,
	summary string,
	detail any,
	status string,
	remoteAddr string,
) {
	detailStr := "{}"
	if detail != nil {
		if raw, err := json.Marshal(detail); err == nil {
			detailStr = string(raw)
		}
	}
	s.Record(ctx, models.AdminAuditLog{
		ActorUserID:  actorID,
		ActorName:    actorName,
		ActorRole:    actorRole,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Summary:      summary,
		DetailJSON:   detailStr,
		Status:       status,
		RemoteAddr:   remoteAddr,
	})
}

// List queries audit log entries matching filter ordered newest first.
func (s *AuditStore) List(ctx context.Context, filter AuditFilter) ([]models.AdminAuditLog, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("admin: audit store database not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultAuditListLimit
	}
	if limit > maxAuditListLimit {
		limit = maxAuditListLimit
	}
	offset := max(filter.Offset, 0)

	q := s.db.WithContext(ctx).Model(&models.AdminAuditLog{})
	if rawUID := strings.TrimSpace(filter.ActorUserID); rawUID != "" {
		uid, err := snowflake.Parse(rawUID)
		if err != nil {
			return nil, 0, nil
		}
		q = q.Where("actor_user_id = ?", uid)
	}
	if act := strings.TrimSpace(filter.Action); act != "" {
		q = q.Where("action = ?", act)
	}
	if rt := strings.TrimSpace(filter.ResourceType); rt != "" {
		q = q.Where("resource_type = ?", rt)
	}
	if st := strings.TrimSpace(filter.Status); st != "" {
		q = q.Where("status = ?", st)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.AdminAuditLog
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
