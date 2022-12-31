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
package interaction

import (
	"sort"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/util"
)

func ComponentPanelRoleCreate(s *discordgo.Session, i *discordgo.InteractionCreate, id string) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	title := i.Message.Embeds[0].Title
	description := i.Message.Embeds[0].Description
	gid := i.GuildID
	cid := i.ChannelID
	var unused string
	rv := i.Interaction.MessageComponentData().Resolved.Roles
	me, _ := util.ErrorCatch(s.GuildMember(i.GuildID, s.State.User.ID))
	var highestPosition int
	for _, v := range me.Roles {
		r, _ := s.State.Role(i.GuildID, v)
		if r.Position > highestPosition {
			highestPosition = r.Position
		}
	}
	roles := []discordgo.Role{}
	for _, role := range rv {
		if role.Position < highestPosition && !role.Managed && role.ID != gid {
			roles = append(roles, *role)
		} else {
			unused += role.Mention() + " "
		}
	}
	if len(roles) == 0 {
		embeds := translate.ErrorEmbed(i.Locale, "error_invalid_roles")
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embeds,
		}))
		return
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position > roles[j].Position
	})
	options := []discordgo.SelectMenuOption{}
	for n, r := range roles {
		options = append(options, discordgo.SelectMenuOption{
			Label: r.Name,
			Value: r.ID,
			Emoji: discordgo.ComponentEmoji{
				ID:   "",
				Name: util.ToEmojiA(n + 1),
			},
		})
	}
	var fields string
	for n, r := range roles {
		fields += util.ToEmojiA(n+1) + " | " + r.Mention() + "\r"
	}
	zero := 0
	content := discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       title,
				Description: description,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "roles",
						Value: fields,
					},
				},
			},
		},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:  product.CommandPanelRole,
						MinValues: &zero,
						MaxValues: len(options),
						Options:   options,
					},
				},
			},
		},
	}
	var embed []*discordgo.MessageEmbed
	if unused != "" {
		embed = append(embed, &discordgo.MessageEmbed{
			Title:       translate.Message(i.Locale, "error_cannot_use_roles"),
			Description: unused,
		})
	}
	util.ErrorCatch(s.ChannelMessageSendComplex(cid, &content))
	str := translate.Message(i.Locale, "command_panel_option_role_message")
	util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
		Embeds:  &embed,
	}))
	i2, _ := util.ErrorCatch(session.InteractionLoad(id))
	util.ErrorCatch("", s.InteractionResponseDelete(i2.Data().Interaction))
	session.InteractionRemove(id)
}
