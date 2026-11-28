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
	"log/slog"
	"net/http"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
	"github.com/sabafly/gobot/internal/i18n"
)

// --- Auth & Session Handlers ---

func (s *Server) handleAuthStart(w http.ResponseWriter, r *http.Request) {
	remoteIP := s.clientRemoteAddr(r)
	if allowed, retryAfter := s.authStartLimiter.Allow(remoteIP, time.Now()); !allowed {
		writeRateLimited(w, retryAfter, "rate_limited", "too many login start requests; please retry later")
		return
	}
	var req AuthStartRequest
	if err := decodeJSONBody(w, r, &req, true); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if strings.TrimSpace(req.ClientInfo) == "" {
		req.ClientInfo = r.UserAgent()
	}
	resp, err := s.oauth.Start(req.ClientInfo, remoteIP)
	if err != nil {
		if errors.Is(err, ErrTooManyPendingLogins) {
			writeRateLimited(w, 60*time.Second, "too_many_pending_logins", "too many pending login requests; please retry later")
			return
		}
		writeError(w, http.StatusInternalServerError, "oauth_start_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	code := q.Get("code")
	state := q.Get("state")
	oauthErr := q.Get("error")
	oauthErrDesc := q.Get("error_description")

	remoteIP := s.clientRemoteAddr(r)
	principal, err := s.oauth.HandleCallback(r.Context(), code, state, oauthErr, oauthErrDesc, remoteIP)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		status := http.StatusBadRequest
		heading := "認可に失敗しました"
		msg := "認証処理を完了できませんでした。ターミナルに戻ってやり直してください。"
		detailCode := "oauth_exchange_failed"
		switch {
		case errors.Is(err, ErrOAuthUnauthorized):
			status = http.StatusForbidden
			heading = "アクセス権限がありません"
			msg = "この Discord アカウントは gobot 管理ダッシュボードの許可ユーザーに登録されていません。"
			detailCode = "unauthorized"
		case errors.Is(err, ErrOAuthInvalidState):
			status = http.StatusBadRequest
			heading = "認証セッションが無効または期限切れです"
			msg = "ターミナル側で再度ログインを開始してください。"
			detailCode = "invalid_or_expired_state"
		}
		slog.Warn("admin oauth callback failed", "code", detailCode, "remote_addr", remoteIP, "err", err)
		w.WriteHeader(status)
		renderCallbackPage(w, false, heading, msg, detailCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	renderCallbackPage(w, true,
		"認証が完了しました",
		"このウィンドウを閉じてターミナル（gobot tui）へ戻ってください。",
		fmt.Sprintf("@%s (%s)", principal.Username, principal.Role),
	)
}

func (s *Server) handleAuthPoll(w http.ResponseWriter, r *http.Request) {
	remoteIP := s.clientRemoteAddr(r)
	if allowed, retryAfter := s.authPollLimiter.Allow(remoteIP, time.Now()); !allowed {
		writeRateLimited(w, retryAfter, "rate_limited", "too many poll requests; please retry later")
		return
	}
	var req AuthPollRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid poll request body")
		return
	}
	if strings.TrimSpace(req.LoginID) == "" || strings.TrimSpace(req.PollSecret) == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "login_id and poll_secret are required")
		return
	}
	resp, err := s.oauth.Poll(req.LoginID, req.PollSecret)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPollSecret):
			writeError(w, http.StatusUnauthorized, "invalid_poll_secret", "invalid poll_secret")
		case errors.Is(err, ErrLoginNotFound):
			writeError(w, http.StatusNotFound, "login_expired", "login request not found or expired")
		default:
			writeError(w, http.StatusInternalServerError, "poll_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthMe(w http.ResponseWriter, _ *http.Request, auth authContext) {
	writeJSON(w, http.StatusOK, auth.Principal)
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request, auth authContext) {
	if err := s.sessions.Revoke(r.Context(), auth.Session.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "logout_failed", err.Error())
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"auth.logout", "session", auth.Session.ID.String(),
		"Operator logged out current session", nil, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request, auth authContext) {
	var filterUID *snowflake.ID
	if !HasPermission(auth.Principal.Permissions, PermSessionsManage) {
		uid := auth.Session.UserID
		filterUID = &uid
	}
	rows, err := s.sessions.List(r.Context(), filterUID, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sessions_list_failed", err.Error())
		return
	}
	out := make([]SessionInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, SessionInfo{
			ID:          row.ID.String(),
			UserID:      row.UserID.String(),
			Username:    row.Username,
			DisplayName: row.DisplayName,
			Role:        row.Role,
			ClientInfo:  row.ClientInfo,
			Current:     row.ID == auth.Session.ID,
			ExpiresAt:   row.ExpiresAt,
			LastUsedAt:  row.LastUsedAt,
			CreatedAt:   row.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, SessionsResponse{Sessions: out})
}

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request, auth authContext) {
	id, err := uuid.Parse(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid session id")
		return
	}
	target, err := s.sessions.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "session_lookup_failed", err.Error())
		return
	}
	if target.UserID != auth.Session.UserID && !HasPermission(auth.Principal.Permissions, PermSessionsManage) {
		writeError(w, http.StatusForbidden, "forbidden", "cannot revoke another user's session")
		return
	}
	if err := s.sessions.Revoke(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "session_revoke_failed", err.Error())
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"session.revoke", "session", id.String(),
		fmt.Sprintf("Revoked session for user %s", target.UserID.String()),
		map[string]any{"target_user_id": target.UserID.String(), "self": target.ID == auth.Session.ID},
		"success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true, "id": id.String()})
}

