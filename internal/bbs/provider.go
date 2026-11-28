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
	"slices"
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
	UpdateCategory(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*Category, error)
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

// FederatedProvider is optionally implemented by cross-instance remote BBS providers.
type FederatedProvider interface {
	Provider
	RemoteKey() string
	DisplayLabel() string
	CanReport() bool
	CanUseCaps() bool
	HasUserDelegation(ctx context.Context, userID snowflake.ID) (bool, error)
	RevokeUserDelegation(ctx context.Context, userID snowflake.ID) error
	CreateRemoteThread(ctx context.Context, input CreateThreadInput) (*Board, *Thread, *Post, error)
	CreateRemotePost(ctx context.Context, input CreatePostInput) (*Board, *Thread, *Post, []uint32, error)
	CreateRemoteReport(ctx context.Context, guildID, reporterID snowflake.ID, postID uuid.UUID, reason string) (*Report, error)
	ListRemoteAvailableCaps(ctx context.Context, userID snowflake.ID, boardID uuid.UUID) ([]Cap, error)
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

// DefaultKey returns the configured default provider key.
func (r *ProviderRegistry) DefaultKey() string {
	return r.defaultKey
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

// All returns all registered providers deterministically, with the default provider placed first.
func (r *ProviderRegistry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	if def, ok := r.providers[r.defaultKey]; ok {
		out = append(out, def)
	}
	otherKeys := make([]string, 0, len(r.providers))
	for k := range r.providers {
		if k == r.defaultKey {
			continue
		}
		otherKeys = append(otherKeys, k)
	}
	slices.Sort(otherKeys)
	for _, k := range otherKeys {
		out = append(out, r.providers[k])
	}
	return out
}

// Store persists Discord mounts, mirrors, opt-in state, consents, references, watches, bans, reports, and federation tokens.
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
	ListThreadMirrorsByMount(ctx context.Context, mountID uuid.UUID) ([]ThreadMirror, error)
	UpdateThreadMirror(ctx context.Context, mirror *ThreadMirror) error
	AdvanceThreadMirrorSync(ctx context.Context, id uuid.UUID, syncedUpTo uint32) error
	UpdateThreadMirrorControl(ctx context.Context, id uuid.UUID, controlMsgID *snowflake.ID) error
	MarkThreadMirrorArchivedSynced(ctx context.Context, id uuid.UUID) (bool, error)
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
	GetBan(ctx context.Context, id uuid.UUID) (*Ban, error)
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

	// Admin queries & mounts
	ListBoardMounts(ctx context.Context, filter BoardMountFilter) ([]BoardMount, error)
	GetBBSStats(ctx context.Context) (*Stats, error)

	// Admin caps (persisted runtime config)
	ListAdminCaps(ctx context.Context) ([]CapDefinition, error)
	GetAdminCap(ctx context.Context, id string) (*CapDefinition, error)
	UpsertAdminCap(ctx context.Context, def CapDefinition) error
	DeleteAdminCap(ctx context.Context, id string) error
	SeedAdminCapsIfEmpty(ctx context.Context, defs []CapDefinition) ([]CapDefinition, error)

	// SLIP events (persisted runtime config)
	ListSlipEvents(ctx context.Context) ([]SlipEventDefinition, error)
	GetSlipEvent(ctx context.Context, id string) (*SlipEventDefinition, error)
	UpsertSlipEvent(ctx context.Context, def SlipEventDefinition) error
	DeleteSlipEvent(ctx context.Context, id string) error
	SeedSlipEventsIfEmpty(ctx context.Context, defs []SlipEventDefinition) ([]SlipEventDefinition, error)

	// Federation Origin OAuth grants & refresh tokens
	CreateFederationGrantWithRefresh(ctx context.Context, grant *FederationGrant, refresh *FederationRefreshToken) error
	GetFederationGrantByAccessHash(ctx context.Context, accessTokenHash string) (*FederationGrant, error)
	GetFederationGrant(ctx context.Context, id uuid.UUID) (*FederationGrant, error)
	ListActiveFederationGrantsByUser(ctx context.Context, userID snowflake.ID, now time.Time) ([]FederationGrant, error)
	RotateFederationRefreshToken(ctx context.Context, peerID, oldRefreshHash, newAccessHash string, newAccessExp time.Time, newRefresh *FederationRefreshToken, now time.Time) (grant *FederationGrant, reused bool, err error)
	RevokeFederationGrant(ctx context.Context, grantID uuid.UUID, userID snowflake.ID, now time.Time) error
	RevokeFederationGrantsByPeerAndUser(ctx context.Context, peerID string, userID snowflake.ID, now time.Time) error
	RevokeAllFederationGrantsByUser(ctx context.Context, userID snowflake.ID, now time.Time) error

	// Federation Listener encrypted user tokens
	GetFederationUserToken(ctx context.Context, remoteKey string, userID snowflake.ID) (*FederationUserToken, error)
	ListFederationUserTokensByUser(ctx context.Context, userID snowflake.ID) ([]FederationUserToken, error)
	UpsertFederationUserToken(ctx context.Context, tok *FederationUserToken) error
	DeleteFederationUserToken(ctx context.Context, remoteKey string, userID snowflake.ID) error
	DeleteAllFederationUserTokensByUser(ctx context.Context, userID snowflake.ID) error
}
