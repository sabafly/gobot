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
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/i18n"
)

const (
	sseHeartbeatInterval    = 25 * time.Second
	userSlipCacheTTL        = 5 * time.Minute
	userSlipCacheMaxEntries = 2048
	userSlipFetchTimeout    = 3 * time.Second
	userFlagActiveDeveloper = discord.UserFlags(1 << 22)
)

type cachedOriginUserSlip struct {
	ctx       bbs.SlipContext
	expiresAt time.Time
}

// ServerDeps groups dependencies needed by Origin's dedicated federation HTTP/SSE + OAuth2 server.
type ServerDeps struct {
	Config             ServerConfig
	Service            *bbs.Service
	Syncer             *discordsync.Syncer
	Client             *bot.Client
	Hub                *EventHub
	OAuthAPIBase       string
	OAuthAuthorizeBase string
	OAuthHTTPClient    *http.Client
}

type userGrantContext struct {
	Grant bbs.FederationGrant
}

// Server serves Origin's read/write/SSE federation API and OAuth 2.0 consent/token endpoints
// on its own dedicated HTTP listener, completely separate from Admin API RBAC and sessions.
type Server struct {
	cfg      ServerConfig
	basePath string
	svc      *bbs.Service
	syncer   *discordsync.Syncer
	hub      *EventHub
	oauth    *oauthManager
	mux      *http.ServeMux
	handler  http.Handler

	clientMu sync.RWMutex
	client   *bot.Client

	userSlipMu    sync.RWMutex
	userSlipCache map[snowflake.ID]cachedOriginUserSlip
}

// NewServer validates configuration and builds Origin's federation server.
func NewServer(deps ServerDeps) (*Server, error) {
	if deps.Service == nil {
		return nil, errors.New("federation server: Service is required")
	}
	if deps.Hub == nil {
		return nil, errors.New("federation server: EventHub is required")
	}
	resolved, err := ResolveServerConfig(deps.Config)
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:           resolved,
		basePath:      resolved.BasePath,
		svc:           deps.Service,
		syncer:        deps.Syncer,
		hub:           deps.Hub,
		mux:           http.NewServeMux(),
		client:        deps.Client,
		userSlipCache: make(map[snowflake.ID]cachedOriginUserSlip),
	}
	s.oauth = newOAuthManager(s, deps.OAuthAPIBase, deps.OAuthAuthorizeBase, deps.OAuthHTTPClient)
	s.registerRoutes()
	s.handler = s.withSecurityHeaders(s.mux)
	return s, nil
}

// SetDiscordClient attaches or updates the Discord client used for Origin-side syncer fanout after federation writes.
func (s *Server) SetDiscordClient(c *bot.Client) {
	s.clientMu.Lock()
	s.client = c
	s.clientMu.Unlock()
}

func (s *Server) discordClient() *bot.Client {
	s.clientMu.RLock()
	defer s.clientMu.RUnlock()
	return s.client
}

// Handler returns the HTTP handler (including basePath routing) for testing.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// ListenAndServe starts the dedicated HTTP server and gracefully shuts down when ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	httpSrv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("BBS federation server listening",
			slog.String("addr", s.cfg.ListenAddr),
			slog.String("base_path", s.basePath),
			slog.Int("peers", len(s.cfg.Peers)),
		)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
		return nil
	}
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handle(method, path string, h http.HandlerFunc) {
	pattern := method + " " + s.basePath + path
	s.mux.HandleFunc(pattern, h)
}

func (s *Server) peerHandle(method, path string, h func(http.ResponseWriter, *http.Request, PeerConfig)) {
	s.handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		peer, ok := s.authenticatePeer(r)
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, CodeUnauthorized, "missing or invalid peer bearer token")
			return
		}
		h(w, r, peer)
	})
}

func (s *Server) grantHandle(method, path, requiredScope string, h func(http.ResponseWriter, *http.Request, PeerConfig, userGrantContext)) {
	s.peerHandle(method, path, func(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
		ug, ok := s.authenticateUserGrant(w, r, peer, requiredScope)
		if !ok {
			return
		}
		h(w, r, peer, ug)
	})
}

