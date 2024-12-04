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
	"encoding/json"
	"entgo.io/ent/dialect/entsql"
	"github.com/sabafly/gobot/internal/uuidv7"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// MessagePin holds the schema definition for the MessagePin entity.
type MessagePin struct {
	ent.Schema
}

// Fields of the MessagePin.
func (MessagePin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuidv7.New()).
			Unique().
			Immutable().
			Default(uuidv7.New),
		field.Uint64("channel_id").
			Unique().
			GoType(snowflake.ID(0)),
		field.Text("content").
			Optional(),
		field.JSON("embeds", []discord.Embed{}).
			Optional(),
		field.Uint64("before_id").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.JSON("rate_limit", RateLimit{}).Default(RateLimit{limit: []time.Time{}}),
	}
}

type RateLimit struct {
	limit []time.Time
}

func (r RateLimit) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.limit)
}

func (r *RateLimit) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &r.limit)
}

func (r *RateLimit) CheckLimit() bool {
	if !((len(r.limit) < 3 || time.Since(r.limit[2]) >= time.Second*5) && (len(r.limit) < 10 || time.Since(r.limit[9]) >= time.Second*30)) {
		return false
	}
	r.limit = append([]time.Time{time.Now()}, r.limit[0:min(10, len(r.limit))]...)
	ok := (len(r.limit) < 3 || time.Since(r.limit[2]) >= time.Second*5) && (len(r.limit) < 10 || time.Since(r.limit[9]) >= time.Second*30)
	return ok
}

// Edges of the MessagePin.
func (MessagePin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("message_pins").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
