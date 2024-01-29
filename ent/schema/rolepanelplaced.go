package schema

import (
	"github.com/sabafly/gobot/internal/uuid"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// RolePanelPlaced holds the schema definition for the RolePanelPlaced entity.
type RolePanelPlaced struct {
	ent.Schema
}

// Fields of the RolePanelPlaced.
func (RolePanelPlaced) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.New()).
			Immutable().
			Unique().
			Default(uuid.New),
		field.Uint64("message_id").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.Uint64("channel_id").
			GoType(snowflake.ID(0)),
		field.Enum("type").
			Values("button", "reaction", "select_menu").
			Optional(),
		field.Int("button_type").
			Min(discord.ButtonStylePrimary).
			Max(discord.ButtonStyleDanger).
			Default(discord.ButtonStylePrimary).
			GoType(discord.ButtonStyle(0)),
		field.Bool("show_name").
			Default(false),
		field.Bool("folding_select_menu").
			Default(true),
		field.Bool("hide_notice").
			Default(false),
		field.Bool("use_display_name").
			Default(false),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
		field.Int("uses").
			Default(0),
		field.String("name").
			NotEmpty(),
		field.String("description"),
		field.JSON("roles", []Role{}).
			Optional(),
		field.Time("updated_at").
			Optional(),
	}
}

// Edges of the RolePanelPlaced.
func (RolePanelPlaced) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("role_panel_placements").
			Required().
			Unique(),
		edge.From("role_panel", RolePanel.Type).
			Ref("placements").
			Required().
			Unique(),
	}
}
