package db

import (
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
)

type TicketPanel struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Emoji       discord.ComponentEmoji `json:"emoji"`
	Label       string                 `json:"label"`
	PlaceHolder string                 `json:"placeholder"`
	Addons      []TicketAddon          `json:"addons"`
}
