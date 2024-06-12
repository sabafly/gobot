package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/internal/uuidv7"
)

// Thread1000 holds the schema definition for the Thread1000 entity.
type Thread1000 struct {
	ent.Schema
}

// Fields of the Thread1000.
func (Thread1000) Fields() []ent.Field {
	return []ent.Field{
		// uuid
		field.UUID("id", uuid.UUID{}).
			Unique().
			Immutable().
			Default(uuidv7.New),
		field.String("name").
			NotEmpty(),
		field.Int("message_count").
			Default(0),
		field.Bool("is_archived").
			Default(false),
		field.Uint64("thread_id").
			GoType(snowflake.ID(0)),
	}
}

// Edges of the Thread1000.
func (Thread1000) Edges() []ent.Edge {
	return []ent.Edge{
		// guild edge
		edge.From("guild", Guild.Type).
			Ref("threads1000").
			Unique().
			Required(),
		edge.From("channel", Thread1000Channel.Type).
			Ref("threads").
			Unique().
			Required(),
	}
}
