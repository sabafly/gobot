// Code generated by ent, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/user"
	"github.com/sabafly/gobot/internal/permissions"
)

// Guild is the model entity for the Guild schema.
type Guild struct {
	config `json:"-"`
	// ID of the ent.
	ID snowflake.ID `json:"id,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Locale holds the value of the "locale" field.
	Locale discord.Locale `json:"locale,omitempty"`
	// LevelUpMessage holds the value of the "level_up_message" field.
	LevelUpMessage string `json:"level_up_message,omitempty"`
	// LevelUpChannel holds the value of the "level_up_channel" field.
	LevelUpChannel *snowflake.ID `json:"level_up_channel,omitempty"`
	// LevelUpExcludeChannel holds the value of the "level_up_exclude_channel" field.
	LevelUpExcludeChannel []snowflake.ID `json:"level_up_exclude_channel,omitempty"`
	// LevelMee6Imported holds the value of the "level_mee6_imported" field.
	LevelMee6Imported bool `json:"level_mee6_imported,omitempty"`
	// LevelRole holds the value of the "level_role" field.
	LevelRole map[int]snowflake.ID `json:"level_role,omitempty"`
	// Permissions holds the value of the "permissions" field.
	Permissions map[snowflake.ID]permissions.Permission `json:"permissions,omitempty"`
	// RemindCount holds the value of the "remind_count" field.
	RemindCount int `json:"remind_count,omitempty"`
	// RolePanelEditTimes holds the value of the "role_panel_edit_times" field.
	RolePanelEditTimes []time.Time `json:"role_panel_edit_times,omitempty"`
	// BumpEnabled holds the value of the "bump_enabled" field.
	BumpEnabled bool `json:"bump_enabled,omitempty"`
	// BumpMessageTitle holds the value of the "bump_message_title" field.
	BumpMessageTitle string `json:"bump_message_title,omitempty"`
	// BumpMessage holds the value of the "bump_message" field.
	BumpMessage string `json:"bump_message,omitempty"`
	// BumpRemindMessageTitle holds the value of the "bump_remind_message_title" field.
	BumpRemindMessageTitle string `json:"bump_remind_message_title,omitempty"`
	// BumpRemindMessage holds the value of the "bump_remind_message" field.
	BumpRemindMessage string `json:"bump_remind_message,omitempty"`
	// UpEnabled holds the value of the "up_enabled" field.
	UpEnabled bool `json:"up_enabled,omitempty"`
	// UpMessageTitle holds the value of the "up_message_title" field.
	UpMessageTitle string `json:"up_message_title,omitempty"`
	// UpMessage holds the value of the "up_message" field.
	UpMessage string `json:"up_message,omitempty"`
	// UpRemindMessageTitle holds the value of the "up_remind_message_title" field.
	UpRemindMessageTitle string `json:"up_remind_message_title,omitempty"`
	// UpRemindMessage holds the value of the "up_remind_message" field.
	UpRemindMessage string `json:"up_remind_message,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the GuildQuery when eager-loading is set.
	Edges           GuildEdges `json:"edges"`
	user_own_guilds *snowflake.ID
	selectValues    sql.SelectValues
}

