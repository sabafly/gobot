/*
	Copyright (C) 2022-2023  sabafly

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
package gobot

import (
	"time"

	"github.com/bwmarrin/discordgo"
	botlib "github.com/sabafly/gobot/lib/bot"
)

var DMPermission = false

func commands() botlib.ApplicationCommands {
	one := 1.0
	return botlib.ApplicationCommands{
		{
			ApplicationCommand: &discordgo.ApplicationCommand{
				Name:         "ping",
				Description:  "pong!",
				DMPermission: &DMPermission,
			},
			Handler: CommandTextPing,
		},
		{
			ApplicationCommand: &discordgo.ApplicationCommand{
				Name:         "vote",
				Description:  "manage votes",
				DMPermission: &DMPermission,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "create",
						Description: "create new vote",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandOption{
							{
								Name:        "title",
								Description: "vote title",
								Type:        discordgo.ApplicationCommandOptionString,
								Required:    true,
								MaxLength:   128,
							},
							{
								Name:        "description",
								Description: "vote description",
								Type:        discordgo.ApplicationCommandOptionString,
								Required:    true,
								MaxLength:   2048,
							},
							{
								Name:        "duration",
								Description: "vote duration",
								Type:        discordgo.ApplicationCommandOptionInteger,
								Required:    true,
								MaxValue:    100,
								MinValue:    &one,
							},
							{
								Name:        "unit",
								Description: "unit of vote duration",
								Type:        discordgo.ApplicationCommandOptionInteger,
								Required:    true,
								Choices: []*discordgo.ApplicationCommandOptionChoice{
									{
										Name:  "day",
										Value: time.Hour * 24,
									},
									{
										Name:  "hour",
										Value: time.Hour,
									},
									{
										Name:  "minute",
										Value: time.Minute,
									},
								},
							},
							{
								Name:        "min-number-limit",
								Description: "min choice of vote",
								Type:        discordgo.ApplicationCommandOptionInteger,
								MaxValue:    25,
								MinValue:    &one,
								Required:    true,
							},
							{
								Name:        "max-number-limit",
								Description: "max choice of vote",
								Type:        discordgo.ApplicationCommandOptionInteger,
								MaxValue:    25,
								MinValue:    &one,
								Required:    true,
							},
							{
								Name:        "show-vote",
								Description: "whether to display what the user is voting for",
								Type:        discordgo.ApplicationCommandOptionBoolean,
							},
						},
					},
				},
			},
			Handler: CommandTextVote,
		},
		{
			ApplicationCommand: &discordgo.ApplicationCommand{
				Name:         "info",
				Type:         discordgo.UserApplicationCommand,
				DMPermission: &DMPermission,
			},
			Handler: CommandUserInfo,
		},
	}
}
