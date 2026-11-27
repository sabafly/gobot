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

import "time"

// Permission represents an RBAC capability checked by the admin API and TUI panels.
type Permission string

const (
	PermDashboardRead  Permission = "dashboard.read"
	PermLogsRead       Permission = "logs.read"
	PermGuildsRead     Permission = "guilds.read"
	PermGuildsWrite    Permission = "guilds.write"
	PermOpsExecute     Permission = "ops.execute"
	PermAuditRead      Permission = "audit.read"
	PermSessionsManage Permission = "sessions.manage"
	PermBBSRead        Permission = "bbs.read"
	PermBBSModerate    Permission = "bbs.moderate"
	PermBBSAdmin       Permission = "bbs.admin"
)

const (
	RoleSuperAdmin   = "super_admin"
	RoleBBSModerator = "bbs_moderator"
	RoleViewer       = "viewer"
)

// AllPermissions returns the full canonical list of admin permission keys.
func AllPermissions() []Permission {
	return []Permission{
		PermDashboardRead,
		PermLogsRead,
		PermGuildsRead,
		PermGuildsWrite,
		PermOpsExecute,
		PermAuditRead,
		PermSessionsManage,
		PermBBSRead,
		PermBBSModerate,
		PermBBSAdmin,
	}
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type AuthStartRequest struct {
	ClientInfo string `json:"client_info,omitempty"`
}

type AuthStartResponse struct {
	LoginID      string    `json:"login_id"`
	AuthorizeURL string    `json:"authorize_url"`
	PollSecret   string    `json:"poll_secret"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type AuthPollRequest struct {
	LoginID    string `json:"login_id"`
	PollSecret string `json:"poll_secret"`
}

type AuthPollResponse struct {
	Status       string         `json:"status"` // pending | approved | denied | expired
	SessionToken string         `json:"session_token,omitempty"`
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"`
	Principal    *PrincipalInfo `json:"principal,omitempty"`
	Error        string         `json:"error,omitempty"`
}

type PrincipalInfo struct {
	UserID      string       `json:"user_id"`
	Username    string       `json:"username"`
	DisplayName string       `json:"display_name"`
	Role        string       `json:"role"`
	Permissions []Permission `json:"permissions"`
}

type SessionInfo struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	ClientInfo  string    `json:"client_info"`
	Current     bool      `json:"current"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type SessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

type LogEntry struct {
	Seq     uint64         `json:"seq"`
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Source  string         `json:"source,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

type LogFilter struct {
	MinLevel string
	Query    string
	AfterSeq uint64
	Limit    int
}

type LogsResponse struct {
	Entries []LogEntry `json:"entries"`
	NextSeq uint64     `json:"next_seq"`
}

type ShardStatus struct {
	ShardID   int    `json:"shard_id"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
}

type BBSSummary struct {
	Available    bool  `json:"available"`
	Categories   int64 `json:"categories"`
	Boards       int64 `json:"boards"`
	ActiveMounts int64 `json:"active_mounts"`
	Threads      int64 `json:"threads"`
	Posts        int64 `json:"posts"`
	OpenReports  int64 `json:"open_reports"`
	ActiveBans   int64 `json:"active_bans"`
}

type OverviewResponse struct {
	Version          string        `json:"version"`
	StartedAt        time.Time     `json:"started_at"`
	UptimeSeconds    int64         `json:"uptime_seconds"`
	GoVersion        string        `json:"go_version"`
	Goroutines       int           `json:"goroutines"`
	MemAllocBytes    uint64        `json:"mem_alloc_bytes"`
	MemSysBytes      uint64        `json:"mem_sys_bytes"`
	DBConnected      bool          `json:"db_connected"`
	DBLatencyMs      int64         `json:"db_latency_ms"`
	DBError          string        `json:"db_error,omitempty"`
	BotConnected     bool          `json:"bot_connected"`
	BotUsername      string        `json:"bot_username,omitempty"`
	CachedGuilds     int           `json:"cached_guilds"`
	DBGuilds         int64         `json:"db_guilds"`
	DBUsers          int64         `json:"db_users"`
	AvgShardPingMs   int64         `json:"avg_shard_ping_ms"`
	Shards           []ShardStatus `json:"shards"`
	LogBufferLen     int           `json:"log_buffer_len"`
	LogBufferCap     int           `json:"log_buffer_cap"`
	ActiveSessions   int64         `json:"active_sessions"`
	BBS              BBSSummary    `json:"bbs"`
}

type GuildSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Cached          bool   `json:"cached"`
	MemberCount     int    `json:"member_count"`
	PreferredLocale string `json:"preferred_locale"`
	LocaleOverride  string `json:"locale_override"`
	EffectiveLocale string `json:"effective_locale"`
	BBSEnabled      bool   `json:"bbs_enabled"`
	BBSMounts       int    `json:"bbs_mounts"`
}

type GuildsResponse struct {
	Guilds           []GuildSummary `json:"guilds"`
	AvailableLocales []string       `json:"available_locales"`
}

type GuildDetailResponse struct {
	Guild            GuildSummary `json:"guild"`
	AvailableLocales []string     `json:"available_locales"`
	BBSForumDefault  *bool        `json:"bbs_forum_default,omitempty"`
	BBSModLogChannel string       `json:"bbs_mod_log_channel,omitempty"`
}

type UpdateGuildRequest struct {
	LocaleOverride *string `json:"locale_override,omitempty"`
	BBSEnabled     *bool   `json:"bbs_enabled,omitempty"`
}

type OpsReloadLocalesResponse struct {
	Locales []string `json:"locales"`
}

type OpsSyncCommandsResponse struct {
	GlobalCommands    int `json:"global_commands"`
	PrivateGuildCount int `json:"private_guild_count"`
}

type OpsDatCheckResponse struct {
	Executed bool `json:"executed"`
}

type AuditLogEntry struct {
	ID           string    `json:"id"`
	ActorUserID  string    `json:"actor_user_id"`
	ActorName    string    `json:"actor_name"`
	ActorRole    string    `json:"actor_role"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Summary      string    `json:"summary"`
	DetailJSON   string    `json:"detail_json"`
	Status       string    `json:"status"`
	RemoteAddr   string    `json:"remote_addr"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditFilter struct {
	ActorUserID  string
	Action       string
	ResourceType string
	Status       string
	Limit        int
	Offset       int
}

type AuditListResponse struct {
	Entries []AuditLogEntry `json:"entries"`
	Total   int64           `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}
