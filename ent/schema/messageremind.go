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
	"github.com/sabafly/gobot/internal/uuidv7"
)

// MessageRemind holds the schema definition for the MessageRemind entity.
type MessageRemind struct {
	ent.Schema
}

// Fields of the MessageRemind.
func (MessageRemind) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuidv7.New()).
			Unique().
			Immutable().
			Default(uuidv7.New),
		field.Uint64("channel_id").
			GoType(snowflake.ID(0)),
		field.Uint64("author_id").
			GoType(snowflake.ID(0)),
		field.Time("time"),
		field.String("content").
			NotEmpty(),
		field.String("name").
			NotEmpty(),
	}
}

// Edges of the MessageRemind.
func (MessageRemind) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("reminds").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
