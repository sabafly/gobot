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
	"os"
	"strings"
	"time"

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
	cfg        components.Config
	adminCfg   components.ConfigAdmin
	listenAddr string
	db         *database.DB
	components *components.Components
	client     *bot.Client
	bbsSvc     *bbs.Service
	bbsSync    *discordsync.Syncer
	logHub     *LogHub
	rbac       *RBAC
	sessions   *SessionStore
	audit      *AuditStore
	oauth      *OAuthManager
	version    string
	startedAt  time.Time
	mux        *http.ServeMux
}

// NewServer validates configuration and initializes the admin HTTP/SSE API server.
func NewServer(deps ServerDeps) (*Server, error) {
	if deps.DB == nil || deps.DB.DB == nil {
		return nil, errors.New("admin server: database is not initialized")
	}

	adminCfg := deps.AdminConfig
	if !adminCfg.Enabled && adminCfg.ListenAddr == "" && len(adminCfg.Users) == 0 && len(adminCfg.Roles) == 0 && adminCfg.OAuth == (components.ConfigAdminOAuth{}) {
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

	redirectURL := strings.TrimSpace(adminCfg.OAuth.RedirectURL)

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
		cfg:        deps.Config,
		adminCfg:   adminCfg,
		listenAddr: listenAddr,
		db:         deps.DB,
		components: deps.Components,
		client:     deps.Client,
		bbsSvc:     deps.BBSService,
		bbsSync:    deps.BBSSyncer,
		logHub:     logHub,
		rbac:       rbac,
		sessions:   sessions,
		audit:      audit,
		oauth:      oauthMgr,
		version:    deps.Version,
		startedAt:  startedAt,
		mux:        http.NewServeMux(),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) registerRoutes() {
	// Unauthenticated OAuth2 handshake endpoints
	s.mux.HandleFunc("POST /api/v1/auth/discord/start", s.handleAuthStart)
	s.mux.HandleFunc("GET /api/v1/auth/discord/callback", s.handleAuthCallback)
	s.mux.HandleFunc("POST /api/v1/auth/discord/poll", s.handleAuthPoll)

	// Authenticated session lifecycle endpoints
	s.mux.HandleFunc("GET /api/v1/auth/me", s.withAuth("", s.handleAuthMe))
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.withAuth("", s.handleAuthLogout))
	s.mux.HandleFunc("GET /api/v1/auth/sessions", s.withAuth("", s.handleListSessions))
	s.mux.HandleFunc("DELETE /api/v1/auth/sessions/{id}", s.withAuth("", s.handleRevokeSession))

	// Common dashboard endpoints
	s.mux.HandleFunc("GET /api/v1/overview", s.withAuth(PermDashboardRead, s.handleOverview))
	s.mux.HandleFunc("GET /api/v1/logs", s.withAuth(PermLogsRead, s.handleLogs))
	s.mux.HandleFunc("GET /api/v1/logs/stream", s.withAuth(PermLogsRead, s.handleLogsStream))
	s.mux.HandleFunc("GET /api/v1/guilds", s.withAuth(PermGuildsRead, s.handleListGuilds))
	s.mux.HandleFunc("GET /api/v1/guilds/{id}", s.withAuth(PermGuildsRead, s.handleGetGuild))
	s.mux.HandleFunc("PATCH /api/v1/guilds/{id}", s.withAuth(PermGuildsWrite, s.handleUpdateGuild))
	s.mux.HandleFunc("POST /api/v1/ops/reload-locales", s.withAuth(PermOpsExecute, s.handleOpsReloadLocales))
	s.mux.HandleFunc("POST /api/v1/ops/sync-commands", s.withAuth(PermOpsExecute, s.handleOpsSyncCommands))
	s.mux.HandleFunc("POST /api/v1/ops/bbs-dat-check", s.withAuth(PermOpsExecute, s.handleOpsBBSDatCheck))
	s.mux.HandleFunc("GET /api/v1/audit", s.withAuth(PermAuditRead, s.handleListAuditLogs))
}

// Handler returns the root HTTP handler wrapped with panic recovery and security headers.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("admin HTTP handler panic recovered", "path", r.URL.Path, "panic", rec)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
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

	errCh := make(chan error, 1)
	go func() {
		slog.Info("管理APIサーバーを起動しました", "addr", ln.Addr().String())
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