func (s *Server) registerRoutes() {
	// Browser OAuth 2.0 endpoints (authenticated by PKCE/client_id/redirect_uri/Discord/session cookie)
	s.handle(http.MethodGet, "/api/v1/bbs/federation/oauth/authorize", s.oauth.handleAuthorize)
	s.handle(http.MethodGet, "/api/v1/bbs/federation/oauth/callback", s.oauth.handleDiscordCallback)
	s.handle(http.MethodGet, "/api/v1/bbs/federation/oauth/consent", s.oauth.handleConsentGet)
	s.handle(http.MethodPost, "/api/v1/bbs/federation/oauth/consent", s.oauth.handleConsentPost)

	// Peer-authenticated OAuth backchannel endpoints
	s.peerHandle(http.MethodPost, "/api/v1/bbs/federation/oauth/token", s.oauth.handleToken)
	s.peerHandle(http.MethodPost, "/api/v1/bbs/federation/oauth/revoke", s.oauth.handleRevoke)

	// Peer-authenticated read & SSE endpoints (global boards only)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/capabilities", s.handleCapabilities)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/categories", s.handleListCategories)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/boards", s.handleListBoards)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/boards/{board_id}", s.handleGetBoard)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/boards/{board_id}/threads", s.handleListThreads)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/threads/{thread_id}", s.handleGetThread)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/threads/{thread_id}/posts", s.handleListPosts)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/posts/{post_id}", s.handleGetPost)
	s.peerHandle(http.MethodGet, "/api/v1/bbs/federation/events", s.handleEventsSSE)

	// Peer + OAuth User-Grant authenticated endpoints
	s.grantHandle(http.MethodGet, "/api/v1/bbs/federation/boards/{board_id}/caps", "", s.handleListBoardCaps)
	s.grantHandle(http.MethodPost, "/api/v1/bbs/federation/boards/{board_id}/threads", bbs.FederationScopeWrite, s.handleCreateThread)
	s.grantHandle(http.MethodPost, "/api/v1/bbs/federation/threads/{thread_id}/posts", bbs.FederationScopeWrite, s.handleCreatePost)
	s.grantHandle(http.MethodPost, "/api/v1/bbs/federation/posts/{post_id}/reports", bbs.FederationScopeReport, s.handleCreateReport)
}

func (s *Server) peerByID(id string) (PeerConfig, bool) {
	for _, p := range s.cfg.Peers {
		if p.ID == id {
			return p, true
		}
	}
	return PeerConfig{}, false
}

func (s *Server) authenticatePeer(r *http.Request) (PeerConfig, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(auth) <= 7 || !strings.EqualFold(auth[:7], "Bearer ") {
		return PeerConfig{}, false
	}
	token := strings.TrimSpace(auth[7:])
	if token == "" {
		return PeerConfig{}, false
	}
	var matched PeerConfig
	var found bool
	for _, p := range s.cfg.Peers {
		if len(token) == len(p.Token) && subtle.ConstantTimeCompare([]byte(token), []byte(p.Token)) == 1 {
			matched = p
			found = true
		}
	}
	return matched, found
}

