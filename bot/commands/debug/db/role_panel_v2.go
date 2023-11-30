package db

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RolePanelV2DB interface {
	Get(id uuid.UUID) (*RolePanelV2, error)
	Set(id uuid.UUID, data *RolePanelV2) error
	Del(id uuid.UUID) error
}

type rolePanelV2DBImpl struct {
	db *redis.Client
}

func (r *rolePanelV2DBImpl) Get(id uuid.UUID) (*RolePanelV2, error) {
	res := r.db.HGet(context.TODO(), "role-panel-v2", id.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	data := &RolePanelV2{}
	if err := json.Unmarshal([]byte(res.Val()), data); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *rolePanelV2DBImpl) Set(id uuid.UUID, data *RolePanelV2) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := r.db.HSet(context.TODO(), "role-panel-v2", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (r *rolePanelV2DBImpl) Del(id uuid.UUID) error {
	res := r.db.HDel(context.TODO(), "role-panel-v2", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewRolePanelV2(name, description string) *RolePanelV2 {
	return &RolePanelV2{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Roles:       []RolePanelV2Role{},
	}
}

type RolePanelV2 struct {
	ID          uuid.UUID         `json:"uuid"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Roles       []RolePanelV2Role `json:"roles"`
}

type rolePanelV2MessageBuilder[T any] interface {
	AddEmbeds(...discord.Embed) T
	AddContainerComponents(...discord.ContainerComponent) T
}

func NewRolePanelV2Config() RolePanelV2Config {
	return RolePanelV2Config{
		PanelType:        RolePanelV2TypeNone,
		ButtonStyle:      discord.ButtonStyleSuccess,
		ButtonShowName:   false,
		SimpleSelectMenu: true,
	}
}

type RolePanelV2Config struct {
	PanelType        RolePanelV2Type     `json:"panel_type"`
	ButtonStyle      discord.ButtonStyle `json:"button_style"`
	ButtonShowName   bool                `json:"show_name"`
	SimpleSelectMenu bool                `json:"simple_select_menu"`
	HideNotice       bool                `json:"hide_notice"`
	UseDisplayName   bool                `json:"use_display_name"`
}

type RolePanelV2Role struct {
	RoleID   snowflake.ID            `json:"role_id"`
	RoleName string                  `json:"role_name"`
	Emoji    *discord.ComponentEmoji `json:"emoji"`
}
