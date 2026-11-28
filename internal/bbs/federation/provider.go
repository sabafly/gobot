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

package federation

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

const tokenRefreshSkew = 2 * time.Minute

// RemoteProvider implements bbs.Provider and bbs.FederatedProvider backed by an Origin bot instance.
type RemoteProvider struct {
	cfg             RemoteConfig
	key             string
	client          *Client
	store           bbs.Store
	cipher          *TokenCipher
	callbackEnabled bool

	capMu    sync.RWMutex
	hasCaps  bool
	lastCaps CapabilitiesResponse

	metaMu      sync.RWMutex
	metaExpires time.Time
	categories  []bbs.Category
	boards      []bbs.Board
	boardByID   map[uuid.UUID]bbs.Board

	userTokenLocks [64]sync.Mutex
}

var _ bbs.Provider = (*RemoteProvider)(nil)
var _ bbs.FederatedProvider = (*RemoteProvider)(nil)

// RemoteProviderDeps configures a RemoteProvider bound to one Origin server.
type RemoteProviderDeps struct {
	Config          RemoteConfig
	Store           bbs.Store
	Cipher          *TokenCipher
	CallbackEnabled bool
	HTTPClient      *Client
}

// NewRemoteProvider validates remote configuration and builds a RemoteProvider.
func NewRemoteProvider(deps RemoteProviderDeps) (*RemoteProvider, error) {
	resolved, err := ResolveRemoteConfig(deps.Config)
	if err != nil {
		return nil, err
	}
	client := deps.HTTPClient
	if client == nil {
		client = NewClient(resolved)
	}
	return &RemoteProvider{
		cfg:             resolved,
		key:             RemoteKeyPrefix + resolved.Key,
		client:          client,
		store:           deps.Store,
		cipher:          deps.Cipher,
		callbackEnabled: deps.CallbackEnabled,
		boardByID:       make(map[uuid.UUID]bbs.Board),
	}, nil
}

func (p *RemoteProvider) Key() string          { return p.key }
func (p *RemoteProvider) RemoteKey() string    { return p.cfg.Key }
func (p *RemoteProvider) DisplayLabel() string { return p.cfg.Label }
func (p *RemoteProvider) Client() *Client      { return p.client }

func (p *RemoteProvider) userLock(userID snowflake.ID) *sync.Mutex {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(userID))
	idx := binary.LittleEndian.Uint64(buf[:]) % uint64(len(p.userTokenLocks))
	return &p.userTokenLocks[idx]
}

// UpdateCapabilities applies the latest Origin /capabilities snapshot.
func (p *RemoteProvider) UpdateCapabilities(caps CapabilitiesResponse) {
	p.capMu.Lock()
	p.hasCaps = true
	p.lastCaps = caps
	p.capMu.Unlock()
}

// EnsureCapabilities returns the cached Origin capabilities or fetches them lazily.
func (p *RemoteProvider) EnsureCapabilities(ctx context.Context) (CapabilitiesResponse, error) {
	p.capMu.RLock()
	has := p.hasCaps
	cached := p.lastCaps
	p.capMu.RUnlock()
	if has {
		return cached, nil
	}
	fetched, err := p.client.GetCapabilities(ctx)
	if err != nil {
		return CapabilitiesResponse{}, err
	}
	p.UpdateCapabilities(*fetched)
	return *fetched, nil
}

func (p *RemoteProvider) Capabilities() bbs.ProviderCapabilities {
	p.capMu.RLock()
	has := p.hasCaps
	caps := p.lastCaps
	p.capMu.RUnlock()

	canWrite := has && caps.AllowWrite && !p.cfg.ReadOnly && p.callbackEnabled
	return bbs.ProviderCapabilities{
		CanCreateBoard:  false,
		CanCreateThread: canWrite,
		CanCreatePost:   canWrite,
		CanModerate:     false,
	}
}

func (p *RemoteProvider) CanReport() bool {
	p.capMu.RLock()
	has := p.hasCaps
	caps := p.lastCaps
	p.capMu.RUnlock()
	return has && caps.AllowReport && !p.cfg.ReadOnly && p.callbackEnabled
}

