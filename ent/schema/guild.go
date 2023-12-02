package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/permissions"
)

// Guild holds the schema definition for the Guild entity.
type Guild struct {
	ent.Schema
}

// Fields of the Guild.
func (Guild) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			Unique().
			Immutable().
			GoType(snowflake.ID(0)),
		field.String("name").
			NotEmpty(),
		field.String("locale").
			NotEmpty().
			Default(string(discord.LocaleJapanese)).
			GoType(discord.Locale("")),
		field.String("level_up_message").
			NotEmpty().
			Default("{user}がレベルアップしたよ！🥳\n**{before_level} レベル → {after_level} レベル**"),
		field.Uint64("level_up_channel").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.JSON("level_up_exclude_channel", []snowflake.ID{}).
			Optional(),
		field.Bool("level_mee6_imported").
			Default(false),
		field.JSON("level_role", map[int]snowflake.ID{}).
			Default(make(map[int]snowflake.ID)).
			Optional(),
		field.JSON("permissions", map[snowflake.ID]permissions.Permission{}).
			Default(make(map[snowflake.ID]permissions.Permission)).
			Optional(),
	}
}

// Edges of the Guild.
func (Guild) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("own_guilds").
			Unique().
			Required(),
		edge.To("members", Member.Type),
		edge.To("message_pins", MessagePin.Type),
		edge.To("reminds", MessageRemind.Type),
		edge.To("role_panels", RolePanel.Type),
		edge.To("role_panel_placements", RolePanelPlaced.Type),
		edge.To("role_panel_edits", RolePanelEdit.Type),
	}
}
