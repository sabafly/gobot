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
	"github.com/sabafly/gobot/ent/thread1000channel"
)

// Thread1000Channel is the model entity for the Thread1000Channel schema.
type Thread1000Channel struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Name holds the value of the "name" field.
	Name *string `json:"name,omitempty"`
	// AnonymousName holds the value of the "anonymous_name" field.
	AnonymousName *string `json:"anonymous_name,omitempty"`
	// ChannelID holds the value of the "channel_id" field.
	ChannelID snowflake.ID `json:"channel_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the Thread1000ChannelQuery when eager-loading is set.
	Edges                     Thread1000ChannelEdges `json:"edges"`
	guild_thread1000_channels *snowflake.ID
	selectValues              sql.SelectValues
}

// Thread1000ChannelEdges holds the relations/edges for other nodes in the graph.
type Thread1000ChannelEdges struct {
	// Guild holds the value of the guild edge.
	Guild *Guild `json:"guild,omitempty"`
	// Threads holds the value of the threads edge.
	Threads []*Thread1000 `json:"threads,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// GuildOrErr returns the Guild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e Thread1000ChannelEdges) GuildOrErr() (*Guild, error) {
	if e.Guild != nil {
		return e.Guild, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: guild.Label}
	}
	return nil, &NotLoadedError{edge: "guild"}
}

// ThreadsOrErr returns the Threads value or an error if the edge
// was not loaded in eager-loading.
func (e Thread1000ChannelEdges) ThreadsOrErr() ([]*Thread1000, error) {
	if e.loadedTypes[1] {
		return e.Threads, nil
	}
	return nil, &NotLoadedError{edge: "threads"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Thread1000Channel) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case thread1000channel.FieldChannelID:
			values[i] = new(sql.NullInt64)
		case thread1000channel.FieldName, thread1000channel.FieldAnonymousName:
			values[i] = new(sql.NullString)
		case thread1000channel.FieldID:
			values[i] = new(uuid.UUID)
		case thread1000channel.ForeignKeys[0]: // guild_thread1000_channels
			values[i] = new(sql.NullInt64)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Thread1000Channel fields.
func (t *Thread1000Channel) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case thread1000channel.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				t.ID = *value
			}
		case thread1000channel.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				t.Name = new(string)
				*t.Name = value.String
			}
		case thread1000channel.FieldAnonymousName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field anonymous_name", values[i])
			} else if value.Valid {
				t.AnonymousName = new(string)
				*t.AnonymousName = value.String
			}
		case thread1000channel.FieldChannelID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field channel_id", values[i])
			} else if value.Valid {
				t.ChannelID = snowflake.ID(value.Int64)
			}
		case thread1000channel.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field guild_thread1000_channels", values[i])
			} else if value.Valid {
				t.guild_thread1000_channels = new(snowflake.ID)
				*t.guild_thread1000_channels = snowflake.ID(value.Int64)
			}
		default:
			t.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Thread1000Channel.
// This includes values selected through modifiers, order, etc.
func (t *Thread1000Channel) Value(name string) (ent.Value, error) {
	return t.selectValues.Get(name)
}

// QueryGuild queries the "guild" edge of the Thread1000Channel entity.
func (t *Thread1000Channel) QueryGuild() *GuildQuery {
	return NewThread1000ChannelClient(t.config).QueryGuild(t)
}

// QueryThreads queries the "threads" edge of the Thread1000Channel entity.
func (t *Thread1000Channel) QueryThreads() *Thread1000Query {
	return NewThread1000ChannelClient(t.config).QueryThreads(t)
}

// Update returns a builder for updating this Thread1000Channel.
// Note that you need to call Thread1000Channel.Unwrap() before calling this method if this Thread1000Channel
// was returned from a transaction, and the transaction was committed or rolled back.
func (t *Thread1000Channel) Update() *Thread1000ChannelUpdateOne {
	return NewThread1000ChannelClient(t.config).UpdateOne(t)
}

// Unwrap unwraps the Thread1000Channel entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (t *Thread1000Channel) Unwrap() *Thread1000Channel {
	_tx, ok := t.config.driver.(*txDriver)
	if !ok {
		panic("ent: Thread1000Channel is not a transactional entity")
	}
	t.config.driver = _tx.drv
	return t
}

// String implements the fmt.Stringer.
func (t *Thread1000Channel) String() string {
	var builder strings.Builder
	builder.WriteString("Thread1000Channel(")
	builder.WriteString(fmt.Sprintf("id=%v, ", t.ID))
	if v := t.Name; v != nil {
		builder.WriteString("name=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := t.AnonymousName; v != nil {
		builder.WriteString("anonymous_name=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("channel_id=")
	builder.WriteString(fmt.Sprintf("%v", t.ChannelID))
	builder.WriteByte(')')
	return builder.String()
}

// Thread1000Channels is a parsable slice of Thread1000Channel.
type Thread1000Channels []*Thread1000Channel
