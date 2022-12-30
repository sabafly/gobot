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
	"encoding/json"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/translate"
)

func ComponentPanelRole(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	component := i.Message.Components
	var content string
	bytes, _ := component[0].MarshalJSON()
	gid := i.GuildID
	uid := i.Member.User.ID
	if component[0].Type() == discordgo.ActionsRowComponent {
		data := &discordgo.ActionsRow{}
		json.Unmarshal(bytes, data)
		bytes, _ := data.Components[0].MarshalJSON()
		if data.Components[0].Type() == discordgo.SelectMenuComponent {
			data := &discordgo.SelectMenu{}
			json.Unmarshal(bytes, data)
			for _, v := range data.Options {
				t := true
				for _, v2 := range i.MessageComponentData().Values {
					if v2 == v.Value {
						t = false
					}
				}
				if t {
					for _, v2 := range i.Member.Roles {
						if v.Value == v2 {
							s.GuildMemberRoleRemove(gid, uid, v.Value)
							content += translate.Message(i.Locale, "panel_role_message_removed") + "<@&" + v.Value + ">\r"
						}
					}
				}
			}
			for _, r := range i.MessageComponentData().Values {
				t := true
				for _, m := range i.Member.Roles {
					if r == m {
						t = false
					}
				}
				if t {
					s.GuildMemberRoleAdd(gid, uid, r)
					content += translate.Message(i.Locale, "panel_role_message_added") + " <@&" + r + ">\r"
				}
			}
		}
	}
	if content != "" {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
	} else {
		s.InteractionResponseDelete(i.Interaction)
	}
}
