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
	"github.com/sabafly/gobot/ent/thread1000"
	"github.com/sabafly/gobot/ent/thread1000channel"
)

// Thread1000 is the model entity for the Thread1000 schema.
type Thread1000 struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// MessageCount holds the value of the "message_count" field.
	MessageCount int `json:"message_count,omitempty"`
	// IsArchived holds the value of the "is_archived" field.
	IsArchived bool `json:"is_archived,omitempty"`
	// ThreadID holds the value of the "thread_id" field.
	ThreadID snowflake.ID `json:"thread_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the Thread1000Query when eager-loading is set.
	Edges                     Thread1000Edges `json:"edges"`
	guild_threads1000         *snowflake.ID
	thread1000channel_threads *uuid.UUID
	selectValues              sql.SelectValues
}

// Thread1000Edges holds the relations/edges for other nodes in the graph.
type Thread1000Edges struct {
	// Guild holds the value of the guild edge.
	Guild *Guild `json:"guild,omitempty"`
	// Channel holds the value of the channel edge.
	Channel *Thread1000Channel `json:"channel,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// GuildOrErr returns the Guild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e Thread1000Edges) GuildOrErr() (*Guild, error) {
	if e.Guild != nil {
		return e.Guild, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: guild.Label}
	}
	return nil, &NotLoadedError{edge: "guild"}
}

// ChannelOrErr returns the Channel value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e Thread1000Edges) ChannelOrErr() (*Thread1000Channel, error) {
	if e.Channel != nil {
		return e.Channel, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: thread1000channel.Label}
	}
	return nil, &NotLoadedError{edge: "channel"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Thread1000) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case thread1000.FieldIsArchived:
			values[i] = new(sql.NullBool)
		case thread1000.FieldMessageCount, thread1000.FieldThreadID:
			values[i] = new(sql.NullInt64)
		case thread1000.FieldName:
			values[i] = new(sql.NullString)
		case thread1000.FieldID:
			values[i] = new(uuid.UUID)
		case thread1000.ForeignKeys[0]: // guild_threads1000
			values[i] = new(sql.NullInt64)
		case thread1000.ForeignKeys[1]: // thread1000channel_threads
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Thread1000 fields.
func (t *Thread1000) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case thread1000.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				t.ID = *value
			}
		case thread1000.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				t.Name = value.String
			}
		case thread1000.FieldMessageCount:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field message_count", values[i])
			} else if value.Valid {
				t.MessageCount = int(value.Int64)
			}
		case thread1000.FieldIsArchived:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field is_archived", values[i])
			} else if value.Valid {
				t.IsArchived = value.Bool
			}
		case thread1000.FieldThreadID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field thread_id", values[i])
			} else if value.Valid {
				t.ThreadID = snowflake.ID(value.Int64)
			}
		case thread1000.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field guild_threads1000", values[i])
			} else if value.Valid {
				t.guild_threads1000 = new(snowflake.ID)
				*t.guild_threads1000 = snowflake.ID(value.Int64)
			}
		case thread1000.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field thread1000channel_threads", values[i])
			} else if value.Valid {
				t.thread1000channel_threads = new(uuid.UUID)
				*t.thread1000channel_threads = *value.S.(*uuid.UUID)
			}
		default:
			t.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Thread1000.
// This includes values selected through modifiers, order, etc.
func (t *Thread1000) Value(name string) (ent.Value, error) {
	return t.selectValues.Get(name)
}

// QueryGuild queries the "guild" edge of the Thread1000 entity.
func (t *Thread1000) QueryGuild() *GuildQuery {
	return NewThread1000Client(t.config).QueryGuild(t)
}

// QueryChannel queries the "channel" edge of the Thread1000 entity.
func (t *Thread1000) QueryChannel() *Thread1000ChannelQuery {
	return NewThread1000Client(t.config).QueryChannel(t)
}

// Update returns a builder for updating this Thread1000.
// Note that you need to call Thread1000.Unwrap() before calling this method if this Thread1000
// was returned from a transaction, and the transaction was committed or rolled back.
func (t *Thread1000) Update() *Thread1000UpdateOne {
	return NewThread1000Client(t.config).UpdateOne(t)
}

// Unwrap unwraps the Thread1000 entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (t *Thread1000) Unwrap() *Thread1000 {
	_tx, ok := t.config.driver.(*txDriver)
	if !ok {
		panic("ent: Thread1000 is not a transactional entity")
	}
	t.config.driver = _tx.drv
	return t
}

// String implements the fmt.Stringer.
func (t *Thread1000) String() string {
	var builder strings.Builder
	builder.WriteString("Thread1000(")
	builder.WriteString(fmt.Sprintf("id=%v, ", t.ID))
	builder.WriteString("name=")
	builder.WriteString(t.Name)
	builder.WriteString(", ")
	builder.WriteString("message_count=")
	builder.WriteString(fmt.Sprintf("%v", t.MessageCount))
	builder.WriteString(", ")
	builder.WriteString("is_archived=")
	builder.WriteString(fmt.Sprintf("%v", t.IsArchived))
	builder.WriteString(", ")
	builder.WriteString("thread_id=")
	builder.WriteString(fmt.Sprintf("%v", t.ThreadID))
	builder.WriteByte(')')
	return builder.String()
}

// Thread1000s is a parsable slice of Thread1000.
type Thread1000s []*Thread1000