/*
	Copyright (C) 2022  ikafly144

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/interaction"
	"github.com/ikafly144/gobot/pkg/product"
)

var (
	modalSubmitHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string){
		product.CommandPanelMinecraftAddModal: func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			interaction.ModalMinecraftPanel(s, i, sessionID)
		},
		product.CommandPanelVoteCreateAddModal: func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			interaction.ModalPanelVoteCreateAdd(s, i, sessionID)
		},
	}
)

func ModalSubmitHandler() map[string]func(*discordgo.Session, *discordgo.InteractionCreate, string) {
	return modalSubmitHandler
}
