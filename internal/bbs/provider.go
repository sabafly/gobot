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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

// CategoryProvider abstracts persistence and retrieval of BBS categories.
type CategoryProvider interface {
	GetCategory(ctx context.Context, id uuid.UUID) (*Category, error)
	ListCategories(ctx context.Context, filter CategoryFilter) ([]Category, error)
	CreateCategory(ctx context.Context, input CreateCategoryInput) (*Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

// BoardProvider abstracts persistence and retrieval of BBS boards.
type BoardProvider interface {
	GetBoard(ctx context.Context, id uuid.UUID) (*Board, error)
	ListBoards(ctx context.Context, filter BoardFilter) ([]Board, error)
	CreateBoard(ctx context.Context, input CreateBoardInput) (*Board, error)
	UpdateBoard(ctx context.Context, id uuid.UUID, input UpdateBoardInput) (*Board, error)
	DeleteBoard(ctx context.Context, id uuid.UUID) error
}

// ThreadProvider abstracts persistence and retrieval of BBS threads.
type ThreadProvider interface {
	GetThread(ctx context.Context, id uuid.UUID) (*Thread, error)
	ListThreads(ctx context.Context, filter ThreadFilter) ([]Thread, error)
	CreateThread(ctx context.Context, input CreateThreadRecord) (*Thread, *Post, error)
	ArchiveThread(ctx context.Context, id uuid.UUID) (*Thread, error)
	ListStaleThreads(ctx context.Context, now time.Time) ([]Thread, error)
}

// PostProvider abstracts persistence and retrieval of BBS posts.
type PostProvider interface {
	GetPost(ctx context.Context, id uuid.UUID) (*Post, error)
	GetPostByNumber(ctx context.Context, threadID uuid.UUID, resNumber uint32) (*Post, error)
	ListPosts(ctx context.Context, threadID uuid.UUID, offset, limit int) ([]Post, error)
	CreatePost(ctx context.Context, input CreatePostRecord) (*Post, *Thread, error)
	AbonePost(ctx context.Context, id uuid.UUID, reason string) (*Post, error)
}

// ProviderCapabilities advertises what operations a backend provider supports.
type ProviderCapabilities struct {
	CanCreateBoard  bool
	CanCreateThread bool
	CanCreatePost   bool
	CanModerate     bool
}

// Provider bundles the four loosely coupled BBS layers under a backend key.
type Provider interface {
	Key() string
	Capabilities() ProviderCapabilities
	CategoryProvider
	BoardProvider
	ThreadProvider
	PostProvider
}

// ProviderRegistry resolves backend providers by key (e.g., "native" or 3rd-party adapters).
type ProviderRegistry struct {
	mu         sync.RWMutex
	providers  map[string]Provider
	defaultKey string
}

// NewProviderRegistry creates a registry initialized with a default provider key.
func NewProviderRegistry(defaultKey string) *ProviderRegistry {
	if defaultKey == "" {
		defaultKey = DefaultProviderKey
	}
	return &ProviderRegistry{
		providers:  make(map[string]Provider),
		defaultKey: defaultKey,
	}
}

// Register adds or replaces a provider implementation.
func (r *ProviderRegistry) Register(p Provider) {
	if p == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Key()] = p
}

// Get returns the provider for the requested key, or default if key is empty.
func (r *ProviderRegistry) Get(key string) (Provider, error) {
	if key == "" {
		key = r.defaultKey
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, key)
	}
	return p, nil
}

// Default returns the default registered provider, or nil if absent.
func (r *ProviderRegistry) Default() Provider {
	p, _ := r.Get(r.defaultKey)
	return p
}

// All returns all registered providers with the default provider placed first.
func (r *ProviderRegistry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	if def, ok := r.providers[r.defaultKey]; ok {
		out = append(out, def)
	}
	for k, p := range r.providers {
		if k == r.defaultKey {
			continue
		}
		out = append(out, p)
	}
	return out
}

