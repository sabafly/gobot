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
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

// Models returns all GORM models managed by the BBS subsystem for AutoMigrate.
func Models() []any {
	return []any{
		&BBSGuildConfig{},
		&BBSUserConsent{},
		&BBSCategory{},
		&BBSBoard{},
		&BBSBoardMount{},
		&BBSThread{},
		&BBSThreadMirror{},
		&BBSPost{},
		&BBSPostMirror{},
		&BBSPostReference{},
		&BBSThreadWatch{},
		&BBSBan{},
		&BBSReport{},
		&BBSSystemSecret{},
	}
}

type BBSSystemSecret struct {
	SecretKey string    `gorm:"column:secret_key;type:varchar(64);primaryKey"`
	SecretHex string    `gorm:"column:secret_hex;type:varchar(256);not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (BBSSystemSecret) TableName() string { return "bbs_system_secrets" }

type BBSGuildConfig struct {
	GuildID         snowflake.ID  `gorm:"primaryKey;autoIncrement:false"`
	Enabled         bool          `gorm:"not null;default:false"`
	AgreedVersion   string        `gorm:"type:varchar(32);not null;default:''"`
	AgreedByUserID  snowflake.ID  `gorm:"not null;default:0"`
	AgreedAt        *time.Time    `gorm:""`
	ViewerRoleID    *snowflake.ID `gorm:""`
	DefaultUseForum bool          `gorm:"not null"`
	ModLogChannelID *snowflake.ID `gorm:""`
	UpdatedAt       time.Time     `gorm:"not null"`
}

func (BBSGuildConfig) TableName() string { return "bbs_guild_configs" }

type BBSUserConsent struct {
	UserID        snowflake.ID `gorm:"primaryKey;autoIncrement:false"`
	PolicyVersion string       `gorm:"type:varchar(32);not null"`
	AgreedAt      time.Time    `gorm:"not null"`
	UpdatedAt     time.Time    `gorm:"not null"`
}

func (BBSUserConsent) TableName() string { return "bbs_user_consents" }

type BBSCategory struct {
	ID                uuid.UUID     `gorm:"type:varchar(36);primaryKey"`
	Scope             string        `gorm:"type:varchar(16);not null;index"`
	GuildID           *snowflake.ID `gorm:"index"`
	ProviderKey       string        `gorm:"type:varchar(64);not null"`
	ExternalID        string        `gorm:"type:varchar(191)"`
	Slug              string        `gorm:"type:varchar(128);not null"`
	Name              string        `gorm:"type:varchar(128);not null"`
	Description       string        `gorm:"type:text"`
	SortOrder         int           `gorm:"not null;default:0"`
	DiscordCategoryID *snowflake.ID `gorm:""`
	CreatedAt         time.Time     `gorm:"not null"`
	UpdatedAt         time.Time     `gorm:"not null"`
}

func (BBSCategory) TableName() string { return "bbs_categories" }

type BBSBoard struct {
	ID                 uuid.UUID     `gorm:"type:varchar(36);primaryKey"`
	CategoryID         uuid.UUID     `gorm:"type:varchar(36);not null;index"`
	Category           BBSCategory   `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE;"`
	Scope              string        `gorm:"type:varchar(16);not null;index"`
	GuildID            *snowflake.ID `gorm:"index"`
	ProviderKey        string        `gorm:"type:varchar(64);not null"`
	ExternalID         string        `gorm:"type:varchar(191)"`
	Slug               string        `gorm:"type:varchar(128);not null"`
	Name               string        `gorm:"type:varchar(128);not null"`
	Description        string        `gorm:"type:text"`
	DefaultName        string        `gorm:"type:varchar(64);not null"`
	HideID             bool          `gorm:"not null;default:false"`
	SLIPLevel          int           `gorm:"not null;default:0"`
	MaxRes             uint32        `gorm:"not null;default:1000"`
	DatOchiIdleHours   int           `gorm:"not null;default:1200"`
	DatOchiLowResHours int           `gorm:"not null;default:240"`
	Locked             bool          `gorm:"not null;default:false"`
	CreatedAt          time.Time     `gorm:"not null"`
	UpdatedAt          time.Time     `gorm:"not null"`
}

