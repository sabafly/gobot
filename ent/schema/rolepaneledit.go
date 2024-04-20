package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/uuidv7"
)

// RolePanelEdit holds the schema definition for the RolePanelEdit entity.
type RolePanelEdit struct {
	ent.Schema
}

// Fields of the RolePanelEdit.
func (RolePanelEdit) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuidv7.New()).
			Immutable().
			Unique().
			Default(uuidv7.New),
		field.Uint64("channel_id").
			GoType(snowflake.ID(0)),
		field.Uint64("emoji_author").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.String("token").
			Optional().
			Nillable(),
		field.Uint64("selected_role").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.Bool("modified").
			Default(false),
		field.String("name").
			Optional().
			Nillable().
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.JSON("roles", []Role{}).
			Optional(),
	}
}

// Edges of the RolePanelEdit.
func (RolePanelEdit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("role_panel_edits").
			Required().
			Unique().
			Immutable(),
		edge.From("parent", RolePanel.Type).
			Ref("edit").
			Required().
			Unique().
			Immutable(),
	}
}
