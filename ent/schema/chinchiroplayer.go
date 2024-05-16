package schema

import (
	"entgo.io/ent"
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
			Required(),
		edge.From("session", ChinchiroSession.Type).
			Ref("players").
			Unique().
			Immutable().
			Required(),
	}
}