// GuildEdges holds the relations/edges for other nodes in the graph.
type GuildEdges struct {
	// Owner holds the value of the owner edge.
	Owner *User `json:"owner,omitempty"`
	// Members holds the value of the members edge.
	Members []*Member `json:"members,omitempty"`
	// MessagePins holds the value of the message_pins edge.
	MessagePins []*MessagePin `json:"message_pins,omitempty"`
	// Reminds holds the value of the reminds edge.
	Reminds []*MessageRemind `json:"reminds,omitempty"`
	// RolePanels holds the value of the role_panels edge.
	RolePanels []*RolePanel `json:"role_panels,omitempty"`
	// RolePanelPlacements holds the value of the role_panel_placements edge.
	RolePanelPlacements []*RolePanelPlaced `json:"role_panel_placements,omitempty"`
	// RolePanelEdits holds the value of the role_panel_edits edge.
	RolePanelEdits []*RolePanelEdit `json:"role_panel_edits,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [7]bool
}

// OwnerOrErr returns the Owner value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e GuildEdges) OwnerOrErr() (*User, error) {
	if e.loadedTypes[0] {
		if e.Owner == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: user.Label}
		}
		return e.Owner, nil
	}
	return nil, &NotLoadedError{edge: "owner"}
}

// MembersOrErr returns the Members value or an error if the edge
// was not loaded in eager-loading.
func (e GuildEdges) MembersOrErr() ([]*Member, error) {
	if e.loadedTypes[1] {
		return e.Members, nil
	}
	return nil, &NotLoadedError{edge: "members"}
}

// MessagePinsOrErr returns the MessagePins value or an error if the edge
// was not loaded in eager-loading.
func (e GuildEdges) MessagePinsOrErr() ([]*MessagePin, error) {
	if e.loadedTypes[2] {
		return e.MessagePins, nil
	}
	return nil, &NotLoadedError{edge: "message_pins"}
}

// RemindsOrErr returns the Reminds value or an error if the edge
// was not loaded in eager-loading.
func (e GuildEdges) RemindsOrErr() ([]*MessageRemind, error) {
	if e.loadedTypes[3] {
		return e.Reminds, nil
	}
	return nil, &NotLoadedError{edge: "reminds"}
}

// RolePanelsOrErr returns the RolePanels value or an error if the edge
// was not loaded in eager-loading.
func (e GuildEdges) RolePanelsOrErr() ([]*RolePanel, error) {
	if e.loadedTypes[4] {
		return e.RolePanels, nil
	}
	return nil, &NotLoadedError{edge: "role_panels"}
}

// RolePanelPlacementsOrErr returns the RolePanelPlacements value or an error if the edge
// was not loaded in eager-loading.
func (e GuildEdges) RolePanelPlacementsOrErr() ([]*RolePanelPlaced, error) {
	if e.loadedTypes[5] {
		return e.RolePanelPlacements, nil
	}
	return nil, &NotLoadedError{edge: "role_panel_placements"}
}

// RolePanelEditsOrErr returns the RolePanelEdits value or an error if the edge
// was not loaded in eager-loading.
func (e GuildEdges) RolePanelEditsOrErr() ([]*RolePanelEdit, error) {
	if e.loadedTypes[6] {
		return e.RolePanelEdits, nil
	}
	return nil, &NotLoadedError{edge: "role_panel_edits"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Guild) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case guild.FieldLevelUpExcludeChannel, guild.FieldLevelRole, guild.FieldPermissions, guild.FieldRolePanelEditTimes:
			values[i] = new([]byte)
		case guild.FieldLevelMee6Imported, guild.FieldBumpEnabled, guild.FieldUpEnabled:
			values[i] = new(sql.NullBool)
		case guild.FieldID, guild.FieldLevelUpChannel, guild.FieldRemindCount:
			values[i] = new(sql.NullInt64)
		case guild.FieldName, guild.FieldLocale, guild.FieldLevelUpMessage, guild.FieldBumpMessageTitle, guild.FieldBumpMessage, guild.FieldBumpRemindMessageTitle, guild.FieldBumpRemindMessage, guild.FieldUpMessageTitle, guild.FieldUpMessage, guild.FieldUpRemindMessageTitle, guild.FieldUpRemindMessage:
			values[i] = new(sql.NullString)
		case guild.ForeignKeys[0]: // user_own_guilds
			values[i] = new(sql.NullInt64)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Guild fields.
func (gu *Guild) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case guild.FieldID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				gu.ID = snowflake.ID(value.Int64)
			}
		case guild.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				gu.Name = value.String
			}
		case guild.FieldLocale:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field locale", values[i])
			} else if value.Valid {
				gu.Locale = discord.Locale(value.String)
			}
		case guild.FieldLevelUpMessage:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field level_up_message", values[i])
			} else if value.Valid {
				gu.LevelUpMessage = value.String
			}
		case guild.FieldLevelUpChannel:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field level_up_channel", values[i])
			} else if value.Valid {
				gu.LevelUpChannel = new(snowflake.ID)
				*gu.LevelUpChannel = snowflake.ID(value.Int64)
			}
		case guild.FieldLevelUpExcludeChannel:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field level_up_exclude_channel", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &gu.LevelUpExcludeChannel); err != nil {
					return fmt.Errorf("unmarshal field level_up_exclude_channel: %w", err)
				}
			}
		case guild.FieldLevelMee6Imported:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field level_mee6_imported", values[i])
			} else if value.Valid {
				gu.LevelMee6Imported = value.Bool
			}
		case guild.FieldLevelRole:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field level_role", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &gu.LevelRole); err != nil {
					return fmt.Errorf("unmarshal field level_role: %w", err)
				}
			}
		case guild.FieldPermissions:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field permissions", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &gu.Permissions); err != nil {
					return fmt.Errorf("unmarshal field permissions: %w", err)
				}
			}
		case guild.FieldRemindCount:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field remind_count", values[i])
			} else if value.Valid {
				gu.RemindCount = int(value.Int64)
			}
		case guild.FieldRolePanelEditTimes:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field role_panel_edit_times", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &gu.RolePanelEditTimes); err != nil {
					return fmt.Errorf("unmarshal field role_panel_edit_times: %w", err)
				}
			}
		case guild.FieldBumpEnabled:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field bump_enabled", values[i])
			} else if value.Valid {
				gu.BumpEnabled = value.Bool
			}
		case guild.FieldBumpMessageTitle:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field bump_message_title", values[i])
			} else if value.Valid {
				gu.BumpMessageTitle = value.String
			}
		case guild.FieldBumpMessage:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field bump_message", values[i])
			} else if value.Valid {
				gu.BumpMessage = value.String
			}
		case guild.FieldBumpRemindMessageTitle:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field bump_remind_message_title", values[i])
			} else if value.Valid {
				gu.BumpRemindMessageTitle = value.String
			}
		case guild.FieldBumpRemindMessage:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field bump_remind_message", values[i])
			} else if value.Valid {
				gu.BumpRemindMessage = value.String
			}
		case guild.FieldUpEnabled:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field up_enabled", values[i])
			} else if value.Valid {
				gu.UpEnabled = value.Bool
			}
		case guild.FieldUpMessageTitle:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field up_message_title", values[i])
			} else if value.Valid {
				gu.UpMessageTitle = value.String
			}
		case guild.FieldUpMessage:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field up_message", values[i])
			} else if value.Valid {
				gu.UpMessage = value.String
			}
		case guild.FieldUpRemindMessageTitle:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field up_remind_message_title", values[i])
			} else if value.Valid {
				gu.UpRemindMessageTitle = value.String
			}
		case guild.FieldUpRemindMessage:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field up_remind_message", values[i])
			} else if value.Valid {
				gu.UpRemindMessage = value.String
			}
		case guild.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field user_own_guilds", values[i])
			} else if value.Valid {
				gu.user_own_guilds = new(snowflake.ID)
				*gu.user_own_guilds = snowflake.ID(value.Int64)
			}
		default:
			gu.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Guild.
// This includes values selected through modifiers, order, etc.
func (gu *Guild) Value(name string) (ent.Value, error) {
	return gu.selectValues.Get(name)
}

// QueryOwner queries the "owner" edge of the Guild entity.
func (gu *Guild) QueryOwner() *UserQuery {
	return NewGuildClient(gu.config).QueryOwner(gu)
}

// QueryMembers queries the "members" edge of the Guild entity.
func (gu *Guild) QueryMembers() *MemberQuery {
	return NewGuildClient(gu.config).QueryMembers(gu)
}

// QueryMessagePins queries the "message_pins" edge of the Guild entity.
func (gu *Guild) QueryMessagePins() *MessagePinQuery {
	return NewGuildClient(gu.config).QueryMessagePins(gu)
}

// QueryReminds queries the "reminds" edge of the Guild entity.
func (gu *Guild) QueryReminds() *MessageRemindQuery {
	return NewGuildClient(gu.config).QueryReminds(gu)
}

// QueryRolePanels queries the "role_panels" edge of the Guild entity.
func (gu *Guild) QueryRolePanels() *RolePanelQuery {
	return NewGuildClient(gu.config).QueryRolePanels(gu)
}

// QueryRolePanelPlacements queries the "role_panel_placements" edge of the Guild entity.
func (gu *Guild) QueryRolePanelPlacements() *RolePanelPlacedQuery {
	return NewGuildClient(gu.config).QueryRolePanelPlacements(gu)
}

// QueryRolePanelEdits queries the "role_panel_edits" edge of the Guild entity.
func (gu *Guild) QueryRolePanelEdits() *RolePanelEditQuery {
	return NewGuildClient(gu.config).QueryRolePanelEdits(gu)
}

// Update returns a builder for updating this Guild.
// Note that you need to call Guild.Unwrap() before calling this method if this Guild
// was returned from a transaction, and the transaction was committed or rolled back.
func (gu *Guild) Update() *GuildUpdateOne {
	return NewGuildClient(gu.config).UpdateOne(gu)
}

// Unwrap unwraps the Guild entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (gu *Guild) Unwrap() *Guild {
	_tx, ok := gu.config.driver.(*txDriver)
	if !ok {
		panic("ent: Guild is not a transactional entity")
	}
	gu.config.driver = _tx.drv
	return gu
}

// String implements the fmt.Stringer.
func (gu *Guild) String() string {
	var builder strings.Builder
	builder.WriteString("Guild(")
	builder.WriteString(fmt.Sprintf("id=%v, ", gu.ID))
	builder.WriteString("name=")
	builder.WriteString(gu.Name)
	builder.WriteString(", ")
	builder.WriteString("locale=")
	builder.WriteString(fmt.Sprintf("%v", gu.Locale))
	builder.WriteString(", ")
	builder.WriteString("level_up_message=")
	builder.WriteString(gu.LevelUpMessage)
	builder.WriteString(", ")
	if v := gu.LevelUpChannel; v != nil {
		builder.WriteString("level_up_channel=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("level_up_exclude_channel=")
	builder.WriteString(fmt.Sprintf("%v", gu.LevelUpExcludeChannel))
	builder.WriteString(", ")
	builder.WriteString("level_mee6_imported=")
	builder.WriteString(fmt.Sprintf("%v", gu.LevelMee6Imported))
	builder.WriteString(", ")
	builder.WriteString("level_role=")
	builder.WriteString(fmt.Sprintf("%v", gu.LevelRole))
	builder.WriteString(", ")
	builder.WriteString("permissions=")
	builder.WriteString(fmt.Sprintf("%v", gu.Permissions))
	builder.WriteString(", ")
	builder.WriteString("remind_count=")
	builder.WriteString(fmt.Sprintf("%v", gu.RemindCount))
	builder.WriteString(", ")
	builder.WriteString("role_panel_edit_times=")
	builder.WriteString(fmt.Sprintf("%v", gu.RolePanelEditTimes))
	builder.WriteString(", ")
	builder.WriteString("bump_enabled=")
	builder.WriteString(fmt.Sprintf("%v", gu.BumpEnabled))
	builder.WriteString(", ")
	builder.WriteString("bump_message_title=")
	builder.WriteString(gu.BumpMessageTitle)
	builder.WriteString(", ")
	builder.WriteString("bump_message=")
	builder.WriteString(gu.BumpMessage)
	builder.WriteString(", ")
	builder.WriteString("bump_remind_message_title=")
	builder.WriteString(gu.BumpRemindMessageTitle)
	builder.WriteString(", ")
	builder.WriteString("bump_remind_message=")
	builder.WriteString(gu.BumpRemindMessage)
	builder.WriteString(", ")
	builder.WriteString("up_enabled=")
	builder.WriteString(fmt.Sprintf("%v", gu.UpEnabled))
	builder.WriteString(", ")
	builder.WriteString("up_message_title=")
	builder.WriteString(gu.UpMessageTitle)
	builder.WriteString(", ")
	builder.WriteString("up_message=")
	builder.WriteString(gu.UpMessage)
	builder.WriteString(", ")
	builder.WriteString("up_remind_message_title=")
	builder.WriteString(gu.UpRemindMessageTitle)
	builder.WriteString(", ")
	builder.WriteString("up_remind_message=")
	builder.WriteString(gu.UpRemindMessage)
	builder.WriteByte(')')
	return builder.String()
}

// Guilds is a parsable slice of Guild.
type Guilds []*Guild
