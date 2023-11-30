package db

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type RolePanelV2Place struct {
	ID        uuid.UUID         `json:"id"`
	CreatedAt time.Time         `json:"created_at"`
	GuildID   snowflake.ID      `json:"guild_id"`
	PanelID   uuid.UUID         `json:"panel_id"`
	Config    RolePanelV2Config `json:"config"`
}

type RolePanelV2Type string

const (
	RolePanelV2TypeNone       = ""
	RolePanelV2TypeReaction   = "reaction"
	RolePanelV2TypeSelectMenu = "select_menu"
	RolePanelV2TypeButton     = "button"
)
