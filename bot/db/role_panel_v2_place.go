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

type RolePanelV2PlaceDB interface {
	Get(id uuid.UUID) (data *RolePanelV2Place, err error)
	Set(id uuid.UUID, data *RolePanelV2Place) (err error)
	Del(id uuid.UUID) (err error)
}

type rolePanelV2PlaceDBImpl struct {
	db *redis.Client
}

func (self rolePanelV2PlaceDBImpl) Get(id uuid.UUID) (*RolePanelV2Place, error) {
	res := self.db.Get(context.TODO(), "role-panel-v2-place"+id.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	data := &RolePanelV2Place{}
	if err := json.Unmarshal([]byte(res.Val()), data); err != nil {
		return nil, err
	}
	return data, nil
}

func (self rolePanelV2PlaceDBImpl) Set(id uuid.UUID, data *RolePanelV2Place) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := self.db.Set(context.TODO(), "role-panel-v2-place"+id.String(), buf, ((time.Minute * 15) - (time.Since(data.CreatedAt))))
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (self rolePanelV2PlaceDBImpl) Del(id uuid.UUID) error {
	res := self.db.Del(context.TODO(), id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewRolePanelV2Place(guild_id snowflake.ID, panel_id uuid.UUID, interaction_token interactions.Token) *RolePanelV2Place {
	return &RolePanelV2Place{
		ID:               uuid.New(),
		CreatedAt:        time.Now(),
		GuildID:          guild_id,
		PanelID:          panel_id,
		Config:           NewRolePanelV2Config(),
		InteractionToken: interaction_token,
	}
}

type RolePanelV2Place struct {
	ID               uuid.UUID          `json:"id"`
	CreatedAt        time.Time          `json:"created_at"`
	GuildID          snowflake.ID       `json:"guild_id"`
	PanelID          uuid.UUID          `json:"panel_id"`
	Config           RolePanelV2Config  `json:"config"`
	InteractionToken interactions.Token `json:"interaction_token"`
}

type RolePanelV2Type string

const (
	RolePanelV2TypeNone       = ""
	RolePanelV2TypeReaction   = "reaction"
	RolePanelV2TypeSelectMenu = "select_menu"
	RolePanelV2TypeButton     = "button"
)
