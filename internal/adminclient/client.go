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

package adminclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sabafly/gobot/internal/admin"
)

// APIError represents a structured error response returned by gobot's admin HTTP API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("admin api error (%d %s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("admin api error (%d): %s", e.StatusCode, e.Message)
}

// IsUnauthorized reports whether err indicates an HTTP 401 authentication failure.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// IsForbidden reports whether err indicates an HTTP 403 authorization failure.
func IsForbidden(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden
}

// Client communicates with gobot's HTTP/SSE admin server.
type Client struct {
	BaseURL          string
	Token            string
	HTTPClient       *http.Client
	StreamHTTPClient *http.Client
	UserAgent        string
}

// New constructs a Client targeting baseURL with an optional bearer session token.
func New(baseURL, token string) *Client {
	normalized := NormalizeBaseURL(baseURL)
	return &Client{
		BaseURL: normalized,
		Token:   strings.TrimSpace(token),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		StreamHTTPClient: &http.Client{
			Timeout: 0, // Controlled via request context for long-lived SSE streams.
		},
		UserAgent: "gobot-adminclient/1.0",
	}
}

// NormalizeBaseURL trims whitespace and trailing slashes from a base server URL.
func NormalizeBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody, dst any) error {
	if c == nil || c.BaseURL == "" {
		return errors.New("adminclient: base URL is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var bodyReader io.Reader
	if reqBody != nil {
		raw, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	rawResp, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errEnv admin.ErrorResponse
		if json.Unmarshal(rawResp, &errEnv) == nil && (errEnv.Error.Code != "" || errEnv.Error.Message != "") {
			return &APIError{
				StatusCode: resp.StatusCode,
				Code:       errEnv.Error.Code,
				Message:    errEnv.Error.Message,
			}
		}
		msg := strings.TrimSpace(string(rawResp))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    msg,
		}
	}
	if dst != nil && len(bytes.TrimSpace(rawResp)) > 0 {
		if err := json.Unmarshal(rawResp, dst); err != nil {
			return err
		}
	}
	return nil
}

// StartDiscordLogin begins a Discord OAuth2 login handshake and returns the authorize URL + poll credentials.
func (c *Client) StartDiscordLogin(ctx context.Context, clientInfo string) (admin.AuthStartResponse, error) {
	var out admin.AuthStartResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/discord/start", admin.AuthStartRequest{
		ClientInfo: clientInfo,
	}, &out)
	return out, err
}

// PollDiscordLogin polls login status and receives the session token once when approved.
func (c *Client) PollDiscordLogin(ctx context.Context, loginID, pollSecret string) (admin.AuthPollResponse, error) {
	var out admin.AuthPollResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/discord/poll", admin.AuthPollRequest{
		LoginID:    loginID,
		PollSecret: pollSecret,
	}, &out)
	return out, err
}

// Me returns the currently authenticated operator identity and permissions.
func (c *Client) Me(ctx context.Context) (admin.PrincipalInfo, error) {
	var out admin.PrincipalInfo
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/auth/me", nil, &out)
	return out, err
}

// Logout revokes the current session on the server.
func (c *Client) Logout(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/auth/logout", nil, nil)
}

// ListSessions returns active sessions visible to the caller.
func (c *Client) ListSessions(ctx context.Context) (admin.SessionsResponse, error) {
	var out admin.SessionsResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/auth/sessions", nil, &out)
	return out, err
}

// RevokeSession revokes the session identified by id.
func (c *Client) RevokeSession(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/auth/sessions/"+url.PathEscape(id), nil, nil)
}

// Overview fetches runtime health, shard latencies, DB ping, and BBS summary metrics.
func (c *Client) Overview(ctx context.Context) (admin.OverviewResponse, error) {
	var out admin.OverviewResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/overview", nil, &out)
	return out, err
}

