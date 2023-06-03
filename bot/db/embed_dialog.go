package db

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/google/uuid"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func NewEmbedDialog() *EmbedDialog {
	return &EmbedDialog{}
}

type EmbedDialog struct {
	ID     uuid.UUID      `json:"id"`
	Locale discord.Locale `json:"locale"`
	Embed  discord.Embed  `json:"embed"`
}

func (e EmbedDialog) Build() discord.Embed {
	return e.Embed
}

func (e EmbedDialog) BaseMenu(btn discord.ButtonComponent) (mes discord.MessageCreate) {
	mes.Embeds = append(mes.Embeds, e.Embed)
	mes.Embeds = append(mes.Embeds,
		discord.Embed{
			Author: &discord.EmbedAuthor{
				Name: translate.Message(e.Locale, "embed_dialog_base_menu_title"),
			},
		},
	)
	mes.Embeds = botlib.SetEmbedsProperties(mes.Embeds)
	return
}

// func (e EmbedDialog) TitleDescriptionMenu() (mes discord.MessageCreate)
