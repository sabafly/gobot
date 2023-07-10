package db

import (
	"context"
	"encoding/json"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/sabafly/disgo/bot"
	"github.com/sabafly/disgo/discord"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
)

type MessagePinDB interface {
	Get(id snowflake.ID) (*GuildMessagePins, error)
	GetAll() (map[snowflake.ID]*GuildMessagePins, error)
	Set(id snowflake.ID, data *GuildMessagePins) error
	Del(id snowflake.ID) error
}

var _ MessagePinDB = (*messagePinDBImpl)(nil)

type messagePinDBImpl struct {
	db *redis.Client
}

func (m *messagePinDBImpl) Get(id snowflake.ID) (*GuildMessagePins, error) {
	res := m.db.HGet(context.TODO(), "message-pin", id.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	val := &GuildMessagePins{}
	if err := json.Unmarshal([]byte(res.Val()), val); err != nil {
		return nil, err
	}
	return val, nil
}

func (m *messagePinDBImpl) GetAll() (map[snowflake.ID]*GuildMessagePins, error) {
	res := m.db.HGetAll(context.TODO(), "message-pin")
	if err := res.Err(); err != nil {
		return nil, err
	}
	val := make(map[snowflake.ID]*GuildMessagePins)
	for k, v := range res.Val() {
		id, err := snowflake.Parse(k)
		if err != nil {
			return nil, err
		}
		data := &GuildMessagePins{}
		if err := json.Unmarshal([]byte(v), data); err != nil {
			return nil, err
		}
		val[id] = data
	}
	return val, nil
}

func (m *messagePinDBImpl) Set(id snowflake.ID, data *GuildMessagePins) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := m.db.HSet(context.TODO(), "message-pin", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (m *messagePinDBImpl) Del(id snowflake.ID) error {
	res := m.db.HDel(context.TODO(), "message-pin", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

type GuildMessagePins struct {
	Enabled bool                        `json:"enabled"`
	Pins    map[snowflake.ID]MessagePin `json:"pins"`
}

type MessagePin struct {
	WebhookMessageCreate discord.WebhookMessageCreate `json:"webhook_message_create"`
	ChannelID            snowflake.ID                 `json:"channel_id"`
	LastMessageID        *snowflake.ID                `json:"last_message_id"`
}

func (self *MessagePin) Update(client bot.Client) error {
	if self.LastMessageID != nil {
		go func() { _ = client.Rest().DeleteMessage(self.ChannelID, *self.LastMessageID) }()
	}
	m, err := botlib.SendWebhook(client, self.ChannelID, self.WebhookMessageCreate)
	if err != nil {
		return err
	}
	self.LastMessageID = &m.ID
	return nil
}

func NewMessagePin() *GuildMessagePins {
	return &GuildMessagePins{
		Enabled: true,
		Pins:    make(map[snowflake.ID]MessagePin),
	}
}
