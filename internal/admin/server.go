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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/disgoorg/disgo/bot"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
)

const (
	defaultListenAddr = "127.0.0.1:8080"
	maxJSONBodyBytes  = 1 << 20 // 1 MiB
)

type authContextKey struct{}

type authContext struct {
	Session   *models.AdminSession
	Principal PrincipalInfo
}

// ServerDeps configures all dependencies required by the admin HTTP/SSE API server.
type ServerDeps struct {
	Config      components.Config
	AdminConfig components.ConfigAdmin
	DB          *database.DB
	Components  *components.Components
	Client      *bot.Client
	BBSService  *bbs.Service
	BBSSyncer   *discordsync.Syncer
	LogHub      *LogHub
	Version     string
	StartedAt   time.Time

	// Optional overrides for Discord OAuth2 endpoints in unit tests.
	OAuthAPIBase       string
	OAuthAuthorizeBase string
	OAuthHTTPClient    *http.Client
}

// Server exposes gobot's management HTTP/SSE API secured by Discord OAuth2 + RBAC.
type Server struct {
	cfg              components.Config
	adminCfg         components.ConfigAdmin
	listenAddr       string
	publicURL        string
	basePath         string
	allowedOrigin    string
	oauthRedirectURL string
	db               *database.DB
	components       *components.Components
	client           *bot.Client
	bbsSvc           *bbs.Service
	bbsSync          *discordsync.Syncer
	logHub           *LogHub
	rbac             *RBAC
	sessions         *SessionStore
	audit            *AuditStore
	oauth            *OAuthManager
	version          string
	startedAt        time.Time
	mux              *http.ServeMux
}

// NormalizeBasePath validates and normalizes a HTTP server base path ("" for root, or "/prefix").
func NormalizeBasePath(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" || s == "/" {
		return "", nil
	}
	if strings.Contains(s, "://") {
		return "", fmt.Errorf("admin server: base_path must be a path, not a full URL %q", raw)
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f || unicode.IsSpace(r) {
			return "", fmt.Errorf("admin server: base_path contains invalid whitespace or control character %q", raw)
		}
	}
	if strings.ContainsAny(s, "?#\\{}") || strings.Contains(s, "%") {
		return "", fmt.Errorf("admin server: base_path contains forbidden characters %q", raw)
	}
	if strings.Contains(s, "//") {
		return "", fmt.Errorf("admin server: base_path must not contain empty path segments %q", raw)
	}
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return "", nil
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	for _, seg := range strings.Split(s[1:], "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Errorf("admin server: invalid base_path segment in %q", raw)
		}
	}
	if path.Clean(s) != s {
		return "", fmt.Errorf("admin server: invalid base_path %q", raw)
	}
	return s, nil
}

func formatCanonicalOrigin(u *url.URL) (string, error) {
	if u == nil {
		return "", errors.New("nil URL")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	hostname := strings.ToLower(u.Hostname())
	if hostname == "" || strings.HasSuffix(u.Host, ":") {
		return "", errors.New("host is required")
	}
	port := u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	var canonHost string
	switch {
	case port != "":
		canonHost = net.JoinHostPort(hostname, port)
	case strings.Contains(hostname, ":"):
		canonHost = "[" + hostname + "]"
	default:
		canonHost = hostname
	}
	return scheme + "://" + canonHost, nil
}

// NormalizePublicURL validates and normalizes the public external URL, returning (publicURL, basePath, origin, err).
func NormalizePublicURL(raw string) (publicURL, basePath, origin string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", "", errors.New("admin server: public_url is required")
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f || unicode.IsSpace(r) {
			return "", "", "", fmt.Errorf("admin server: public_url contains invalid whitespace or control character %q", raw)
		}
	}
	if strings.ContainsAny(s, "?#") {
		return "", "", "", fmt.Errorf("admin server: public_url must not contain query or fragment %q", raw)
	}
	u, err := url.Parse(s)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Host == "" || u.Hostname() == "" || strings.HasSuffix(u.Host, ":") {
		return "", "", "", fmt.Errorf("admin server: invalid public_url %q", raw)
	}
	origin, err = formatCanonicalOrigin(u)
	if err != nil {
		return "", "", "", fmt.Errorf("admin server: invalid public_url %q: %w", raw, err)
	}
	basePath, err = NormalizeBasePath(u.EscapedPath())
	if err != nil {
		return "", "", "", fmt.Errorf("admin server: invalid path in public_url %q: %w", raw, err)
	}
	return origin + basePath, basePath, origin, nil
}

func canonicalOrigin(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" || strings.EqualFold(s, "null") {
		return "", false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f || unicode.IsSpace(r) {
			return "", false
		}
	}
	if strings.ContainsAny(s, "?#") {
		return "", false
	}
	u, err := url.Parse(s)
	if err != nil || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Host == "" || u.Hostname() == "" || strings.HasSuffix(u.Host, ":") {
		return "", false
	}
	orig, err := formatCanonicalOrigin(u)
	return orig, err == nil
}

