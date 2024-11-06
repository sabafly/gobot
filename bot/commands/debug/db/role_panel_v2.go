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
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type RolePanelV2 struct {
	ID          uuid.UUID         `json:"uuid"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Roles       []RolePanelV2Role `json:"roles"`
}

type RolePanelV2Config struct {
	PanelType        RolePanelV2Type     `json:"panel_type"`
	ButtonStyle      discord.ButtonStyle `json:"button_style"`
	ButtonShowName   bool                `json:"show_name"`
	SimpleSelectMenu bool                `json:"simple_select_menu"`
	HideNotice       bool                `json:"hide_notice"`
	UseDisplayName   bool                `json:"use_display_name"`
}

type RolePanelV2Role struct {
	RoleID   snowflake.ID            `json:"role_id"`
	RoleName string                  `json:"role_name"`
	Emoji    *discord.ComponentEmoji `json:"emoji"`
}
