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

package bbs

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

const (
	DefaultProviderKey     = "native"
	DefaultAnonymousName   = "風吹けば名無し"
	DefaultMaxRes          = uint32(1000)
	MinSlashBoardMaxRes    = uint32(50)
	MaxBoardMaxRes         = uint32(1000)
	DatOchiDisabled        = -1
	DefaultDatOchiIdleHrs  = 24 * 50 // 50 days without activity
	DefaultDatOchiLowResHr = 24 * 10 // 10 days with fewer than 10 posts
	MaxArchiveDays         = 365
	DefaultPolicyVersion   = "2026-11-25"
	MaxPolicyVersionLen    = 64
	MaxBanHours            = 24 * 365 * 10
)

type Scope string

const (
	ScopeLocal  Scope = "local"
	ScopeGlobal Scope = "global"
)

type SLIPLevel int

const (
	SLIPNone  SLIPLevel = 0
	SLIPVVVVV SLIPLevel = 1
)

type ThreadStatus string

const (
	ThreadStatusActive   ThreadStatus = "active"
	ThreadStatusArchived ThreadStatus = "archived"
	ThreadStatusDeleted  ThreadStatus = "deleted"
)

type ReportStatus string

const (
	ReportStatusOpen      ReportStatus = "open"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

var (
	ErrNotFound             = errors.New("bbs: resource not found")
	ErrGuildNotEnabled      = errors.New("bbs: feature is not enabled in this guild")
	ErrConsentRequired      = errors.New("bbs: user policy consent is required")
	ErrBanned               = errors.New("bbs: user is banned from posting")
	ErrRateLimited          = errors.New("bbs: posting too fast")
	ErrBoardLocked          = errors.New("bbs: board is locked")
	ErrThreadArchived       = errors.New("bbs: thread is archived")
	ErrMaxResReached        = errors.New("bbs: thread reached maximum post count")
	ErrUnauthorized         = errors.New("bbs: unauthorized action")
	ErrInvalidInput         = errors.New("bbs: invalid input")
	ErrProviderNotFound     = errors.New("bbs: provider not found")
	ErrUnsupportedOperation = errors.New("bbs: operation not supported by provider")
	ErrAlreadyExists        = errors.New("bbs: resource already exists")

	ErrInvalidCategoryName   = fmt.Errorf("%w: invalid category name", ErrInvalidInput)
	ErrInvalidBoardName      = fmt.Errorf("%w: invalid board name", ErrInvalidInput)
	ErrInvalidTitleLength    = fmt.Errorf("%w: title length invalid", ErrInvalidInput)
	ErrInvalidNameLength     = fmt.Errorf("%w: name exceeds max length", ErrInvalidInput)
	ErrInvalidBodyLength     = fmt.Errorf("%w: body length invalid", ErrInvalidInput)
	ErrReportReasonRequired  = fmt.Errorf("%w: report reason required", ErrInvalidInput)
	ErrGuildIDRequired       = fmt.Errorf("%w: guild_id required for local category", ErrInvalidInput)
	ErrBoardScopeMismatch    = fmt.Errorf("%w: board scope must match category scope", ErrInvalidInput)
	ErrInvalidBanDuration    = fmt.Errorf("%w: ban duration out of range", ErrInvalidInput)
	ErrBoardProviderMismatch = fmt.Errorf("%w: board provider must match category provider", ErrInvalidInput)
	ErrInvalidBoardRules     = fmt.Errorf("%w: invalid board rules", ErrInvalidInput)
	ErrInvalidMail           = fmt.Errorf("%w: invalid mail field", ErrInvalidInput)
	ErrInvalidNameCommand    = fmt.Errorf("%w: unsupported name command", ErrInvalidInput)
	ErrInvalidExtendCommand  = fmt.Errorf("%w: invalid !extend syntax", ErrInvalidInput)
	ErrFormattedBodyTooLong  = fmt.Errorf("%w: formatted body exceeds display limit", ErrInvalidInput)
	ErrInvalidReaderJump     = fmt.Errorf("%w: invalid reader jump target", ErrInvalidInput)
	ErrStalePolicyVersion    = fmt.Errorf("%w: outdated policy panel version", ErrInvalidInput)
	ErrInvalidPolicyVersion  = fmt.Errorf("%w: policy version exceeds max length", ErrInvalidInput)
)

// BoardRules configures anonymous posting behavior for a board or thread.
type BoardRules struct {
	DefaultName        string
	HideID             bool
	SLIP               SLIPLevel
	MaxRes             uint32
	DatOchiIdleHours   int
	DatOchiLowResHours int
}

// Normalize fills zero-value rule fields with sensible defaults while preserving DatOchiDisabled (-1).
func (r BoardRules) Normalize() BoardRules {
	if r.DefaultName == "" {
		r.DefaultName = DefaultAnonymousName
	}
	if r.MaxRes == 0 {
		r.MaxRes = DefaultMaxRes
	}
	if r.DatOchiIdleHours == 0 {
		r.DatOchiIdleHours = DefaultDatOchiIdleHrs
	}
	if r.DatOchiLowResHours == 0 {
		r.DatOchiLowResHours = DefaultDatOchiLowResHr
	}
	return r
}

// Validate rejects out-of-range or unsupported board rule values.
func (r BoardRules) Validate() error {
	if r.DefaultName != "" && utf8.RuneCountInString(SanitizeSingleLine(r.DefaultName)) > MaxNameRunes {
		return ErrInvalidNameLength
	}
	if r.SLIP != SLIPNone && r.SLIP != SLIPVVVVV {
		return ErrInvalidBoardRules
	}
	if r.MaxRes > MaxBoardMaxRes {
		return ErrInvalidBoardRules
	}
	if r.DatOchiIdleHours < DatOchiDisabled || r.DatOchiIdleHours > MaxArchiveDays*24 {
		return ErrInvalidBoardRules
	}
	if r.DatOchiLowResHours < DatOchiDisabled || r.DatOchiLowResHours > MaxArchiveDays*24 {
		return ErrInvalidBoardRules
	}
	return nil
}

// Category represents a top-level board grouping (local to a guild or global).
type Category struct {
	ID                uuid.UUID
	Scope             Scope
	GuildID           *snowflake.ID
	ProviderKey       string
	ExternalID        string
	Slug              string
	Name              string
	Description       string
	SortOrder         int
	DiscordCategoryID *snowflake.ID
	CreatedAt         time.Time
}

// Board represents a 5ch-style bulletin board belonging to a category.
type Board struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Scope       Scope
	GuildID     *snowflake.ID
	ProviderKey string
	ExternalID  string
	Slug        string
	Name        string
	Description string
	Rules       BoardRules
	Locked      bool
	CreatedAt   time.Time
}

