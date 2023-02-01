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
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sabafly/gobot/pkg/lib/constants"
	"github.com/sabafly/gobot/pkg/lib/translate"
)

func setEmbedProperties(embeds []*discordgo.MessageEmbed) []*discordgo.MessageEmbed {
	for i := range embeds {
		if embeds[i].Color == 0 {
			embeds[i].Color = constants.Color
		}
		if i == len(embeds)-1 {
			embeds[i].Footer = &discordgo.MessageEmbedFooter{
				Text: constants.BotName,
			}
			embeds[i].Timestamp = time.Now().Format(time.RFC3339)
		}
	}
	return embeds
}

func ErrorTraceEmbed(locale discordgo.Locale, err error) []*discordgo.MessageEmbed {
	stack := debug.Stack()
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "💥" + translate.Message(locale, "error_occurred_embed_message"),
			Description: "```" + string(stack) + "```",
			Color:       0xff0000,
		},
	}
	embeds = setEmbedProperties(embeds)
	return embeds
}

func ErrorRespond(i *discordgo.InteractionCreate, err error) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: ErrorTraceEmbed(i.Locale, err),
		},
	}
}

func StatusString(status discordgo.Status) (str string) {
	switch status {
	case discordgo.StatusOnline:
		return "<:online:1055430359363354644>"
	case discordgo.StatusDoNotDisturb:
		return "<:dnd:1055434290629980220>"
	case discordgo.StatusIdle:
		return "<:idle:1055433789020586035> "
	case discordgo.StatusInvisible:
		return "<:offline:1055434315514785792>"
	case discordgo.StatusOffline:
		return "<:offline:1055434315514785792>"
	}
	return
}
