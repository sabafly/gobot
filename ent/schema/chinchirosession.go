package schema

import (
	"entgo.io/ent"
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
			Unique(),
		edge.To("players", ChinchiroPlayer.Type),
	}
}