// --- Overview ---

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request, _ authContext) {
	ctx := r.Context()
	now := time.Now().UTC()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	dbConnected := false
	var dbLatencyMs int64
	var dbErrStr string
	var dbGuilds, dbUsers int64

	if s.db != nil && s.db.DB != nil {
		pingStart := time.Now()
		if sqlDB, err := s.db.DB.DB(); err != nil {
			dbErrStr = err.Error()
		} else if err := sqlDB.PingContext(ctx); err != nil {
			dbErrStr = err.Error()
		} else {
			dbConnected = true
			dbLatencyMs = time.Since(pingStart).Milliseconds()
		}

		gdb := s.db.DB.WithContext(ctx)
		if gdb.Migrator().HasTable(&models.Guild{}) {
			_ = gdb.Model(&models.Guild{}).Count(&dbGuilds).Error
		}
		if gdb.Migrator().HasTable(&models.User{}) {
			_ = gdb.Model(&models.User{}).Count(&dbUsers).Error
		}
	}

	activeSessions, _ := s.sessions.CountActive(ctx)
	logLen, logCap := s.logHub.Stats()

	var botConnected bool
	var botUsername string
	var cachedGuilds int
	var shards []ShardStatus
	var avgShardPingMs int64

	if s.client != nil {
		if s.client.Caches != nil {
			cachedGuilds = s.client.Caches.GuildsLen()
			if self, ok := s.client.Caches.SelfUser(); ok {
				botUsername = self.Username
			}
		}
		if s.client.ShardManager != nil {
			var pingSum int64
			var pingCount int64
			for shard := range s.client.ShardManager.Shards() {
				lat := shard.Latency().Milliseconds()
				shards = append(shards, ShardStatus{
					ShardID:   shard.ShardID(),
					Status:    shard.Status().String(),
					LatencyMs: lat,
				})
				if shard.Status().IsConnected() {
					botConnected = true
				}
				if lat >= 0 {
					pingSum += lat
					pingCount++
				}
			}
			slices.SortFunc(shards, func(a, b ShardStatus) int {
				return a.ShardID - b.ShardID
			})
			if pingCount > 0 {
				avgShardPingMs = pingSum / pingCount
			}
		} else if s.client.Gateway != nil {
			lat := s.client.Gateway.Latency().Milliseconds()
			shards = append(shards, ShardStatus{
				ShardID:   s.client.Gateway.ShardID(),
				Status:    s.client.Gateway.Status().String(),
				LatencyMs: lat,
			})
			botConnected = s.client.Gateway.Status().IsConnected()
			if lat >= 0 {
				avgShardPingMs = lat
			}
		}
	}

	bbsSummary := s.collectBBSSummary(ctx)

	writeJSON(w, http.StatusOK, OverviewResponse{
		Version:        s.version,
		StartedAt:      s.startedAt,
		UptimeSeconds:  max(int64(now.Sub(s.startedAt).Seconds()), 0),
		GoVersion:      runtime.Version(),
		Goroutines:     runtime.NumGoroutine(),
		MemAllocBytes:  ms.Alloc,
		MemSysBytes:    ms.Sys,
		DBConnected:    dbConnected,
		DBLatencyMs:    dbLatencyMs,
		DBError:        dbErrStr,
		BotConnected:   botConnected,
		BotUsername:    botUsername,
		CachedGuilds:   cachedGuilds,
		DBGuilds:       dbGuilds,
		DBUsers:        dbUsers,
		AvgShardPingMs: avgShardPingMs,
		Shards:         shards,
		LogBufferLen:   logLen,
		LogBufferCap:   logCap,
		ActiveSessions: activeSessions,
		BBS:            bbsSummary,
	})
}

