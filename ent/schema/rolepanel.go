package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

// RolePanel holds the schema definition for the RolePanel entity.
type RolePanel struct {
	ent.Schema
}

// Fields of the RolePanel.
func (RolePanel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.New()).
			Immutable().
			Unique().
			Default(uuid.New),
		field.String("name").
			NotEmpty().
			MaxLen(32),
		field.String("description").
			MaxLen(140),
		field.JSON("roles", []Role{}).
			Optional(),
		field.Time("updated_at").Optional(),
		field.Time("applied_at").Optional(),
	}
}

type Role struct {
	ID    snowflake.ID            `json:"id"`
	Name  string                  `json:"name"`
	Emoji *discord.ComponentEmoji `json:"emoji"`
}

// Edges of the RolePanel.
func (RolePanel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("role_panels").
			Required().
			Unique(),
		edge.To("placements", RolePanelPlaced.Type),
		edge.To("edit", RolePanelEdit.Type).
			Unique(),
	}
}
