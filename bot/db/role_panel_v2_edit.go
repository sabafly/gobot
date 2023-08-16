package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-lib/v2/handler/interactions"
)

type RolePanelV2EditDB interface {
	Get(id uuid.UUID) (data *RolePanelV2Edit, err error)
	Set(id uuid.UUID, data *RolePanelV2Edit) (err error)
	Del(id uuid.UUID) (err error)
}

type rolePanelV2EditDBImpl struct {
	db *redis.Client
}

func (self rolePanelV2EditDBImpl) Get(id uuid.UUID) (*RolePanelV2Edit, error) {
	res := self.db.Get(context.TODO(), "role-panel-v2-edit"+id.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	data := &RolePanelV2Edit{}
	if err := json.Unmarshal([]byte(res.Val()), data); err != nil {
		return nil, err
	}
	return data, nil
}

func (self rolePanelV2EditDBImpl) Set(id uuid.UUID, data *RolePanelV2Edit) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := self.db.Set(context.TODO(), "role-panel-v2-edit"+id.String(), buf, time.Minute*15)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (self rolePanelV2EditDBImpl) Del(id uuid.UUID) error {
	res := self.db.Del(context.TODO(), "role-panel-v2-edit"+id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewRolePanelV2Edit(rolePanelID uuid.UUID, guildID, channelID snowflake.ID, token interactions.Token) *RolePanelV2Edit {
	return &RolePanelV2Edit{
		ID:               uuid.New(),
		CreatedAt:        time.Now(),
		GuildID:          guildID,
		ChannelID:        channelID,
		InteractionToken: token,
		RolePanelID:      rolePanelID,
	}
}

type RolePanelV2Edit struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	RolePanelID uuid.UUID `json:"role_panel_id"`

	GuildID          snowflake.ID       `json:"guild_id"`
	ChannelID        snowflake.ID       `json:"channel_id"`
	MessageID        snowflake.ID       `json:"message_id"`
	InteractionToken interactions.Token `json:"interaction_token"`

	SelectedID *snowflake.ID
}

func (r RolePanelV2Edit) IsSelected(id snowflake.ID) bool {
	return r.SelectedID != nil && *r.SelectedID == id
}

func (r RolePanelV2Edit) HasSelectedRole() bool {
	return r.SelectedID != nil
}
