package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/permissions"
	"github.com/sabafly/gobot/internal/xppoint"
)

// Member holds the schema definition for the Member entity.
type Member struct {
	ent.Schema
}

// Fields of the Member.
func (Member) Fields() []ent.Field {
	return []ent.Field{
		field.JSON("permission", permissions.Permission{}).
			Default(permissions.Permission{}).
			Optional(),
		field.Uint64("xp").
			Default(0).
			GoType(xppoint.XP(0)),
		field.Uint64("user_id").
			Immutable().
			GoType(snowflake.ID(0)),
		field.Time("last_xp").
			Optional(),
		field.Uint64("message_count").
			Default(0),
		field.Uint64("last_notified_level").
			Default(0).
			Nillable().
			Optional(),
	}
}

// Edges of the Member.
func (Member) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("members").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("guilds").
			Field("user_id").
			Immutable().
			Unique().
			Required(),
	}
}
