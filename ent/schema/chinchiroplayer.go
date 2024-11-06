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

// ChinchiroPlayer holds the schema definition for the ChinchiroPlayer entity.
type ChinchiroPlayer struct {
	ent.Schema
}

// Fields of the ChinchiroPlayer.
func (ChinchiroPlayer) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuidv7.New),
		field.Int("point").Default(0),
		field.Bool("is_owner").Default(false),
		field.Uint64("user_id").
			Immutable().
			GoType(snowflake.ID(0)),
		field.Int("bet").Optional().Nillable(),
		field.Ints("dices").Optional().Comment("サイコロの目"),
	}
}

// Edges of the ChinchiroPlayer.
func (ChinchiroPlayer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("chinchiro_players").
			Field("user_id").
			Unique().
			Immutable().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("session", ChinchiroSession.Type).
			Ref("players").
			Unique().
			Immutable().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