// PublicURL returns the normalized public base URL configured for this server.
func (s *Server) PublicURL() string {
	return s.publicURL
}

// BasePath returns the normalized effective HTTP route prefix ("" when mounted at root).
func (s *Server) BasePath() string {
	return s.basePath
}

// NewServer validates configuration and initializes the admin HTTP/SSE API server.
func NewServer(deps ServerDeps) (*Server, error) {
	if deps.DB == nil || deps.DB.DB == nil {
		return nil, errors.New("admin server: database is not initialized")
	}

	adminCfg := deps.AdminConfig
	if !adminCfg.Enabled && adminCfg.ListenAddr == "" && adminCfg.PublicURL == "" && adminCfg.BasePath == "" && len(adminCfg.Users) == 0 && len(adminCfg.Roles) == 0 && adminCfg.OAuth == (components.ConfigAdminOAuth{}) {
		adminCfg = deps.Config.Admin
	}

	rbac, err := NewRBAC(adminCfg)
	if err != nil {
		return nil, err
	}

	listenAddr := strings.TrimSpace(adminCfg.ListenAddr)
	if listenAddr == "" {
		listenAddr = defaultListenAddr
	}

	rawPublicURL := strings.TrimSpace(os.Getenv("ADMIN_PUBLIC_URL"))
	if rawPublicURL == "" {
		rawPublicURL = strings.TrimSpace(adminCfg.PublicURL)
	}
	publicURL, publicPath, allowedOrigin, err := NormalizePublicURL(rawPublicURL)
	if err != nil {
		return nil, err
	}

	rawBasePath := strings.TrimSpace(os.Getenv("ADMIN_BASE_PATH"))
	if rawBasePath == "" {
		rawBasePath = strings.TrimSpace(adminCfg.BasePath)
	}
	basePath := publicPath
	if rawBasePath != "" {
		basePath, err = NormalizeBasePath(rawBasePath)
		if err != nil {
			return nil, err
		}
	}

	sessionTTL := defaultSessionTTL
	if adminCfg.SessionTTLHours > 0 {
		sessionTTL = time.Duration(adminCfg.SessionTTLHours) * time.Hour
	}

	clientID := strings.TrimSpace(adminCfg.OAuth.ClientID)
	if clientID == "" && deps.Client != nil && deps.Client.ApplicationID != 0 {
		clientID = deps.Client.ApplicationID.String()
	}

	clientSecret := strings.TrimSpace(os.Getenv("ADMIN_DISCORD_CLIENT_SECRET"))
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET"))
	}
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(adminCfg.OAuth.ClientSecret)
	}

	redirectURL := publicURL + "/api/v1/auth/discord/callback"

	logHub := deps.LogHub
	if logHub == nil {
		logHub = NewLogHub(adminCfg.LogBufferSize)
	} else if adminCfg.LogBufferSize > 0 {
		logHub.SetCapacity(adminCfg.LogBufferSize)
	}

	sessions := NewSessionStore(deps.DB.DB)
	audit := NewAuditStore(deps.DB.DB)
	oauthMgr, err := NewOAuthManager(OAuthConfig{
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		RedirectURL:          redirectURL,
		DiscordAPIBase:       deps.OAuthAPIBase,
		DiscordAuthorizeBase: deps.OAuthAuthorizeBase,
		HTTPClient:           deps.OAuthHTTPClient,
		SessionTTL:           sessionTTL,
	}, rbac, sessions, audit)
	if err != nil {
		return nil, err
	}

	startedAt := deps.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	s := &Server{
		cfg:              deps.Config,
		adminCfg:         adminCfg,
		listenAddr:       listenAddr,
		publicURL:        publicURL,
		basePath:         basePath,
		allowedOrigin:    allowedOrigin,
		oauthRedirectURL: redirectURL,
		db:               deps.DB,
		components:       deps.Components,
		client:           deps.Client,
		bbsSvc:           deps.BBSService,
		bbsSync:          deps.BBSSyncer,
		logHub:           logHub,
		rbac:             rbac,
		sessions:         sessions,
		audit:            audit,
		oauth:            oauthMgr,
		version:          deps.Version,
		startedAt:        startedAt,
		mux:              http.NewServeMux(),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) handle(method, pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(method+" "+s.basePath+pattern, h)
}