func (s *Server) authenticateUserGrant(w http.ResponseWriter, r *http.Request, peer PeerConfig, requiredScope string) (userGrantContext, bool) {
	rawUserTok := strings.TrimSpace(r.Header.Get(HeaderUserToken))
	if rawUserTok == "" {
		writeAPIError(w, http.StatusUnauthorized, CodeAuthRequired, "user delegation token required")
		return userGrantContext{}, false
	}
	grant, err := s.svc.Store.GetFederationGrantByAccessHash(r.Context(), HashToken(rawUserTok))
	if err != nil {
		if errors.Is(err, bbs.ErrNotFound) {
			writeAPIError(w, http.StatusUnauthorized, CodeTokenRevoked, "user delegation token not found or revoked")
			return userGrantContext{}, false
		}
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "failed to verify user delegation")
		return userGrantContext{}, false
	}
	now := time.Now().UTC()
	if grant.PeerID != peer.ID {
		writeAPIError(w, http.StatusUnauthorized, CodeUnauthorized, "token belongs to a different peer")
		return userGrantContext{}, false
	}
	if grant.RevokedAt != nil {
		writeAPIError(w, http.StatusUnauthorized, CodeTokenRevoked, "user delegation has been revoked")
		return userGrantContext{}, false
	}
	if !now.Before(grant.AccessExpiresAt) {
		writeAPIError(w, http.StatusUnauthorized, CodeTokenExpired, "user delegation access token expired")
		return userGrantContext{}, false
	}
	if grant.PolicyVersion != s.svc.Policy.CurrentVersion {
		_ = s.svc.Store.RevokeFederationGrant(r.Context(), grant.ID, grant.UserID, now)
		writeAPIError(w, http.StatusForbidden, CodePolicyConsentRequired, "origin policy version changed; re-consent required")
		return userGrantContext{}, false
	}
	if _, cErr := s.svc.CheckConsent(r.Context(), grant.UserID); cErr != nil {
		_ = s.svc.Store.RevokeFederationGrant(r.Context(), grant.ID, grant.UserID, now)
		writeAPIError(w, http.StatusForbidden, CodePolicyConsentRequired, "origin policy consent withdrawn")
		return userGrantContext{}, false
	}
	if requiredScope != "" && !grant.HasScope(requiredScope) {
		writeAPIError(w, http.StatusForbidden, CodeReadOnly, "delegation grant lacks required scope: "+requiredScope)
		return userGrantContext{}, false
	}
	posterHash := ""
	if s.svc.Identity != nil {
		posterHash = s.svc.Identity.GeneratePosterHash(bbs.ScopeGlobal, 0, grant.UserID)
	}
	banned, err := s.svc.Store.IsBanned(r.Context(), bbs.ScopeGlobal, 0, posterHash, grant.UserID, now)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "ban check failed")
		return userGrantContext{}, false
	}
	if banned {
		writeAPIError(w, http.StatusForbidden, CodeBanned, "user is globally banned on origin")
		return userGrantContext{}, false
	}
	return userGrantContext{Grant: *grant}, true
}

func (s *Server) isOriginGlobalBoard(b *bbs.Board) bool {
	if b == nil || b.Scope != bbs.ScopeGlobal {
		return false
	}
	return b.ProviderKey == "" || b.ProviderKey == s.svc.Registry.DefaultKey()
}

func (s *Server) loadAllowedGlobalBoard(ctx context.Context, peer PeerConfig, boardID uuid.UUID) (*bbs.Board, error) {
	b, err := s.svc.GetBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}
	if !s.isOriginGlobalBoard(b) || !peer.AllowsBoard(b.ID) {
		return nil, bbs.ErrNotFound
	}
	return b, nil
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	allowedIDs := make([]uuid.UUID, 0, len(peer.ParsedBoardIDs))
	for id := range peer.ParsedBoardIDs {
		allowedIDs = append(allowedIDs, id)
	}
	writeJSON(w, http.StatusOK, CapabilitiesResponse{
		ProtocolVersion: ProtocolVersion,
		InstanceEpoch:   s.hub.Epoch(),
		LatestSeq:       s.hub.LatestSeq(),
		PeerID:          peer.ID,
		AllowWrite:      peer.AllowWrite,
		AllowReport:     peer.EffectiveAllowReport(),
		AllowCaps:       peer.AllowCaps,
		AllowedBoardIDs: allowedIDs,
		PolicyVersion:   s.svc.Policy.CurrentVersion,
	})
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	def := s.svc.Registry.Default()
	if def == nil {
		writeAPIError(w, http.StatusServiceUnavailable, CodeUnavailable, "default bbs provider unavailable")
		return
	}
	scope := bbs.ScopeGlobal
	cats, err := def.ListCategories(r.Context(), bbs.CategoryFilter{Scope: &scope})
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if len(peer.ParsedBoardIDs) > 0 {
		boards, bErr := def.ListBoards(r.Context(), bbs.BoardFilter{Scope: &scope})
		if bErr == nil {
			usedCats := make(map[uuid.UUID]struct{})
			for _, b := range boards {
				if peer.AllowsBoard(b.ID) {
					usedCats[b.CategoryID] = struct{}{}
				}
			}
			filtered := make([]bbs.Category, 0, len(cats))
			for _, c := range cats {
				if _, ok := usedCats[c.ID]; ok {
					filtered = append(filtered, c)
				}
			}
			cats = filtered
		}
	}
	if cats == nil {
		cats = []bbs.Category{}
	}
	writeJSON(w, http.StatusOK, CategoryListResponse{Categories: cats})
}

