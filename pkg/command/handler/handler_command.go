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
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command"
)

var (
	commandHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			contents := map[discordgo.Locale]string{
				discordgo.Japanese: "ポング！\r" + s.HeartbeatLatency().String(),
			}
			content := "pong!\r" + s.HeartbeatLatency().String()
			if c, ok := contents[i.Locale]; ok {
				content = c
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
			if err != nil {
				log.Printf("例外: %v", err)
			}
		},
		"admin": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.Admin(s, i)
		},
		"panel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.Panel(s, i)
		},
		"tracker": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.Feed(s, i)
		},
		"role": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.Role(s, i)
		},
		"modify": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.MessageModify(s, i)
		},
		"select": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.MessageSelect(s, i)
		},
		"info": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.UserInfo(s, i)
		},
	}
)

func CommandHandler() map[string]func(*discordgo.Session, *discordgo.InteractionCreate) {
	return commandHandler
}
