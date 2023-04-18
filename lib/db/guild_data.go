package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type GuildDataDB interface {
	Get(snowflake.ID) (GuildData, error)
	Set(snowflake.ID, GuildData) error
	Remove(snowflake.ID) error
}

type guildDataDBImpl struct {
	db *redis.Client
}

func (g *guildDataDBImpl) Get(id snowflake.ID) (GuildData, error) {
	res := g.db.HGet(context.TODO(), "guild-data", id.String())
	if err := res.Err(); err != nil {
		return GuildData{}, err
	}
	buf := []byte(res.Val())
	val := GuildData{}
	if err := json.Unmarshal(buf, &val); err != nil {
		return GuildData{}, err
	}
	return val, nil
}

func (g *guildDataDBImpl) Set(id snowflake.ID, data GuildData) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := g.db.HSet(context.TODO(), "guild-data", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (g *guildDataDBImpl) Remove(id snowflake.ID) error {
	res := g.db.HDel(context.TODO(), "guild-data", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewGuildData(id snowflake.ID) GuildData {
	return GuildData{
		ID:             id,
		RolePanel:      make(map[uuid.UUID]GuildDataRolePanel),
		RolePanelLimit: 10,
		Member:         make(map[snowflake.ID]GuildDataMember),
	}
}

type GuildData struct {
	ID             snowflake.ID                     `json:"id"`
	RolePanel      map[uuid.UUID]GuildDataRolePanel `json:"role_panel"`
	RolePanelLimit int                              `json:"role_panel_limit"`
	Member         map[snowflake.ID]GuildDataMember `json:"message"`
}

type GuildDataMember struct {
	LastMessageID snowflake.ID `json:"last_message_id"`
	LastMessage   time.Time    `json:"last_message"`
	LastVoice     time.Time    `json:"last_voice"`
}

type GuildDataRolePanel struct {
	OnList bool `json:"on_list"`
}