func (s *Server) registerRoutes() {
	// Unauthenticated OAuth2 handshake endpoints
	s.handle(http.MethodPost, "/api/v1/auth/discord/start", s.handleAuthStart)
	s.handle(http.MethodGet, "/api/v1/auth/discord/callback", s.handleAuthCallback)
	s.handle(http.MethodPost, "/api/v1/auth/discord/poll", s.handleAuthPoll)

	// Authenticated session lifecycle endpoints
	s.handle(http.MethodGet, "/api/v1/auth/me", s.withAuth("", s.handleAuthMe))
	s.handle(http.MethodPost, "/api/v1/auth/logout", s.withAuth("", s.handleAuthLogout))
	s.handle(http.MethodGet, "/api/v1/auth/sessions", s.withAuth("", s.handleListSessions))
	s.handle(http.MethodDelete, "/api/v1/auth/sessions/{id}", s.withAuth("", s.handleRevokeSession))

	// Common dashboard endpoints
	s.handle(http.MethodGet, "/api/v1/overview", s.withAuth(PermDashboardRead, s.handleOverview))
	s.handle(http.MethodGet, "/api/v1/logs", s.withAuth(PermLogsRead, s.handleLogs))
	s.handle(http.MethodGet, "/api/v1/logs/stream", s.withAuth(PermLogsRead, s.handleLogsStream))
	s.handle(http.MethodGet, "/api/v1/guilds", s.withAuth(PermGuildsRead, s.handleListGuilds))
	s.handle(http.MethodGet, "/api/v1/guilds/{id}", s.withAuth(PermGuildsRead, s.handleGetGuild))
	s.handle(http.MethodPatch, "/api/v1/guilds/{id}", s.withAuth(PermGuildsWrite, s.handleUpdateGuild))
	s.handle(http.MethodPost, "/api/v1/ops/reload-locales", s.withAuth(PermOpsExecute, s.handleOpsReloadLocales))
	s.handle(http.MethodPost, "/api/v1/ops/sync-commands", s.withAuth(PermOpsExecute, s.handleOpsSyncCommands))
	s.handle(http.MethodPost, "/api/v1/ops/bbs-dat-check", s.withAuth(PermOpsExecute, s.handleOpsBBSDatCheck))
	s.handle(http.MethodGet, "/api/v1/audit", s.withAuth(PermAuditRead, s.handleListAuditLogs))
}

// Handler returns the root HTTP handler wrapped with panic recovery, CORS/Origin enforcement, and security headers.
func (s *Server) Handler() http.Handler {
	callbackPath := s.basePath + "/api/v1/auth/discord/callback"
	apiPrefix := s.basePath + "/api/v1/"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("admin HTTP handler panic recovered", "path", r.URL.Path, "panic", rec)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")

		if rawOrigin := strings.TrimSpace(r.Header.Get("Origin")); rawOrigin != "" {
			if orig, ok := canonicalOrigin(rawOrigin); ok && orig == s.allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", rawOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
				if r.Method == http.MethodOptions {
					w.Header().Add("Vary", "Access-Control-Request-Method")
					w.Header().Add("Vary", "Access-Control-Request-Headers")
					if strings.HasPrefix(r.URL.Path, apiPrefix) {
						w.WriteHeader(http.StatusNoContent)
						return
					}
				}
			} else if !(r.Method == http.MethodGet && r.URL.Path == callbackPath) {
				writeError(w, http.StatusForbidden, "forbidden_origin", "origin is not allowed")
				return
			}
		}

		s.mux.ServeHTTP(w, r)
	})
}

// ListenAndServe runs the admin HTTP/SSE server until ctx is canceled, then shuts down cleanly.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return fmt.Errorf("admin server listen %s failed: %w", s.listenAddr, err)
	}
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	displayBasePath := s.basePath
	if displayBasePath == "" {
		displayBasePath = "/"
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("管理APIサーバーを起動しました",
			"addr", ln.Addr().String(),
			"base_path", displayBasePath,
			"public_url", s.publicURL,
			"oauth_redirect_url", s.oauthRedirectURL,
		)
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		return <-errCh
	case serveErr := <-errCh:
		return serveErr
	}
}

type authedHandler func(w http.ResponseWriter, r *http.Request, auth authContext)

func (s *Server) withAuth(required Permission, next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer session token")
			return
		}
		sess, err := s.sessions.Authenticate(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired session token")
			return
		}
		// Re-evaluate RBAC on every request so role/user removals take effect immediately.
		role, perms, ok := s.rbac.ResolveUser(sess.UserID)
		if !ok {
			writeError(w, http.StatusForbidden, "forbidden", "operator account is no longer permitted")
			return
		}
		if required != "" && !HasPermission(perms, required) {
			writeError(w, http.StatusForbidden, "forbidden", fmt.Sprintf("missing required permission: %s", required))
			return
		}
		auth := authContext{
			Session: sess,
			Principal: PrincipalInfo{
				UserID:      sess.UserID.String(),
				Username:    sess.Username,
				DisplayName: sess.DisplayName,
				Role:        role,
				Permissions: perms,
			},
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, auth)
		next(w, r.WithContext(ctx), auth)
	}
}

func extractBearerToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(h) < 7 || !strings.EqualFold(h[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: msg,
		},
	})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		if allowEmpty {
			return nil
		}
		return errors.New("request body is empty")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return err
	}
	return nil
}

func clientRemoteAddr(r *http.Request) string {
	if r == nil {
		return ""
	}
	if host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil && host != "" {
		return truncateRunes(host, 64)
	}
	return truncateRunes(r.RemoteAddr, 64)
}
