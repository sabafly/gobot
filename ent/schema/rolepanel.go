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
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/uuidv7"
)

// RolePanel holds the schema definition for the RolePanel entity.
type RolePanel struct {
	ent.Schema
}

// Fields of the RolePanel.
func (RolePanel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuidv7.New()).
			Immutable().
			Unique().
			Default(uuidv7.New),
		field.Text("name").
			NotEmpty(),
		field.Text("description"),
		field.JSON("roles", []Role{}).
			Optional(),
		field.Time("updated_at").Optional(),
		field.Time("applied_at").Optional(),
	}
}

type Role struct {
	ID    snowflake.ID            `json:"id"`
	Name  string                  `json:"name"`
	Emoji *discord.ComponentEmoji `json:"emoji"`
}

// Edges of the RolePanel.
func (RolePanel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("role_panels").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("placements", RolePanelPlaced.Type),
		edge.To("edit", RolePanelEdit.Type).
			Unique(),
	}
}
