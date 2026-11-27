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

package bbs

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

// UpdateCategoryInput modifies mutable category settings.
type UpdateCategoryInput struct {
	Name        *string
	Slug        *string
	Description *string
	SortOrder   *int
}

// BoardMountFilter selects board-to-channel mounts across guilds.
type BoardMountFilter struct {
	GuildID *snowflake.ID
	BoardID *uuid.UUID
	Active  *bool
}

// Stats holds aggregate operational counts for the BBS admin summary.
type Stats struct {
	Categories    int64 `json:"categories"`
	Boards        int64 `json:"boards"`
	ActiveThreads int64 `json:"active_threads"`
	OpenReports   int64 `json:"open_reports"`
	ActiveBans    int64 `json:"active_bans"`
}

// AdminBanInput captures ban creation parameters submitted via the admin API.
type AdminBanInput struct {
	Scope            Scope
	GuildID          snowflake.ID
	PostID           *uuid.UUID
	TargetUserID     *snowflake.ID
	TargetPosterHash string
	Reason           string
	DurationHours    int
	ActorID          snowflake.ID
}