func (s *Server) handleListBoards(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	def := s.svc.Registry.Default()
	if def == nil {
		writeAPIError(w, http.StatusServiceUnavailable, CodeUnavailable, "default bbs provider unavailable")
		return
	}
	scope := bbs.ScopeGlobal
	var filter bbs.BoardFilter
	filter.Scope = &scope
	if rawCat := strings.TrimSpace(r.URL.Query().Get("category_id")); rawCat != "" {
		catID, err := uuid.Parse(rawCat)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid category_id")
			return
		}
		filter.CategoryID = &catID
	}
	boards, err := def.ListBoards(r.Context(), filter)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	out := make([]bbs.Board, 0, len(boards))
	for _, b := range boards {
		if s.isOriginGlobalBoard(&b) && peer.AllowsBoard(b.ID) {
			out = append(out, b)
		}
	}
	writeJSON(w, http.StatusOK, BoardListResponse{Boards: out})
}

func (s *Server) handleGetBoard(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	boardID, err := uuid.Parse(r.PathValue("board_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid board_id")
		return
	}
	board, err := s.loadAllowedGlobalBoard(r.Context(), peer, boardID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, board)
}

func (s *Server) handleListThreads(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	boardID, err := uuid.Parse(r.PathValue("board_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid board_id")
		return
	}
	board, err := s.loadAllowedGlobalBoard(r.Context(), peer, boardID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	filter := bbs.ThreadFilter{BoardID: board.ID, Limit: 50}
	if stRaw := strings.TrimSpace(r.URL.Query().Get("status")); stRaw != "" {
		st := bbs.ThreadStatus(stRaw)
		switch st {
		case bbs.ThreadStatusActive, bbs.ThreadStatusArchived, bbs.ThreadStatusDeleted:
			filter.Status = &st
		default:
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid status")
			return
		}
	}
	if qRaw := strings.TrimSpace(r.URL.Query().Get("q")); qRaw != "" {
		filter.Query = qRaw
	}
	if offRaw := strings.TrimSpace(r.URL.Query().Get("offset")); offRaw != "" {
		off, pErr := strconv.Atoi(offRaw)
		if pErr != nil || off < 0 {
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid offset")
			return
		}
		filter.Offset = off
	}
	if limRaw := strings.TrimSpace(r.URL.Query().Get("limit")); limRaw != "" {
		lim, pErr := strconv.Atoi(limRaw)
		if pErr != nil || lim < 1 {
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid limit")
			return
		}
		filter.Limit = min(lim, 200)
	}
	def := s.svc.Registry.Default()
	threads, err := def.ListThreads(r.Context(), filter)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if threads == nil {
		threads = []bbs.Thread{}
	}
	writeJSON(w, http.StatusOK, ThreadListResponse{Threads: threads})
}

func (s *Server) handleGetThread(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	threadID, err := uuid.Parse(r.PathValue("thread_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid thread_id")
		return
	}
	def := s.svc.Registry.Default()
	th, err := def.GetThread(r.Context(), threadID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if _, err := s.loadAllowedGlobalBoard(r.Context(), peer, th.BoardID); err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, th)
}

func (s *Server) handleListPosts(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	threadID, err := uuid.Parse(r.PathValue("thread_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid thread_id")
		return
	}
	def := s.svc.Registry.Default()
	th, err := def.GetThread(r.Context(), threadID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if _, err := s.loadAllowedGlobalBoard(r.Context(), peer, th.BoardID); err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}

	offset := 0
	limit := 200
	var afterRes uint32
	var hasAfterRes bool
	if arRaw := strings.TrimSpace(r.URL.Query().Get("after_res")); arRaw != "" {
		v, ok := parsePositiveUint32(arRaw)
		if !ok {
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid after_res")
			return
		}
		afterRes = v
		hasAfterRes = true
		offset = int(afterRes)
		limit = 1000
	}
	if offRaw := strings.TrimSpace(r.URL.Query().Get("offset")); offRaw != "" {
		off, pErr := strconv.Atoi(offRaw)
		if pErr != nil || off < 0 {
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid offset")
			return
		}
		offset = off
	}
	if limRaw := strings.TrimSpace(r.URL.Query().Get("limit")); limRaw != "" {
		lim, pErr := strconv.Atoi(limRaw)
		if pErr != nil || lim < 1 {
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid limit")
			return
		}
		limit = min(lim, 1000)
	}
	posts, err := def.ListPosts(r.Context(), threadID, offset, limit)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if hasAfterRes {
		filtered := make([]bbs.Post, 0, len(posts))
		for _, p := range posts {
			if p.ResNumber > afterRes {
				filtered = append(filtered, p)
			}
		}
		posts = filtered
	}
	if posts == nil {
		posts = []bbs.Post{}
	}
	writeJSON(w, http.StatusOK, PostListResponse{Posts: posts})
}

func (s *Server) handleGetPost(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	postID, err := uuid.Parse(r.PathValue("post_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid post_id")
		return
	}
	def := s.svc.Registry.Default()
	post, err := def.GetPost(r.Context(), postID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	th, err := def.GetThread(r.Context(), post.ThreadID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if _, err := s.loadAllowedGlobalBoard(r.Context(), peer, th.BoardID); err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (s *Server) handleListBoardCaps(w http.ResponseWriter, r *http.Request, peer PeerConfig, ug userGrantContext) {
	boardID, err := uuid.Parse(r.PathValue("board_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid board_id")
		return
	}
	if _, err := s.loadAllowedGlobalBoard(r.Context(), peer, boardID); err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if !peer.AllowCaps || !ug.Grant.HasScope(bbs.FederationScopeCaps) {
		writeJSON(w, http.StatusOK, CapsListResponse{Caps: []bbs.Cap{}})
		return
	}
	caps := s.svc.ListAvailableCaps(r.Context(), ug.Grant.UserID, boardID)
	if caps == nil {
		caps = []bbs.Cap{}
	}
	writeJSON(w, http.StatusOK, CapsListResponse{Caps: caps})
}

func (s *Server) handleCreateThread(w http.ResponseWriter, r *http.Request, peer PeerConfig, ug userGrantContext) {
	if !peer.AllowWrite {
		writeAPIError(w, http.StatusForbidden, CodeReadOnly, "peer is read-only")
		return
	}
	boardID, err := uuid.Parse(r.PathValue("board_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid board_id")
		return
	}
	if _, err := s.loadAllowedGlobalBoard(r.Context(), peer, boardID); err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}

	var req CreateThreadRequest
	if err := readJSONBody(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid json body")
		return
	}

	allowCaps := peer.AllowCaps && ug.Grant.HasScope(bbs.FederationScopeCaps)
	slipCtx := s.buildOriginSlipContext(r.Context(), ug.Grant.UserID, peer.SlipLabel)
	board, thread, post, err := s.svc.CreateThreadFromFederation(r.Context(), peer.ID, allowCaps, bbs.CreateThreadInput{
		BoardID:     boardID,
		UserID:      ug.Grant.UserID,
		Title:       req.Title,
		RawName:     req.RawName,
		CapID:       req.CapID,
		RawBody:     req.RawBody,
		SlipContext: slipCtx,
		Now:         time.Now().UTC(),
	})
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}

	if s.syncer != nil {
		if dc := s.discordClient(); dc != nil {
			bCopy, tCopy, pCopy := *board, *thread, *post
			go s.syncer.FanoutThread(context.Background(), dc, bCopy, tCopy, pCopy, i18n.DefaultLocale())
		}
	}

	writeJSON(w, http.StatusCreated, CreateThreadResponse{
		Board:  *board,
		Thread: *thread,
		Post:   *post,
	})
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request, peer PeerConfig, ug userGrantContext) {
	if !peer.AllowWrite {
		writeAPIError(w, http.StatusForbidden, CodeReadOnly, "peer is read-only")
		return
	}
	threadID, err := uuid.Parse(r.PathValue("thread_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid thread_id")
		return
	}
	def := s.svc.Registry.Default()
	th, err := def.GetThread(r.Context(), threadID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if _, err := s.loadAllowedGlobalBoard(r.Context(), peer, th.BoardID); err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}

	var req CreatePostRequest
	if err := readJSONBody(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid json body")
		return
	}

	allowCaps := peer.AllowCaps && ug.Grant.HasScope(bbs.FederationScopeCaps)
	slipCtx := s.buildOriginSlipContext(r.Context(), ug.Grant.UserID, peer.SlipLabel)
	board, thread, post, backwardRefs, err := s.svc.CreatePostFromFederation(r.Context(), peer.ID, allowCaps, bbs.CreatePostInput{
		ThreadID:    threadID,
		UserID:      ug.Grant.UserID,
		RawName:     req.RawName,
		CapID:       req.CapID,
		RawBody:     req.RawBody,
		SlipContext: slipCtx,
		Now:         time.Now().UTC(),
	})
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}

	if s.syncer != nil {
		if dc := s.discordClient(); dc != nil {
			bCopy, tCopy, pCopy := *board, *thread, *post
			refsCopy := append([]uint32(nil), backwardRefs...)
			go s.syncer.FanoutPost(context.Background(), dc, bCopy, tCopy, pCopy, refsCopy, i18n.DefaultLocale())
		}
	}

	writeJSON(w, http.StatusCreated, CreatePostResponse{
		Board:        *board,
		Thread:       *thread,
		Post:         *post,
		BackwardRefs: backwardRefs,
	})
}

func (s *Server) handleCreateReport(w http.ResponseWriter, r *http.Request, peer PeerConfig, ug userGrantContext) {
	if !peer.EffectiveAllowReport() {
		writeAPIError(w, http.StatusForbidden, CodeReadOnly, "peer reporting is disabled")
		return
	}
	postID, err := uuid.Parse(r.PathValue("post_id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid post_id")
		return
	}
	def := s.svc.Registry.Default()
	post, err := def.GetPost(r.Context(), postID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	th, err := def.GetThread(r.Context(), post.ThreadID)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	if _, err := s.loadAllowedGlobalBoard(r.Context(), peer, th.BoardID); err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}

	var req CreateReportRequest
	if err := readJSONBody(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid json body")
		return
	}
	rep, err := s.svc.CreateReportFromFederation(r.Context(), peer.ID, req.GuildID, ug.Grant.UserID, postID, req.Reason)
	if err != nil {
		st, code := MapBBSErrorToHTTP(err)
		writeAPIError(w, st, code, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, CreateReportResponse{
		ID:         rep.ID,
		Scope:      rep.Scope,
		OriginPeer: rep.OriginPeer,
		GuildID:    rep.GuildID,
		PostID:     rep.PostID,
		ThreadID:   rep.ThreadID,
		BoardID:    rep.BoardID,
		Reason:     rep.Reason,
		Status:     rep.Status,
		CreatedAt:  rep.CreatedAt,
		UpdatedAt:  rep.UpdatedAt,
	})
}

func (s *Server) handleEventsSSE(w http.ResponseWriter, r *http.Request, peer PeerConfig) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, CodeUnavailable, "streaming unsupported")
		return
	}

	var afterSeq uint64
	if raw := strings.TrimSpace(r.URL.Query().Get("after_seq")); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, CodeInvalidInput, "invalid after_seq")
			return
		}
		afterSeq = v
	} else if lastID := strings.TrimSpace(r.Header.Get("Last-Event-ID")); lastID != "" {
		if v, err := strconv.ParseUint(lastID, 10, 64); err == nil {
			afterSeq = v
		}
	}
	clientEpoch := strings.TrimSpace(r.URL.Query().Get("epoch"))

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	epoch, latestSeq, replay, live, resyncRequired, unsub := s.hub.Subscribe(afterSeq, clientEpoch)
	defer unsub()

	writeSSE := func(event string, id uint64, payload any) bool {
		b, err := json.Marshal(payload)
		if err != nil {
			return true
		}
		if id > 0 {
			if _, wErr := fmt.Fprintf(w, "id: %d\n", id); wErr != nil {
				return false
			}
		}
		if event != "" {
			if _, wErr := fmt.Fprintf(w, "event: %s\n", event); wErr != nil {
				return false
			}
		}
		if _, wErr := fmt.Fprintf(w, "data: %s\n\n", b); wErr != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if resyncRequired {
		if !writeSSE("resync", latestSeq, map[string]any{
			"epoch":      epoch,
			"latest_seq": latestSeq,
		}) {
			return
		}
	} else {
		if !writeSSE("ready", latestSeq, map[string]any{
			"epoch":      epoch,
			"latest_seq": latestSeq,
		}) {
			return
		}
		for _, we := range replay {
			if !peer.AllowsBoard(we.BoardID) {
				continue
			}
			if !writeSSE(string(we.Type), we.Seq, we) {
				return
			}
		}
	}

	ticker := time.NewTicker(sseHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case we, open := <-live:
			if !open {
				_ = writeSSE("resync", s.hub.LatestSeq(), map[string]any{
					"epoch":      epoch,
					"latest_seq": s.hub.LatestSeq(),
					"reason":     "overflow",
				})
				return
			}
			if !peer.AllowsBoard(we.BoardID) {
				continue
			}
			if !writeSSE(string(we.Type), we.Seq, we) {
				return
			}
		}
	}
}

