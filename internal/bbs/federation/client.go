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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

// Client calls Origin's dedicated federation HTTP/SSE + OAuth2 backchannel API.
type Client struct {
	baseURL    string
	peerToken  string
	restClient *http.Client
	sseClient  *http.Client
}

// NewClient creates a federation API client with separate timeouts for REST RPCs and SSE streams.
func NewClient(cfg RemoteConfig, customHTTPClient ...*http.Client) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultRemoteTimeout
	}
	var restClient *http.Client
	var transport http.RoundTripper
	if len(customHTTPClient) > 0 && customHTTPClient[0] != nil {
		base := customHTTPClient[0]
		transport = base.Transport
		restClient = &http.Client{
			Timeout:   timeout,
			Transport: base.Transport,
		}
	} else {
		restClient = &http.Client{Timeout: timeout}
		transport = &http.Transport{
			DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		}
	}
	sseClient := &http.Client{
		Timeout:   0,
		Transport: transport,
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		peerToken:  strings.TrimSpace(cfg.Token),
		restClient: restClient,
		sseClient:  sseClient,
	}
}

func (c *Client) doJSON(ctx context.Context, method, path, userToken string, reqBody, respBody any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.peerToken)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if userToken != "" {
		req.Header.Set(HeaderUserToken, userToken)
	}

	resp, err := c.restClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", bbs.ErrFederationUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%w: read response: %v", bbs.ErrFederationUnavailable, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		_ = json.Unmarshal(raw, &errResp)
		msg := strings.TrimSpace(errResp.Error)
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       errResp.Code,
			Message:    msg,
		}
	}
	if respBody != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, respBody); err != nil {
			return fmt.Errorf("%w: invalid json: %v", bbs.ErrFederationUnavailable, err)
		}
	}
	return nil
}

func (c *Client) GetCapabilities(ctx context.Context) (*CapabilitiesResponse, error) {
	var out CapabilitiesResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/federation/capabilities", "", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListCategories(ctx context.Context) ([]bbs.Category, error) {
	var out CategoryListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/federation/categories", "", nil, &out); err != nil {
		return nil, err
	}
	return out.Categories, nil
}

func (c *Client) ListBoards(ctx context.Context, categoryID *uuid.UUID) ([]bbs.Board, error) {
	path := "/api/v1/bbs/federation/boards"
	if categoryID != nil && *categoryID != uuid.Nil {
		path += "?category_id=" + url.QueryEscape(categoryID.String())
	}
	var out BoardListResponse
	if err := c.doJSON(ctx, http.MethodGet, path, "", nil, &out); err != nil {
		return nil, err
	}
	return out.Boards, nil
}

func (c *Client) GetBoard(ctx context.Context, boardID uuid.UUID) (*bbs.Board, error) {
	var out bbs.Board
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/federation/boards/"+url.PathEscape(boardID.String()), "", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListThreads(ctx context.Context, filter bbs.ThreadFilter) ([]bbs.Thread, error) {
	q := url.Values{}
	if filter.Status != nil {
		q.Set("status", string(*filter.Status))
	}
	if strings.TrimSpace(filter.Query) != "" {
		q.Set("q", strings.TrimSpace(filter.Query))
	}
	if filter.Offset > 0 {
		q.Set("offset", strconv.Itoa(filter.Offset))
	}
	if filter.Limit > 0 {
		q.Set("limit", strconv.Itoa(filter.Limit))
	}
	path := "/api/v1/bbs/federation/boards/" + url.PathEscape(filter.BoardID.String()) + "/threads"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out ThreadListResponse
	if err := c.doJSON(ctx, http.MethodGet, path, "", nil, &out); err != nil {
		return nil, err
	}
	return out.Threads, nil
}