func (p *RemoteProvider) CanUseCaps() bool {
	p.capMu.RLock()
	has := p.hasCaps
	caps := p.lastCaps
	p.capMu.RUnlock()
	return has && caps.AllowCaps && !p.cfg.ReadOnly && p.callbackEnabled
}

// InvalidateMetadataCache drops the cached category/board lists (e.g. on SSE board.updated/deleted).
func (p *RemoteProvider) InvalidateMetadataCache() {
	p.metaMu.Lock()
	p.metaExpires = time.Time{}
	p.categories = nil
	p.boards = nil
	p.boardByID = make(map[uuid.UUID]bbs.Board)
	p.metaMu.Unlock()
}

func (p *RemoteProvider) rewriteCategory(c bbs.Category) bbs.Category {
	c.ProviderKey = p.key
	return c
}

func (p *RemoteProvider) rewriteBoard(b bbs.Board) bbs.Board {
	b.ProviderKey = p.key
	return b
}

func (p *RemoteProvider) rewriteThread(t bbs.Thread) bbs.Thread {
	t.ProviderKey = p.key
	return t
}

func (p *RemoteProvider) refreshMetadataCache(ctx context.Context) error {
	// Opportunistically seed capabilities on first metadata sync so Capabilities() reflects Origin before first write.
	_, _ = p.EnsureCapabilities(ctx)

	cats, err := p.client.ListCategories(ctx)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return MapAPIErrorToBBS(apiErr)
		}
		return err
	}
	boards, err := p.client.ListBoards(ctx, nil)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return MapAPIErrorToBBS(apiErr)
		}
		return err
	}

	rwCats := make([]bbs.Category, len(cats))
	for i, c := range cats {
		rwCats[i] = p.rewriteCategory(c)
	}
	rwBoards := make([]bbs.Board, len(boards))
	byID := make(map[uuid.UUID]bbs.Board, len(boards))
	for i, b := range boards {
		rb := p.rewriteBoard(b)
		rwBoards[i] = rb
		byID[rb.ID] = rb
	}

	p.metaMu.Lock()
	p.categories = rwCats
	p.boards = rwBoards
	p.boardByID = byID
	p.metaExpires = time.Now().Add(p.cfg.CacheTTL)
	p.metaMu.Unlock()
	return nil
}

// --- CategoryProvider ---

func (p *RemoteProvider) GetCategory(ctx context.Context, id uuid.UUID) (*bbs.Category, error) {
	p.metaMu.RLock()
	fresh := time.Now().Before(p.metaExpires)
	var found *bbs.Category
	if fresh {
		for _, c := range p.categories {
			if c.ID == id {
				cp := c
				found = &cp
				break
			}
		}
	}
	p.metaMu.RUnlock()

	if found != nil {
		return found, nil
	}
	if !fresh {
		if err := p.refreshMetadataCache(ctx); err != nil {
			return nil, err
		}
		p.metaMu.RLock()
		for _, c := range p.categories {
			if c.ID == id {
				cp := c
				found = &cp
				break
			}
		}
		p.metaMu.RUnlock()
		if found != nil {
			return found, nil
		}
	}
	return nil, bbs.ErrNotFound
}

func (p *RemoteProvider) ListCategories(ctx context.Context, filter bbs.CategoryFilter) ([]bbs.Category, error) {
	if filter.Scope != nil && *filter.Scope != bbs.ScopeGlobal {
		return nil, nil
	}
	if filter.GuildID != nil && *filter.GuildID != 0 {
		return nil, nil
	}
	p.metaMu.RLock()
	fresh := time.Now().Before(p.metaExpires)
	cached := slices.Clone(p.categories)
	p.metaMu.RUnlock()

	if !fresh {
		if err := p.refreshMetadataCache(ctx); err != nil {
			return nil, err
		}
		p.metaMu.RLock()
		cached = slices.Clone(p.categories)
		p.metaMu.RUnlock()
	}
	return cached, nil
}

func (p *RemoteProvider) CreateCategory(_ context.Context, _ bbs.CreateCategoryInput) (*bbs.Category, error) {
	return nil, bbs.ErrUnsupportedOperation
}

func (p *RemoteProvider) UpdateCategory(_ context.Context, _ uuid.UUID, _ bbs.UpdateCategoryInput) (*bbs.Category, error) {
	return nil, bbs.ErrUnsupportedOperation
}