func readJSONBody(r *http.Request, dst any) error {
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, ErrorResponse{Code: code, Error: msg})
}

func slipContextFromDiscordUser(user discord.User) bbs.SlipContext {
	sc := bbs.SlipContext{
		Username:        user.Username,
		UserCreatedAt:   user.ID.Time(),
		IsDefaultAvatar: user.Avatar == nil,
		HasProfileDecor: user.AvatarDecorationData != nil || user.Banner != nil || user.AccentColor != nil,
	}
	if user.GlobalName != nil {
		sc.GlobalName = *user.GlobalName
	}
	switch {
	case user.PublicFlags.Has(discord.UserFlagHouseBravery):
		sc.HypeSquadHouse = "bravery"
	case user.PublicFlags.Has(discord.UserFlagHouseBrilliance):
		sc.HypeSquadHouse = "brilliance"
	case user.PublicFlags.Has(discord.UserFlagHouseBalance):
		sc.HypeSquadHouse = "balance"
	}
	specialMask := discord.UserFlagDiscordEmployee |
		discord.UserFlagPartneredServerOwner |
		discord.UserFlagHypeSquadEvents |
		discord.UserFlagBugHunterLevel1 |
		discord.UserFlagBugHunterLevel2 |
		discord.UserFlagEarlySupporter |
		discord.UserFlagEarlyVerifiedBotDeveloper |
		discord.UserFlagDiscordCertifiedModerator |
		userFlagActiveDeveloper
	sc.HasSpecialBadge = (user.PublicFlags & specialMask) != 0
	return sc
}

