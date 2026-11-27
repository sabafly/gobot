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
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sabafly/gobot/internal/admin"
)

func (c *Client) BBSStats(ctx context.Context) (admin.BBSStatsResponse, error) {
	var out admin.BBSStatsResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/stats", nil, &out)
	return out, err
}

func (c *Client) ListBBSCategories(ctx context.Context, filter admin.BBSCategoryFilter) (admin.BBSCategoriesResponse, error) {
	v := url.Values{}
	if strings.TrimSpace(filter.Scope) != "" {
		v.Set("scope", strings.TrimSpace(filter.Scope))
	}
	if strings.TrimSpace(filter.GuildID) != "" {
		v.Set("guild_id", strings.TrimSpace(filter.GuildID))
	}
	path := "/api/v1/bbs/categories"
	if enc := v.Encode(); enc != "" {
		path += "?" + enc
	}
	var out admin.BBSCategoriesResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) CreateBBSCategory(ctx context.Context, req admin.CreateBBSCategoryRequest) (admin.BBSCategoryDTO, error) {
	var out admin.BBSCategoryDTO
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/categories", req, &out)
	return out, err
}

func (c *Client) UpdateBBSCategory(ctx context.Context, id string, req admin.UpdateBBSCategoryRequest) (admin.BBSCategoryDTO, error) {
	var out admin.BBSCategoryDTO
	err := c.doJSON(ctx, http.MethodPatch, "/api/v1/bbs/categories/"+url.PathEscape(id), req, &out)
	return out, err
}

