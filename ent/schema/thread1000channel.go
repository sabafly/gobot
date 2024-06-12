package schema

import (
	"entgo.io/ent"
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
			Unique().Required(),
		edge.To("threads", Thread1000.Type),
	}
}