func (BBSBoard) TableName() string { return "bbs_boards" }

type BBSBoardMount struct {
	ID               uuid.UUID     `gorm:"type:varchar(36);primaryKey"`
	BoardID          uuid.UUID     `gorm:"type:varchar(36);not null;index"`
	Board            BBSBoard      `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE;"`
	GuildID          snowflake.ID  `gorm:"not null;index"`
	ChannelID        snowflake.ID  `gorm:"not null;uniqueIndex"`
	Locale           string        `gorm:"type:varchar(16);not null;default:'ja'"`
	IsForum          bool          `gorm:"not null"`
	PinnedPostID     *snowflake.ID `gorm:""`
	ControlMessageID *snowflake.ID `gorm:""`
	Active           bool          `gorm:"not null"`
	CreatedAt        time.Time     `gorm:"not null"`
}

func (BBSBoardMount) TableName() string { return "bbs_board_mounts" }

type BBSThread struct {
	ID                 uuid.UUID  `gorm:"type:varchar(36);primaryKey"`
	BoardID            uuid.UUID  `gorm:"type:varchar(36);not null;index"`
	Board              BBSBoard   `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE;"`
	ProviderKey        string     `gorm:"type:varchar(64);not null"`
	ExternalID         string     `gorm:"type:varchar(191)"`
	Title              string     `gorm:"type:varchar(128);not null"`
	Status             string     `gorm:"type:varchar(16);not null;index"`
	ResCount           uint32     `gorm:"not null;default:0"`
	DefaultName        string     `gorm:"type:varchar(64);not null"`
	HideID             bool       `gorm:"not null;default:false"`
	SLIPLevel          int        `gorm:"not null;default:0"`
	MaxRes             uint32     `gorm:"not null;default:1000"`
	DatOchiIdleHours   int        `gorm:"not null;default:1200"`
	DatOchiLowResHours int        `gorm:"not null;default:240"`
	ExtDat             string     `gorm:"type:varchar(191)"`
	OwnerPosterHash    string     `gorm:"type:varchar(32);not null"`
	OwnerAnonID        string     `gorm:"type:varchar(32);not null"`
	BumpedAt           time.Time  `gorm:"not null;index"`
	ArchivedAt         *time.Time `gorm:""`
	CreatedAt          time.Time  `gorm:"not null"`
	UpdatedAt          time.Time  `gorm:"not null"`
}

func (BBSThread) TableName() string { return "bbs_threads" }

type BBSThreadMirror struct {
	ID               uuid.UUID     `gorm:"type:varchar(36);primaryKey"`
	ThreadID         uuid.UUID     `gorm:"type:varchar(36);not null;index"`
	Thread           BBSThread     `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE;"`
	BoardMountID     uuid.UUID     `gorm:"type:varchar(36);not null;index"`
	BoardMount       BBSBoardMount `gorm:"foreignKey:BoardMountID;constraint:OnDelete:CASCADE;"`
	GuildID          snowflake.ID  `gorm:"not null;index"`
	DiscordThreadID  snowflake.ID  `gorm:"not null;uniqueIndex"`
	ControlMessageID *snowflake.ID `gorm:""`
	SyncedUpToRes    uint32        `gorm:"not null;default:0"`
	CreatedAt        time.Time     `gorm:"not null"`
}

func (BBSThreadMirror) TableName() string { return "bbs_thread_mirrors" }

type BBSPost struct {
	ID           uuid.UUID    `gorm:"type:varchar(36);primaryKey"`
	ThreadID     uuid.UUID    `gorm:"type:varchar(36);not null;uniqueIndex:idx_bbs_thread_res"`
	Thread       BBSThread    `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE;"`
	ResNumber    uint32       `gorm:"not null;uniqueIndex:idx_bbs_thread_res"`
	Name         string       `gorm:"type:varchar(64);not null"`
	Trip         string       `gorm:"type:varchar(32)"`
	SlipText     string       `gorm:"type:varchar(32)"`
	AnonID       string       `gorm:"type:varchar(32)"`
	PosterHash   string       `gorm:"type:varchar(32);not null;index"`
	IsOP         bool         `gorm:"not null;default:false"`
	IsSage       bool         `gorm:"not null;default:false"`
	Body         string       `gorm:"type:text;not null"`
	Abone        bool         `gorm:"not null;default:false"`
	AboneReason  string       `gorm:"type:varchar(255)"`
	AuthorUserID snowflake.ID `gorm:"not null;index"`
	CreatedAt    time.Time    `gorm:"not null"`
}