func (p *RemoteProvider) DeleteCategory(_ context.Context, _ uuid.UUID) error {
	return bbs.ErrUnsupportedOperation
}

// --- BoardProvider ---

func (p *RemoteProvider) GetBoard(ctx context.Context, id uuid.UUID) (*bbs.Board, error) {
	p.metaMu.RLock()
	if time.Now().Before(p.metaExpires) {
		if b, ok := p.boardByID[id]; ok {
			p.metaMu.RUnlock()
			return &b, nil
		}
	}
	p.metaMu.RUnlock()

	_, _ = p.EnsureCapabilities(ctx)
	b, err := p.client.GetBoard(ctx, id)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return nil, MapAPIErrorToBBS(apiErr)
		}
		return nil, err
	}
	rb := p.rewriteBoard(*b)
	p.metaMu.Lock()
	if p.boardByID == nil {
		p.boardByID = make(map[uuid.UUID]bbs.Board)
	}
	p.boardByID[rb.ID] = rb
	p.metaMu.Unlock()
	return &rb, nil
}

func (p *RemoteProvider) ListBoards(ctx context.Context, filter bbs.BoardFilter) ([]bbs.Board, error) {
	if filter.Scope != nil && *filter.Scope != bbs.ScopeGlobal {
		return nil, nil
	}
	if filter.GuildID != nil && *filter.GuildID != 0 {
		return nil, nil
	}

	p.metaMu.RLock()
	fresh := time.Now().Before(p.metaExpires)
	cached := slices.Clone(p.boards)
	p.metaMu.RUnlock()

	if !fresh {
		if err := p.refreshMetadataCache(ctx); err != nil {
			return nil, err
		}
		p.metaMu.RLock()
		cached = slices.Clone(p.boards)
		p.metaMu.RUnlock()
	}

	if filter.CategoryID == nil {
		return cached, nil
	}
	var filtered []bbs.Board
	for _, b := range cached {
		if b.CategoryID == *filter.CategoryID {
			filtered = append(filtered, b)
		}
	}
	return filtered, nil
}

func (p *RemoteProvider) CreateBoard(_ context.Context, _ bbs.CreateBoardInput) (*bbs.Board, error) {
	return nil, bbs.ErrUnsupportedOperation
}

func (p *RemoteProvider) UpdateBoard(_ context.Context, _ uuid.UUID, _ bbs.UpdateBoardInput) (*bbs.Board, error) {
	return nil, bbs.ErrUnsupportedOperation
}

func (p *RemoteProvider) DeleteBoard(_ context.Context, _ uuid.UUID) error {
	return bbs.ErrUnsupportedOperation
}

// --- ThreadProvider ---

func (p *RemoteProvider) GetThread(ctx context.Context, id uuid.UUID) (*bbs.Thread, error) {
	th, err := p.client.GetThread(ctx, id)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return nil, MapAPIErrorToBBS(apiErr)
		}
		return nil, err
	}
	rt := p.rewriteThread(*th)
	return &rt, nil
}

func (p *RemoteProvider) ListThreads(ctx context.Context, filter bbs.ThreadFilter) ([]bbs.Thread, error) {
	threads, err := p.client.ListThreads(ctx, filter)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return nil, MapAPIErrorToBBS(apiErr)
		}
		return nil, err
	}
	for i := range threads {
		threads[i] = p.rewriteThread(threads[i])
	}
	return threads, nil
}

func (p *RemoteProvider) CreateThread(_ context.Context, _ bbs.CreateThreadRecord) (*bbs.Thread, *bbs.Post, error) {
	return nil, nil, bbs.ErrUnsupportedOperation
}

func (p *RemoteProvider) ArchiveThread(_ context.Context, _ uuid.UUID) (*bbs.Thread, error) {
	return nil, bbs.ErrUnsupportedOperation
}

func (p *RemoteProvider) ListStaleThreads(_ context.Context, _ time.Time) ([]bbs.Thread, error) {
	return nil, nil
}

// --- PostProvider ---

func (p *RemoteProvider) GetPost(ctx context.Context, id uuid.UUID) (*bbs.Post, error) {
	post, err := p.client.GetPost(ctx, id)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return nil, MapAPIErrorToBBS(apiErr)
		}
		return nil, err
	}
	return post, nil
}