func (s *Server) buildOriginSlipContext(ctx context.Context, userID snowflake.ID, slipLabel string) bbs.SlipContext {
	base := bbs.SlipContext{
		BaseNicknameOverride: slipLabel,
		UserCreatedAt:        userID.Time(),
	}
	if userID == 0 {
		return base
	}
	now := time.Now()
	s.userSlipMu.RLock()
	cached, ok := s.userSlipCache[userID]
	s.userSlipMu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		sc := cached.ctx
		sc.BaseNicknameOverride = slipLabel
		return sc
	}

	dc := s.discordClient()
	if dc == nil {
		return base
	}
	var resolved *discord.User
	if dc.Rest != nil {
		fetchCtx, cancel := context.WithTimeout(ctx, userSlipFetchTimeout)
		u, err := dc.Rest.GetUser(userID, rest.WithCtx(fetchCtx))
		cancel()
		if err == nil && u != nil {
			resolved = u
		}
	}
	if resolved == nil {
		return base
	}
	sc := slipContextFromDiscordUser(*resolved)
	s.userSlipMu.Lock()
	if len(s.userSlipCache) >= userSlipCacheMaxEntries {
		for k, v := range s.userSlipCache {
			if !now.Before(v.expiresAt) {
				delete(s.userSlipCache, k)
			}
		}
		if len(s.userSlipCache) >= userSlipCacheMaxEntries {
			for k := range s.userSlipCache {
				delete(s.userSlipCache, k)
				break
			}
		}
	}
	s.userSlipCache[userID] = cachedOriginUserSlip{ctx: sc, expiresAt: now.Add(userSlipCacheTTL)}
	s.userSlipMu.Unlock()

	sc.BaseNicknameOverride = slipLabel
	return sc
}
