// Code generated by ent, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/user"
	"github.com/sabafly/gobot/internal/permissions"
)

// Member is the model entity for the Member schema.
type Member struct {
	config `json:"-"`
	// ID of the ent.
	ID int `json:"id,omitempty"`
	// Permission holds the value of the "permission" field.
	Permission permissions.Permission `json:"permission,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the MemberQuery when eager-loading is set.
	Edges         MemberEdges `json:"edges"`
	guild_members *snowflake.ID
	user_guilds   *snowflake.ID
	selectValues  sql.SelectValues
}

// MemberEdges holds the relations/edges for other nodes in the graph.
type MemberEdges struct {
	// Guild holds the value of the guild edge.
	Guild *Guild `json:"guild,omitempty"`
	// User holds the value of the user edge.
	User *User `json:"user,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// GuildOrErr returns the Guild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e MemberEdges) GuildOrErr() (*Guild, error) {
	if e.loadedTypes[0] {
		if e.Guild == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: guild.Label}
		}
		return e.Guild, nil
	}
	return nil, &NotLoadedError{edge: "guild"}
}

// UserOrErr returns the User value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e MemberEdges) UserOrErr() (*User, error) {
	if e.loadedTypes[1] {
		if e.User == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: user.Label}
		}
		return e.User, nil
	}
	return nil, &NotLoadedError{edge: "user"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Member) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case member.FieldPermission:
			values[i] = new([]byte)
		case member.FieldID:
			values[i] = new(sql.NullInt64)
		case member.ForeignKeys[0]: // guild_members
			values[i] = new(sql.NullInt64)
		case member.ForeignKeys[1]: // user_guilds
			values[i] = new(sql.NullInt64)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Member fields.
func (m *Member) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case member.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			m.ID = int(value.Int64)
		case member.FieldPermission:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field permission", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &m.Permission); err != nil {
					return fmt.Errorf("unmarshal field permission: %w", err)
				}
			}
		case member.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field guild_members", values[i])
			} else if value.Valid {
				m.guild_members = new(snowflake.ID)
				*m.guild_members = snowflake.ID(value.Int64)
			}
		case member.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field user_guilds", values[i])
			} else if value.Valid {
				m.user_guilds = new(snowflake.ID)
				*m.user_guilds = snowflake.ID(value.Int64)
			}
		default:
			m.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Member.
// This includes values selected through modifiers, order, etc.
func (m *Member) Value(name string) (ent.Value, error) {
	return m.selectValues.Get(name)
}

// QueryGuild queries the "guild" edge of the Member entity.
func (m *Member) QueryGuild() *GuildQuery {
	return NewMemberClient(m.config).QueryGuild(m)
}

// QueryUser queries the "user" edge of the Member entity.
func (m *Member) QueryUser() *UserQuery {
	return NewMemberClient(m.config).QueryUser(m)
}

// Update returns a builder for updating this Member.
// Note that you need to call Member.Unwrap() before calling this method if this Member
// was returned from a transaction, and the transaction was committed or rolled back.
func (m *Member) Update() *MemberUpdateOne {
	return NewMemberClient(m.config).UpdateOne(m)
}

// Unwrap unwraps the Member entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (m *Member) Unwrap() *Member {
	_tx, ok := m.config.driver.(*txDriver)
	if !ok {
		panic("ent: Member is not a transactional entity")
	}
	m.config.driver = _tx.drv
	return m
}

// String implements the fmt.Stringer.
func (m *Member) String() string {
	var builder strings.Builder
	builder.WriteString("Member(")
	builder.WriteString(fmt.Sprintf("id=%v, ", m.ID))
	builder.WriteString("permission=")
	builder.WriteString(fmt.Sprintf("%v", m.Permission))
	builder.WriteByte(')')
	return builder.String()
}

// Members is a parsable slice of Member.
type Members []*Member