// Store persists Discord mounts, mirrors, opt-in state, consents, references, watches, bans, and reports.
type Store interface {
	// Guild configuration
	GetGuildConfig(ctx context.Context, guildID snowflake.ID) (*GuildConfig, error)
	SaveGuildConfig(ctx context.Context, cfg *GuildConfig) error

	// User consent
	GetUserConsent(ctx context.Context, userID snowflake.ID) (*UserConsent, error)
	SaveUserConsent(ctx context.Context, consent *UserConsent) error
	DeleteUserConsent(ctx context.Context, userID snowflake.ID) error

	// Board mounts
	CreateBoardMount(ctx context.Context, mount *BoardMount) error
	GetBoardMount(ctx context.Context, id uuid.UUID) (*BoardMount, error)
	GetBoardMountByGuildAndBoard(ctx context.Context, guildID snowflake.ID, boardID uuid.UUID) (*BoardMount, error)
	GetBoardMountByChannel(ctx context.Context, channelID snowflake.ID) (*BoardMount, error)
	ListActiveBoardMountsByBoard(ctx context.Context, boardID uuid.UUID) ([]BoardMount, error)
	ListBoardMountsByGuild(ctx context.Context, guildID snowflake.ID) ([]BoardMount, error)
	UpdateBoardMount(ctx context.Context, mount *BoardMount) error
	UpdateBoardMountControl(ctx context.Context, id uuid.UUID, controlMsgID *snowflake.ID) (bool, error)
	DeactivateBoardMount(ctx context.Context, id uuid.UUID) error

	// Thread mirrors
	CreateThreadMirror(ctx context.Context, mirror *ThreadMirror) error
	CreateThreadAndPostMirror(ctx context.Context, threadMirror *ThreadMirror, postMirror *PostMirror) error
	GetThreadMirror(ctx context.Context, id uuid.UUID) (*ThreadMirror, error)
	GetThreadMirrorByDiscordThread(ctx context.Context, discordThreadID snowflake.ID) (*ThreadMirror, error)
	GetThreadMirrorByMountAndThread(ctx context.Context, mountID, threadID uuid.UUID) (*ThreadMirror, error)
	ListThreadMirrorsByThread(ctx context.Context, threadID uuid.UUID) ([]ThreadMirror, error)
	UpdateThreadMirror(ctx context.Context, mirror *ThreadMirror) error
	AdvanceThreadMirrorSync(ctx context.Context, id uuid.UUID, syncedUpTo uint32) error
	UpdateThreadMirrorControl(ctx context.Context, id uuid.UUID, controlMsgID *snowflake.ID) error
	DeleteThreadMirror(ctx context.Context, id uuid.UUID) error

	// Post mirrors
	CreatePostMirror(ctx context.Context, mirror *PostMirror) error
	GetPostMirrorByDiscordMessage(ctx context.Context, discordMessageID snowflake.ID) (*PostMirror, error)
	GetPostMirrorByPostAndGuild(ctx context.Context, postID uuid.UUID, guildID snowflake.ID) (*PostMirror, error)
	GetPostMirrorByResAndGuild(ctx context.Context, threadID uuid.UUID, resNumber uint32, guildID snowflake.ID) (*PostMirror, error)
	ListPostMirrorsByPost(ctx context.Context, postID uuid.UUID) ([]PostMirror, error)

	// Anchor forward references
	SavePostReferences(ctx context.Context, refs []PostReference) error
	ListReferencingRes(ctx context.Context, threadID uuid.UUID, targetRes uint32) ([]uint32, error)

	// Thread watch subscriptions
	GetThreadWatch(ctx context.Context, threadID uuid.UUID, userID snowflake.ID) (*ThreadWatch, error)
	SaveThreadWatch(ctx context.Context, watch *ThreadWatch) error
	DeleteThreadWatch(ctx context.Context, threadID uuid.UUID, userID snowflake.ID) error
	ListThreadWatches(ctx context.Context, threadID uuid.UUID) ([]ThreadWatch, error)

	// Bans
	CreateBan(ctx context.Context, ban *Ban) error
	DeleteBan(ctx context.Context, id uuid.UUID) error
	ListBans(ctx context.Context, filter BanFilter) ([]Ban, error)
	IsBanned(ctx context.Context, scope Scope, guildID snowflake.ID, posterHash string, userID snowflake.ID, now time.Time) (bool, error)

	// Reports & operator-only audit
	CreateReport(ctx context.Context, report *Report) error
	GetReport(ctx context.Context, id uuid.UUID) (*Report, error)
	ListReports(ctx context.Context, filter ReportFilter) ([]Report, error)
	UpdateReport(ctx context.Context, report *Report) error
	GetAuditPostInfo(ctx context.Context, postID uuid.UUID) (*AuditPostInfo, error)

	// System secrets (persisted fallback HMAC key)
	GetOrCreateSystemSecret(ctx context.Context, secretKey string, fallback []byte) ([]byte, error)
}
