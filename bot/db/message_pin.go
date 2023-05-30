package db

import (
	"context"
	"encoding/json"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
)

type MessagePinDB interface {
	Get(id snowflake.ID) (MessagePin, error)
	Set(id snowflake.ID, data MessagePin) error
	Del(id snowflake.ID) error
}

var _ MessagePinDB = (*messagePinDBImpl)(nil)

type messagePinDBImpl struct {
	db *redis.Client
}

func (m *messagePinDBImpl) Get(id snowflake.ID) (MessagePin, error) {
	res := m.db.HGet(context.TODO(), "message-pin", id.String())
	if err := res.Err(); err != nil {
		return MessagePin{}, err
	}
	val := MessagePin{}
	if err := json.Unmarshal([]byte(res.Val()), &val); err != nil {
		return MessagePin{}, err
	}
	return val, nil
}

func (m *messagePinDBImpl) Set(id snowflake.ID, data MessagePin) error {
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

type MessagePin struct {
	Enabled bool                                   `json:"enabled"`
	Pins    map[snowflake.ID]discord.MessageCreate `json:"pins"`
}

func NewMessagePin() MessagePin {
	return MessagePin{
		Enabled: false,
		Pins:    make(map[snowflake.ID]discord.MessageCreate),
	}
}
