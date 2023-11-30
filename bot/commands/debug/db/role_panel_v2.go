package db

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type RolePanelV2 struct {
	ID          uuid.UUID         `json:"uuid"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Roles       []RolePanelV2Role `json:"roles"`
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
