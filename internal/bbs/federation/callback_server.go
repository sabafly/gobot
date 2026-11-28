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
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/internal/bbs"
)

const (
	pendingClientAuthTTL   = 10 * time.Minute
	maxPendingClientAuths  = 512
	callbackPathSuffix     = "/api/v1/bbs/federation/client/callback"
	defaultFollowupMessage = 5 * time.Second
)

// BeginAuthRequest carries Discord context needed to start an OAuth 2.0 PKCE user delegation flow.
type BeginAuthRequest struct {
	ProviderKey      string
	UserID           snowflake.ID
	ApplicationID    snowflake.ID
	InteractionToken string
	ContinueCustomID string
	Locale           discord.Locale
}

// AuthCompleteCallback updates the triggering Discord ephemeral prompt after OAuth completes in browser.
type AuthCompleteCallback func(ctx context.Context, req BeginAuthRequest, providerLabel string)

type pendingClientAuth struct {
	State        string
	CodeVerifier string
	RemoteKey    string
	Label        string
	BeginReq     BeginAuthRequest
	ExpiresAt    time.Time
}

// CallbackServerDeps configures the Listener's dedicated OAuth 2.0 callback HTTP server.
type CallbackServerDeps struct {
	Config         ClientConfig
	Store          bbs.Store
	Cipher         *TokenCipher
	OnAuthComplete AuthCompleteCallback
}

// CallbackServer receives OAuth 2.0 authorization code redirects from Origin on a dedicated port,
// completes PKCE token exchange, stores encrypted user tokens, and notifies Discord UI waiters.
type CallbackServer struct {
	cfg            ClientConfig
	basePath       string
	store          bbs.Store
	cipher         *TokenCipher
	onAuthComplete AuthCompleteCallback
	mux            *http.ServeMux
	handler        http.Handler

	mu        sync.RWMutex
	remotes   map[string]*RemoteProvider
	pendingMu sync.Mutex
	pending   map[string]*pendingClientAuth
}

// NewCallbackServer builds and validates a Listener OAuth callback server.
func NewCallbackServer(deps CallbackServerDeps) (*CallbackServer, error) {
	if deps.Store == nil {
		return nil, errors.New("federation callback server: Store is required")
	}
	if deps.Cipher == nil {
		return nil, errors.New("federation callback server: TokenCipher is required")
	}
	resolved, err := ResolveClientConfig(deps.Config)
	if err != nil {
		return nil, err
	}
	s := &CallbackServer{
		cfg:            resolved,
		basePath:       resolved.BasePath,
		store:          deps.Store,
		cipher:         deps.Cipher,
		onAuthComplete: deps.OnAuthComplete,
		mux:            http.NewServeMux(),
		remotes:        make(map[string]*RemoteProvider),
		pending:        make(map[string]*pendingClientAuth),
	}
	s.mux.HandleFunc(http.MethodGet+" "+s.basePath+callbackPathSuffix, s.handleCallback)
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		s.mux.ServeHTTP(w, r)
	})
	return s, nil
}

// SetOnAuthComplete updates the completion hook after Discord bot initialization.
func (s *CallbackServer) SetOnAuthComplete(fn AuthCompleteCallback) {
	s.mu.Lock()
	s.onAuthComplete = fn
	s.mu.Unlock()
}

// RegisterRemote attaches a RemoteProvider handled by this callback server.
func (s *CallbackServer) RegisterRemote(rp *RemoteProvider) {
	if rp == nil {
		return
	}
	s.mu.Lock()
	s.remotes[rp.RemoteKey()] = rp
	s.mu.Unlock()
}

// Enabled reports whether the callback HTTP listener is configured and enabled.
func (s *CallbackServer) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

// CallbackURL returns the full public redirect_uri expected by Origin.
func (s *CallbackServer) CallbackURL() string {
	return s.cfg.PublicURL + s.basePath + callbackPathSuffix
}