func (s *Server) collectBBSSummary(ctx context.Context) BBSSummary {
	if s.bbsSvc == nil || s.db == nil || s.db.DB == nil {
		return BBSSummary{}
	}
	gdb := s.db.DB.WithContext(ctx)
	summary := BBSSummary{Available: true}
	if gdb.Migrator().HasTable(&gormstore.BBSCategory{}) {
		_ = gdb.Model(&gormstore.BBSCategory{}).Count(&summary.Categories).Error
	}
	if gdb.Migrator().HasTable(&gormstore.BBSBoard{}) {
		_ = gdb.Model(&gormstore.BBSBoard{}).Count(&summary.Boards).Error
	}
	if gdb.Migrator().HasTable(&gormstore.BBSBoardMount{}) {
		_ = gdb.Model(&gormstore.BBSBoardMount{}).Where("active = ?", true).Count(&summary.ActiveMounts).Error
	}
	if gdb.Migrator().HasTable(&gormstore.BBSThread{}) {
		_ = gdb.Model(&gormstore.BBSThread{}).Count(&summary.Threads).Error
	}
	if gdb.Migrator().HasTable(&gormstore.BBSPost{}) {
		_ = gdb.Model(&gormstore.BBSPost{}).Count(&summary.Posts).Error
	}
	if gdb.Migrator().HasTable(&gormstore.BBSReport{}) {
		_ = gdb.Model(&gormstore.BBSReport{}).Where("status = ?", string(bbs.ReportStatusOpen)).Count(&summary.OpenReports).Error
	}
	if gdb.Migrator().HasTable(&gormstore.BBSBan{}) {
		_ = gdb.Model(&gormstore.BBSBan{}).Where("expires_at IS NULL OR expires_at > ?", time.Now().UTC()).Count(&summary.ActiveBans).Error
	}
	return summary
}

// --- Logs ---

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request, _ authContext) {
	q := r.URL.Query()
	var afterSeq uint64
	if raw := strings.TrimSpace(q.Get("after_seq")); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
			afterSeq = v
		}
	}
	var limit int
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	entries, nextSeq := s.logHub.List(LogFilter{
		MinLevel: q.Get("level"),
		Query:    q.Get("q"),
		AfterSeq: afterSeq,
		Limit:    limit,
	})
	if entries == nil {
		entries = []LogEntry{}
	}
	writeJSON(w, http.StatusOK, LogsResponse{
		Entries: entries,
		NextSeq: nextSeq,
	})
}

