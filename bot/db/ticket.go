package db

import (
	"encoding/json"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
)

func NewTicket(guildID snowflake.ID, author discord.User) *Ticket {
	return &Ticket{
		id:       uuid.New(),
		guildID:  guildID,
		isClosed: false,
	}
}

type Ticket struct {
	id                   uuid.UUID
	guildID              snowflake.ID
	channelID, messageID snowflake.ID
	isClosed             bool
	author               discord.User
	subject              string
	content              string
	routine              *TicketPanelTaskAddonRoutine
	hasThread            bool
}

func (t Ticket) MarshalJSON() ([]byte, error) {
	v := struct {
		ID        uuid.UUID                    `json:"id"`
		GuildID   snowflake.ID                 `json:"guild_id"`
		ChannelID snowflake.ID                 `json:"channel_id"`
		MessageID snowflake.ID                 `json:"message_id"`
		IsClosed  bool                         `json:"is_closed"`
		Author    discord.User                 `json:"author"`
		Subject   string                       `json:"subject"`
		Content   string                       `json:"content,omitempty"`
		Routine   *TicketPanelTaskAddonRoutine `json:"routine,omitempty"`
		HasThread bool                         `json:"has_thread"`
	}{
		ID:        t.id,
		GuildID:   t.guildID,
		ChannelID: t.channelID,
		MessageID: t.messageID,
		IsClosed:  t.isClosed,
		Author:    t.author,
		Subject:   t.subject,
		Content:   t.content,
		Routine:   t.routine,
		HasThread: t.hasThread,
	}
	return json.Marshal(v)
}

func (t *Ticket) UnmarshalJSON(b []byte) error {
	v := struct {
		ID        uuid.UUID                    `json:"id"`
		GuildID   snowflake.ID                 `json:"guild_id"`
		ChannelID snowflake.ID                 `json:"channel_id"`
		MessageID snowflake.ID                 `json:"message_id"`
		IsClosed  bool                         `json:"is_closed"`
		Author    discord.User                 `json:"author"`
		Subject   string                       `json:"subject"`
		Content   string                       `json:"content,omitempty"`
		Routine   *TicketPanelTaskAddonRoutine `json:"routine,omitempty"`
		HasThread bool                         `json:"has_thread"`
	}{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	t.id = v.ID
	t.guildID = v.GuildID
	t.channelID = v.ChannelID
	t.messageID = v.MessageID
	t.isClosed = v.IsClosed
	t.author = v.Author
	t.subject = v.Subject
	t.content = v.Content
	t.routine = v.Routine
	t.hasThread = v.HasThread
	return nil
}

// 完璧で究極なゲッター！

func (t Ticket) ID() uuid.UUID         { return t.id }
func (t Ticket) GuildID() snowflake.ID { return t.guildID }
func (t Ticket) IsClosed() bool        { return t.isClosed }
func (t Ticket) Author() discord.User  { return t.author }
func (t Ticket) Subject() string       { return t.subject }
func (t Ticket) Content() string       { return t.content }
func (t Ticket) HasThread() bool       { return t.hasThread }

func (t *Ticket) SetSubject(s string) { t.subject = s }
func (t *Ticket) SetContent(s string) { t.content = s }
