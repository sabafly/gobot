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
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/permissions"
	"github.com/sabafly/gobot/internal/xppoint"
)

// Member holds the schema definition for the Member entity.
type Member struct {
	ent.Schema
}

// Fields of the Member.
func (Member) Fields() []ent.Field {
	return []ent.Field{
		field.JSON("permission", permissions.Permission{}).
			Default(permissions.Permission{}).
			Optional(),
		field.Uint64("xp").
			Default(0).
			GoType(xppoint.XP(0)),
		field.Uint64("user_id").
			Immutable().
			GoType(snowflake.ID(0)),
		field.Time("last_xp").
			Optional(),
		field.Uint64("message_count").
			Default(0),
		field.Uint64("last_notified_level").
			Nillable().
			Optional(),
	}
}

// Edges of the Member.
func (Member) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("members").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("user", User.Type).
			Ref("guilds").
			Field("user_id").
			Immutable().
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
