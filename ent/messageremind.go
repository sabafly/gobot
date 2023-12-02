// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/messageremind"
)

// MessageRemind is the model entity for the MessageRemind schema.
type MessageRemind struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// ChannelID holds the value of the "channel_id" field.
	ChannelID snowflake.ID `json:"channel_id,omitempty"`
	// AuthorID holds the value of the "author_id" field.
	AuthorID snowflake.ID `json:"author_id,omitempty"`
	// Time holds the value of the "time" field.
	Time time.Time `json:"time,omitempty"`
	// Content holds the value of the "content" field.
	Content string `json:"content,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the MessageRemindQuery when eager-loading is set.
	Edges         MessageRemindEdges `json:"edges"`
	guild_reminds *snowflake.ID
	selectValues  sql.SelectValues
}

// MessageRemindEdges holds the relations/edges for other nodes in the graph.
type MessageRemindEdges struct {
	// Guild holds the value of the guild edge.
	Guild *Guild `json:"guild,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// GuildOrErr returns the Guild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e MessageRemindEdges) GuildOrErr() (*Guild, error) {
	if e.loadedTypes[0] {
		if e.Guild == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: guild.Label}
		}
		return e.Guild, nil
	}
	return nil, &NotLoadedError{edge: "guild"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*MessageRemind) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case messageremind.FieldChannelID, messageremind.FieldAuthorID:
			values[i] = new(sql.NullInt64)
		case messageremind.FieldContent:
			values[i] = new(sql.NullString)
		case messageremind.FieldTime:
			values[i] = new(sql.NullTime)
		case messageremind.FieldID:
			values[i] = new(uuid.UUID)
		case messageremind.ForeignKeys[0]: // guild_reminds
			values[i] = new(sql.NullInt64)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the MessageRemind fields.
func (mr *MessageRemind) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case messageremind.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				mr.ID = *value
			}
		case messageremind.FieldChannelID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field channel_id", values[i])
			} else if value.Valid {
				mr.ChannelID = snowflake.ID(value.Int64)
			}
		case messageremind.FieldAuthorID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field author_id", values[i])
			} else if value.Valid {
				mr.AuthorID = snowflake.ID(value.Int64)
			}
		case messageremind.FieldTime:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field time", values[i])
			} else if value.Valid {
				mr.Time = value.Time
			}
		case messageremind.FieldContent:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field content", values[i])
			} else if value.Valid {
				mr.Content = value.String
			}
		case messageremind.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field guild_reminds", values[i])
			} else if value.Valid {
				mr.guild_reminds = new(snowflake.ID)
				*mr.guild_reminds = snowflake.ID(value.Int64)
			}
		default:
			mr.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the MessageRemind.
// This includes values selected through modifiers, order, etc.
func (mr *MessageRemind) Value(name string) (ent.Value, error) {
	return mr.selectValues.Get(name)
}

// QueryGuild queries the "guild" edge of the MessageRemind entity.
func (mr *MessageRemind) QueryGuild() *GuildQuery {
	return NewMessageRemindClient(mr.config).QueryGuild(mr)
}

// Update returns a builder for updating this MessageRemind.
// Note that you need to call MessageRemind.Unwrap() before calling this method if this MessageRemind
// was returned from a transaction, and the transaction was committed or rolled back.
func (mr *MessageRemind) Update() *MessageRemindUpdateOne {
	return NewMessageRemindClient(mr.config).UpdateOne(mr)
}

// Unwrap unwraps the MessageRemind entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (mr *MessageRemind) Unwrap() *MessageRemind {
	_tx, ok := mr.config.driver.(*txDriver)
	if !ok {
		panic("ent: MessageRemind is not a transactional entity")
	}
	mr.config.driver = _tx.drv
	return mr
}

// String implements the fmt.Stringer.
func (mr *MessageRemind) String() string {
	var builder strings.Builder
	builder.WriteString("MessageRemind(")
	builder.WriteString(fmt.Sprintf("id=%v, ", mr.ID))
	builder.WriteString("channel_id=")
	builder.WriteString(fmt.Sprintf("%v", mr.ChannelID))
	builder.WriteString(", ")
	builder.WriteString("author_id=")
	builder.WriteString(fmt.Sprintf("%v", mr.AuthorID))
	builder.WriteString(", ")
	builder.WriteString("time=")
	builder.WriteString(mr.Time.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("content=")
	builder.WriteString(mr.Content)
	builder.WriteByte(')')
	return builder.String()
}

// MessageReminds is a parsable slice of MessageRemind.
type MessageReminds []*MessageRemind
