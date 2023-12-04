// Code generated by ent, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/schema"
)

// MessagePin is the model entity for the MessagePin schema.
type MessagePin struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// ChannelID holds the value of the "channel_id" field.
	ChannelID snowflake.ID `json:"channel_id,omitempty"`
	// Content holds the value of the "content" field.
	Content string `json:"content,omitempty"`
	// Embeds holds the value of the "embeds" field.
	Embeds []discord.Embed `json:"embeds,omitempty"`
	// BeforeID holds the value of the "before_id" field.
	BeforeID *snowflake.ID `json:"before_id,omitempty"`
	// RateLimit holds the value of the "rate_limit" field.
	RateLimit schema.RateLimit `json:"rate_limit,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the MessagePinQuery when eager-loading is set.
	Edges              MessagePinEdges `json:"edges"`
	guild_message_pins *snowflake.ID
	selectValues       sql.SelectValues
}

// MessagePinEdges holds the relations/edges for other nodes in the graph.
type MessagePinEdges struct {
	// Guild holds the value of the guild edge.
	Guild *Guild `json:"guild,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// GuildOrErr returns the Guild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e MessagePinEdges) GuildOrErr() (*Guild, error) {
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
func (*MessagePin) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case messagepin.FieldEmbeds, messagepin.FieldRateLimit:
			values[i] = new([]byte)
		case messagepin.FieldChannelID, messagepin.FieldBeforeID:
			values[i] = new(sql.NullInt64)
		case messagepin.FieldContent:
			values[i] = new(sql.NullString)
		case messagepin.FieldID:
			values[i] = new(uuid.UUID)
		case messagepin.ForeignKeys[0]: // guild_message_pins
			values[i] = new(sql.NullInt64)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the MessagePin fields.
func (mp *MessagePin) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case messagepin.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				mp.ID = *value
			}
		case messagepin.FieldChannelID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field channel_id", values[i])
			} else if value.Valid {
				mp.ChannelID = snowflake.ID(value.Int64)
			}
		case messagepin.FieldContent:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field content", values[i])
			} else if value.Valid {
				mp.Content = value.String
			}
		case messagepin.FieldEmbeds:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field embeds", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &mp.Embeds); err != nil {
					return fmt.Errorf("unmarshal field embeds: %w", err)
				}
			}
		case messagepin.FieldBeforeID:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field before_id", values[i])
			} else if value.Valid {
				mp.BeforeID = new(snowflake.ID)
				*mp.BeforeID = snowflake.ID(value.Int64)
			}
		case messagepin.FieldRateLimit:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field rate_limit", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &mp.RateLimit); err != nil {
					return fmt.Errorf("unmarshal field rate_limit: %w", err)
				}
			}
		case messagepin.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field guild_message_pins", values[i])
			} else if value.Valid {
				mp.guild_message_pins = new(snowflake.ID)
				*mp.guild_message_pins = snowflake.ID(value.Int64)
			}
		default:
			mp.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the MessagePin.
// This includes values selected through modifiers, order, etc.
func (mp *MessagePin) Value(name string) (ent.Value, error) {
	return mp.selectValues.Get(name)
}

// QueryGuild queries the "guild" edge of the MessagePin entity.
func (mp *MessagePin) QueryGuild() *GuildQuery {
	return NewMessagePinClient(mp.config).QueryGuild(mp)
}

// Update returns a builder for updating this MessagePin.
// Note that you need to call MessagePin.Unwrap() before calling this method if this MessagePin
// was returned from a transaction, and the transaction was committed or rolled back.
func (mp *MessagePin) Update() *MessagePinUpdateOne {
	return NewMessagePinClient(mp.config).UpdateOne(mp)
}

// Unwrap unwraps the MessagePin entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (mp *MessagePin) Unwrap() *MessagePin {
	_tx, ok := mp.config.driver.(*txDriver)
	if !ok {
		panic("ent: MessagePin is not a transactional entity")
	}
	mp.config.driver = _tx.drv
	return mp
}

// String implements the fmt.Stringer.
func (mp *MessagePin) String() string {
	var builder strings.Builder
	builder.WriteString("MessagePin(")
	builder.WriteString(fmt.Sprintf("id=%v, ", mp.ID))
	builder.WriteString("channel_id=")
	builder.WriteString(fmt.Sprintf("%v", mp.ChannelID))
	builder.WriteString(", ")
	builder.WriteString("content=")
	builder.WriteString(mp.Content)
	builder.WriteString(", ")
	builder.WriteString("embeds=")
	builder.WriteString(fmt.Sprintf("%v", mp.Embeds))
	builder.WriteString(", ")
	if v := mp.BeforeID; v != nil {
		builder.WriteString("before_id=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("rate_limit=")
	builder.WriteString(fmt.Sprintf("%v", mp.RateLimit))
	builder.WriteByte(')')
	return builder.String()
}

// MessagePins is a parsable slice of MessagePin.
type MessagePins []*MessagePin