func (s *Server) handleGetLogLevel(w http.ResponseWriter, r *http.Request, _ authContext) {
	writeJSON(w, http.StatusOK, s.logHub.CaptureLevel(r.Context()))
}

func (s *Server) handleUpdateLogLevel(w http.ResponseWriter, r *http.Request, auth authContext) {
	var req UpdateLogLevelRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid log level payload")
		return
	}
	resp, err := s.logHub.SetCaptureLevel(req.Level)
	if err != nil {
		s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
			"logs.level.update", "logs", "capture_level", "Rejected invalid log capture level",
			map[string]any{"level": req.Level, "error": err.Error()}, "error", s.clientRemoteAddr(r))
		writeError(w, http.StatusBadRequest, "invalid_log_level", err.Error())
		return
	}
	s.audit.RecordAction(r.Context(), auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"logs.level.update", "logs", "capture_level",
		fmt.Sprintf("Updated log capture level to %s (effective: %s)", resp.Mode, resp.EffectiveLevel),
		map[string]any{"mode": resp.Mode, "effective_level": resp.EffectiveLevel},
		"success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request, _ authContext) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported", "streaming is not supported by response writer")
		return
	}

	q := r.URL.Query()
	minLevel := q.Get("level")
	queryStr := q.Get("q")
	var afterSeq uint64
	hasAfterSeq := false
	if raw := strings.TrimSpace(q.Get("after_seq")); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
			afterSeq = v
			hasAfterSeq = true
		}
	}

	// Subscribe before reading initial backlog so no entries are lost in between.
	subCh, unsub := s.logHub.Subscribe(256)
	defer unsub()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	var lastSentSeq uint64
	if hasAfterSeq {
		backlog, _ := s.logHub.List(LogFilter{
			MinLevel: minLevel,
			Query:    queryStr,
			AfterSeq: afterSeq,
			Limit:    maxLogListLimit,
		})
		for _, entry := range backlog {
			if err := writeSSELogEvent(w, entry); err != nil {
				return
			}
			lastSentSeq = entry.Seq
		}
	} else {
		// Emit an initial heartbeat comment so clients confirm the stream is live immediately.
		if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
			return
		}
	}
	flusher.Flush()

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case entry, open := <-subCh:
			if !open {
				return
			}
			if entry.Seq <= lastSentSeq {
				continue
			}
			if !MatchLogEntry(entry, minLevel, queryStr) {
				continue
			}
			if err := writeSSELogEvent(w, entry); err != nil {
				return
			}
			lastSentSeq = entry.Seq
			flusher.Flush()
		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSELogEvent(w http.ResponseWriter, entry LogEntry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "id: %d\nevent: log\ndata: %s\n\n", entry.Seq, raw)
	return err
}

// --- Guilds ---

func (s *Server) handleListGuilds(w http.ResponseWriter, r *http.Request, _ authContext) {
	guilds, err := s.collectGuildSummaries(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "guilds_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, GuildsResponse{
		Guilds:           guilds,
		AvailableLocales: availableLocaleCodes(),
	})
}