func (c *Client) DeleteBBSCategory(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/bbs/categories/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListBBSBoards(ctx context.Context, filter admin.BBSBoardFilter) (admin.BBSBoardsResponse, error) {
	v := url.Values{}
	if strings.TrimSpace(filter.Scope) != "" {
		v.Set("scope", strings.TrimSpace(filter.Scope))
	}
	if strings.TrimSpace(filter.GuildID) != "" {
		v.Set("guild_id", strings.TrimSpace(filter.GuildID))
	}
	if strings.TrimSpace(filter.CategoryID) != "" {
		v.Set("category_id", strings.TrimSpace(filter.CategoryID))
	}
	path := "/api/v1/bbs/boards"
	if enc := v.Encode(); enc != "" {
		path += "?" + enc
	}
	var out admin.BBSBoardsResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) CreateBBSBoard(ctx context.Context, req admin.CreateBBSBoardRequest) (admin.BBSBoardDTO, error) {
	var out admin.BBSBoardDTO
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/boards", req, &out)
	return out, err
}

func (c *Client) UpdateBBSBoard(ctx context.Context, id string, req admin.UpdateBBSBoardRequest) (admin.BBSBoardDTO, error) {
	var out admin.BBSBoardDTO
	err := c.doJSON(ctx, http.MethodPatch, "/api/v1/bbs/boards/"+url.PathEscape(id), req, &out)
	return out, err
}

func (c *Client) DeleteBBSBoard(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/bbs/boards/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListBBSBoardMounts(ctx context.Context, boardID string) (admin.BBSBoardMountsResponse, error) {
	var out admin.BBSBoardMountsResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/boards/"+url.PathEscape(boardID)+"/mounts", nil, &out)
	return out, err
}

func (c *Client) ListBBSThreads(ctx context.Context, filter admin.BBSThreadFilter) (admin.BBSThreadsResponse, error) {
	v := url.Values{}
	if strings.TrimSpace(filter.BoardID) != "" {
		v.Set("board_id", strings.TrimSpace(filter.BoardID))
	}
	if strings.TrimSpace(filter.Status) != "" {
		v.Set("status", strings.TrimSpace(filter.Status))
	}
	if strings.TrimSpace(filter.Query) != "" {
		v.Set("q", strings.TrimSpace(filter.Query))
	}
	if filter.Limit > 0 {
		v.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		v.Set("offset", strconv.Itoa(filter.Offset))
	}
	path := "/api/v1/bbs/threads"
	if enc := v.Encode(); enc != "" {
		path += "?" + enc
	}
	var out admin.BBSThreadsResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) GetBBSThreadDetail(ctx context.Context, threadID string, offset, limit int) (admin.BBSThreadDetailResponse, error) {
	v := url.Values{}
	if offset > 0 {
		v.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	path := "/api/v1/bbs/threads/" + url.PathEscape(threadID)
	if enc := v.Encode(); enc != "" {
		path += "?" + enc
	}
	var out admin.BBSThreadDetailResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) ArchiveBBSThread(ctx context.Context, threadID string) (admin.BBSArchiveThreadResponse, error) {
	var out admin.BBSArchiveThreadResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/threads/"+url.PathEscape(threadID)+"/archive", nil, &out)
	return out, err
}

func (c *Client) AboneBBSPost(ctx context.Context, postID, reason string) (admin.BBSAbonePostResponse, error) {
	var out admin.BBSAbonePostResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/posts/"+url.PathEscape(postID)+"/abone", admin.AboneBBSPostRequest{Reason: reason}, &out)
	return out, err
}

func (c *Client) ListBBSReports(ctx context.Context, filter admin.BBSReportFilter) (admin.BBSReportsResponse, error) {
	v := url.Values{}
	if strings.TrimSpace(filter.Scope) != "" {
		v.Set("scope", strings.TrimSpace(filter.Scope))
	}
	if strings.TrimSpace(filter.Status) != "" {
		v.Set("status", strings.TrimSpace(filter.Status))
	}
	if filter.Limit > 0 {
		v.Set("limit", strconv.Itoa(filter.Limit))
	}
	path := "/api/v1/bbs/reports"
	if enc := v.Encode(); enc != "" {
		path += "?" + enc
	}
	var out admin.BBSReportsResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) InspectBBSReport(ctx context.Context, reportID string) (admin.BBSReportInspectResponse, error) {
	var out admin.BBSReportInspectResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/reports/"+url.PathEscape(reportID)+"/inspect", nil, &out)
	return out, err
}

func (c *Client) ResolveBBSReport(ctx context.Context, reportID string, req admin.ResolveBBSReportRequest) (admin.BBSReportDTO, error) {
	var out admin.BBSReportDTO
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/reports/"+url.PathEscape(reportID)+"/resolve", req, &out)
	return out, err
}

func (c *Client) ListBBSBans(ctx context.Context, filter admin.BBSBanFilter) (admin.BBSBansResponse, error) {
	v := url.Values{}
	if strings.TrimSpace(filter.Scope) != "" {
		v.Set("scope", strings.TrimSpace(filter.Scope))
	}
	if strings.TrimSpace(filter.GuildID) != "" {
		v.Set("guild_id", strings.TrimSpace(filter.GuildID))
	}
	if filter.Limit > 0 {
		v.Set("limit", strconv.Itoa(filter.Limit))
	}
	path := "/api/v1/bbs/bans"
	if enc := v.Encode(); enc != "" {
		path += "?" + enc
	}
	var out admin.BBSBansResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) CreateBBSBan(ctx context.Context, req admin.CreateBBSBanRequest) (admin.BBSBanDTO, error) {
	var out admin.BBSBanDTO
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/bans", req, &out)
	return out, err
}

func (c *Client) DeleteBBSBan(ctx context.Context, banID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/bbs/bans/"+url.PathEscape(banID), nil, nil)
}

func (c *Client) ListBBSCaps(ctx context.Context) (admin.BBSCapsResponse, error) {
	var out admin.BBSCapsResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/config/caps", nil, &out)
	return out, err
}

func (c *Client) CreateBBSCap(ctx context.Context, req admin.SaveBBSCapRequest) (admin.BBSCapDTO, error) {
	var out admin.BBSCapDTO
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/config/caps", req, &out)
	return out, err
}

func (c *Client) UpdateBBSCap(ctx context.Context, id string, req admin.SaveBBSCapRequest) (admin.BBSCapDTO, error) {
	var out admin.BBSCapDTO
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/bbs/config/caps/"+url.PathEscape(id), req, &out)
	return out, err
}

func (c *Client) DeleteBBSCap(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/bbs/config/caps/"+url.PathEscape(id), nil, nil)
}

func (c *Client) ListBBSSlipEvents(ctx context.Context) (admin.BBSSlipEventsResponse, error) {
	var out admin.BBSSlipEventsResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/bbs/config/slip-events", nil, &out)
	return out, err
}

func (c *Client) CreateBBSSlipEvent(ctx context.Context, req admin.SaveBBSSlipEventRequest) (admin.BBSSlipEventDTO, error) {
	var out admin.BBSSlipEventDTO
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/bbs/config/slip-events", req, &out)
	return out, err
}

func (c *Client) UpdateBBSSlipEvent(ctx context.Context, id string, req admin.SaveBBSSlipEventRequest) (admin.BBSSlipEventDTO, error) {
	var out admin.BBSSlipEventDTO
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/bbs/config/slip-events/"+url.PathEscape(id), req, &out)
	return out, err
}

func (c *Client) DeleteBBSSlipEvent(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/bbs/config/slip-events/"+url.PathEscape(id), nil, nil)
}
