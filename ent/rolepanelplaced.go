// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
)

// RolePanelPlaced is the model entity for the RolePanelPlaced schema.
type RolePanelPlaced struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// MessageID holds the value of the "message_id" field.
	MessageID *snowflake.ID `json:"message_id,omitempty"`
	// ChannelID holds the value of the "channel_id" field.
	ChannelID snowflake.ID `json:"channel_id,omitempty"`
	// Type holds the value of the "type" field.
	Type rolepanelplaced.Type `json:"type,omitempty"`
	// ButtonType holds the value of the "button_type" field.
	ButtonType discord.ButtonStyle `json:"button_type,omitempty"`
	// ShowName holds the value of the "show_name" field.
	ShowName bool `json:"show_name,omitempty"`
	// FoldingSelectMenu holds the value of the "folding_select_menu" field.
	FoldingSelectMenu bool `json:"folding_select_menu,omitempty"`
	// HideNotice holds the value of the "hide_notice" field.
	HideNotice bool `json:"hide_notice,omitempty"`
	// UseDisplayName holds the value of the "use_display_name" field.
	UseDisplayName bool `json:"use_display_name,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// Uses holds the value of the "uses" field.
	Uses int `json:"uses,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the RolePanelPlacedQuery when eager-loading is set.
	Edges                       RolePanelPlacedEdges `json:"edges"`
	guild_role_panel_placements *snowflake.ID
	role_panel_placements       *uuid.UUID
	selectValues                sql.SelectValues
}

