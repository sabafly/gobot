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
 *
 */

package db

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type RolePanelV2Place struct {
	ID        uuid.UUID         `json:"id"`
	CreatedAt time.Time         `json:"created_at"`
	GuildID   snowflake.ID      `json:"guild_id"`
	PanelID   uuid.UUID         `json:"panel_id"`
	Config    RolePanelV2Config `json:"config"`
}

type RolePanelV2Type string
