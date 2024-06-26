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