func (p *RemoteProvider) GetPostByNumber(ctx context.Context, threadID uuid.UUID, resNumber uint32) (*bbs.Post, error) {
	if resNumber == 0 {
		return nil, bbs.ErrNotFound
	}
	posts, err := p.client.ListPosts(ctx, threadID, int(resNumber-1), 1)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return nil, MapAPIErrorToBBS(apiErr)
		}
		return nil, err
	}
	for _, post := range posts {
		if post.ResNumber == resNumber {
			cp := post
			return &cp, nil
		}
	}
	return nil, bbs.ErrNotFound
}

func (p *RemoteProvider) ListPosts(ctx context.Context, threadID uuid.UUID, offset, limit int) ([]bbs.Post, error) {
	posts, err := p.client.ListPosts(ctx, threadID, offset, limit)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			return nil, MapAPIErrorToBBS(apiErr)
		}
		return nil, err
	}
	return posts, nil
}

func (p *RemoteProvider) CreatePost(_ context.Context, _ bbs.CreatePostRecord) (*bbs.Post, *bbs.Thread, error) {
	return nil, nil, bbs.ErrUnsupportedOperation
}

func (p *RemoteProvider) AbonePost(_ context.Context, _ uuid.UUID, _ string) (*bbs.Post, error) {
	return nil, bbs.ErrUnsupportedOperation
}

// --- Delegated OAuth Token Management & Federated Write/Report/Caps ---

func (p *RemoteProvider) deleteLocalUserToken(ctx context.Context, userID snowflake.ID) {
	if p.store == nil || userID == 0 {
		return
	}
	if err := p.store.DeleteFederationUserToken(ctx, p.cfg.Key, userID); err != nil {
		slog.Warn("federation failed to delete stale local user token", "remote", p.cfg.Key, "user_id", userID, "err", err)
	}
}

var errDelegationScopeMissing = errors.New("federation delegation lacks required scope")

func (p *RemoteProvider) ensureAccessToken(ctx context.Context, userID snowflake.ID, forceRefresh bool) (string, []string, error) {
	return p.ensureAccessTokenForScope(ctx, userID, forceRefresh, "")
}

func (p *RemoteProvider) ensureAccessTokenForScope(ctx context.Context, userID snowflake.ID, forceRefresh bool, requiredScope string) (string, []string, error) {
	if p.store == nil || p.cipher == nil || userID == 0 {
		return "", nil, bbs.ErrFederationAuthRequired
	}
	mu := p.userLock(userID)
	mu.Lock()
	defer mu.Unlock()

	tok, err := p.store.GetFederationUserToken(ctx, p.cfg.Key, userID)
	if err != nil {
		if errors.Is(err, bbs.ErrNotFound) {
			return "", nil, bbs.ErrFederationAuthRequired
		}
		return "", nil, err
	}
	if requiredScope != "" && !tok.HasScope(requiredScope) {
		return "", nil, errDelegationScopeMissing
	}

	caps, cErr := p.EnsureCapabilities(ctx)
	if cErr != nil {
		return "", nil, cErr
	}
	if caps.PolicyVersion != "" && tok.PolicyVersion != caps.PolicyVersion {
		p.deleteLocalUserToken(ctx, userID)
		return "", nil, bbs.ErrFederationAuthRequired
	}

	now := time.Now().UTC()
	if !forceRefresh && now.Add(tokenRefreshSkew).Before(tok.AccessExpiresAt) {
		plainAccess, dErr := p.cipher.Decrypt(tok.AccessTokenEnc)
		if dErr != nil {
			p.deleteLocalUserToken(ctx, userID)
			return "", nil, bbs.ErrFederationAuthRequired
		}
		return plainAccess, slices.Clone(tok.Scopes), nil
	}

	if !now.Before(tok.RefreshExpiresAt) {
		p.deleteLocalUserToken(ctx, userID)
		return "", nil, bbs.ErrFederationAuthRequired
	}
	plainRefresh, dErr := p.cipher.Decrypt(tok.RefreshTokenEnc)
	if dErr != nil {
		p.deleteLocalUserToken(ctx, userID)
		return "", nil, bbs.ErrFederationAuthRequired
	}

	refResp, rErr := p.client.RefreshToken(ctx, plainRefresh)
	if rErr != nil {
		if apiErr, ok := asAPIError(rErr); ok {
			switch apiErr.Code {
			case CodeBanned:
				p.deleteLocalUserToken(ctx, userID)
				return "", nil, MapAPIErrorToBBS(apiErr)
			case CodeInvalidGrant, CodeTokenRevoked, CodePolicyConsentRequired, CodeAuthRequired, CodeUnauthorized:
				p.deleteLocalUserToken(ctx, userID)
				return "", nil, bbs.ErrFederationAuthRequired
			}
			return "", nil, MapAPIErrorToBBS(apiErr)
		}
		return "", nil, rErr
	}

	accEnc, err := p.cipher.Encrypt(refResp.AccessToken)
	if err != nil {
		return "", nil, err
	}
	refEnc, err := p.cipher.Encrypt(refResp.RefreshToken)
	if err != nil {
		return "", nil, err
	}
	updated := &bbs.FederationUserToken{
		RemoteKey:        p.cfg.Key,
		UserID:           userID,
		AccessTokenEnc:   accEnc,
		AccessExpiresAt:  now.Add(time.Duration(refResp.ExpiresIn) * time.Second),
		RefreshTokenEnc:  refEnc,
		RefreshExpiresAt: now.Add(time.Duration(refResp.RefreshExpiresIn) * time.Second),
		Scopes:           refResp.Scopes,
		PolicyVersion:    refResp.PolicyVersion,
	}
	if err := p.store.UpsertFederationUserToken(ctx, updated); err != nil {
		return "", nil, err
	}
	return refResp.AccessToken, slices.Clone(refResp.Scopes), nil
}

