package db

import (
	"encoding/json"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
)

type Ticket struct {
	id       uuid.UUID
	guildID  snowflake.ID
	isClosed bool
	author   discord.User
	Subject  string  `json:"subject"`
	Content  *string `json:"content,omitempty"`
}

func (t Ticket) MarshalJSON() ([]byte, error) {
	v := struct {
		ID       uuid.UUID    `json:"id"`
		GuildID  snowflake.ID `json:"guild_id"`
		IsClosed bool         `json:"is_closed"`
		Author   discord.User `json:"author"`
		Ticket
	}{
		ID:       t.id,
		GuildID:  t.guildID,
		IsClosed: t.isClosed,
		Author:   t.author,
		Ticket:   t,
	}
	return json.Marshal(v)
}

func (t *Ticket) UnmarshalJSON(b []byte) error {
	v := struct {
		ID       uuid.UUID    `json:"id"`
		GuildID  snowflake.ID `json:"guild_id"`
		IsClosed bool         `json:"is_closed"`
		Author   discord.User `json:"author"`
		Ticket
	}{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*t = v.Ticket
	t.id = v.ID
	t.guildID = v.GuildID
	t.isClosed = v.IsClosed
	t.author = v.Author
	return nil
}

func (t Ticket) ID() uuid.UUID         { return t.id }
func (t Ticket) GuildID() snowflake.ID { return t.guildID }
func (t Ticket) IsClosed() bool        { return t.isClosed }
func (t Ticket) Author() discord.User  { return t.author }

type TicketAddon interface {
	Type() TicketAddonType
}

type TicketAddonType int

const (
	TicketAddonTypeTask = iota + 1
	TicketAddonTypeDiscussion
	TicketAddonTypeParliament
)

type TaskTicketAddon struct {
	Deadline time.Time `json:"deadline"`
	Routine  struct {
		Weekday  time.Weekday `json:"weekday"`
		Interval int          `json:"interval"`
	}
}

func (t TaskTicketAddon) Type() TicketAddonType { return TicketAddonTypeTask }
