package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command"
	"github.com/ikafly144/gobot/pkg/product"
)

var (
	modalSubmitHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, mid string){
		product.CommandPanelMinecraftAddModal: func(s *discordgo.Session, i *discordgo.InteractionCreate, mid string) {
			command.ModalMinecraftPanel(s, i, mid)
		},
	}
)

func ModalSubmitHandler() map[string]func(*discordgo.Session, *discordgo.InteractionCreate, string) {
	return modalSubmitHandler
}