func (p *RemoteProvider) HasUserDelegation(ctx context.Context, userID snowflake.ID) (bool, error) {
	_, _, err := p.ensureAccessToken(ctx, userID, false)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bbs.ErrFederationAuthRequired) {
		return false, nil
	}
	return false, err
}

func (p *RemoteProvider) RevokeUserDelegation(ctx context.Context, userID snowflake.ID) error {
	if userID == 0 {
		return nil
	}
	mu := p.userLock(userID)
	mu.Lock()
	defer mu.Unlock()

	var errs []error
	if err := p.client.RevokeUserGrant(ctx, userID); err != nil {
		if apiErr, ok := asAPIError(err); ok {
			errs = append(errs, MapAPIErrorToBBS(apiErr))
		} else {
			errs = append(errs, err)
		}
	}
	if p.store != nil {
		if err := p.store.DeleteFederationUserToken(ctx, p.cfg.Key, userID); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *RemoteProvider) mapRemoteWriteErr(ctx context.Context, userID snowflake.ID, err error) error {
	apiErr, ok := asAPIError(err)
	if !ok {
		return err
	}
	switch apiErr.Code {
	case CodeAuthRequired, CodeTokenRevoked, CodeInvalidGrant, CodePolicyConsentRequired:
		p.deleteLocalUserToken(ctx, userID)
		return fmt.Errorf("%w: %s", bbs.ErrFederationAuthRequired, apiErr.Message)
	case CodeBanned:
		p.deleteLocalUserToken(ctx, userID)
	}
	return MapAPIErrorToBBS(apiErr)
}

func (p *RemoteProvider) CreateRemoteThread(ctx context.Context, input bbs.CreateThreadInput) (*bbs.Board, *bbs.Thread, *bbs.Post, error) {
	if !p.Capabilities().CanCreateThread {
		return nil, nil, nil, bbs.ErrFederationReadOnly
	}
	accessTok, _, err := p.ensureAccessToken(ctx, input.UserID, false)
	if err != nil {
		return nil, nil, nil, err
	}
	req := CreateThreadRequest{
		Title:   input.Title,
		RawName: input.RawName,
		CapID:   input.CapID,
		RawBody: input.RawBody,
	}
	resp, err := p.client.CreateThread(ctx, input.BoardID, accessTok, req)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok && apiErr.Code == CodeTokenExpired {
			var rErr error
			accessTok, _, rErr = p.ensureAccessToken(ctx, input.UserID, true)
			if rErr != nil {
				return nil, nil, nil, rErr
			}
			resp, err = p.client.CreateThread(ctx, input.BoardID, accessTok, req)
		}
	}
	if err != nil {
		return nil, nil, nil, p.mapRemoteWriteErr(ctx, input.UserID, err)
	}
	rb := p.rewriteBoard(resp.Board)
	rt := p.rewriteThread(resp.Thread)
	rp := resp.Post
	return &rb, &rt, &rp, nil
}