// RolePanelPlacedEdges holds the relations/edges for other nodes in the graph.
type RolePanelPlacedEdges struct {
	// Guild holds the value of the guild edge.
	Guild *Guild `json:"guild,omitempty"`
	// RolePanel holds the value of the role_panel edge.
	RolePanel *RolePanel `json:"role_panel,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// GuildOrErr returns the Guild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e RolePanelPlacedEdges) GuildOrErr() (*Guild, error) {
	if e.loadedTypes[0] {
		if e.Guild == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: guild.Label}
		}
		return e.Guild, nil
	}
	return nil, &NotLoadedError{edge: "guild"}
}

// RolePanelOrErr returns the RolePanel value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e RolePanelPlacedEdges) RolePanelOrErr() (*RolePanel, error) {
	if e.loadedTypes[1] {
		if e.RolePanel == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: rolepanel.Label}
		}
		return e.RolePanel, nil
	}
	return nil, &NotLoadedError{edge: "role_panel"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*RolePanelPlaced) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case rolepanelplaced.FieldShowName, rolepanelplaced.FieldFoldingSelectMenu, rolepanelplaced.FieldHideNotice, rolepanelplaced.FieldUseDisplayName:
			values[i] = new(sql.NullBool)
		case rolepanelplaced.FieldMessageID, rolepanelplaced.FieldChannelID, rolepanelplaced.FieldButtonType, rolepanelplaced.FieldUses:
			values[i] = new(sql.NullInt64)
		case rolepanelplaced.FieldType:
			values[i] = new(sql.NullString)
		case rolepanelplaced.FieldCreatedAt:
			values[i] = new(sql.NullTime)
		case rolepanelplaced.FieldID:
			values[i] = new(uuid.UUID)
		case rolepanelplaced.ForeignKeys[0]: // guild_role_panel_placements
			values[i] = new(sql.NullInt64)
		case rolepanelplaced.ForeignKeys[1]: // role_panel_placements
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the RolePanelPlaced fields.
func (rpp *RolePanelPlaced) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case rolepanelplaced.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				rpp.ID = *value
			}
		case rolepanelplaced.FieldMessageID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field message_id", values[i])
			} else if value.Valid {
				rpp.MessageID = new(snowflake.ID)
				*rpp.MessageID = snowflake.ID(value.Int64)
			}
		case rolepanelplaced.FieldChannelID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field channel_id", values[i])
			} else if value.Valid {
				rpp.ChannelID = snowflake.ID(value.Int64)
			}
		case rolepanelplaced.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				rpp.Type = rolepanelplaced.Type(value.String)
			}
		case rolepanelplaced.FieldButtonType:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field button_type", values[i])
			} else if value.Valid {
				rpp.ButtonType = discord.ButtonStyle(value.Int64)
			}
		case rolepanelplaced.FieldShowName:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field show_name", values[i])
			} else if value.Valid {
				rpp.ShowName = value.Bool
			}
		case rolepanelplaced.FieldFoldingSelectMenu:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field folding_select_menu", values[i])
			} else if value.Valid {
				rpp.FoldingSelectMenu = value.Bool
			}
		case rolepanelplaced.FieldHideNotice:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field hide_notice", values[i])
			} else if value.Valid {
				rpp.HideNotice = value.Bool
			}
		case rolepanelplaced.FieldUseDisplayName:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field use_display_name", values[i])
			} else if value.Valid {
				rpp.UseDisplayName = value.Bool
			}
		case rolepanelplaced.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				rpp.CreatedAt = value.Time
			}
		case rolepanelplaced.FieldUses:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field uses", values[i])
			} else if value.Valid {
				rpp.Uses = int(value.Int64)
			}
		case rolepanelplaced.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field guild_role_panel_placements", values[i])
			} else if value.Valid {
				rpp.guild_role_panel_placements = new(snowflake.ID)
				*rpp.guild_role_panel_placements = snowflake.ID(value.Int64)
			}
		case rolepanelplaced.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field role_panel_placements", values[i])
			} else if value.Valid {
				rpp.role_panel_placements = new(uuid.UUID)
				*rpp.role_panel_placements = *value.S.(*uuid.UUID)
			}
		default:
			rpp.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the RolePanelPlaced.
// This includes values selected through modifiers, order, etc.
func (rpp *RolePanelPlaced) Value(name string) (ent.Value, error) {
	return rpp.selectValues.Get(name)
}

// QueryGuild queries the "guild" edge of the RolePanelPlaced entity.
func (rpp *RolePanelPlaced) QueryGuild() *GuildQuery {
	return NewRolePanelPlacedClient(rpp.config).QueryGuild(rpp)
}

// QueryRolePanel queries the "role_panel" edge of the RolePanelPlaced entity.
func (rpp *RolePanelPlaced) QueryRolePanel() *RolePanelQuery {
	return NewRolePanelPlacedClient(rpp.config).QueryRolePanel(rpp)
}

// Update returns a builder for updating this RolePanelPlaced.
// Note that you need to call RolePanelPlaced.Unwrap() before calling this method if this RolePanelPlaced
// was returned from a transaction, and the transaction was committed or rolled back.
func (rpp *RolePanelPlaced) Update() *RolePanelPlacedUpdateOne {
	return NewRolePanelPlacedClient(rpp.config).UpdateOne(rpp)
}

// Unwrap unwraps the RolePanelPlaced entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (rpp *RolePanelPlaced) Unwrap() *RolePanelPlaced {
	_tx, ok := rpp.config.driver.(*txDriver)
	if !ok {
		panic("ent: RolePanelPlaced is not a transactional entity")
	}
	rpp.config.driver = _tx.drv
	return rpp
}

// String implements the fmt.Stringer.
func (rpp *RolePanelPlaced) String() string {
	var builder strings.Builder
	builder.WriteString("RolePanelPlaced(")
	builder.WriteString(fmt.Sprintf("id=%v, ", rpp.ID))
	if v := rpp.MessageID; v != nil {
		builder.WriteString("message_id=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("channel_id=")
	builder.WriteString(fmt.Sprintf("%v", rpp.ChannelID))
	builder.WriteString(", ")
	builder.WriteString("type=")
	builder.WriteString(fmt.Sprintf("%v", rpp.Type))
	builder.WriteString(", ")
	builder.WriteString("button_type=")
	builder.WriteString(fmt.Sprintf("%v", rpp.ButtonType))
	builder.WriteString(", ")
	builder.WriteString("show_name=")
	builder.WriteString(fmt.Sprintf("%v", rpp.ShowName))
	builder.WriteString(", ")
	builder.WriteString("folding_select_menu=")
	builder.WriteString(fmt.Sprintf("%v", rpp.FoldingSelectMenu))
	builder.WriteString(", ")
	builder.WriteString("hide_notice=")
	builder.WriteString(fmt.Sprintf("%v", rpp.HideNotice))
	builder.WriteString(", ")
	builder.WriteString("use_display_name=")
	builder.WriteString(fmt.Sprintf("%v", rpp.UseDisplayName))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(rpp.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("uses=")
	builder.WriteString(fmt.Sprintf("%v", rpp.Uses))
	builder.WriteByte(')')
	return builder.String()
}

// RolePanelPlaceds is a parsable slice of RolePanelPlaced.
type RolePanelPlaceds []*RolePanelPlaced
