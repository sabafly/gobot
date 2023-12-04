// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
)

// RolePanelEdit is the model entity for the RolePanelEdit schema.
type RolePanelEdit struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// ChannelID holds the value of the "channel_id" field.
	ChannelID snowflake.ID `json:"channel_id,omitempty"`
	// EmojiAuthor holds the value of the "emoji_author" field.
	EmojiAuthor *snowflake.ID `json:"emoji_author,omitempty"`
	// Token holds the value of the "token" field.
	Token *string `json:"token,omitempty"`
	// SelectedRole holds the value of the "selected_role" field.
	SelectedRole *snowflake.ID `json:"selected_role,omitempty"`
	// Modified holds the value of the "modified" field.
	Modified bool `json:"modified,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the RolePanelEditQuery when eager-loading is set.
	Edges                  RolePanelEditEdges `json:"edges"`
	guild_role_panel_edits *snowflake.ID
	role_panel_edit        *uuid.UUID
	selectValues           sql.SelectValues
}

// RolePanelEditEdges holds the relations/edges for other nodes in the graph.
type RolePanelEditEdges struct {
	// Guild holds the value of the guild edge.
	Guild *Guild `json:"guild,omitempty"`
	// Parent holds the value of the parent edge.
	Parent *RolePanel `json:"parent,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// GuildOrErr returns the Guild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e RolePanelEditEdges) GuildOrErr() (*Guild, error) {
	if e.loadedTypes[0] {
		if e.Guild == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: guild.Label}
		}
		return e.Guild, nil
	}
	return nil, &NotLoadedError{edge: "guild"}
}

// ParentOrErr returns the Parent value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e RolePanelEditEdges) ParentOrErr() (*RolePanel, error) {
	if e.loadedTypes[1] {
		if e.Parent == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: rolepanel.Label}
		}
		return e.Parent, nil
	}
	return nil, &NotLoadedError{edge: "parent"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*RolePanelEdit) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case rolepaneledit.FieldModified:
			values[i] = new(sql.NullBool)
		case rolepaneledit.FieldChannelID, rolepaneledit.FieldEmojiAuthor, rolepaneledit.FieldSelectedRole:
			values[i] = new(sql.NullInt64)
		case rolepaneledit.FieldToken:
			values[i] = new(sql.NullString)
		case rolepaneledit.FieldID:
			values[i] = new(uuid.UUID)
		case rolepaneledit.ForeignKeys[0]: // guild_role_panel_edits
			values[i] = new(sql.NullInt64)
		case rolepaneledit.ForeignKeys[1]: // role_panel_edit
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the RolePanelEdit fields.
func (rpe *RolePanelEdit) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case rolepaneledit.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				rpe.ID = *value
			}
		case rolepaneledit.FieldChannelID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field channel_id", values[i])
			} else if value.Valid {
				rpe.ChannelID = snowflake.ID(value.Int64)
			}
		case rolepaneledit.FieldEmojiAuthor:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field emoji_author", values[i])
			} else if value.Valid {
				rpe.EmojiAuthor = new(snowflake.ID)
				*rpe.EmojiAuthor = snowflake.ID(value.Int64)
			}
		case rolepaneledit.FieldToken:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field token", values[i])
			} else if value.Valid {
				rpe.Token = new(string)
				*rpe.Token = value.String
			}
		case rolepaneledit.FieldSelectedRole:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field selected_role", values[i])
			} else if value.Valid {
				rpe.SelectedRole = new(snowflake.ID)
				*rpe.SelectedRole = snowflake.ID(value.Int64)
			}
		case rolepaneledit.FieldModified:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field modified", values[i])
			} else if value.Valid {
				rpe.Modified = value.Bool
			}
		case rolepaneledit.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field guild_role_panel_edits", values[i])
			} else if value.Valid {
				rpe.guild_role_panel_edits = new(snowflake.ID)
				*rpe.guild_role_panel_edits = snowflake.ID(value.Int64)
			}
		case rolepaneledit.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field role_panel_edit", values[i])
			} else if value.Valid {
				rpe.role_panel_edit = new(uuid.UUID)
				*rpe.role_panel_edit = *value.S.(*uuid.UUID)
			}
		default:
			rpe.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the RolePanelEdit.
// This includes values selected through modifiers, order, etc.
func (rpe *RolePanelEdit) Value(name string) (ent.Value, error) {
	return rpe.selectValues.Get(name)
}

// QueryGuild queries the "guild" edge of the RolePanelEdit entity.
func (rpe *RolePanelEdit) QueryGuild() *GuildQuery {
	return NewRolePanelEditClient(rpe.config).QueryGuild(rpe)
}

// QueryParent queries the "parent" edge of the RolePanelEdit entity.
func (rpe *RolePanelEdit) QueryParent() *RolePanelQuery {
	return NewRolePanelEditClient(rpe.config).QueryParent(rpe)
}

// Update returns a builder for updating this RolePanelEdit.
// Note that you need to call RolePanelEdit.Unwrap() before calling this method if this RolePanelEdit
// was returned from a transaction, and the transaction was committed or rolled back.
func (rpe *RolePanelEdit) Update() *RolePanelEditUpdateOne {
	return NewRolePanelEditClient(rpe.config).UpdateOne(rpe)
}

// Unwrap unwraps the RolePanelEdit entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (rpe *RolePanelEdit) Unwrap() *RolePanelEdit {
	_tx, ok := rpe.config.driver.(*txDriver)
	if !ok {
		panic("ent: RolePanelEdit is not a transactional entity")
	}
	rpe.config.driver = _tx.drv
	return rpe
}

// String implements the fmt.Stringer.
func (rpe *RolePanelEdit) String() string {
	var builder strings.Builder
	builder.WriteString("RolePanelEdit(")
	builder.WriteString(fmt.Sprintf("id=%v, ", rpe.ID))
	builder.WriteString("channel_id=")
	builder.WriteString(fmt.Sprintf("%v", rpe.ChannelID))
	builder.WriteString(", ")
	if v := rpe.EmojiAuthor; v != nil {
		builder.WriteString("emoji_author=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := rpe.Token; v != nil {
		builder.WriteString("token=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := rpe.SelectedRole; v != nil {
		builder.WriteString("selected_role=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("modified=")
	builder.WriteString(fmt.Sprintf("%v", rpe.Modified))
	builder.WriteByte(')')
	return builder.String()
}

// RolePanelEdits is a parsable slice of RolePanelEdit.
type RolePanelEdits []*RolePanelEdit