func (p *RemoteProvider) CreateRemotePost(ctx context.Context, input bbs.CreatePostInput) (*bbs.Board, *bbs.Thread, *bbs.Post, []uint32, error) {
	if !p.Capabilities().CanCreatePost {
		return nil, nil, nil, nil, bbs.ErrFederationReadOnly
	}
	accessTok, _, err := p.ensureAccessToken(ctx, input.UserID, false)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	req := CreatePostRequest{
		RawName: input.RawName,
		CapID:   input.CapID,
		RawBody: input.RawBody,
	}
	resp, err := p.client.CreatePost(ctx, input.ThreadID, accessTok, req)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok && apiErr.Code == CodeTokenExpired {
			var rErr error
			accessTok, _, rErr = p.ensureAccessToken(ctx, input.UserID, true)
			if rErr != nil {
				return nil, nil, nil, nil, rErr
			}
			resp, err = p.client.CreatePost(ctx, input.ThreadID, accessTok, req)
		}
	}
	if err != nil {
		return nil, nil, nil, nil, p.mapRemoteWriteErr(ctx, input.UserID, err)
	}
	rb := p.rewriteBoard(resp.Board)
	rt := p.rewriteThread(resp.Thread)
	rp := resp.Post
	return &rb, &rt, &rp, resp.BackwardRefs, nil
}

func (p *RemoteProvider) CreateRemoteReport(ctx context.Context, guildID, reporterID snowflake.ID, postID uuid.UUID, reason string) (*bbs.Report, error) {
	if !p.CanReport() {
		return nil, bbs.ErrFederationReadOnly
	}
	accessTok, _, err := p.ensureAccessToken(ctx, reporterID, false)
	if err != nil {
		return nil, err
	}
	req := CreateReportRequest{
		GuildID: guildID,
		Reason:  reason,
	}
	resp, err := p.client.CreateReport(ctx, postID, accessTok, req)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok && apiErr.Code == CodeTokenExpired {
			var rErr error
			accessTok, _, rErr = p.ensureAccessToken(ctx, reporterID, true)
			if rErr != nil {
				return nil, rErr
			}
			resp, err = p.client.CreateReport(ctx, postID, accessTok, req)
		}
	}
	if err != nil {
		return nil, p.mapRemoteWriteErr(ctx, reporterID, err)
	}
	return &bbs.Report{
		ID:             resp.ID,
		Scope:          resp.Scope,
		OriginPeer:     resp.OriginPeer,
		GuildID:        resp.GuildID,
		PostID:         resp.PostID,
		ThreadID:       resp.ThreadID,
		BoardID:        resp.BoardID,
		ReporterUserID: reporterID,
		Reason:         resp.Reason,
		Status:         resp.Status,
		CreatedAt:      resp.CreatedAt,
		UpdatedAt:      resp.UpdatedAt,
	}, nil
}

func (p *RemoteProvider) ListRemoteAvailableCaps(ctx context.Context, userID snowflake.ID, boardID uuid.UUID) ([]bbs.Cap, error) {
	if !p.CanUseCaps() || p.store == nil || userID == 0 {
		return nil, nil
	}
	accessTok, scopes, err := p.ensureAccessTokenForScope(ctx, userID, false, bbs.FederationScopeCaps)
	if err != nil {
		return nil, nil
	}
	if !slices.Contains(scopes, bbs.FederationScopeCaps) {
		return nil, nil
	}
	caps, err := p.client.ListBoardCaps(ctx, boardID, accessTok)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok && apiErr.Code == CodeTokenExpired {
			var rErr error
			accessTok, _, rErr = p.ensureAccessToken(ctx, userID, true)
			if rErr != nil {
				return nil, rErr
			}
			caps, err = p.client.ListBoardCaps(ctx, boardID, accessTok)
		}
	}
	if err != nil {
		return nil, p.mapRemoteWriteErr(ctx, userID, err)
	}
	return caps, nil
}
