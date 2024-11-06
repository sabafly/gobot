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

package schema

import (
	"entgo.io/ent/dialect/entsql"
	"github.com/sabafly/gobot/internal/uuidv7"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// RolePanelPlaced holds the schema definition for the RolePanelPlaced entity.
type RolePanelPlaced struct {
	ent.Schema
}

// Fields of the RolePanelPlaced.
func (RolePanelPlaced) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuidv7.New()).
			Immutable().
			Unique().
			Default(uuidv7.New),
		field.Uint64("message_id").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.Uint64("channel_id").
			GoType(snowflake.ID(0)),
		field.Enum("type").
			Values("button", "reaction", "select_menu").
			Optional(),
		field.Int("button_type").
			Min(int(discord.ButtonStylePrimary)).
			Max(int(discord.ButtonStyleDanger)).
			Default(int(discord.ButtonStylePrimary)).
			GoType(discord.ButtonStyle(0)),
		field.Bool("show_name").
			Default(false),
		field.Bool("folding_select_menu").
			Default(true),
		field.Bool("hide_notice").
			Default(false),
		field.Bool("use_display_name").
			Default(false),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
		field.Int("uses").
			Default(0),
		field.String("name").
			NotEmpty(),
		field.String("description"),
		field.JSON("roles", []Role{}).
			Optional(),
		field.Time("updated_at").
			Optional(),
	}
}

// Edges of the RolePanelPlaced.
func (RolePanelPlaced) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("role_panel_placements").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("role_panel", RolePanel.Type).
			Ref("placements").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
