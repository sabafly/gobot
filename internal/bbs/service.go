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
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/ratelimit"
	"github.com/sabafly/gobot/internal/uuidv7"
)

const (
	MaxTitleRunes       = 100
	MaxBodyRunes        = 1800
	MaxEscapedBodyRunes = 2200
	MaxNameRunes        = 48
)

// Service orchestrates BBS domain rules, opt-in checks, consent, rate limits, and providers.
type Service struct {
	Registry  *ProviderRegistry
	Store     Store
	Identity  *IdentityGenerator
	Policy    *PolicyManager
	Operators []snowflake.ID

	capMu     sync.RWMutex
	configMu  sync.Mutex
	adminCaps []CapDefinition

	threadRules   []ratelimit.Rule
	postRules     []ratelimit.Rule
	threadRLMu    sync.Mutex
	postRLMu      sync.Mutex
	threadRL      map[snowflake.ID][]time.Time
	postRL        map[snowflake.ID][]time.Time
	threadRLSweep time.Time
	postRLSweep   time.Time

	threadLocks [256]sync.Mutex
	guildLocks  [64]sync.Mutex

	eventSinkMu sync.RWMutex
	eventSink   EventSink
}

// NewService constructs a Service with standard anti-spam rate limits.
func NewService(
	registry *ProviderRegistry,
	store Store,
	identity *IdentityGenerator,
	policy *PolicyManager,
	operators []snowflake.ID,
) *Service {
	return &Service{
		Registry:  registry,
		Store:     store,
		Identity:  identity,
		Policy:    policy,
		Operators: slices.Clone(operators),
		threadRules: []ratelimit.Rule{
			{Limit: 1, Unit: 30 * time.Second},
			{Limit: 5, Unit: 10 * time.Minute},
		},
		postRules: []ratelimit.Rule{
			{Limit: 1, Unit: 3 * time.Second},
			{Limit: 5, Unit: 30 * time.Second},
		},
		threadRL: make(map[snowflake.ID][]time.Time),
		postRL:   make(map[snowflake.ID][]time.Time),
	}
}

// IsOperator returns true if userID is configured as a gobot BBS operator.
func (s *Service) IsOperator(userID snowflake.ID) bool {
	return slices.Contains(s.Operators, userID)
}

// SetAdminCaps validates and configures static operator/admin caps on the service.
func (s *Service) SetAdminCaps(defs []CapDefinition) error {
	validated, err := ValidateCapDefinitions(defs)
	if err != nil {
		return err
	}
	s.capMu.Lock()
	s.adminCaps = validated
	s.capMu.Unlock()
	return nil
}

// SetSlipEvents validates and configures custom date-based SLIP events on the service.
func (s *Service) SetSlipEvents(defs []SlipEventDefinition) error {
	return s.Identity.SetSlipEvents(defs)
}

func (s *Service) userHasAdminCap(userID snowflake.ID, def CapDefinition) bool {
	if userID == 0 {
		return false
	}
	if def.GrantOperators && s.IsOperator(userID) {
		return true
	}
	return slices.Contains(def.Users, userID)
}

// ListAvailableCaps returns caps held by userID that can be used when posting on boardID.
func (s *Service) ListAvailableCaps(ctx context.Context, userID snowflake.ID, boardID uuid.UUID) []Cap {
	if userID == 0 {
		return nil
	}
	if boardID != uuid.Nil {
		if b, err := s.GetBoard(ctx, boardID); err == nil && b != nil {
			if p, pErr := s.Registry.Get(b.ProviderKey); pErr == nil {
				if fp, ok := p.(FederatedProvider); ok {
					caps, rErr := fp.ListRemoteAvailableCaps(ctx, userID, boardID)
					if rErr != nil {
						slog.Warn("bbs failed to fetch remote caps", "provider", fp.Key(), "board_id", boardID, "err", rErr)
						return nil
					}
					return caps
				}
			}
		}
	}
	s.capMu.RLock()
	defs := s.adminCaps
	s.capMu.RUnlock()
	if len(defs) == 0 {
		return nil
	}
	var caps []Cap
	for _, def := range defs {
		if !s.userHasAdminCap(userID, def) {
			continue
		}
		caps = append(caps, Cap{
			ID:          def.ID,
			Name:        def.Name,
			Description: def.Description,
		})
	}
	return caps
}

// ResolveUserCap validates that userID holds capID on boardID.
// An empty capID or CapSelectNoneValue resolves to (nil, nil).
func (s *Service) ResolveUserCap(ctx context.Context, userID snowflake.ID, boardID uuid.UUID, capID string) (*Cap, error) {
	cleanID := SanitizeSingleLine(capID)
	if cleanID == "" || cleanID == CapSelectNoneValue {
		return nil, nil
	}
	for _, c := range s.ListAvailableCaps(ctx, userID, boardID) {
		if c.ID == cleanID {
			cp := c
			return &cp, nil
		}
	}
	return nil, ErrCapNotAuthorized
}

