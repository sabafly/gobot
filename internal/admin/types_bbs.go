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

type BBSStatsResponse struct {
	Available     bool  `json:"available"`
	Categories    int64 `json:"categories"`
	Boards        int64 `json:"boards"`
	ActiveThreads int64 `json:"active_threads"`
	OpenReports   int64 `json:"open_reports"`
	ActiveBans    int64 `json:"active_bans"`
}

type BBSCategoryFilter struct {
	Scope   string
	GuildID string
}

type BBSCategoryDTO struct {
	ID          string    `json:"id"`
	Scope       string    `json:"scope"`
	GuildID     *string   `json:"guild_id,omitempty"`
	GuildName   string    `json:"guild_name,omitempty"`
	ProviderKey string    `json:"provider_key"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

type BBSCategoriesResponse struct {
	Categories []BBSCategoryDTO `json:"categories"`
}

type CreateBBSCategoryRequest struct {
	Scope       string `json:"scope"`
	GuildID     string `json:"guild_id,omitempty"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateBBSCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

type BBSBoardFilter struct {
	Scope      string
	GuildID    string
	CategoryID string
}

type BBSBoardRulesDTO struct {
	DefaultName        string `json:"default_name"`
	HideID             bool   `json:"hide_id"`
	SLIPLevel          int    `json:"slip_level"`
	MaxRes             uint32 `json:"max_res"`
	DatOchiIdleHours   int    `json:"dat_ochi_idle_hours"`
	DatOchiLowResHours int    `json:"dat_ochi_low_res_hours"`
}

type BBSBoardDTO struct {
	ID           string           `json:"id"`
	CategoryID   string           `json:"category_id"`
	CategoryName string           `json:"category_name,omitempty"`
	Scope        string           `json:"scope"`
	GuildID      *string          `json:"guild_id,omitempty"`
	GuildName    string           `json:"guild_name,omitempty"`
	ProviderKey  string           `json:"provider_key"`
	Slug         string           `json:"slug"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Rules        BBSBoardRulesDTO `json:"rules"`
	Locked       bool             `json:"locked"`
	CreatedAt    time.Time        `json:"created_at"`
}

type BBSBoardsResponse struct {
	Boards []BBSBoardDTO `json:"boards"`
}

type CreateBBSBoardRequest struct {
	CategoryID  string           `json:"category_id"`
	Scope       string           `json:"scope,omitempty"`
	GuildID     string           `json:"guild_id,omitempty"`
	Slug        string           `json:"slug"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Rules       BBSBoardRulesDTO `json:"rules"`
}

type UpdateBBSBoardRequest struct {
	Name               *string `json:"name,omitempty"`
	Description        *string `json:"description,omitempty"`
	DefaultName        *string `json:"default_name,omitempty"`
	HideID             *bool   `json:"hide_id,omitempty"`
	SLIPLevel          *int    `json:"slip_level,omitempty"`
	MaxRes             *uint32 `json:"max_res,omitempty"`
	DatOchiIdleHours   *int    `json:"dat_ochi_idle_hours,omitempty"`
	DatOchiLowResHours *int    `json:"dat_ochi_low_res_hours,omitempty"`
	Locked             *bool   `json:"locked,omitempty"`
}

type BBSBoardMountDTO struct {
	ID          string    `json:"id"`
	BoardID     string    `json:"board_id"`
	GuildID     string    `json:"guild_id"`
	GuildName   string    `json:"guild_name,omitempty"`
	ChannelID   string    `json:"channel_id"`
	ChannelName string    `json:"channel_name,omitempty"`
	Locale      string    `json:"locale"`
	IsForum     bool      `json:"is_forum"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

type BBSBoardMountsResponse struct {
	Mounts []BBSBoardMountDTO `json:"mounts"`
}

type BBSThreadFilter struct {
	BoardID string
	Status  string
	Query   string
	Limit   int
	Offset  int
}

type BBSThreadDTO struct {
	ID          string           `json:"id"`
	BoardID     string           `json:"board_id"`
	ProviderKey string           `json:"provider_key"`
	ExternalID  string           `json:"external_id"`
	Title       string           `json:"title"`
	Status      string           `json:"status"`
	ResCount    uint32           `json:"res_count"`
	Rules       BBSBoardRulesDTO `json:"rules"`
	BumpedAt    time.Time        `json:"bumped_at"`
	ArchivedAt  *time.Time       `json:"archived_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type BBSThreadsResponse struct {
	Threads []BBSThreadDTO `json:"threads"`
}

type BBSPostDTO struct {
	ID          string    `json:"id"`
	ThreadID    string    `json:"thread_id"`
	ResNumber   uint32    `json:"res_number"`
	Name        string    `json:"name"`
	Trip        string    `json:"trip"`
	Cap         string    `json:"cap"`
	SlipText    string    `json:"slip_text"`
	AnonID      string    `json:"anon_id"`
	PosterHash  string    `json:"poster_hash"`
	Body        string    `json:"body"`
	Abone       bool      `json:"abone"`
	AboneReason string    `json:"abone_reason"`
	CreatedAt   time.Time `json:"created_at"`
}

type BBSThreadDetailResponse struct {
	Thread BBSThreadDTO `json:"thread"`
	Posts  []BBSPostDTO `json:"posts"`
}

type BBSArchiveThreadResponse struct {
	Thread BBSThreadDTO `json:"thread"`
}

type AboneBBSPostRequest struct {
	Reason string `json:"reason"`
}

type BBSAbonePostResponse struct {
	Post BBSPostDTO `json:"post"`
}

type BBSReportFilter struct {
	Scope  string
	Status string
	Limit  int
}

type BBSReportDTO struct {
	ID             string    `json:"id"`
	Scope          string    `json:"scope"`
	GuildID        string    `json:"guild_id"`
	GuildName      string    `json:"guild_name,omitempty"`
	PostID         string    `json:"post_id"`
	ThreadID       string    `json:"thread_id"`
	ThreadTitle    string    `json:"thread_title,omitempty"`
	BoardID        string    `json:"board_id"`
	BoardName      string    `json:"board_name,omitempty"`
	ReporterUserID string    `json:"reporter_user_id,omitempty"`
	ReporterName   string    `json:"reporter_name,omitempty"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"`
	BanID          *string   `json:"ban_id,omitempty"`
	HandledBy      *string   `json:"handled_by,omitempty"`
	HandledByName  string    `json:"handled_by_name,omitempty"`
	ResolutionNote string    `json:"resolution_note"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BBSReportsResponse struct {
	Reports []BBSReportDTO `json:"reports"`
}

type BBSReportInspectResponse struct {
	Report         BBSReportDTO `json:"report"`
	Post           *BBSPostDTO  `json:"post,omitempty"`
	AuthorUserID   string       `json:"author_user_id,omitempty"`
	AuthorUserName string       `json:"author_user_name,omitempty"`
	PostGuildID    string       `json:"post_guild_id,omitempty"`
	PostGuildName  string       `json:"post_guild_name,omitempty"`
	PosterHash     string       `json:"poster_hash,omitempty"`
}

type ResolveBBSReportRequest struct {
	Status         string `json:"status"`
	ResolutionNote string `json:"resolution_note"`
}

type BBSBanFilter struct {
	Scope   string
	GuildID string
	Limit   int
}

type BBSBanDTO struct {
	ID             string     `json:"id"`
	Scope          string     `json:"scope"`
	GuildID        *string    `json:"guild_id,omitempty"`
	GuildName      string     `json:"guild_name,omitempty"`
	PosterHash     string     `json:"poster_hash"`
	TargetUserID   *string    `json:"target_user_id,omitempty"`
	TargetUserName string     `json:"target_user_name,omitempty"`
	Reason         string     `json:"reason"`
	CreatedBy      string     `json:"created_by"`
	CreatedByName  string     `json:"created_by_name,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type BBSBansResponse struct {
	Bans []BBSBanDTO `json:"bans"`
}

type CreateBBSBanRequest struct {
	Scope            string `json:"scope"`
	GuildID          string `json:"guild_id,omitempty"`
	PostID           string `json:"post_id,omitempty"`
	TargetUserID     string `json:"target_user_id,omitempty"`
	TargetPosterHash string `json:"target_poster_hash,omitempty"`
	Reason           string `json:"reason"`
	DurationHours    int    `json:"duration_hours"`
}

type BBSCapDTO struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	GrantOperators bool              `json:"grant_operators"`
	Users          []string          `json:"users"`
	UserNames      map[string]string `json:"user_names,omitempty"`
}

type BBSCapsResponse struct {
	Caps []BBSCapDTO `json:"caps"`
}

type SaveBBSCapRequest struct {
	ID             string   `json:"id,omitempty"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	GrantOperators bool     `json:"grant_operators"`
	Users          []string `json:"users"`
}

type BBSSlipEventDTO struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Mode      string `json:"mode"`
}

type BBSSlipEventsResponse struct {
	Events []BBSSlipEventDTO `json:"events"`
}

type SaveBBSSlipEventRequest struct {
	ID        string `json:"id,omitempty"`
	Label     string `json:"label"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Mode      string `json:"mode"`
}