func (BBSPost) TableName() string { return "bbs_posts" }

type BBSPostMirror struct {
	ID               uuid.UUID       `gorm:"type:varchar(36);primaryKey"`
	PostID           uuid.UUID       `gorm:"type:varchar(36);not null;index"`
	Post             BBSPost         `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE;"`
	ThreadMirrorID   uuid.UUID       `gorm:"type:varchar(36);not null;index"`
	ThreadMirror     BBSThreadMirror `gorm:"foreignKey:ThreadMirrorID;constraint:OnDelete:CASCADE;"`
	GuildID          snowflake.ID    `gorm:"not null;index"`
	DiscordMessageID snowflake.ID    `gorm:"not null;uniqueIndex"`
	CreatedAt        time.Time       `gorm:"not null"`
}

func (BBSPostMirror) TableName() string { return "bbs_post_mirrors" }

type BBSPostReference struct {
	ThreadID uuid.UUID `gorm:"type:varchar(36);primaryKey"`
	Thread   BBSThread `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE;"`
	FromRes  uint32    `gorm:"primaryKey;autoIncrement:false"`
	ToRes    uint32    `gorm:"primaryKey;autoIncrement:false;index"`
}

func (BBSPostReference) TableName() string { return "bbs_post_references" }

type BBSThreadWatch struct {
	ID        uuid.UUID    `gorm:"type:varchar(36);primaryKey"`
	ThreadID  uuid.UUID    `gorm:"type:varchar(36);not null;uniqueIndex:idx_bbs_watch_thread_user"`
	Thread    BBSThread    `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE;"`
	UserID    snowflake.ID `gorm:"not null;uniqueIndex:idx_bbs_watch_thread_user"`
	Locale    string       `gorm:"type:varchar(16);not null;default:'ja'"`
	CreatedAt time.Time    `gorm:"not null"`
}

func (BBSThreadWatch) TableName() string { return "bbs_thread_watches" }

type BBSBan struct {
	ID           uuid.UUID     `gorm:"type:varchar(36);primaryKey"`
	Scope        string        `gorm:"type:varchar(16);not null;index"`
	GuildID      *snowflake.ID `gorm:"index"`
	PosterHash   string        `gorm:"type:varchar(32);not null;index"`
	TargetUserID *snowflake.ID `gorm:"index"`
	Reason       string        `gorm:"type:varchar(255)"`
	CreatedBy    snowflake.ID  `gorm:"not null"`
	ExpiresAt    *time.Time    `gorm:""`
	CreatedAt    time.Time     `gorm:"not null"`
}

func (BBSBan) TableName() string { return "bbs_bans" }

type BBSReport struct {
	ID             uuid.UUID     `gorm:"type:varchar(36);primaryKey"`
	Scope          string        `gorm:"type:varchar(16);not null;index"`
	GuildID        snowflake.ID  `gorm:"not null;index"`
	PostID         uuid.UUID     `gorm:"type:varchar(36);not null;index"`
	Post           BBSPost       `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE;"`
	ThreadID       uuid.UUID     `gorm:"type:varchar(36);not null;index"`
	BoardID        uuid.UUID     `gorm:"type:varchar(36);not null;index"`
	ReporterUserID snowflake.ID  `gorm:"not null"`
	Reason         string        `gorm:"type:text;not null"`
	Status         string        `gorm:"type:varchar(16);not null;index"`
	HandledBy      *snowflake.ID `gorm:""`
	ResolutionNote string        `gorm:"type:text"`
	CreatedAt      time.Time     `gorm:"not null"`
	UpdatedAt      time.Time     `gorm:"not null"`
}

func (BBSReport) TableName() string { return "bbs_reports" }
