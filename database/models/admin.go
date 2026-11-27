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

package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

// AdminSession persists an authenticated TUI/API operator session backed by a SHA-256 token hash.
type AdminSession struct {
	ID          uuid.UUID    `gorm:"type:varchar(36);primaryKey"`
	TokenHash   string       `gorm:"type:varchar(64);uniqueIndex;not null"`
	UserID      snowflake.ID `gorm:"not null;index"`
	Username    string       `gorm:"type:varchar(100);not null"`
	DisplayName string       `gorm:"type:varchar(100);not null"`
	Role        string       `gorm:"type:varchar(64);not null"`
	ClientInfo  string       `gorm:"type:varchar(200);not null;default:''"`
	ExpiresAt   time.Time    `gorm:"not null;index"`
	LastUsedAt  time.Time    `gorm:"not null"`
	RevokedAt   *time.Time   `gorm:"index"`
	CreatedAt   time.Time    `gorm:"not null"`
}

func (AdminSession) TableName() string { return "admin_sessions" }

// AdminAuditLog records security-relevant management and moderation actions performed via the admin API.
type AdminAuditLog struct {
	ID           uuid.UUID    `gorm:"type:varchar(36);primaryKey"`
	ActorUserID  snowflake.ID `gorm:"not null;index"`
	ActorName    string       `gorm:"type:varchar(100);not null"`
	ActorRole    string       `gorm:"type:varchar(64);not null"`
	Action       string       `gorm:"type:varchar(100);not null;index"`
	ResourceType string       `gorm:"type:varchar(64);not null;index"`
	ResourceID   string       `gorm:"type:varchar(128);not null;default:''"`
	Summary      string       `gorm:"type:varchar(255);not null"`
	DetailJSON   string       `gorm:"type:text;not null"`
	Status       string       `gorm:"type:varchar(32);not null"`
	RemoteAddr   string       `gorm:"type:varchar(64);not null;default:''"`
	CreatedAt    time.Time    `gorm:"not null;index"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_logs" }