// Handler returns the HTTP handler for testing.
func (s *CallbackServer) Handler() http.Handler {
	return s.handler
}

// ListenAndServe starts the dedicated callback HTTP server until ctx is cancelled.
func (s *CallbackServer) ListenAndServe(ctx context.Context) error {
	httpSrv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("BBS federation OAuth callback server listening",
			slog.String("addr", s.cfg.ListenAddr),
			slog.String("callback_url", s.CallbackURL()),
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

func (s *CallbackServer) findRemote(keyOrProviderKey string) (*RemoteProvider, bool) {
	k := strings.TrimPrefix(strings.TrimSpace(keyOrProviderKey), RemoteKeyPrefix)
	s.mu.RLock()
	defer s.mu.RUnlock()
	rp, ok := s.remotes[k]
	return rp, ok
}

// BeginUserAuth starts a PKCE authorization flow for userID against Origin and returns the browser URL.
func (s *CallbackServer) BeginUserAuth(ctx context.Context, req BeginAuthRequest) (authorizeURL string, providerLabel string, err error) {
	if !s.Enabled() {
		return "", "", bbs.ErrFederationReadOnly
	}
	rp, ok := s.findRemote(req.ProviderKey)
	if !ok {
		return "", "", bbs.ErrProviderNotFound
	}
	caps, err := rp.EnsureCapabilities(ctx)
	if err != nil {
		return "", rp.DisplayLabel(), err
	}
	if rp.cfg.ReadOnly || (!caps.AllowWrite && !caps.AllowReport) {
		return "", rp.DisplayLabel(), bbs.ErrFederationReadOnly
	}
	peerID := strings.TrimSpace(rp.cfg.PeerID)
	if peerID == "" {
		peerID = caps.PeerID
	}
	if peerID == "" {
		return "", rp.DisplayLabel(), fmt.Errorf("%w: remote peer_id unknown", bbs.ErrFederationUnavailable)
	}

	var scopes []string
	if caps.AllowWrite {
		scopes = append(scopes, bbs.FederationScopeWrite)
	}
	if caps.AllowReport {
		scopes = append(scopes, bbs.FederationScopeReport)
	}
	if caps.AllowCaps {
		scopes = append(scopes, bbs.FederationScopeCaps)
	}

	state, err := GenerateRandomToken(24)
	if err != nil {
		return "", rp.DisplayLabel(), err
	}
	verifier, err := GenerateRandomToken(32)
	if err != nil {
		return "", rp.DisplayLabel(), err
	}
	challenge := ComputePKCES256(verifier)

	now := time.Now().UTC()
	s.pendingMu.Lock()
	for k, p := range s.pending {
		if now.After(p.ExpiresAt) || len(s.pending) >= maxPendingClientAuths {
			delete(s.pending, k)
		}
	}
	s.pending[state] = &pendingClientAuth{
		State:        state,
		CodeVerifier: verifier,
		RemoteKey:    rp.RemoteKey(),
		Label:        rp.DisplayLabel(),
		BeginReq:     req,
		ExpiresAt:    now.Add(pendingClientAuthTTL),
	}
	s.pendingMu.Unlock()

	authURL, err := url.Parse(rp.cfg.BaseURL + "/api/v1/bbs/federation/oauth/authorize")
	if err != nil {
		return "", rp.DisplayLabel(), err
	}
	q := authURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", peerID)
	q.Set("redirect_uri", s.CallbackURL())
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	if req.UserID != 0 {
		q.Set("login_hint", req.UserID.String())
	}
	authURL.RawQuery = q.Encode()
	return authURL.String(), rp.DisplayLabel(), nil
}

func (s *CallbackServer) popPending(state string) (*pendingClientAuth, bool) {
	now := time.Now().UTC()
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	p, ok := s.pending[state]
	if ok {
		delete(s.pending, state)
	}
	if !ok || now.After(p.ExpiresAt) {
		return nil, false
	}
	return p, true
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	state := q.Get("state")
	pending, ok := s.popPending(state)
	if !ok {
		renderStatusPage(w, http.StatusBadRequest, false, "連携エラー / Invalid State", "認可状態が見つからないか期限切れです。Discord からやり直してください。", "")
		return
	}

	if errParam := strings.TrimSpace(q.Get("error")); errParam != "" {
		renderStatusPage(w, http.StatusOK, false, "連携が拒否されました / Denied", "本番インスタンスでの連携許可がキャンセルされました。必要な場合は Discord から再度実行できます。", errParam)
		return
	}

	code := strings.TrimSpace(q.Get("code"))
	if code == "" {
		renderStatusPage(w, http.StatusBadRequest, false, "認可コード不足 / Missing Code", "認可コードが見つかりませんでした。", "")
		return
	}

	rp, exists := s.findRemote(pending.RemoteKey)
	if !exists {
		renderStatusPage(w, http.StatusBadGateway, false, "リモート未登録", "対象のリモートプロバイダーが見つかりません。", pending.RemoteKey)
		return
	}

	tokResp, err := rp.client.ExchangeAuthCode(r.Context(), code, s.CallbackURL(), pending.CodeVerifier)
	if err != nil {
		slog.Warn("federation client oauth code exchange failed", "remote", pending.RemoteKey, "err", err)
		renderStatusPage(w, http.StatusBadGateway, false, "トークン交換失敗 / Exchange Failed", "本番インスタンスとの連携完了処理に失敗しました。", err.Error())
		return
	}
	if pending.BeginReq.UserID != 0 && tokResp.UserID != pending.BeginReq.UserID {
		renderStatusPage(w, http.StatusForbidden, false, "ユーザー不一致 / User Mismatch", "Discord でリクエストしたユーザーと一致しないためトークンを破棄しました。", "")
		return
	}

	accessEnc, err := s.cipher.Encrypt(tokResp.AccessToken)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "暗号化エラー", "トークンの保護に失敗しました。", "")
		return
	}
	refreshEnc, err := s.cipher.Encrypt(tokResp.RefreshToken)
	if err != nil {
		renderStatusPage(w, http.StatusInternalServerError, false, "暗号化エラー", "トークンの保護に失敗しました。", "")
		return
	}

	now := time.Now().UTC()
	if err := s.store.UpsertFederationUserToken(r.Context(), &bbs.FederationUserToken{
		RemoteKey:        rp.RemoteKey(),
		UserID:           tokResp.UserID,
		AccessTokenEnc:   accessEnc,
		AccessExpiresAt:  now.Add(time.Duration(tokResp.ExpiresIn) * time.Second),
		RefreshTokenEnc:  refreshEnc,
		RefreshExpiresAt: now.Add(time.Duration(tokResp.RefreshExpiresIn) * time.Second),
		Scopes:           tokResp.Scopes,
		PolicyVersion:    tokResp.PolicyVersion,
	}); err != nil {
		slog.Error("federation client failed to persist encrypted user token", "remote", rp.RemoteKey, "user_id", tokResp.UserID, "err", err)
		renderStatusPage(w, http.StatusInternalServerError, false, "保存エラー", "連携トークンの保存に失敗しました。", "")
		return
	}

	s.mu.RLock()
	cb := s.onAuthComplete
	s.mu.RUnlock()
	if cb != nil && pending.BeginReq.InteractionToken != "" && pending.BeginReq.ApplicationID != 0 {
		reqCopy := pending.BeginReq
		label := pending.Label
		go func() {
			cbCtx, cancel := context.WithTimeout(context.Background(), defaultFollowupMessage)
			defer cancel()
			cb(cbCtx, reqCopy, label)
		}()
	}

	renderStatusPage(w, http.StatusOK, true, "連携が完了しました / Authorization Complete",
		fmt.Sprintf("[%s] の規約への同意と権限連携が完了しました。このタブを閉じて Discord に戻り、操作を続けてください。", pending.Label), "")
}