func (s *Server) handleGetGuild(w http.ResponseWriter, r *http.Request, _ authContext) {
	gid, err := snowflake.Parse(strings.TrimSpace(r.PathValue("id")))
	if err != nil || gid == 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid guild id")
		return
	}
	detail, found, err := s.collectGuildDetail(r.Context(), gid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "guild_lookup_failed", err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "not_found", "guild not found")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleUpdateGuild(w http.ResponseWriter, r *http.Request, auth authContext) {
	ctx := r.Context()
	gid, err := snowflake.Parse(strings.TrimSpace(r.PathValue("id")))
	if err != nil || gid == 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid guild id")
		return
	}
	var req UpdateGuildRequest
	if err := decodeJSONBody(w, r, &req, false); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid guild update payload")
		return
	}
	if req.LocaleOverride == nil && req.BBSEnabled == nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "at least one field must be provided")
		return
	}

	// Pre-validate both requested updates before mutating state to avoid partial updates.
	var targetLocale discord.Locale
	var prevLocale discord.Locale
	var rawLocale string
	if req.LocaleOverride != nil {
		if s.components == nil {
			writeError(w, http.StatusServiceUnavailable, "unavailable", "components not initialized")
			return
		}
		rawLocale = strings.TrimSpace(*req.LocaleOverride)
		if rawLocale != "" && rawLocale != "auto" {
			loc := discord.Locale(rawLocale)
			if !i18n.IsSupportedLocale(loc) {
				writeError(w, http.StatusBadRequest, "invalid_locale", fmt.Sprintf("unsupported locale %q", rawLocale))
				return
			}
			targetLocale = loc
		}
		prevLocale = s.components.GuildLocaleOverride(ctx, gid)
	}

	var setupIn bbs.GuildSetupInput
	if req.BBSEnabled != nil {
		if s.bbsSvc == nil {
			writeError(w, http.StatusServiceUnavailable, "bbs_unavailable", "BBS system is not initialized")
			return
		}
		existingCfg, cfgErr := s.bbsSvc.GetGuildConfig(ctx, gid)
		if cfgErr != nil {
			writeError(w, http.StatusInternalServerError, "bbs_config_load_failed", cfgErr.Error())
			return
		}
		setupIn = bbs.GuildSetupInput{
			GuildID:     gid,
			Enabled:     *req.BBSEnabled,
			ActorUserID: auth.Session.UserID,
		}
		if existingCfg != nil {
			defForum := existingCfg.DefaultUseForum
			setupIn.DefaultUseForum = &defForum
			setupIn.ModLogChannelID = existingCfg.ModLogChannelID
		}
	}

	if req.LocaleOverride != nil {
		if err := s.components.SetGuildLocaleOverride(ctx, gid, targetLocale); err != nil {
			s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
				"guild.update", "guild", gid.String(), "Failed to update guild locale override",
				map[string]any{"locale_override": rawLocale, "error": err.Error()}, "error", s.clientRemoteAddr(r))
			writeError(w, http.StatusInternalServerError, "guild_update_failed", err.Error())
			return
		}
	}

	if req.BBSEnabled != nil {
		if _, err := s.bbsSvc.SetupGuild(ctx, setupIn); err != nil {
			if req.LocaleOverride != nil {
				if rbErr := s.components.SetGuildLocaleOverride(ctx, gid, prevLocale); rbErr != nil {
					slog.Error("failed to rollback guild locale override after bbs setup error",
						"guild_id", gid, "prev_locale", prevLocale, "err", rbErr)
				}
			}
			s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
				"guild.update", "guild", gid.String(), "Failed to update guild BBS enabled state",
				map[string]any{"bbs_enabled": *req.BBSEnabled, "error": err.Error()}, "error", s.clientRemoteAddr(r))
			writeError(w, http.StatusInternalServerError, "bbs_setup_failed", err.Error())
			return
		}
	}

	if req.LocaleOverride != nil {
		effective := s.components.ResolveGuildLocale(ctx, s.client, gid, "")
		s.components.NotifyGuildLocaleChanged(ctx, s.client, gid, effective)
	}

	s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"guild.update", "guild", gid.String(),
		fmt.Sprintf("Updated settings for guild %s", gid.String()),
		req, "success", s.clientRemoteAddr(r))

	detail, _, err := s.collectGuildDetail(ctx, gid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "guild_lookup_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) collectGuildSummaries(ctx context.Context, query string) ([]GuildSummary, error) {
	items := make(map[snowflake.ID]*GuildSummary)
	ensure := func(id snowflake.ID) *GuildSummary {
		if cur, ok := items[id]; ok {
			return cur
		}
		g := &GuildSummary{ID: id.String()}
		items[id] = g
		return g
	}

	if s.client != nil && s.client.Caches != nil {
		for g := range s.client.Caches.Guilds() {
			item := ensure(g.ID)
			item.Name = g.Name
			item.Cached = true
			item.MemberCount = g.MemberCount
			item.PreferredLocale = g.PreferredLocale
		}
	}

	if s.db != nil && s.db.DB != nil {
		gdb := s.db.DB.WithContext(ctx)
		if gdb.Migrator().HasTable(&models.Guild{}) {
			var dbGuilds []models.Guild
			if err := gdb.Select("id", "name", "locale_override").Find(&dbGuilds).Error; err != nil {
				return nil, err
			}
			for _, g := range dbGuilds {
				item := ensure(g.ID)
				if item.Name == "" {
					item.Name = g.Name
				}
				item.LocaleOverride = string(g.LocaleOverride)
			}
		}
		if gdb.Migrator().HasTable(&gormstore.BBSGuildConfig{}) {
			var cfgs []gormstore.BBSGuildConfig
			if err := gdb.Select("guild_id", "enabled").Find(&cfgs).Error; err == nil {
				for _, cfg := range cfgs {
					item := ensure(cfg.GuildID)
					item.BBSEnabled = cfg.Enabled
				}
			}
		}
		if gdb.Migrator().HasTable(&gormstore.BBSBoardMount{}) {
			type mountCount struct {
				GuildID snowflake.ID
				Cnt     int
			}
			var counts []mountCount
			if err := gdb.Model(&gormstore.BBSBoardMount{}).
				Select("guild_id, COUNT(*) AS cnt").
				Where("active = ?", true).
				Group("guild_id").
				Scan(&counts).Error; err == nil {
				for _, mc := range counts {
					item := ensure(mc.GuildID)
					item.BBSMounts = mc.Cnt
				}
			}
		}
	}

	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]GuildSummary, 0, len(items))
	for id, item := range items {
		item.EffectiveLocale = string(s.resolveGuildLocale(ctx, id, item.LocaleOverride, item.PreferredLocale))
		if q != "" {
			if !strings.Contains(strings.ToLower(item.Name), q) &&
				!strings.Contains(strings.ToLower(item.ID), q) &&
				!strings.Contains(strings.ToLower(item.EffectiveLocale), q) {
				continue
			}
		}
		out = append(out, *item)
	}
	slices.SortFunc(out, func(a, b GuildSummary) int {
		if a.Cached != b.Cached {
			if a.Cached {
				return -1
			}
			return 1
		}
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out, nil
}

func (s *Server) collectGuildDetail(ctx context.Context, gid snowflake.ID) (GuildDetailResponse, bool, error) {
	var summary GuildSummary
	summary.ID = gid.String()
	found := false

	if s.client != nil && s.client.Caches != nil {
		if cg, ok := s.client.Caches.Guild(gid); ok {
			found = true
			summary.Name = cg.Name
			summary.Cached = true
			summary.MemberCount = cg.MemberCount
			summary.PreferredLocale = cg.PreferredLocale
		}
	}

	var bbsForumDefault *bool
	var bbsModLog string
	var bbsModLogName string
	if s.db != nil && s.db.DB != nil {
		gdb := s.db.DB.WithContext(ctx)
		if gdb.Migrator().HasTable(&models.Guild{}) {
			var g models.Guild
			if err := gdb.Select("id", "name", "locale_override").Where("id = ?", gid).First(&g).Error; err == nil {
				found = true
				if summary.Name == "" {
					summary.Name = g.Name
				}
				summary.LocaleOverride = string(g.LocaleOverride)
			}
		}
		if gdb.Migrator().HasTable(&gormstore.BBSGuildConfig{}) {
			var cfg gormstore.BBSGuildConfig
			if err := gdb.Where("guild_id = ?", gid).First(&cfg).Error; err == nil {
				found = true
				summary.BBSEnabled = cfg.Enabled
				v := cfg.DefaultUseForum
				bbsForumDefault = &v
				if cfg.ModLogChannelID != nil {
					bbsModLog = cfg.ModLogChannelID.String()
					if names := s.resolveChannelNames([]snowflake.ID{*cfg.ModLogChannelID}); len(names) > 0 {
						bbsModLogName = names[*cfg.ModLogChannelID]
					}
				}
			}
		}
		if gdb.Migrator().HasTable(&gormstore.BBSBoardMount{}) {
			var cnt int64
			if err := gdb.Model(&gormstore.BBSBoardMount{}).Where("guild_id = ? AND active = ?", gid, true).Count(&cnt).Error; err == nil {
				summary.BBSMounts = int(cnt)
			}
		}
	}
	summary.EffectiveLocale = string(s.resolveGuildLocale(ctx, gid, summary.LocaleOverride, summary.PreferredLocale))
	return GuildDetailResponse{
		Guild:                summary,
		AvailableLocales:     availableLocaleCodes(),
		BBSForumDefault:      bbsForumDefault,
		BBSModLogChannel:     bbsModLog,
		BBSModLogChannelName: bbsModLogName,
	}, found, nil
}

func (s *Server) resolveGuildLocale(_ context.Context, _ snowflake.ID, override, preferred string) discord.Locale {
	if override != "" && override != discord.LocaleUnknown.Code() {
		return discord.Locale(override)
	}
	if preferred != "" && preferred != discord.LocaleUnknown.Code() {
		return discord.Locale(preferred)
	}
	return i18n.DefaultLocale()
}

func availableLocaleCodes() []string {
	locs := i18n.AvailableLocales()
	out := make([]string, 0, len(locs))
	for _, loc := range locs {
		out = append(out, loc.Code())
	}
	return out
}

// --- Ops ---

func (s *Server) handleOpsReloadLocales(w http.ResponseWriter, r *http.Request, auth authContext) {
	ctx := r.Context()
	dir := strings.TrimSpace(s.cfg.LocaleDir)
	if dir == "" {
		dir = "locales"
	}
	if err := i18n.LoadLocales(dir); err != nil {
		s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
			"ops.reload_locales", "locales", dir, "Failed to reload locale files",
			map[string]any{"dir": dir, "error": err.Error()}, "error", s.clientRemoteAddr(r))
		writeError(w, http.StatusInternalServerError, "reload_locales_failed", err.Error())
		return
	}
	codes := availableLocaleCodes()
	s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"ops.reload_locales", "locales", dir, "Reloaded locale files",
		map[string]any{"dir": dir, "locales": codes}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, OpsReloadLocalesResponse{Locales: codes})
}

