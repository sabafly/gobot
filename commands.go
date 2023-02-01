/*
	Copyright (C) 2022-2023  ikafly144

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
package main

import (
	"github.com/bwmarrin/discordgo"
	gobot "github.com/sabafly/gobot/pkg/bot"
)

func commands() gobot.ApplicationCommands {
	return gobot.ApplicationCommands{
		{
			ApplicationCommand: &discordgo.ApplicationCommand{
				Name:        "ping",
				Description: "pong!",
			},
			Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: setEmbedProperties([]*discordgo.MessageEmbed{
							{
								Title: "🏓 pong!",
								Fields: []*discordgo.MessageEmbedField{
									{
										Name:  "Discord API",
										Value: s.HeartbeatLatency().String(),
									},
								},
							},
						}),
					},
				})
			},
		},
	}
}