func (s *Service) threadMutex(threadID uuid.UUID) *sync.Mutex {
	v := binary.LittleEndian.Uint64(threadID[8:16]) ^ binary.BigEndian.Uint64(threadID[0:8])
	v ^= v >> 32
	v ^= v >> 16
	v ^= v >> 8
	idx := v % uint64(len(s.threadLocks))
	return &s.threadLocks[idx]
}

func (s *Service) guildMutex(guildID snowflake.ID) *sync.Mutex {
	idx := uint64(guildID) % uint64(len(s.guildLocks))
	return &s.guildLocks[idx]
}

// EnsureGuildEnabled checks whether a guild has opted in to BBS.
func (s *Service) EnsureGuildEnabled(ctx context.Context, guildID snowflake.ID) (*GuildConfig, error) {
	cfg, err := s.Store.GetGuildConfig(ctx, guildID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrGuildNotEnabled
		}
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrGuildNotEnabled
	}
	return cfg, nil
}

// GetGuildConfig returns the guild configuration or nil if not yet initialized.
func (s *Service) GetGuildConfig(ctx context.Context, guildID snowflake.ID) (*GuildConfig, error) {
	cfg, err := s.Store.GetGuildConfig(ctx, guildID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return cfg, err
}

// SetupGuild applies guild opt-in settings and records administrator policy agreement.
func (s *Service) SetupGuild(ctx context.Context, input GuildSetupInput) (*GuildConfig, error) {
	mu := s.guildMutex(input.GuildID)
	mu.Lock()
	defer mu.Unlock()

	cfg, err := s.Store.GetGuildConfig(ctx, input.GuildID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if cfg == nil {
		cfg = &GuildConfig{
			GuildID:         input.GuildID,
			DefaultUseForum: true,
		}
	}
	cfg.Enabled = input.Enabled
	now := time.Now()
	if input.Enabled {
		if err := ValidatePolicyConfig(PolicyConfig{Version: s.Policy.CurrentVersion}); err != nil {
			return nil, err
		}
		cfg.AgreedVersion = s.Policy.CurrentVersion
		cfg.AgreedByUserID = input.ActorUserID
		cfg.AgreedAt = &now
	}
	if input.DefaultUseForum != nil {
		cfg.DefaultUseForum = *input.DefaultUseForum
	}
	if input.ClearModLog {
		cfg.ModLogChannelID = nil
	} else if input.ModLogChannelID != nil {
		cfg.ModLogChannelID = input.ModLogChannelID
	}
	cfg.UpdatedAt = now
	if err := s.Store.SaveGuildConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// CheckConsent verifies whether the user has agreed to the current policy version.
func (s *Service) CheckConsent(ctx context.Context, userID snowflake.ID) (*UserConsent, error) {
	consent, err := s.Store.GetUserConsent(ctx, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if s.Policy.NeedsConsent(consent) {
		return consent, ErrConsentRequired
	}
	return consent, nil
}

// AcceptPolicy records a user's acceptance of the current policy version.
func (s *Service) AcceptPolicy(ctx context.Context, userID snowflake.ID) (*UserConsent, error) {
	if err := ValidatePolicyConfig(PolicyConfig{Version: s.Policy.CurrentVersion}); err != nil {
		return nil, err
	}
	consent := &UserConsent{
		UserID:        userID,
		PolicyVersion: s.Policy.CurrentVersion,
		AgreedAt:      time.Now(),
	}
	if err := s.Store.SaveUserConsent(ctx, consent); err != nil {
		return nil, err
	}
	return consent, nil
}

// AcceptPolicyVersion validates that versionToken matches the active policy version before recording consent.
func (s *Service) AcceptPolicyVersion(ctx context.Context, userID snowflake.ID, versionToken string) (*UserConsent, error) {
	if versionToken == "" || versionToken != PolicyVersionToken(s.Policy.CurrentVersion) {
		return nil, ErrStalePolicyVersion
	}
	return s.AcceptPolicy(ctx, userID)
}

// RevokePolicy withdraws user consent, removes active thread watches, and cascades federation delegation revocation.
func (s *Service) RevokePolicy(ctx context.Context, userID snowflake.ID) error {
	if err := s.Store.DeleteUserConsent(ctx, userID); err != nil {
		return err
	}
	if err := s.Store.RevokeAllFederationGrantsByUser(ctx, userID, time.Now().UTC()); err != nil {
		slog.Warn("bbs failed to revoke origin federation grants on policy revoke", "user_id", userID, "err", err)
	}
	for _, p := range s.Registry.All() {
		if fp, ok := p.(FederatedProvider); ok {
			if rErr := fp.RevokeUserDelegation(ctx, userID); rErr != nil {
				slog.Warn("bbs failed to revoke remote user delegation on policy revoke", "provider", fp.Key(), "user_id", userID, "err", rErr)
			}
		}
	}
	if err := s.Store.DeleteAllFederationUserTokensByUser(ctx, userID); err != nil {
		slog.Warn("bbs failed to delete listener federation user tokens on policy revoke", "user_id", userID, "err", err)
	}
	return nil
}

// CreateCategory validates scope/inputs and creates a local or global category via its provider.
func (s *Service) CreateCategory(ctx context.Context, input CreateCategoryInput) (*Category, error) {
	input.Name = SanitizeSingleLine(input.Name)
	input.Slug = strings.ToLower(SanitizeSingleLine(input.Slug))
	input.Description = SanitizeSingleLine(input.Description)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 64 {
		return nil, ErrInvalidCategoryName
	}
	if input.Slug == "" {
		input.Slug = strings.ReplaceAll(strings.ToLower(input.Name), " ", "-")
	}
	switch input.Scope {
	case ScopeLocal:
		if input.GuildID == nil || *input.GuildID == 0 {
			return nil, ErrGuildIDRequired
		}
		if _, err := s.EnsureGuildEnabled(ctx, *input.GuildID); err != nil {
			return nil, err
		}
	case ScopeGlobal:
		input.GuildID = nil
	default:
		return nil, fmt.Errorf("%w: invalid scope", ErrInvalidInput)
	}

	p, err := s.Registry.Get(input.ProviderKey)
	if err != nil {
		return nil, err
	}
	input.ProviderKey = p.Key()
	return p.CreateCategory(ctx, input)
}

// ListCategories returns categories matching the given scope/guild filter across providers.
// Non-default remote providers are fault-isolated so unreachable remotes do not break local listings.
func (s *Service) ListCategories(ctx context.Context, filter CategoryFilter) ([]Category, error) {
	providers := s.Registry.All()
	if len(providers) == 0 {
		return nil, ErrProviderNotFound
	}
	defaultKey := s.Registry.DefaultKey()
	var out []Category
	seen := make(map[uuid.UUID]struct{})
	for _, p := range providers {
		cats, err := p.ListCategories(ctx, filter)
		if err != nil {
			if errors.Is(err, ErrUnsupportedOperation) {
				continue
			}
			if p.Key() != defaultKey {
				slog.Warn("bbs remote ListCategories skipped due to error", "provider", p.Key(), "err", err)
				continue
			}
			return nil, err
		}
		for _, c := range cats {
			if _, ok := seen[c.ID]; ok {
				continue
			}
			seen[c.ID] = struct{}{}
			out = append(out, c)
		}
	}
	return out, nil
}

// GetCategory resolves a category across registered providers.
func (s *Service) GetCategory(ctx context.Context, id uuid.UUID) (*Category, error) {
	providers := s.Registry.All()
	if len(providers) == 0 {
		return nil, ErrProviderNotFound
	}
	defaultKey := s.Registry.DefaultKey()
	var remoteErr error
	for _, p := range providers {
		cat, err := p.GetCategory(ctx, id)
		if err == nil {
			return cat, nil
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrUnsupportedOperation) {
			if p.Key() != defaultKey {
				if remoteErr == nil {
					remoteErr = err
				}
				continue
			}
			return nil, err
		}
	}
	if remoteErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrFederationUnavailable, remoteErr)
	}
	return nil, ErrNotFound
}

// DeleteCategory removes a category by ID via its owning provider.
func (s *Service) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	cat, err := s.GetCategory(ctx, id)
	if err != nil {
		return err
	}
	p, err := s.Registry.Get(cat.ProviderKey)
	if err != nil {
		return err
	}
	return p.DeleteCategory(ctx, id)
}

// CreateBoard validates category ownership/scope and creates a board under the category's provider.
func (s *Service) CreateBoard(ctx context.Context, input CreateBoardInput) (*Board, error) {
	input.Name = SanitizeSingleLine(input.Name)
	input.Slug = strings.ToLower(SanitizeSingleLine(input.Slug))
	input.Description = SanitizeBody(input.Description)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 80 {
		return nil, ErrInvalidBoardName
	}
	if input.Slug == "" {
		input.Slug = strings.ReplaceAll(strings.ToLower(input.Name), " ", "-")
	}
	input.Rules.DefaultName = SanitizeDisplayName(input.Rules.DefaultName)
	if err := input.Rules.Validate(); err != nil {
		return nil, err
	}
	input.Rules = input.Rules.Normalize()
	if err := input.Rules.Validate(); err != nil {
		return nil, err
	}

	cat, err := s.GetCategory(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}
	if cat.Scope != input.Scope {
		return nil, ErrBoardScopeMismatch
	}
	if input.Scope == ScopeLocal {
		if input.GuildID == nil || cat.GuildID == nil || *input.GuildID != *cat.GuildID {
			return nil, fmt.Errorf("%w: local board guild mismatch", ErrUnauthorized)
		}
		if _, err := s.EnsureGuildEnabled(ctx, *input.GuildID); err != nil {
			return nil, err
		}
	} else {
		input.GuildID = nil
	}

	if input.ProviderKey != "" && input.ProviderKey != cat.ProviderKey {
		return nil, ErrBoardProviderMismatch
	}
	provKey := cat.ProviderKey
	p, err := s.Registry.Get(provKey)
	if err != nil {
		return nil, err
	}
	if !p.Capabilities().CanCreateBoard {
		return nil, ErrUnsupportedOperation
	}
	input.ProviderKey = p.Key()
	return p.CreateBoard(ctx, input)
}

// GetBoard resolves a board across registered providers.
func (s *Service) GetBoard(ctx context.Context, boardID uuid.UUID) (*Board, error) {
	providers := s.Registry.All()
	if len(providers) == 0 {
		return nil, ErrProviderNotFound
	}
	defaultKey := s.Registry.DefaultKey()
	var remoteErr error
	for _, p := range providers {
		b, err := p.GetBoard(ctx, boardID)
		if err == nil {
			return b, nil
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrUnsupportedOperation) {
			if p.Key() != defaultKey {
				if remoteErr == nil {
					remoteErr = err
				}
				continue
			}
			return nil, err
		}
	}
	if remoteErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrFederationUnavailable, remoteErr)
	}
	return nil, ErrNotFound
}

