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
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

const (
	defaultDiscordAPIBase       = "https://discord.com/api/v10"
	defaultDiscordAuthorizeBase = "https://discord.com/oauth2/authorize"
	defaultPendingLoginTTL      = 5 * time.Minute
	maxPendingLogins            = 256
	maxPendingLoginsPerIP       = 8
)

var (
	ErrLoginNotFound        = errors.New("admin oauth: login request not found or expired")
	ErrInvalidPollSecret    = errors.New("admin oauth: invalid poll secret")
	ErrOAuthUnauthorized    = errors.New("admin oauth: discord user is not authorized for admin dashboard")
	ErrOAuthInvalidState    = errors.New("admin oauth: invalid or expired state parameter")
	ErrOAuthUpstreamError   = errors.New("admin oauth: discord oauth exchange failed")
	ErrTooManyPendingLogins = errors.New("admin oauth: too many pending login requests")
)

var callbackHTMLTmpl = template.Must(template.New("oauth_callback").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: system-ui, -apple-system, sans-serif; background: #111827; color: #f9fafb; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
    .card { background: #1f2937; border: 1px solid #374151; border-radius: 10px; padding: 28px 32px; max-width: 480px; width: 90%; box-shadow: 0 10px 25px rgba(0,0,0,0.4); }
    h1 { margin: 0 0 12px; font-size: 1.25rem; color: {{if .Success}}#34d399{{else}}#f87171{{end}}; }
    p { margin: 0 0 8px; line-height: 1.55; color: #d1d5db; }
    code { background: #111827; padding: 2px 6px; border-radius: 4px; color: #93c5fd; }
  </style>
</head>
<body>
  <div class="card">
    <h1>{{.Heading}}</h1>
    <p>{{.Message}}</p>
    {{if .Detail}}<p><code>{{.Detail}}</code></p>{{end}}
  </div>
</body>
</html>`))

// OAuthConfig configures server-side Discord OAuth2 authorization code exchange for TUI clients.
type OAuthConfig struct {
	ClientID             string
	ClientSecret         string
	RedirectURL          string
	DiscordAPIBase       string
	DiscordAuthorizeBase string
	HTTPClient           *http.Client
	PendingTTL           time.Duration
	SessionTTL           time.Duration
}

type pendingLogin struct {
	LoginID          string
	State            string
	PollSecretHash   string
	ClientInfo       string
	RemoteAddr       string
	Status           string // pending | approved | denied
	ErrMessage       string
	SessionToken     string
	SessionExpiresAt *time.Time
	Principal        *PrincipalInfo
	ExpiresAt        time.Time
}

// OAuthManager coordinates Discord OAuth2 start -> callback -> poll login handshakes.
type OAuthManager struct {
	cfg      OAuthConfig
	rbac     *RBAC
	sessions *SessionStore
	audit    *AuditStore

	mu             sync.Mutex
	pendingByLogin map[string]*pendingLogin
	loginByState   map[string]string
}

// NewOAuthManager creates an OAuthManager after normalizing default endpoints and timeouts.
func NewOAuthManager(cfg OAuthConfig, rbac *RBAC, sessions *SessionStore, audit *AuditStore) (*OAuthManager, error) {
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.RedirectURL = strings.TrimSpace(cfg.RedirectURL)
	if cfg.ClientID == "" {
		return nil, errors.New("admin oauth: client_id is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("admin oauth: client_secret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("admin oauth: redirect_url is required")
	}
	parsedRedirect, err := url.Parse(cfg.RedirectURL)
	if err != nil || (parsedRedirect.Scheme != "http" && parsedRedirect.Scheme != "https") || parsedRedirect.Host == "" {
		return nil, fmt.Errorf("admin oauth: invalid redirect_url %q", cfg.RedirectURL)
	}
	if cfg.DiscordAPIBase == "" {
		cfg.DiscordAPIBase = defaultDiscordAPIBase
	}
	cfg.DiscordAPIBase = strings.TrimRight(cfg.DiscordAPIBase, "/")
	if cfg.DiscordAuthorizeBase == "" {
		cfg.DiscordAuthorizeBase = defaultDiscordAuthorizeBase
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.PendingTTL <= 0 {
		cfg.PendingTTL = defaultPendingLoginTTL
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = defaultSessionTTL
	}
	return &OAuthManager{
		cfg:            cfg,
		rbac:           rbac,
		sessions:       sessions,
		audit:          audit,
		pendingByLogin: make(map[string]*pendingLogin),
		loginByState:   make(map[string]string),
	}, nil
}

func (m *OAuthManager) pruneExpiredLocked(now time.Time) {
	for id, p := range m.pendingByLogin {
		if now.After(p.ExpiresAt) {
			delete(m.loginByState, p.State)
			delete(m.pendingByLogin, id)
		}
	}
}

// Start creates a pending OAuth2 login entry and returns the Discord authorization URL and poll credentials.
func (m *OAuthManager) Start(clientInfo, remoteAddr string) (AuthStartResponse, error) {
	loginID, err := generateRandomToken(16)
	if err != nil {
		return AuthStartResponse{}, err
	}
	state, err := generateRandomToken(24)
	if err != nil {
		return AuthStartResponse{}, err
	}
	pollSecret, err := generateRandomToken(32)
	if err != nil {
		return AuthStartResponse{}, err
	}

	authURL, err := url.Parse(m.cfg.DiscordAuthorizeBase)
	if err != nil {
		return AuthStartResponse{}, err
	}
	q := authURL.Query()
	q.Set("client_id", m.cfg.ClientID)
	q.Set("redirect_uri", m.cfg.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "identify")
	q.Set("state", state)
	q.Set("prompt", "consent")
	authURL.RawQuery = q.Encode()

	now := time.Now().UTC()
	expiresAt := now.Add(m.cfg.PendingTTL)
	normRemote := truncateRunes(strings.TrimSpace(remoteAddr), 64)

	m.mu.Lock()
	m.pruneExpiredLocked(now)
	if len(m.pendingByLogin) >= maxPendingLogins {
		m.mu.Unlock()
		return AuthStartResponse{}, ErrTooManyPendingLogins
	}
	if normRemote != "" {
		perIPCount := 0
		for _, p := range m.pendingByLogin {
			if p.Status == "pending" && p.RemoteAddr == normRemote {
				perIPCount++
			}
		}
		if perIPCount >= maxPendingLoginsPerIP {
			m.mu.Unlock()
			return AuthStartResponse{}, ErrTooManyPendingLogins
		}
	}
	m.pendingByLogin[loginID] = &pendingLogin{
		LoginID:        loginID,
		State:          state,
		PollSecretHash: HashToken(pollSecret),
		ClientInfo:     truncateRunes(clientInfo, 200),
		RemoteAddr:     normRemote,
		Status:         "pending",
		ExpiresAt:      expiresAt,
	}
	m.loginByState[state] = loginID
	m.mu.Unlock()

	return AuthStartResponse{
		LoginID:      loginID,
		AuthorizeURL: authURL.String(),
		PollSecret:   pollSecret,
		ExpiresAt:    expiresAt,
	}, nil
}

type discordTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type discordUserResponse struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
}

// HandleCallback validates Discord's OAuth2 callback, enforces RBAC membership, and creates a DB session.
func (m *OAuthManager) HandleCallback(ctx context.Context, code, state, oauthErr, oauthErrDesc, remoteAddr string) (*PrincipalInfo, error) {
	now := time.Now().UTC()

	m.mu.Lock()
	m.pruneExpiredLocked(now)
	loginID, ok := m.loginByState[state]
	if ok {
		// Enforce single-use CSRF state token.
		delete(m.loginByState, state)
	}
	var p *pendingLogin
	if ok {
		p = m.pendingByLogin[loginID]
	}
	var clientInfo string
	if p != nil {
		if now.After(p.ExpiresAt) {
			delete(m.pendingByLogin, loginID)
			p = nil
		} else {
			clientInfo = p.ClientInfo
		}
	}
	m.mu.Unlock()

	if p == nil {
		return nil, ErrOAuthInvalidState
	}

	failPending := func(msg string, err error) (*PrincipalInfo, error) {
		m.mu.Lock()
		if cur, exists := m.pendingByLogin[loginID]; exists {
			cur.Status = "denied"
			cur.ErrMessage = msg
		}
		m.mu.Unlock()
		return nil, err
	}

	if oauthErr != "" {
		msg := oauthErr
		if oauthErrDesc != "" {
			msg = oauthErr + ": " + oauthErrDesc
		}
		return failPending(msg, fmt.Errorf("%w: %s", ErrOAuthUnauthorized, msg))
	}
	if strings.TrimSpace(code) == "" {
		return failPending("missing authorization code", fmt.Errorf("%w: missing code", ErrOAuthUpstreamError))
	}

	accessToken, err := m.exchangeCode(ctx, code)
	if err != nil {
		return failPending("Discord token exchange failed", fmt.Errorf("%w: %v", ErrOAuthUpstreamError, err))
	}
	user, err := m.fetchCurrentUser(ctx, accessToken)
	if err != nil {
		return failPending("Discord user lookup failed", fmt.Errorf("%w: %v", ErrOAuthUpstreamError, err))
	}
	uid, err := snowflake.Parse(user.ID)
	if err != nil || uid == 0 {
		return failPending("invalid Discord user ID", fmt.Errorf("%w: invalid user id %q", ErrOAuthUpstreamError, user.ID))
	}
	displayName := user.Username
	if user.GlobalName != nil && strings.TrimSpace(*user.GlobalName) != "" {
		displayName = strings.TrimSpace(*user.GlobalName)
	}

	role, perms, allowed := m.rbac.ResolveUser(uid)
	if !allowed {
		if m.audit != nil {
			m.audit.RecordAction(ctx, 0, user.Username, "unauthorized", "auth.login_denied", "user", uid.String(),
				fmt.Sprintf("Unauthorized OAuth login attempt by @%s", user.Username),
				map[string]any{"user_id": uid.String(), "username": user.Username},
				"error", remoteAddr)
		}
		return failPending("Discord account is not configured in admin.users", ErrOAuthUnauthorized)
	}

	rawToken, sess, err := m.sessions.Create(ctx, uid, user.Username, displayName, role, clientInfo, m.cfg.SessionTTL)
	if err != nil {
		return failPending("failed to persist admin session", err)
	}

	principal := &PrincipalInfo{
		UserID:      uid.String(),
		Username:    user.Username,
		DisplayName: displayName,
		Role:        role,
		Permissions: perms,
	}

	if m.audit != nil {
		m.audit.RecordAction(ctx, uid, displayName, role, "auth.login", "session", sess.ID.String(),
			fmt.Sprintf("Operator @%s (%s) logged in via Discord OAuth2", user.Username, role),
			map[string]any{"client_info": clientInfo},
			"success", remoteAddr)
	}

	m.mu.Lock()
	if cur, exists := m.pendingByLogin[loginID]; exists {
		exp := sess.ExpiresAt
		cur.Status = "approved"
		cur.SessionToken = rawToken
		cur.SessionExpiresAt = &exp
		cur.Principal = principal
	}
	m.mu.Unlock()

	return principal, nil
}

// Poll checks the status of loginID using pollSecret and returns a session token once when approved.
func (m *OAuthManager) Poll(loginID, pollSecret string) (AuthPollResponse, error) {
	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.pendingByLogin[loginID]
	if !ok {
		return AuthPollResponse{Status: "expired"}, ErrLoginNotFound
	}
	if subtle.ConstantTimeCompare([]byte(HashToken(pollSecret)), []byte(p.PollSecretHash)) != 1 {
		return AuthPollResponse{}, ErrInvalidPollSecret
	}
	if now.After(p.ExpiresAt) {
		delete(m.loginByState, p.State)
		delete(m.pendingByLogin, loginID)
		return AuthPollResponse{Status: "expired"}, ErrLoginNotFound
	}

	switch p.Status {
	case "pending":
		return AuthPollResponse{Status: "pending"}, nil
	case "denied":
		msg := p.ErrMessage
		delete(m.loginByState, p.State)
		delete(m.pendingByLogin, loginID)
		return AuthPollResponse{Status: "denied", Error: msg}, nil
	case "approved":
		resp := AuthPollResponse{
			Status:       "approved",
			SessionToken: p.SessionToken,
			ExpiresAt:    p.SessionExpiresAt,
			Principal:    p.Principal,
		}
		// Zero token and consume pending record on first claim.
		p.SessionToken = ""
		delete(m.loginByState, p.State)
		delete(m.pendingByLogin, loginID)
		return resp, nil
	default:
		delete(m.loginByState, p.State)
		delete(m.pendingByLogin, loginID)
		return AuthPollResponse{Status: "expired"}, ErrLoginNotFound
	}
}

func (m *OAuthManager) exchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", m.cfg.ClientID)
	form.Set("client_secret", m.cfg.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", m.cfg.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.DiscordAPIBase+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed discordTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", errors.New("empty access_token in Discord OAuth response")
	}
	return parsed.AccessToken, nil
}

func (m *OAuthManager) fetchCurrentUser(ctx context.Context, accessToken string) (discordUserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.DiscordAPIBase+"/users/@me", nil)
	if err != nil {
		return discordUserResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := m.cfg.HTTPClient.Do(req)
	if err != nil {
		return discordUserResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return discordUserResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return discordUserResponse{}, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var user discordUserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		return discordUserResponse{}, err
	}
	return user, nil
}

func renderCallbackPage(w io.Writer, success bool, heading, message, detail string) {
	_ = callbackHTMLTmpl.Execute(w, map[string]any{
		"Title":   "gobot admin OAuth2",
		"Success": success,
		"Heading": heading,
		"Message": message,
		"Detail":  detail,
	})
}
