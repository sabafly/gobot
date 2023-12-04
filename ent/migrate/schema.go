// Code generated by ent, DO NOT EDIT.

package migrate

import (
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// GuildsColumns holds the columns for the "guilds" table.
	GuildsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUint64, Increment: true},
		{Name: "name", Type: field.TypeString},
		{Name: "locale", Type: field.TypeString, Default: "ja"},
		{Name: "level_up_message", Type: field.TypeString, Default: "{user}がレベルアップしたよ！🥳\n**{before_level} レベル → {after_level} レベル**"},
		{Name: "level_up_channel", Type: field.TypeUint64, Nullable: true},
		{Name: "level_up_exclude_channel", Type: field.TypeJSON, Nullable: true},
		{Name: "level_mee6_imported", Type: field.TypeBool, Default: false},
		{Name: "level_role", Type: field.TypeJSON, Nullable: true},
		{Name: "permissions", Type: field.TypeJSON},
		{Name: "remind_count", Type: field.TypeInt, Default: 0},
		{Name: "user_own_guilds", Type: field.TypeUint64},
	}
	// GuildsTable holds the schema information for the "guilds" table.
	GuildsTable = &schema.Table{
		Name:       "guilds",
		Columns:    GuildsColumns,
		PrimaryKey: []*schema.Column{GuildsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "guilds_users_own_guilds",
				Columns:    []*schema.Column{GuildsColumns[10]},
				RefColumns: []*schema.Column{UsersColumns[0]},
				OnDelete:   schema.NoAction,
			},
		},
	}
	// MembersColumns holds the columns for the "members" table.
	MembersColumns = []*schema.Column{
		{Name: "id", Type: field.TypeInt, Increment: true},
		{Name: "permission", Type: field.TypeJSON, Nullable: true},
		{Name: "xp", Type: field.TypeUint64, Default: 0},
		{Name: "last_xp", Type: field.TypeTime, Nullable: true},
		{Name: "message_count", Type: field.TypeUint64, Default: 0},
		{Name: "guild_members", Type: field.TypeUint64},
		{Name: "user_id", Type: field.TypeUint64},
	}
	// MembersTable holds the schema information for the "members" table.
	MembersTable = &schema.Table{
		Name:       "members",
		Columns:    MembersColumns,
		PrimaryKey: []*schema.Column{MembersColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "members_guilds_members",
				Columns:    []*schema.Column{MembersColumns[5]},
				RefColumns: []*schema.Column{GuildsColumns[0]},
				OnDelete:   schema.NoAction,
			},
			{
				Symbol:     "members_users_guilds",
				Columns:    []*schema.Column{MembersColumns[6]},
				RefColumns: []*schema.Column{UsersColumns[0]},
				OnDelete:   schema.NoAction,
			},
		},
	}
	// MessagePinsColumns holds the columns for the "message_pins" table.
	MessagePinsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUUID, Unique: true},
		{Name: "channel_id", Type: field.TypeUint64, Unique: true},
		{Name: "content", Type: field.TypeString, Nullable: true},
		{Name: "embeds", Type: field.TypeJSON, Nullable: true},
		{Name: "before_id", Type: field.TypeUint64, Nullable: true},
		{Name: "rate_limit", Type: field.TypeJSON},
		{Name: "guild_message_pins", Type: field.TypeUint64},
	}
	// MessagePinsTable holds the schema information for the "message_pins" table.
	MessagePinsTable = &schema.Table{
		Name:       "message_pins",
		Columns:    MessagePinsColumns,
		PrimaryKey: []*schema.Column{MessagePinsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "message_pins_guilds_message_pins",
				Columns:    []*schema.Column{MessagePinsColumns[6]},
				RefColumns: []*schema.Column{GuildsColumns[0]},
				OnDelete:   schema.NoAction,
			},
		},
	}
	// MessageRemindsColumns holds the columns for the "message_reminds" table.
	MessageRemindsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUUID, Unique: true},
		{Name: "channel_id", Type: field.TypeUint64},
		{Name: "author_id", Type: field.TypeUint64},
		{Name: "time", Type: field.TypeTime},
		{Name: "content", Type: field.TypeString},
		{Name: "name", Type: field.TypeString},
		{Name: "guild_reminds", Type: field.TypeUint64},
	}
	// MessageRemindsTable holds the schema information for the "message_reminds" table.
	MessageRemindsTable = &schema.Table{
		Name:       "message_reminds",
		Columns:    MessageRemindsColumns,
		PrimaryKey: []*schema.Column{MessageRemindsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "message_reminds_guilds_reminds",
				Columns:    []*schema.Column{MessageRemindsColumns[6]},
				RefColumns: []*schema.Column{GuildsColumns[0]},
				OnDelete:   schema.NoAction,
			},
		},
	}
	// RolePanelsColumns holds the columns for the "role_panels" table.
	RolePanelsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUUID, Unique: true},
		{Name: "name", Type: field.TypeString, Size: 32},
		{Name: "description", Type: field.TypeString, Size: 140},
		{Name: "roles", Type: field.TypeJSON, Nullable: true},
		{Name: "guild_role_panels", Type: field.TypeUint64},
	}
	// RolePanelsTable holds the schema information for the "role_panels" table.
	RolePanelsTable = &schema.Table{
		Name:       "role_panels",
		Columns:    RolePanelsColumns,
		PrimaryKey: []*schema.Column{RolePanelsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "role_panels_guilds_role_panels",
				Columns:    []*schema.Column{RolePanelsColumns[4]},
				RefColumns: []*schema.Column{GuildsColumns[0]},
				OnDelete:   schema.NoAction,
			},
		},
	}
	// RolePanelEditsColumns holds the columns for the "role_panel_edits" table.
	RolePanelEditsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUUID, Unique: true},
		{Name: "channel_id", Type: field.TypeUint64},
		{Name: "emoji_author", Type: field.TypeUint64, Nullable: true},
		{Name: "token", Type: field.TypeString, Nullable: true},
		{Name: "selected_role", Type: field.TypeUint64, Nullable: true},
		{Name: "modified", Type: field.TypeBool, Default: false},
		{Name: "guild_role_panel_edits", Type: field.TypeUint64},
		{Name: "role_panel_edit", Type: field.TypeUUID, Unique: true},
	}
	// RolePanelEditsTable holds the schema information for the "role_panel_edits" table.
	RolePanelEditsTable = &schema.Table{
		Name:       "role_panel_edits",
		Columns:    RolePanelEditsColumns,
		PrimaryKey: []*schema.Column{RolePanelEditsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "role_panel_edits_guilds_role_panel_edits",
				Columns:    []*schema.Column{RolePanelEditsColumns[6]},
				RefColumns: []*schema.Column{GuildsColumns[0]},
				OnDelete:   schema.NoAction,
			},
			{
				Symbol:     "role_panel_edits_role_panels_edit",
				Columns:    []*schema.Column{RolePanelEditsColumns[7]},
				RefColumns: []*schema.Column{RolePanelsColumns[0]},
				OnDelete:   schema.NoAction,
			},
		},
	}
	// RolePanelPlacedsColumns holds the columns for the "role_panel_placeds" table.
	RolePanelPlacedsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUUID, Unique: true},
		{Name: "message_id", Type: field.TypeUint64, Nullable: true},
		{Name: "channel_id", Type: field.TypeUint64},
		{Name: "type", Type: field.TypeEnum, Nullable: true, Enums: []string{"button", "reaction", "select_menu"}},
		{Name: "button_type", Type: field.TypeInt, Default: 1},
		{Name: "show_name", Type: field.TypeBool, Default: false},
		{Name: "folding_select_menu", Type: field.TypeBool, Default: true},
		{Name: "hide_notice", Type: field.TypeBool, Default: false},
		{Name: "use_display_name", Type: field.TypeBool, Default: false},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "uses", Type: field.TypeInt, Default: 0},
		{Name: "guild_role_panel_placements", Type: field.TypeUint64},
		{Name: "role_panel_placements", Type: field.TypeUUID},
	}
	// RolePanelPlacedsTable holds the schema information for the "role_panel_placeds" table.
	RolePanelPlacedsTable = &schema.Table{
		Name:       "role_panel_placeds",
		Columns:    RolePanelPlacedsColumns,
		PrimaryKey: []*schema.Column{RolePanelPlacedsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "role_panel_placeds_guilds_role_panel_placements",
				Columns:    []*schema.Column{RolePanelPlacedsColumns[11]},
				RefColumns: []*schema.Column{GuildsColumns[0]},
				OnDelete:   schema.NoAction,
			},
			{
				Symbol:     "role_panel_placeds_role_panels_placements",
				Columns:    []*schema.Column{RolePanelPlacedsColumns[12]},
				RefColumns: []*schema.Column{RolePanelsColumns[0]},
				OnDelete:   schema.NoAction,
			},
		},
	}
	// UsersColumns holds the columns for the "users" table.
	UsersColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUint64, Increment: true},
		{Name: "name", Type: field.TypeString},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "locale", Type: field.TypeString, Default: "ja"},
		{Name: "xp", Type: field.TypeUint64, Default: 0},
	}
	// UsersTable holds the schema information for the "users" table.
	UsersTable = &schema.Table{
		Name:       "users",
		Columns:    UsersColumns,
		PrimaryKey: []*schema.Column{UsersColumns[0]},
	}
	// WordSuffixesColumns holds the columns for the "word_suffixes" table.
	WordSuffixesColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUUID},
		{Name: "suffix", Type: field.TypeString, Size: 120},
		{Name: "expired", Type: field.TypeTime, Nullable: true},
		{Name: "rule", Type: field.TypeEnum, Enums: []string{"webhook", "warn", "delete"}, Default: "webhook"},
		{Name: "user_word_suffix", Type: field.TypeUint64},
		{Name: "guild_id", Type: field.TypeUint64, Nullable: true},
	}
	// WordSuffixesTable holds the schema information for the "word_suffixes" table.
	WordSuffixesTable = &schema.Table{
		Name:       "word_suffixes",
		Columns:    WordSuffixesColumns,
		PrimaryKey: []*schema.Column{WordSuffixesColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "word_suffixes_users_word_suffix",
				Columns:    []*schema.Column{WordSuffixesColumns[4]},
				RefColumns: []*schema.Column{UsersColumns[0]},
				OnDelete:   schema.NoAction,
			},
			{
				Symbol:     "word_suffixes_guilds_guild",
				Columns:    []*schema.Column{WordSuffixesColumns[5]},
				RefColumns: []*schema.Column{GuildsColumns[0]},
				OnDelete:   schema.SetNull,
			},
		},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		GuildsTable,
		MembersTable,
		MessagePinsTable,
		MessageRemindsTable,
		RolePanelsTable,
		RolePanelEditsTable,
		RolePanelPlacedsTable,
		UsersTable,
		WordSuffixesTable,
	}
)

func init() {
	GuildsTable.ForeignKeys[0].RefTable = UsersTable
	MembersTable.ForeignKeys[0].RefTable = GuildsTable
	MembersTable.ForeignKeys[1].RefTable = UsersTable
	MessagePinsTable.ForeignKeys[0].RefTable = GuildsTable
	MessageRemindsTable.ForeignKeys[0].RefTable = GuildsTable
	RolePanelsTable.ForeignKeys[0].RefTable = GuildsTable
	RolePanelEditsTable.ForeignKeys[0].RefTable = GuildsTable
	RolePanelEditsTable.ForeignKeys[1].RefTable = RolePanelsTable
	RolePanelPlacedsTable.ForeignKeys[0].RefTable = GuildsTable
	RolePanelPlacedsTable.ForeignKeys[1].RefTable = RolePanelsTable
	WordSuffixesTable.ForeignKeys[0].RefTable = UsersTable
	WordSuffixesTable.ForeignKeys[1].RefTable = GuildsTable
}