func (s *Server) handleOpsSyncCommands(w http.ResponseWriter, r *http.Request, auth authContext) {
	ctx := r.Context()
	if s.components == nil || s.client == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "bot client or components not initialized")
		return
	}
	globalCnt, privGuilds, err := s.components.SyncCommands(s.client)
	if err != nil {
		s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
			"ops.sync_commands", "commands", "", "Failed to sync Discord slash commands",
			map[string]any{"error": err.Error()}, "error", s.clientRemoteAddr(r))
		writeError(w, http.StatusInternalServerError, "sync_commands_failed", err.Error())
		return
	}
	s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"ops.sync_commands", "commands", "",
		fmt.Sprintf("Synced %d global commands and %d private guilds", globalCnt, privGuilds),
		map[string]any{"global_commands": globalCnt, "private_guild_count": privGuilds}, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, OpsSyncCommandsResponse{
		GlobalCommands:    globalCnt,
		PrivateGuildCount: privGuilds,
	})
}

func (s *Server) handleOpsBBSDatCheck(w http.ResponseWriter, r *http.Request, auth authContext) {
	ctx := r.Context()
	if s.bbsSync == nil || s.bbsSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "bbs_unavailable", "BBS system is not initialized")
		return
	}
	if err := s.bbsSync.RunDatOchiCheck(ctx, s.client); err != nil {
		s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
			"ops.bbs_dat_check", "bbs", "", "BBS dat-ochi check failed",
			map[string]any{"error": err.Error()}, "error", s.clientRemoteAddr(r))
		writeError(w, http.StatusInternalServerError, "bbs_dat_check_failed", err.Error())
		return
	}
	s.audit.RecordAction(ctx, auth.Session.UserID, auth.Principal.DisplayName, auth.Principal.Role,
		"ops.bbs_dat_check", "bbs", "", "Executed manual BBS dat-ochi archive check",
		nil, "success", s.clientRemoteAddr(r))
	writeJSON(w, http.StatusOK, OpsDatCheckResponse{Executed: true})
}