// ListBoards returns boards matching filter across registered providers.
// Non-default remote providers are fault-isolated so unreachable remotes do not break local listings.
func (s *Service) ListBoards(ctx context.Context, filter BoardFilter) ([]Board, error) {
	providers := s.Registry.All()
	if len(providers) == 0 {
		return nil, ErrProviderNotFound
	}
	defaultKey := s.Registry.DefaultKey()
	var out []Board
	seen := make(map[uuid.UUID]struct{})
	for _, p := range providers {
		boards, err := p.ListBoards(ctx, filter)
		if err != nil {
			if errors.Is(err, ErrUnsupportedOperation) {
				continue
			}
			if p.Key() != defaultKey {
				slog.Warn("bbs remote ListBoards skipped due to error", "provider", p.Key(), "err", err)
				continue
			}
			return nil, err
		}
		for _, b := range boards {
			if _, ok := seen[b.ID]; ok {
				continue
			}
			seen[b.ID] = struct{}{}
			out = append(out, b)
		}
	}
	return out, nil
}

// UpdateBoard validates and updates board configuration fields.
func (s *Service) UpdateBoard(ctx context.Context, boardID uuid.UUID, input UpdateBoardInput) (*Board, error) {
	if input.Name != nil {
		clean := SanitizeSingleLine(*input.Name)
		if clean == "" || utf8.RuneCountInString(clean) > 80 {
			return nil, ErrInvalidBoardName
		}
		input.Name = &clean
	}
	if input.Description != nil {
		clean := SanitizeBody(*input.Description)
		input.Description = &clean
	}
	if input.DefaultName != nil {
		clean := SanitizeDisplayName(*input.DefaultName)
		if clean == "" || utf8.RuneCountInString(clean) > MaxNameRunes {
			return nil, ErrInvalidNameLength
		}
		input.DefaultName = &clean
	}
	if input.SLIP != nil && !input.SLIP.IsValid() {
		return nil, ErrInvalidBoardRules
	}
	b, err := s.GetBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}
	p, err := s.Registry.Get(b.ProviderKey)
	if err != nil {
		return nil, err
	}
	updated, err := p.UpdateBoard(ctx, boardID, input)
	if err != nil {
		return nil, err
	}
	if s.isNativeGlobalBoard(updated) {
		bCopy := *updated
		s.emitEvent(Event{
			Type:    EventBoardUpdated,
			BoardID: updated.ID,
			Board:   &bCopy,
		})
	}
	return updated, nil
}