// ListLogs fetches buffered slog entries matching filter.
func (c *Client) ListLogs(ctx context.Context, filter admin.LogFilter) (admin.LogsResponse, error) {
	q := url.Values{}
	if filter.MinLevel != "" {
		q.Set("level", filter.MinLevel)
	}
	if filter.Query != "" {
		q.Set("q", filter.Query)
	}
	if filter.AfterSeq > 0 {
		q.Set("after_seq", strconv.FormatUint(filter.AfterSeq, 10))
	}
	if filter.Limit > 0 {
		q.Set("limit", strconv.Itoa(filter.Limit))
	}
	path := "/api/v1/logs"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out admin.LogsResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// GetLogLevel returns the current server LogHub capture level configuration and effective level.
func (c *Client) GetLogLevel(ctx context.Context) (admin.LogLevelResponse, error) {
	var out admin.LogLevelResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/logs/level", nil, &out)
	return out, err
}

// SetLogLevel updates the server LogHub capture level in memory (auto/debug/info/warn/error).
func (c *Client) SetLogLevel(ctx context.Context, level string) (admin.LogLevelResponse, error) {
	var out admin.LogLevelResponse
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/logs/level", admin.UpdateLogLevelRequest{Level: level}, &out)
	return out, err
}

// StreamLogs subscribes to the SSE log stream and invokes onEntry until ctx is canceled or an error occurs.
func (c *Client) StreamLogs(ctx context.Context, filter admin.LogFilter, onEntry func(admin.LogEntry) error) error {
	if c == nil || c.BaseURL == "" {
		return errors.New("adminclient: base URL is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	q := url.Values{}
	if filter.MinLevel != "" {
		q.Set("level", filter.MinLevel)
	}
	if filter.Query != "" {
		q.Set("q", filter.Query)
	}
	if filter.AfterSeq > 0 {
		q.Set("after_seq", strconv.FormatUint(filter.AfterSeq, 10))
	}
	path := "/api/v1/logs/stream"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	hc := c.StreamHTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 0}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rawResp, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var errEnv admin.ErrorResponse
		if json.Unmarshal(rawResp, &errEnv) == nil && (errEnv.Error.Code != "" || errEnv.Error.Message != "") {
			return &APIError{
				StatusCode: resp.StatusCode,
				Code:       errEnv.Error.Code,
				Message:    errEnv.Error.Message,
			}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(string(rawResp)),
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventName string
	var dataLines []string

	flushEvent := func() error {
		if len(dataLines) == 0 {
			eventName = ""
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = nil
		ev := eventName
		eventName = ""
		if ev != "" && ev != "log" {
			return nil
		}
		var entry admin.LogEntry
		if err := json.Unmarshal([]byte(payload), &entry); err != nil {
			return nil
		}
		if onEntry != nil {
			return onEntry(entry)
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flushEvent(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		val = strings.TrimPrefix(val, " ")
		switch field {
		case "event":
			eventName = val
		case "data":
			dataLines = append(dataLines, val)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flushEvent()
}

// ListGuilds lists guilds visible in Discord cache and DB, optionally filtered by query.
func (c *Client) ListGuilds(ctx context.Context, query string) (admin.GuildsResponse, error) {
	path := "/api/v1/guilds"
	if strings.TrimSpace(query) != "" {
		path += "?q=" + url.QueryEscape(strings.TrimSpace(query))
	}
	var out admin.GuildsResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// GetGuild fetches detail for a single guild.
func (c *Client) GetGuild(ctx context.Context, guildID string) (admin.GuildDetailResponse, error) {
	var out admin.GuildDetailResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/guilds/"+url.PathEscape(guildID), nil, &out)
	return out, err
}

// UpdateGuild updates guild locale override and/or BBS opt-in state.
func (c *Client) UpdateGuild(ctx context.Context, guildID string, req admin.UpdateGuildRequest) (admin.GuildDetailResponse, error) {
	var out admin.GuildDetailResponse
	err := c.doJSON(ctx, http.MethodPatch, "/api/v1/guilds/"+url.PathEscape(guildID), req, &out)
	return out, err
}

// ReloadLocales triggers runtime i18n locale file reload.
func (c *Client) ReloadLocales(ctx context.Context) (admin.OpsReloadLocalesResponse, error) {
	var out admin.OpsReloadLocalesResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/ops/reload-locales", nil, &out)
	return out, err
}

// SyncCommands pushes application slash commands to Discord.
func (c *Client) SyncCommands(ctx context.Context) (admin.OpsSyncCommandsResponse, error) {
	var out admin.OpsSyncCommandsResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/ops/sync-commands", nil, &out)
	return out, err
}

// TriggerBBSDatCheck triggers a manual BBS dat-ochi archive pass.
func (c *Client) TriggerBBSDatCheck(ctx context.Context) (admin.OpsDatCheckResponse, error) {
	var out admin.OpsDatCheckResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/ops/bbs-dat-check", nil, &out)
	return out, err
}

// ListAuditLogs lists operator management audit records matching filter.
func (c *Client) ListAuditLogs(ctx context.Context, filter admin.AuditFilter) (admin.AuditListResponse, error) {
	q := url.Values{}
	if filter.ActorUserID != "" {
		q.Set("actor_user_id", filter.ActorUserID)
	}
	if filter.Action != "" {
		q.Set("action", filter.Action)
	}
	if filter.ResourceType != "" {
		q.Set("resource_type", filter.ResourceType)
	}
	if filter.Status != "" {
		q.Set("status", filter.Status)
	}
	if filter.Limit > 0 {
		q.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		q.Set("offset", strconv.Itoa(filter.Offset))
	}
	path := "/api/v1/audit"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out admin.AuditListResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}
