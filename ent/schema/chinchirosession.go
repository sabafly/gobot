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
	"github.com/google/uuid"
	"github.com/sabafly/gobot/internal/uuidv7"
)

// ChinchiroSession holds the schema definition for the ChinchiroSession entity.
type ChinchiroSession struct {
	ent.Schema
}

// Fields of the ChinchiroSession.
func (ChinchiroSession) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuidv7.New),
		field.Int("turn").Default(0).Comment("親を決めるための回の数"),
		field.Int("loop").Default(0).Comment("その回でサイコロを振った数"),
	}
}

// Edges of the ChinchiroSession.
func (ChinchiroSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("chinchiro_sessions").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("players", ChinchiroPlayer.Type),
	}
}
