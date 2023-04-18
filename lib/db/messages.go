package db

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
)

type MessageDB interface {
	Get(guildID snowflake.ID) (MessageGuild, error)
	Set(guildID snowflake.ID, messages MessageGuild) error
	Append(guildID snowflake.ID, message Message) error
	Remove(guildID snowflake.ID) error
}

type MessageDBImpl struct {
	db *redis.Client
	*sync.Mutex
}

func (m *MessageDBImpl) Get(guildID snowflake.ID) (*MessageGuild, error) {
	m.Lock()
	defer m.Unlock()
	res := m.db.HGet(context.TODO(), "message", guildID.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	data := &MessageGuild{}
	err := json.Unmarshal([]byte(res.Val()), data)
	if err != nil {
		return nil, err
	}
	if data.Members == nil {
		return nil, errors.New("nil map error")
	}
	return data, nil
}

func (m *MessageDBImpl) Set(guildID snowflake.ID, message MessageGuild) error {
	m.Lock()
	defer m.Unlock()
	buf, err := json.Marshal(message)
	if err != nil {
		return err
	}
	res := m.db.HSet(context.TODO(), "message", guildID.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (m *MessageDBImpl) Remove(guildID snowflake.ID) error {
	m.Lock()
	defer m.Unlock()
	res := m.db.HDel(context.TODO(), "message", guildID.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (m *MessageDBImpl) Append(guildID, userID snowflake.ID, message Message) error {
	m.Lock()
	defer m.Unlock()
	data, err := m.Get(guildID)
	if err != nil {
		return err
	}
	data.Members[userID] = append(data.Members[userID], message)
	if err := m.Set(guildID, *data); err != nil {
		return err
	}
	return nil
}

type MessageGuild struct {
	Members map[snowflake.ID][]Message `json:"members"`
}

type Message struct {
	ID        snowflake.ID     `json:"id"`
	ChannelID snowflake.ID     `json:"channel_id"`
	GuildID   snowflake.ID     `json:"guild_id"`
	Data      *discord.Message `json:"data"`
}
