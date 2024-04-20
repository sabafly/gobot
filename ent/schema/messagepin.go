package schema

import (
	"encoding/json"
	"github.com/sabafly/gobot/internal/uuidv7"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// MessagePin holds the schema definition for the MessagePin entity.
type MessagePin struct {
	ent.Schema
}

// Fields of the MessagePin.
func (MessagePin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuidv7.New()).
			Unique().
			Immutable().
			Default(uuidv7.New),
		field.Uint64("channel_id").
			Unique().
			GoType(snowflake.ID(0)),
		field.String("content").
			Optional(),
		field.JSON("embeds", []discord.Embed{}).
			Optional(),
		field.Uint64("before_id").
			Optional().
			Nillable().
			GoType(snowflake.ID(0)),
		field.JSON("rate_limit", RateLimit{}).Default(RateLimit{limit: []time.Time{}}),
	}
}

type RateLimit struct {
	limit []time.Time
}

func (r RateLimit) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.limit)
}

func (r *RateLimit) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &r.limit)
}

func (r *RateLimit) CheckLimit() bool {
	if !((len(r.limit) < 3 || time.Since(r.limit[2]) >= time.Second*5) && (len(r.limit) < 10 || time.Since(r.limit[9]) >= time.Second*30)) {
		return false
	}
	r.limit = append([]time.Time{time.Now()}, r.limit[0:min(10, len(r.limit))]...)
	ok := (len(r.limit) < 3 || time.Since(r.limit[2]) >= time.Second*5) && (len(r.limit) < 10 || time.Since(r.limit[9]) >= time.Second*30)
	return ok
}

// Edges of the MessagePin.
func (MessagePin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("guild", Guild.Type).
			Ref("message_pins").
			Required().
			Unique(),
	}
}
