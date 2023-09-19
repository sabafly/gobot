package db

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type GuildTicketData struct {
	GuildID               snowflake.ID  `json:"id"`
	DefaultMessageChannel *snowflake.ID `json:"default_message_channel"`
	DefaultThreadChannel  *snowflake.ID `json:"default_thread_channel"`
	Templates             []uuid.UUID   `json:"templates"`
	Tickets               []uuid.UUID   `json:"tickets"`
}

func (g GuildTicketData) ID() snowflake.ID { return g.GuildID }

func (g GuildTicketData) ChannelID() (*snowflake.ID, bool) {
	if g.DefaultMessageChannel != nil {
		return g.DefaultMessageChannel, true
	} else {
		return nil, false
	}
}
