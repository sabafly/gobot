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
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/xppoint"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			Unique().
			Immutable().
			GoType(snowflake.ID(0)),
		field.String("name").
			NotEmpty(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.String("locale").
			NotEmpty().
			Default(string(discord.LocaleJapanese)).
			GoType(discord.Locale("")),
		field.Uint64("xp").
			Default(0).
			GoType(xppoint.XP(0)),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("own_guilds", Guild.Type),
		edge.To("guilds", Member.Type),
		edge.To("word_suffix", WordSuffix.Type),
		edge.To("chinchiro_sessions", ChinchiroSession.Type),
		edge.To("chinchiro_players", ChinchiroPlayer.Type),
	}
}
