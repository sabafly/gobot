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
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/uuidv7"
)

// WordSuffix holds the schema definition for the WordSuffix entity.
type WordSuffix struct {
	ent.Schema
}

// Fields of the WordSuffix.
func (WordSuffix) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuidv7.New()).
			Default(uuidv7.New),
		field.String("suffix").
			NotEmpty(),
		field.Time("expired").
			Optional().
			Nillable(),
		field.Uint64("guild_id").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.Enum("rule").
			Values("webhook", "warn", "delete").
			Default("webhook"),
	}
}

// Edges of the WordSuffix.
func (WordSuffix) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("guild", Guild.Type).
			Field("guild_id").
			Unique(),
		edge.From("owner", User.Type).
			Ref("word_suffix").
			Unique().
			Required(),
	}
}
