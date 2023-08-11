package db

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-lib/v2/handler/interactions"
)

func NewRolePanelV2Edit(guildID, channelID, MessageID snowflake.ID, token interactions.Token) RolePanelV2Edit {
	return RolePanelV2Edit{
		ID:               uuid.New(),
		CreatedAt:        time.Now(),
		GuildID:          guildID,
		ChannelID:        channelID,
		MessageID:        MessageID,
		InteractionToken: token,
	}
}

type RolePanelV2Edit struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	GuildID          snowflake.ID       `json:"guild_id"`
	ChannelID        snowflake.ID       `json:"channel_id"`
	MessageID        snowflake.ID       `json:"message_id"`
	InteractionToken interactions.Token `json:"interaction_token"`

	SelectedID *snowflake.ID
}

func (r RolePanelV2Edit) IsSelected(id snowflake.ID) bool {
	return r.SelectedID != nil && *r.SelectedID == id
}