// DeleteBoard deletes a board.
func (s *Service) DeleteBoard(ctx context.Context, boardID uuid.UUID) error {
	b, err := s.GetBoard(ctx, boardID)
	if err != nil {
		return err
	}
	p, err := s.Registry.Get(b.ProviderKey)
	if err != nil {
		return err
	}
	if err := p.DeleteBoard(ctx, boardID); err != nil {
		return err
	}
	if s.isNativeGlobalBoard(b) {
		bCopy := *b
		s.emitEvent(Event{
			Type:    EventBoardDeleted,
			BoardID: b.ID,
			Board:   &bCopy,
		})
	}
	return nil
}

// CreateThread validates opt-in, consent, bans, rate limit, and opens a new thread with >>1.
func (s *Service) CreateThread(ctx context.Context, input CreateThreadInput) (*Board, *Thread, *Post, error) {
	if _, err := s.EnsureGuildEnabled(ctx, input.GuildID); err != nil {
		return nil, nil, nil, err
	}
	if _, err := s.CheckConsent(ctx, input.UserID); err != nil {
		return nil, nil, nil, err
	}

	title, err := validateBasicThreadInput(input.Title, input.RawName, input.RawBody)
	if err != nil {
		return nil, nil, nil, err
	}

	board, err := s.GetBoard(ctx, input.BoardID)
	if err != nil {
		return nil, nil, nil, err
	}
	if board.Scope == ScopeLocal && (board.GuildID == nil || *board.GuildID != input.GuildID) {
		return nil, nil, nil, fmt.Errorf("%w: local board guild mismatch", ErrUnauthorized)
	}
	if board.Locked {
		return nil, nil, nil, ErrBoardLocked
	}
	p, err := s.Registry.Get(board.ProviderKey)
	if err != nil {
		return nil, nil, nil, err
	}
	if fp, ok := p.(FederatedProvider); ok {
		if !fp.Capabilities().CanCreateThread {
			return nil, nil, nil, ErrFederationReadOnly
		}
		if SanitizeSingleLine(input.CapID) == "" && !s.checkThreadRateLimit(input.UserID) {
			return nil, nil, nil, ErrRateLimited
		}
		input.Title = title
		return fp.CreateRemoteThread(ctx, input)
	}
	if !p.Capabilities().CanCreateThread {
		return nil, nil, nil, ErrUnsupportedOperation
	}

	cap, err := s.ResolveUserCap(ctx, input.UserID, board.ID, input.CapID)
	if err != nil {
		return nil, nil, nil, err
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	parsed, err := s.Identity.ParsePostIdentity(
		board.ID,
		board.Scope,
		input.GuildID,
		input.UserID,
		input.RawName,
		input.RawBody,
		board.Rules,
		true,
		cap,
		input.SlipContext,
		now,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if parsed.CleanBody == "" || utf8.RuneCountInString(parsed.CleanBody) > MaxBodyRunes {
		return nil, nil, nil, ErrInvalidBodyLength
	}
	formattedBody := ClearFormat(parsed.CleanBody)
	if strings.TrimSpace(formattedBody) == "" {
		return nil, nil, nil, ErrInvalidBodyLength
	}
	if utf8.RuneCountInString(formattedBody) > MaxEscapedBodyRunes {
		return nil, nil, nil, ErrFormattedBodyTooLong
	}

	banned, err := s.Store.IsBanned(ctx, board.Scope, input.GuildID, parsed.PosterHash, input.UserID, now)
	if err != nil {
		return nil, nil, nil, err
	}
	if banned {
		return nil, nil, nil, ErrBanned
	}

	if cap == nil && !s.checkThreadRateLimit(input.UserID) {
		return nil, nil, nil, ErrRateLimited
	}

	thread, post, err := p.CreateThread(ctx, CreateThreadRecord{
		BoardID:         board.ID,
		Title:           title,
		Rules:           parsed.ThreadRules,
		ExtDat:          parsed.ExtDat,
		OwnerPosterHash: parsed.PosterHash,
		OwnerAnonID:     parsed.AnonID,
		FirstPost: CreatePostRecord{
			Name:         parsed.DisplayName,
			Trip:         parsed.Trip,
			Cap:          parsed.CapName,
			SlipText:     parsed.SlipText,
			AnonID:       parsed.AnonID,
			PosterHash:   parsed.PosterHash,
			IsOP:         true,
			Body:         parsed.CleanBody,
			AuthorUserID: input.UserID,
		},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	s.recordPostReferences(ctx, thread.ID, 1, parsed.CleanBody, parsed.ThreadRules.MaxRes)
	if s.isNativeGlobalBoard(board) {
		bCopy := *board
		tCopy := *thread
		pCopy := *post
		s.emitEvent(Event{
			Type:       EventThreadCreated,
			BoardID:    board.ID,
			ThreadID:   thread.ID,
			PostID:     post.ID,
			Board:      &bCopy,
			Thread:     &tCopy,
			Post:       &pCopy,
			OccurredAt: now,
		})
	}
	return board, thread, post, nil
}

// CreatePost validates opt-in, consent, bans, rate limit, thread status, and appends a reply atomically.
func (s *Service) CreatePost(ctx context.Context, input CreatePostInput) (*Board, *Thread, *Post, []uint32, error) {
	if _, err := s.EnsureGuildEnabled(ctx, input.GuildID); err != nil {
		return nil, nil, nil, nil, err
	}
	if _, err := s.CheckConsent(ctx, input.UserID); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := validateBasicPostInput(input.RawName, input.RawBody); err != nil {
		return nil, nil, nil, nil, err
	}

	thread, err := s.GetThread(ctx, input.ThreadID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	board, err := s.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if board.Scope == ScopeLocal && (board.GuildID == nil || *board.GuildID != input.GuildID) {
		return nil, nil, nil, nil, fmt.Errorf("%w: local board guild mismatch", ErrUnauthorized)
	}
	if board.Locked {
		return nil, nil, nil, nil, ErrBoardLocked
	}
	p, err := s.Registry.Get(board.ProviderKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if fp, ok := p.(FederatedProvider); ok {
		if !fp.Capabilities().CanCreatePost {
			return nil, nil, nil, nil, ErrFederationReadOnly
		}
		if thread.Status != ThreadStatusActive {
			return nil, nil, nil, nil, ErrThreadArchived
		}
		if thread.ResCount >= thread.Rules.Normalize().MaxRes {
			return nil, nil, nil, nil, ErrMaxResReached
		}
		if SanitizeSingleLine(input.CapID) == "" && !s.checkPostRateLimit(input.UserID) {
			return nil, nil, nil, nil, ErrRateLimited
		}
		return fp.CreateRemotePost(ctx, input)
	}
	if !p.Capabilities().CanCreatePost {
		return nil, nil, nil, nil, ErrUnsupportedOperation
	}

	cap, err := s.ResolveUserCap(ctx, input.UserID, board.ID, input.CapID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	mu := s.threadMutex(thread.ID)
	mu.Lock()
	defer mu.Unlock()

	// Re-read thread inside lock to avoid stale status/count checks.
	thread, err = p.GetThread(ctx, input.ThreadID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if thread.Status != ThreadStatusActive {
		return nil, nil, nil, nil, ErrThreadArchived
	}
	maxRes := thread.Rules.Normalize().MaxRes
	if thread.ResCount >= maxRes {
		return nil, nil, nil, nil, ErrMaxResReached
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	parsed, err := s.Identity.ParsePostIdentity(
		board.ID,
		board.Scope,
		input.GuildID,
		input.UserID,
		input.RawName,
		input.RawBody,
		thread.Rules,
		false,
		cap,
		input.SlipContext,
		now,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if parsed.CleanBody == "" || utf8.RuneCountInString(parsed.CleanBody) > MaxBodyRunes {
		return nil, nil, nil, nil, ErrInvalidBodyLength
	}
	formattedBody := ClearFormat(parsed.CleanBody)
	if strings.TrimSpace(formattedBody) == "" {
		return nil, nil, nil, nil, ErrInvalidBodyLength
	}
	if utf8.RuneCountInString(formattedBody) > MaxEscapedBodyRunes {
		return nil, nil, nil, nil, ErrFormattedBodyTooLong
	}

	banned, err := s.Store.IsBanned(ctx, board.Scope, input.GuildID, parsed.PosterHash, input.UserID, now)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if banned {
		return nil, nil, nil, nil, ErrBanned
	}

	if cap == nil && !s.checkPostRateLimit(input.UserID) {
		return nil, nil, nil, nil, ErrRateLimited
	}

	isOP := thread.OwnerPosterHash != "" && parsed.PosterHash == thread.OwnerPosterHash
	post, updatedThread, err := p.CreatePost(ctx, CreatePostRecord{
		ThreadID:     thread.ID,
		Name:         parsed.DisplayName,
		Trip:         parsed.Trip,
		Cap:          parsed.CapName,
		SlipText:     parsed.SlipText,
		AnonID:       parsed.AnonID,
		PosterHash:   parsed.PosterHash,
		IsOP:         isOP,
		Body:         parsed.CleanBody,
		AuthorUserID: input.UserID,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	s.recordPostReferences(ctx, thread.ID, post.ResNumber, parsed.CleanBody, maxRes)
	backwardRefs, refErr := s.Store.ListReferencingRes(ctx, thread.ID, post.ResNumber)
	if refErr != nil {
		slog.Warn("bbs failed to list referencing res", "thread_id", thread.ID, "res_number", post.ResNumber, "err", refErr)
	}
	if s.isNativeGlobalBoard(board) {
		bCopy := *board
		tCopy := *updatedThread
		pCopy := *post
		s.emitEvent(Event{
			Type:         EventPostCreated,
			BoardID:      board.ID,
			ThreadID:     updatedThread.ID,
			PostID:       post.ID,
			Board:        &bCopy,
			Thread:       &tCopy,
			Post:         &pCopy,
			BackwardRefs: slices.Clone(backwardRefs),
			OccurredAt:   now,
		})
		if updatedThread.Status == ThreadStatusArchived {
			s.emitEvent(Event{
				Type:       EventThreadArchived,
				BoardID:    board.ID,
				ThreadID:   updatedThread.ID,
				Board:      &bCopy,
				Thread:     &tCopy,
				HitMaxRes:  post.ResNumber >= maxRes,
				OccurredAt: now,
			})
		}
	}
	return board, updatedThread, post, backwardRefs, nil
}

// ListPosts lists posts for a thread via its board provider.
func (s *Service) ListPosts(ctx context.Context, threadID uuid.UUID, offset, limit int) ([]Post, error) {
	thread, err := s.GetThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	p, err := s.Registry.Get(thread.ProviderKey)
	if err != nil {
		return nil, err
	}
	return p.ListPosts(ctx, threadID, offset, limit)
}

// GetThread resolves thread metadata across registered providers.
func (s *Service) GetThread(ctx context.Context, threadID uuid.UUID) (*Thread, error) {
	providers := s.Registry.All()
	if len(providers) == 0 {
		return nil, ErrProviderNotFound
	}
	defaultKey := s.Registry.DefaultKey()
	var remoteErr error
	for _, p := range providers {
		th, err := p.GetThread(ctx, threadID)
		if err == nil {
			return th, nil
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrUnsupportedOperation) {
			if p.Key() != defaultKey {
				if remoteErr == nil {
					remoteErr = err
				}
				continue
			}
			return nil, err
		}
	}
	if remoteErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrFederationUnavailable, remoteErr)
	}
	return nil, ErrNotFound
}

// GetPost resolves a single post across registered providers.
func (s *Service) GetPost(ctx context.Context, postID uuid.UUID) (*Post, error) {
	providers := s.Registry.All()
	if len(providers) == 0 {
		return nil, ErrProviderNotFound
	}
	defaultKey := s.Registry.DefaultKey()
	var remoteErr error
	for _, p := range providers {
		post, err := p.GetPost(ctx, postID)
		if err == nil {
			return post, nil
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrUnsupportedOperation) {
			if p.Key() != defaultKey {
				if remoteErr == nil {
					remoteErr = err
				}
				continue
			}
			return nil, err
		}
	}
	if remoteErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrFederationUnavailable, remoteErr)
	}
	return nil, ErrNotFound
}

// ListThreads lists threads matching filter via the target board's provider (or default provider when BoardID is zero).
func (s *Service) ListThreads(ctx context.Context, filter ThreadFilter) ([]Thread, error) {
	var providerKey string
	if filter.BoardID != uuid.Nil {
		b, err := s.GetBoard(ctx, filter.BoardID)
		if err != nil {
			return nil, err
		}
		providerKey = b.ProviderKey
	}
	p, err := s.Registry.Get(providerKey)
	if err != nil {
		return nil, err
	}
	return p.ListThreads(ctx, filter)
}

// AbonePostLocal soft-deletes a post on a local board owned by guildID.
func (s *Service) AbonePostLocal(ctx context.Context, guildID snowflake.ID, postID uuid.UUID, reason string) (*Post, error) {
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	thread, err := s.GetThread(ctx, post.ThreadID)
	if err != nil {
		return nil, err
	}
	board, err := s.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, err
	}
	if board.Scope != ScopeLocal || board.GuildID == nil || *board.GuildID != guildID {
		return nil, fmt.Errorf("%w: local moderators cannot abone global or foreign posts", ErrUnauthorized)
	}
	p, err := s.Registry.Get(board.ProviderKey)
	if err != nil {
		return nil, err
	}
	return p.AbonePost(ctx, postID, SanitizeSingleLine(reason))
}

// AbonePostGlobal soft-deletes any post across all mirrored guilds (operator-only).
func (s *Service) AbonePostGlobal(ctx context.Context, operatorID snowflake.ID, postID uuid.UUID, reason string) (*Post, error) {
	if !s.IsOperator(operatorID) {
		return nil, ErrUnauthorized
	}
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	thread, err := s.GetThread(ctx, post.ThreadID)
	if err != nil {
		return nil, err
	}
	p, err := s.Registry.Get(thread.ProviderKey)
	if err != nil {
		return nil, err
	}
	aboned, err := p.AbonePost(ctx, postID, SanitizeSingleLine(reason))
	if err != nil {
		return nil, err
	}
	if board, bErr := s.GetBoard(ctx, thread.BoardID); bErr == nil && s.isNativeGlobalBoard(board) {
		pCopy := *aboned
		tCopy := *thread
		s.emitEvent(Event{
			Type:     EventPostAboned,
			BoardID:  board.ID,
			ThreadID: thread.ID,
			PostID:   aboned.ID,
			Thread:   &tCopy,
			Post:     &pCopy,
		})
	}
	return aboned, nil
}

// ArchiveThreadLocal archives a local board thread owned by guildID.
func (s *Service) ArchiveThreadLocal(ctx context.Context, guildID snowflake.ID, threadID uuid.UUID) (*Thread, error) {
	thread, err := s.GetThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	board, err := s.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, err
	}
	if board.Scope != ScopeLocal || board.GuildID == nil || *board.GuildID != guildID {
		return nil, ErrUnauthorized
	}
	p, err := s.Registry.Get(board.ProviderKey)
	if err != nil {
		return nil, err
	}
	return p.ArchiveThread(ctx, threadID)
}

// BanLocal bans an anonymous PosterHash within a guild's local boards.
func (s *Service) BanLocal(ctx context.Context, guildID snowflake.ID, actorID snowflake.ID, posterHash string, reason string, duration *time.Duration) (*Ban, error) {
	if posterHash == "" {
		return nil, fmt.Errorf("%w: poster hash required", ErrInvalidInput)
	}
	var expiresAt *time.Time
	if duration != nil && *duration > 0 {
		t := time.Now().Add(*duration)
		expiresAt = &t
	}
	ban := &Ban{
		ID:         uuidv7.New(),
		Scope:      ScopeLocal,
		GuildID:    &guildID,
		PosterHash: posterHash,
		Reason:     SanitizeSingleLine(reason),
		CreatedBy:  actorID,
		ExpiresAt:  expiresAt,
		CreatedAt:  time.Now(),
	}
	if err := s.Store.CreateBan(ctx, ban); err != nil {
		return nil, err
	}
	return ban, nil
}

// BanGlobal bans a user/poster hash globally (operator-only).
func (s *Service) BanGlobal(ctx context.Context, operatorID snowflake.ID, postID uuid.UUID, reason string, duration *time.Duration) (*Ban, error) {
	if !s.IsOperator(operatorID) {
		return nil, ErrUnauthorized
	}
	audit, err := s.Store.GetAuditPostInfo(ctx, postID)
	if err != nil {
		return nil, err
	}
	var expiresAt *time.Time
	if duration != nil && *duration > 0 {
		t := time.Now().Add(*duration)
		expiresAt = &t
	}
	targetID := audit.AuthorUserID
	ban := &Ban{
		ID:           uuidv7.New(),
		Scope:        ScopeGlobal,
		PosterHash:   audit.Post.PosterHash,
		TargetUserID: &targetID,
		Reason:       SanitizeSingleLine(reason),
		CreatedBy:    operatorID,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}
	if err := s.Store.CreateBan(ctx, ban); err != nil {
		return nil, err
	}
	return ban, nil
}

// CreateReport records a user report against a post.
func (s *Service) CreateReport(ctx context.Context, guildID snowflake.ID, reporterID snowflake.ID, postID uuid.UUID, reason string) (*Report, error) {
	if _, err := s.CheckConsent(ctx, reporterID); err != nil {
		return nil, err
	}
	reason = SanitizeSingleLine(reason)
	if reason == "" {
		return nil, ErrReportReasonRequired
	}
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	thread, err := s.GetThread(ctx, post.ThreadID)
	if err != nil {
		return nil, err
	}
	board, err := s.GetBoard(ctx, thread.BoardID)
	if err != nil {
		return nil, err
	}
	if board.Scope == ScopeLocal && (board.GuildID == nil || *board.GuildID != guildID) {
		return nil, fmt.Errorf("%w: local post guild mismatch", ErrUnauthorized)
	}
	if p, pErr := s.Registry.Get(board.ProviderKey); pErr == nil {
		if fp, ok := p.(FederatedProvider); ok {
			if !fp.CanReport() {
				return nil, ErrFederationReadOnly
			}
			return fp.CreateRemoteReport(ctx, guildID, reporterID, postID, reason)
		}
	}
	now := time.Now()
	report := &Report{
		ID:             uuidv7.New(),
		Scope:          board.Scope,
		GuildID:        guildID,
		PostID:         post.ID,
		ThreadID:       thread.ID,
		BoardID:        board.ID,
		ReporterUserID: reporterID,
		Reason:         reason,
		Status:         ReportStatusOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.Store.CreateReport(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

// InspectReportForOperator returns operator-only audit attribution for a report.
func (s *Service) InspectReportForOperator(ctx context.Context, operatorID snowflake.ID, reportID uuid.UUID) (*Report, *AuditPostInfo, error) {
	if !s.IsOperator(operatorID) {
		return nil, nil, ErrUnauthorized
	}
	report, err := s.Store.GetReport(ctx, reportID)
	if err != nil {
		return nil, nil, err
	}
	audit, err := s.Store.GetAuditPostInfo(ctx, report.PostID)
	if err != nil {
		return nil, nil, err
	}
	return report, audit, nil
}

func (s *Service) recordPostReferences(ctx context.Context, threadID uuid.UUID, fromRes uint32, body string, maxRes uint32) {
	anchors := ExtractAnchors(body, maxRes)
	if len(anchors) == 0 {
		return
	}
	refs := make([]PostReference, 0, len(anchors))
	for _, toRes := range anchors {
		if toRes == fromRes {
			continue
		}
		refs = append(refs, PostReference{
			ThreadID: threadID,
			FromRes:  fromRes,
			ToRes:    toRes,
		})
	}
	if len(refs) > 0 {
		if err := s.Store.SavePostReferences(ctx, refs); err != nil {
			slog.Warn("bbs failed to save post references", "thread_id", threadID, "from_res", fromRes, "err", err)
		}
	}
}

func pruneRateLimitMapLocked(m map[snowflake.ID][]time.Time, maxUnit time.Duration, now time.Time) {
	for uid, times := range m {
		if len(times) == 0 || now.Sub(times[0]) > maxUnit {
			delete(m, uid)
		}
	}
}

// PruneRateLimits removes expired per-user rate-limit entries from memory.
func (s *Service) PruneRateLimits(now time.Time) {
	s.threadRLMu.Lock()
	pruneRateLimitMapLocked(s.threadRL, 10*time.Minute, now)
	s.threadRLSweep = now
	s.threadRLMu.Unlock()

	s.postRLMu.Lock()
	pruneRateLimitMapLocked(s.postRL, 30*time.Second, now)
	s.postRLSweep = now
	s.postRLMu.Unlock()
}

func (s *Service) checkThreadRateLimit(userID snowflake.ID) bool {
	s.threadRLMu.Lock()
	defer s.threadRLMu.Unlock()
	now := time.Now()
	if s.threadRLSweep.IsZero() || now.Sub(s.threadRLSweep) >= 10*time.Minute {
		pruneRateLimitMapLocked(s.threadRL, 10*time.Minute, now)
		s.threadRLSweep = now
	}
	next, ok := ratelimit.CheckLimit(s.threadRL[userID], s.threadRules)
	s.threadRL[userID] = next
	return ok
}

func (s *Service) checkPostRateLimit(userID snowflake.ID) bool {
	s.postRLMu.Lock()
	defer s.postRLMu.Unlock()
	now := time.Now()
	if s.postRLSweep.IsZero() || now.Sub(s.postRLSweep) >= 10*time.Minute {
		pruneRateLimitMapLocked(s.postRL, 30*time.Second, now)
		s.postRLSweep = now
	}
	next, ok := ratelimit.CheckLimit(s.postRL[userID], s.postRules)
	s.postRL[userID] = next
	return ok
}
