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

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type RolePanelV2EditDB interface {
	Get(id uuid.UUID) (data *RolePanelV2Edit, err error)
	Set(id uuid.UUID, data *RolePanelV2Edit) (err error)
	Del(id uuid.UUID) (err error)
}

type RolePanelV2Edit struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	RolePanelID uuid.UUID `json:"role_panel_id"`

	GuildID     snowflake.ID   `json:"guild_id"`
	ChannelID   snowflake.ID   `json:"channel_id"`
	MessageID   snowflake.ID   `json:"message_id"`
	EmojiMode   bool           `json:"emoji_mode"`
	EmojiLocale discord.Locale `json:"emoji_locale"`

	SelectedID *snowflake.ID
}

func (r RolePanelV2Edit) IsSelected(id snowflake.ID) bool {
	return r.SelectedID != nil && *r.SelectedID == id
}

func (r RolePanelV2Edit) HasSelectedRole() bool {
	return r.SelectedID != nil
}
