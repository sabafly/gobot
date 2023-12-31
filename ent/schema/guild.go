package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
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
			Default(make(map[snowflake.ID]permissions.Permission)),
		field.Int("remind_count").
			Default(0),
		field.JSON("role_panel_edit_times", []time.Time{}).
			Default([]time.Time{}).
			Annotations(
				entsql.Default(`[]`),
			),
		field.Bool("bump_enabled").
			Default(true),
		field.String("bump_message_title").
			NotEmpty().
			Default("Bumpを検知しました"),
		field.String("bump_message").
			NotEmpty().
			Default("２時間後に通知します"),
		field.String("bump_remind_message_title").
			NotEmpty().
			Default("Bumpの時間です"),
		field.String("bump_remind_message").
			NotEmpty().
			Default("</bump:947088344167366698>でBumpしましょう"),
		field.Bool("up_enabled").
			Default(true),
		field.String("up_message_title").
			NotEmpty().
			Default("UPを検知しました"),
		field.String("up_message").
			NotEmpty().
			Default("１時間後に通知します"),
		field.String("up_remind_message_title").
			NotEmpty().
			Default("UPの時間です"),
		field.String("up_remind_message").
			NotEmpty().
			Default("</dissoku up:828002256690610256>でUPしましょう"),
		field.Uint64("bump_mention").
			Nillable().
			Optional().
			GoType(snowflake.ID(0)),
		field.Uint64("up_mention").
			Nillable().
			Optional().
			GoType(snowflake.ID(0)),
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
