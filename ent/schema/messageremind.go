package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

// MessageRemind holds the schema definition for the MessageRemind entity.
type MessageRemind struct {
	ent.Schema
}

// Fields of the MessageRemind.
func (MessageRemind) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.New()).
			Unique().
			Immutable().
			Default(uuid.New),
		field.Uint64("channel_id").
			GoType(snowflake.ID(0)),
		field.Uint64("author_id").
			GoType(snowflake.ID(0)),
		field.Time("time"),
		field.String("content").
			NotEmpty(),
	}
}

// Edges of the MessageRemind.
func (MessageRemind) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("reminds").
			Required().
			Unique(),
	}
}
