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