// Thread represents a discussion thread within a board.
// Author Discord IDs are intentionally omitted from this public/guild DTO.
type Thread struct {
	ID              uuid.UUID
	BoardID         uuid.UUID
	ProviderKey     string
	ExternalID      string
	Title           string
	Status          ThreadStatus
	ResCount        uint32
	Rules           BoardRules
	ExtDat          string
	OwnerPosterHash string
	OwnerAnonID     string
	BumpedAt        time.Time
	ArchivedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Post represents an anonymous reply inside a thread.
// AuthorUserID is deliberately excluded to enforce anonymity in guild layers.
type Post struct {
	ID          uuid.UUID
	ThreadID    uuid.UUID
	ResNumber   uint32
	Name        string
	Trip        string
	SlipText    string
	AnonID      string
	PosterHash  string
	IsOP        bool
	IsSage      bool
	Body        string
	Abone       bool
	AboneReason string
	CreatedAt   time.Time
}

// AuditPostInfo contains operator-only attribution for abuse/legal investigations.
type AuditPostInfo struct {
	Post         Post
	AuthorUserID snowflake.ID
	GuildID      snowflake.ID
}

// GuildConfig stores per-guild opt-in and moderation settings.
type GuildConfig struct {
	GuildID         snowflake.ID
	Enabled         bool
	AgreedVersion   string
	AgreedByUserID  snowflake.ID
	AgreedAt        *time.Time
	DefaultUseForum bool
	ModLogChannelID *snowflake.ID
	UpdatedAt       time.Time
}

// UserConsent stores a user's acceptance of the BBS policy documents.
type UserConsent struct {
	UserID        snowflake.ID
	PolicyVersion string
	AgreedAt      time.Time
}

// BoardMount maps a board to a Discord channel in a subscribed/owning guild.
type BoardMount struct {
	ID               uuid.UUID
	BoardID          uuid.UUID
	GuildID          snowflake.ID
	ChannelID        snowflake.ID
	Locale           string
	IsForum          bool
	PinnedPostID     *snowflake.ID
	ControlMessageID *snowflake.ID
	Active           bool
	CreatedAt        time.Time
}

// ThreadMirror maps a logical thread to a Discord thread inside a mounted channel.
type ThreadMirror struct {
	ID               uuid.UUID
	ThreadID         uuid.UUID
	BoardMountID     uuid.UUID
	GuildID          snowflake.ID
	DiscordThreadID  snowflake.ID
	ControlMessageID *snowflake.ID
	SyncedUpToRes    uint32
	CreatedAt        time.Time
}

// PostMirror maps a logical post to its Discord message ID in a specific guild.
type PostMirror struct {
	ID               uuid.UUID
	PostID           uuid.UUID
	ThreadMirrorID   uuid.UUID
	GuildID          snowflake.ID
	DiscordMessageID snowflake.ID
	CreatedAt        time.Time
}

// PostReference tracks >>N anchors inside a thread for forward-reference updates.
type PostReference struct {
	ThreadID uuid.UUID
	FromRes  uint32
	ToRes    uint32
}

// ThreadWatch tracks DM update subscriptions for a thread.
type ThreadWatch struct {
	ID        uuid.UUID
	ThreadID  uuid.UUID
	UserID    snowflake.ID
	Locale    string
	CreatedAt time.Time
}

// Ban blocks a poster hash (or user ID for global bans) from creating threads/posts.
type Ban struct {
	ID           uuid.UUID
	Scope        Scope
	GuildID      *snowflake.ID
	PosterHash   string
	TargetUserID *snowflake.ID
	Reason       string
	CreatedBy    snowflake.ID
	ExpiresAt    *time.Time
	CreatedAt    time.Time
}

// Report captures a user-submitted moderation report on a post or thread.
type Report struct {
	ID             uuid.UUID
	Scope          Scope
	GuildID        snowflake.ID
	PostID         uuid.UUID
	ThreadID       uuid.UUID
	BoardID        uuid.UUID
	ReporterUserID snowflake.ID
	Reason         string
	Status         ReportStatus
	BanID          *uuid.UUID
	HandledBy      *snowflake.ID
	ResolutionNote string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CategoryFilter selects categories.
type CategoryFilter struct {
	Scope   *Scope
	GuildID *snowflake.ID
}

// CreateCategoryInput defines parameters for creating a category.
type CreateCategoryInput struct {
	Scope             Scope
	GuildID           *snowflake.ID
	ProviderKey       string
	ExternalID        string
	Slug              string
	Name              string
	Description       string
	SortOrder         int
	DiscordCategoryID *snowflake.ID
}

// BoardFilter selects boards.
type BoardFilter struct {
	Scope      *Scope
	GuildID    *snowflake.ID
	CategoryID *uuid.UUID
}

// CreateBoardInput defines parameters for creating a board.
type CreateBoardInput struct {
	CategoryID  uuid.UUID
	Scope       Scope
	GuildID     *snowflake.ID
	ProviderKey string
	ExternalID  string
	Slug        string
	Name        string
	Description string
	Rules       BoardRules
}

// UpdateBoardInput modifies mutable board settings.
type UpdateBoardInput struct {
	Name        *string
	Description *string
	DefaultName *string
	HideID      *bool
	SLIP        *SLIPLevel
	Locked      *bool
}

// ThreadFilter selects threads within a board.
type ThreadFilter struct {
	BoardID uuid.UUID
	Status  *ThreadStatus
	Limit   int
	Offset  int
}

// CreateThreadRecord is passed from Service to ThreadProvider.
type CreateThreadRecord struct {
	BoardID         uuid.UUID
	Title           string
	Rules           BoardRules
	ExtDat          string
	OwnerPosterHash string
	OwnerAnonID     string
	FirstPost       CreatePostRecord
}

// CreatePostRecord is passed from Service to PostProvider.
type CreatePostRecord struct {
	ThreadID     uuid.UUID
	Name         string
	Trip         string
	SlipText     string
	AnonID       string
	PosterHash   string
	IsOP         bool
	IsSage       bool
	Body         string
	AuthorUserID snowflake.ID
}

// CreateThreadInput is the high-level user request to open a thread.
type CreateThreadInput struct {
	BoardID uuid.UUID
	GuildID snowflake.ID
	UserID  snowflake.ID
	Title   string
	RawName string
	RawMail string
	RawBody string
	Now     time.Time
}

// CreatePostInput is the high-level user request to post a reply.
type CreatePostInput struct {
	ThreadID uuid.UUID
	GuildID  snowflake.ID
	UserID   snowflake.ID
	RawName  string
	RawMail  string
	RawBody  string
	Now      time.Time
}

// GuildSetupInput updates guild-level BBS opt-in configuration.
type GuildSetupInput struct {
	GuildID         snowflake.ID
	Enabled         bool
	ActorUserID     snowflake.ID
	DefaultUseForum *bool
	ModLogChannelID *snowflake.ID
	ClearModLog     bool
}

// ReportFilter selects moderation reports.
type ReportFilter struct {
	Scope  *Scope
	Status *ReportStatus
	Limit  int
}

// BanFilter selects active bans as of Now (defaults to current time when zero).
type BanFilter struct {
	Scope   *Scope
	GuildID *snowflake.ID
	Limit   int
	Now     time.Time
}
