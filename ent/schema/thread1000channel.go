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
	"github.com/google/uuid"
	"github.com/sabafly/gobot/internal/uuidv7"
)

// Thread1000Channel holds the schema definition for the Thread1000Channel entity.
type Thread1000Channel struct {
	ent.Schema
}

// Fields of the Thread1000Channel.
func (Thread1000Channel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Unique().Immutable().
			Default(uuidv7.New),
		field.String("name").
			Nillable().
			Optional(),
		field.String("anonymous_name").
			Nillable().
			Optional(),
		field.Uint64("channel_id").
			GoType(snowflake.ID(0)),
	}
}

// Edges of the Thread1000Channel.
func (Thread1000Channel) Edges() []ent.Edge {
	return []ent.Edge{
		// guild edge
		edge.From("guild", Guild.Type).
			Ref("thread1000_channels").
			Unique().Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("threads", Thread1000.Type),
	}
}
