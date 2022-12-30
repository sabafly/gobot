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
package command

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/util"
)

func Role(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "color":
		roleColor(s, i, options)
	}
}

func roleColor(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	options = options[0].Options
	var raw string
	var name string
	for _, o := range options {
		switch o.Name {
		case "rgb":
			raw = "0x" + o.StringValue()
		case "name":
			name = o.StringValue()
		}
	}
	col, err := strconv.ParseInt(raw, 0, 32)
	if name == "" {
		name = strconv.FormatInt(col, 16)
	}
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		})
		return
	}
	colInt := int(col)
	r, err := s.GuildRoleCreate(i.GuildID, &discordgo.RoleParams{
		Name:  name,
		Color: &colInt,
	})
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		})
		return
	}
	s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, r.ID)
	str := "OK " + r.Mention()
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
	})
}
