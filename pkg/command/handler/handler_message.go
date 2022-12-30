package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command"
	"github.com/ikafly144/gobot/pkg/product"
)

var (
	messageComponentHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string){
		product.CommandPanelRole: func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.ComponentPanelRole(s, i)
		},
		product.CommandPanelAdd: func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.ComponentPanelRoleAdd(s, i, sessionID)
		},
		product.CommandPanelRoleCreate: func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.ComponentPanelRoleCreate(s, i, sessionID)
		},
		product.CommandPanelMinecraft: func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.ComponentPanelMinecraft(s, i)
		},
	}
)

func MessageComponentHandler() map[string]func(*discordgo.Session, *discordgo.InteractionCreate, string) {
	return messageComponentHandler
}
