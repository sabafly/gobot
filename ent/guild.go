// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/user"
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
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
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

// scanValues returns the types for scanning values from sql.Rows.
func (*Guild) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case guild.FieldID:
			values[i] = new(sql.NullInt64)
		case guild.FieldName, guild.FieldLocale:
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
	builder.WriteByte(')')
	return builder.String()
}

// Guilds is a parsable slice of Guild.
type Guilds []*Guild