func (c *Client) GetThread(ctx context.Context, threadID uuid.UUID) (*bbs.Thread, error) {
	var out bbs.Thread
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/federation/threads/"+url.PathEscape(threadID.String()), "", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListPosts(ctx context.Context, threadID uuid.UUID, offset, limit int) ([]bbs.Post, error) {
	q := url.Values{}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/api/v1/bbs/federation/threads/" + url.PathEscape(threadID.String()) + "/posts"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out PostListResponse
	if err := c.doJSON(ctx, http.MethodGet, path, "", nil, &out); err != nil {
		return nil, err
	}
	return out.Posts, nil
}

func (c *Client) ListPostsAfterRes(ctx context.Context, threadID uuid.UUID, afterRes uint32) ([]bbs.Post, error) {
	path := fmt.Sprintf("/api/v1/bbs/federation/threads/%s/posts?after_res=%d", url.PathEscape(threadID.String()), afterRes)
	var out PostListResponse
	if err := c.doJSON(ctx, http.MethodGet, path, "", nil, &out); err != nil {
		return nil, err
	}
	return out.Posts, nil
}

func (c *Client) GetPost(ctx context.Context, postID uuid.UUID) (*bbs.Post, error) {
	var out bbs.Post
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/federation/posts/"+url.PathEscape(postID.String()), "", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListBoardCaps(ctx context.Context, boardID uuid.UUID, userToken string) ([]bbs.Cap, error) {
	path := "/api/v1/bbs/federation/boards/" + url.PathEscape(boardID.String()) + "/caps"
	var out CapsListResponse
	if err := c.doJSON(ctx, http.MethodGet, path, userToken, nil, &out); err != nil {
		return nil, err
	}
	return out.Caps, nil
}

func (c *Client) CreateThread(ctx context.Context, boardID uuid.UUID, userToken string, req CreateThreadRequest) (*CreateThreadResponse, error) {
	path := "/api/v1/bbs/federation/boards/" + url.PathEscape(boardID.String()) + "/threads"
	var out CreateThreadResponse
	if err := c.doJSON(ctx, http.MethodPost, path, userToken, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreatePost(ctx context.Context, threadID uuid.UUID, userToken string, req CreatePostRequest) (*CreatePostResponse, error) {
	path := "/api/v1/bbs/federation/threads/" + url.PathEscape(threadID.String()) + "/posts"
	var out CreatePostResponse
	if err := c.doJSON(ctx, http.MethodPost, path, userToken, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateReport(ctx context.Context, postID uuid.UUID, userToken string, req CreateReportRequest) (*CreateReportResponse, error) {
	path := "/api/v1/bbs/federation/posts/" + url.PathEscape(postID.String()) + "/reports"
	var out CreateReportResponse
	if err := c.doJSON(ctx, http.MethodPost, path, userToken, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ExchangeAuthCode(ctx context.Context, code, redirectURI, codeVerifier string) (*TokenResponse, error) {
	body := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  redirectURI,
		"code_verifier": codeVerifier,
	}
	var out TokenResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/federation/oauth/token", "", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}
	var out TokenResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/federation/oauth/token", "", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RevokeUserGrant(ctx context.Context, userID snowflake.ID) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/federation/oauth/revoke", "", RevokeRequest{
		UserID: userID,
	}, nil)
}

// OpenEventsStream opens the Origin SSE event stream starting after afterSeq for clientEpoch.
func (c *Client) OpenEventsStream(ctx context.Context, afterSeq uint64, clientEpoch string) (*http.Response, error) {
	q := url.Values{}
	if afterSeq > 0 {
		q.Set("after_seq", strconv.FormatUint(afterSeq, 10))
	}
	if clientEpoch != "" {
		q.Set("epoch", clientEpoch)
	}
	u := c.baseURL + "/api/v1/bbs/federation/events"
	if encoded := q.Encode(); encoded != "" {
		u += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.peerToken)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.sseClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		var errResp ErrorResponse
		_ = json.Unmarshal(raw, &errResp)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       errResp.Code,
			Message:    strings.TrimSpace(errResp.Error),
		}
	}
	return resp, nil
}

func asAPIError(err error) (*APIError, bool) {
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr, true
	}
	return nil, false
}
