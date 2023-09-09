package db

import (
	"encoding/json"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-disgo/discord"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
)

func NewTicket(guildID snowflake.ID, author discord.User) *Ticket {
	return &Ticket{
		id:       uuid.New(),
		guildID:  guildID,
		isClosed: false,
	}
}

type Ticket struct {
	id                      uuid.UUID
	guildID                 snowflake.ID
	channelID, messageID    snowflake.ID
	threadChannel, threadID *snowflake.ID
	isClosed                bool
	author                  discord.User
	subject                 string
	content                 string
	hasThread               bool
	template                *TicketTemplate
}

type TicketStatus int

const (
	TicketStatusCancel TicketStatus = iota
	TicketStatusDone
	TicketStatusOpen
)

func (t Ticket) MarshalJSON() ([]byte, error) {
	v := struct {
		ID            uuid.UUID       `json:"id"`
		GuildID       snowflake.ID    `json:"guild_id"`
		ChannelID     snowflake.ID    `json:"channel_id"`
		MessageID     snowflake.ID    `json:"message_id"`
		ThreadChannel *snowflake.ID   `json:"thread_channel"`
		ThreadID      *snowflake.ID   `json:"thread_id"`
		IsClosed      bool            `json:"is_closed"`
		Author        discord.User    `json:"author"`
		Subject       string          `json:"subject"`
		Content       string          `json:"content,omitempty"`
		HasThread     bool            `json:"has_thread"`
		Template      *TicketTemplate `json:"template,omitempty"`
	}{
		ID:            t.id,
		GuildID:       t.guildID,
		ChannelID:     t.channelID,
		MessageID:     t.messageID,
		ThreadChannel: t.threadChannel,
		ThreadID:      t.threadID,
		IsClosed:      t.isClosed,
		Author:        t.author,
		Subject:       t.subject,
		Content:       t.content,
		HasThread:     t.hasThread,
		Template:      t.template,
	}
	return json.Marshal(v)
}

func (t *Ticket) UnmarshalJSON(b []byte) error {
	v := struct {
		ID            uuid.UUID       `json:"id"`
		GuildID       snowflake.ID    `json:"guild_id"`
		ChannelID     snowflake.ID    `json:"channel_id"`
		MessageID     snowflake.ID    `json:"message_id"`
		ThreadChannel *snowflake.ID   `json:"thread_channel"`
		ThreadID      *snowflake.ID   `json:"thread_id"`
		IsClosed      bool            `json:"is_closed"`
		Author        discord.User    `json:"author"`
		Subject       string          `json:"subject"`
		Content       string          `json:"content,omitempty"`
		HasThread     bool            `json:"has_thread"`
		Template      *TicketTemplate `json:"template"`
	}{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	t.id = v.ID
	t.guildID = v.GuildID
	t.channelID = v.ChannelID
	t.messageID = v.MessageID
	t.threadChannel = v.ThreadChannel
	t.threadID = v.ThreadID
	t.isClosed = v.IsClosed
	t.author = v.Author
	t.subject = v.Subject
	t.content = v.Content
	t.hasThread = v.HasThread
	t.template = v.Template
	return nil
}

// 君は完璧で究極のゲッター！

// 明日の希望を取り戻そうぜ
// 強く今を生きる人間の腕に
// 赤き血潮が激しくうねる
// 正義の疾風が 荒れるぜゲッター
// 俺の嵐が巻き起こるとき
// 悪の炎なんて全て消すさ

func (t Ticket) ID() uuid.UUID                { return t.id }
func (t Ticket) GuildID() snowflake.ID        { return t.guildID }
func (t Ticket) ChannelID() snowflake.ID      { return t.channelID }
func (t Ticket) MessageID() snowflake.ID      { return t.messageID }
func (t Ticket) ThreadID() *snowflake.ID      { return t.threadID }
func (t Ticket) ThreadChannel() *snowflake.ID { return t.threadChannel }
func (t Ticket) IsClosed() bool               { return t.isClosed }
func (t Ticket) Author() discord.User         { return t.author }
func (t Ticket) Subject() string              { return t.subject }
func (t Ticket) Content() string              { return t.content }
func (t Ticket) HasThread() bool              { return t.hasThread }
func (t Ticket) Template() *TicketTemplate    { return t.template }

func (t *Ticket) SetSubject(s string)                     { t.subject = s }
func (t *Ticket) SetContent(s string)                     { t.content = s }
func (t *Ticket) SetChannelMessage(cid, mid snowflake.ID) { t.channelID, t.messageID = cid, mid }
func (t *Ticket) SetTemplate(template *TicketTemplate)    { t.template = template }
func (t *Ticket) SetHasThread(b bool)                     { t.hasThread = b }

type ticketMessageBuilder[T any] interface {
	SetContent(string) T
	AddEmbeds(...discord.Embed) T
	AddContainerComponents(...discord.ContainerComponent) T
}

func TicketMessage[T ticketMessageBuilder[T]](message T, ticket *Ticket) T {
	embed := discord.NewEmbedBuilder()
	embed.SetTitle(ticket.content)
	embed.SetDescription(ticket.subject)
	embed.SetAuthorName(ticket.author.EffectiveName())
	embed.SetAuthorIcon(ticket.author.EffectiveAvatarURL())
	embed.Embed = botlib.SetEmbedProperties(embed.Embed)
	message.AddEmbeds(embed.Build())
	return message
}
