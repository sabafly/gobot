package db

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type RolePanelV2EditDB interface {
	Get(id uuid.UUID) (data *RolePanelV2Edit, err error)
	Set(id uuid.UUID, data *RolePanelV2Edit) (err error)
	Del(id uuid.UUID) (err error)
}

type RolePanelV2Edit struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	RolePanelID uuid.UUID `json:"role_panel_id"`

	GuildID     snowflake.ID   `json:"guild_id"`
	ChannelID   snowflake.ID   `json:"channel_id"`
	MessageID   snowflake.ID   `json:"message_id"`
	EmojiMode   bool           `json:"emoji_mode"`
	EmojiLocale discord.Locale `json:"emoji_locale"`

	SelectedID *snowflake.ID
}

func (r RolePanelV2Edit) IsSelected(id snowflake.ID) bool {
	return r.SelectedID != nil && *r.SelectedID == id
}

func (r RolePanelV2Edit) HasSelectedRole() bool {
	return r.SelectedID != nil
}