// --- Audit ---

func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request, _ authContext) {
	q := r.URL.Query()
	var limit, offset int
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	if raw := strings.TrimSpace(q.Get("offset")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			offset = v
		}
	}
	if limit <= 0 {
		limit = defaultAuditListLimit
	}
	if limit > maxAuditListLimit {
		limit = maxAuditListLimit
	}
	if offset < 0 {
		offset = 0
	}

	rows, total, err := s.audit.List(r.Context(), AuditFilter{
		ActorUserID:  q.Get("actor_user_id"),
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
		Status:       q.Get("status"),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit_list_failed", err.Error())
		return
	}

	var guildIDs, userIDs []snowflake.ID
	var catIDs, boardIDs, threadIDs []uuid.UUID
	for _, row := range rows {
		rid := strings.TrimSpace(row.ResourceID)
		if rid == "" {
			continue
		}
		switch row.ResourceType {
		case "guild":
			if id, pErr := snowflake.Parse(rid); pErr == nil && id != 0 {
				guildIDs = append(guildIDs, id)
			}
		case "user":
			if id, pErr := snowflake.Parse(rid); pErr == nil && id != 0 {
				userIDs = append(userIDs, id)
			}
		case "bbs_category":
			if id, pErr := uuid.Parse(rid); pErr == nil {
				catIDs = append(catIDs, id)
			}
		case "bbs_board":
			if id, pErr := uuid.Parse(rid); pErr == nil {
				boardIDs = append(boardIDs, id)
			}
		case "bbs_thread":
			if id, pErr := uuid.Parse(rid); pErr == nil {
				threadIDs = append(threadIDs, id)
			}
		}
	}
	guildNames := s.resolveGuildNames(r.Context(), guildIDs)
	userNames := s.resolveUserNames(r.Context(), userIDs, nil, false)
	catNames, boardNames, threadNames := s.resolveBBSNames(r.Context(), catIDs, boardIDs, threadIDs)

	entries := make([]AuditLogEntry, 0, len(rows))
	for _, row := range rows {
		var resName string
		if rid := strings.TrimSpace(row.ResourceID); rid != "" {
			switch row.ResourceType {
			case "guild":
				if id, pErr := snowflake.Parse(rid); pErr == nil {
					resName = guildNames[id]
				}
			case "user":
				if id, pErr := snowflake.Parse(rid); pErr == nil {
					resName = userNames[id]
				}
			case "bbs_category":
				if id, pErr := uuid.Parse(rid); pErr == nil {
					resName = catNames[id]
				}
			case "bbs_board":
				if id, pErr := uuid.Parse(rid); pErr == nil {
					resName = boardNames[id]
				}
			case "bbs_thread":
				if id, pErr := uuid.Parse(rid); pErr == nil {
					resName = threadNames[id]
				}
			}
		}
		entries = append(entries, AuditLogEntry{
			ID:           row.ID.String(),
			ActorUserID:  row.ActorUserID.String(),
			ActorName:    row.ActorName,
			ActorRole:    row.ActorRole,
			Action:       row.Action,
			ResourceType: row.ResourceType,
			ResourceID:   row.ResourceID,
			ResourceName: resName,
			Summary:      row.Summary,
			DetailJSON:   row.DetailJSON,
			Status:       row.Status,
			RemoteAddr:   row.RemoteAddr,
			CreatedAt:    row.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, AuditListResponse{
		Entries: entries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}
