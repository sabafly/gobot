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
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/util"
)

func MessageModify(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := &discordgo.ApplicationCommandInteractionData{}
	byte, _ := util.ErrorCatch(json.Marshal(i.Interaction.Data))
	util.ErrorCatch("", json.Unmarshal(byte, data))
	mes, _ := util.ErrorCatch(s.ChannelMessage(i.ChannelID, data.TargetID))
	if mes.Author.ID == s.State.User.ID {
		if len(mes.Components) != 0 {
			for _, v := range mes.Components {
				if v.Type() == discordgo.ActionsRowComponent {
					byte, _ := util.ErrorCatch(v.MarshalJSON())
					data := &discordgo.ActionsRow{}
					util.ErrorCatch("", json.Unmarshal(byte, data))
					for _, v := range data.Components {
						if v.Type() == discordgo.SelectMenuComponent {
							byte, _ := util.ErrorCatch(v.MarshalJSON())
							data := &discordgo.SelectMenu{}
							util.ErrorCatch("", json.Unmarshal(byte, data))
							switch data.CustomID {
							case product.CommandPanelRole:
								modifyPanelRole(s, i, mes)
								return
							case product.CommandPanelMinecraft:
								modifyPanelMinecraft(s, i, mes)
								return
							}
						}
					}
				}
			}
		}
	}
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: translate.Message(i.Locale, "message_modify_cant_use_this_message_error"),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}))
}

func modifyPanelRole(s *discordgo.Session, i *discordgo.InteractionCreate, mes *discordgo.Message) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	one := 1
	str := translate.Message(i.Locale, "message_modify_role_add_message")
	util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title: mes.ID,
			},
		},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:  discordgo.RoleSelectMenu,
						CustomID:  product.CommandPanelAdd + ":" + session.InteractionSave(i),
						MinValues: &one,
						MaxValues: 25,
					},
				},
			},
		},
	}))
}

func modifyPanelMinecraft(s *discordgo.Session, i *discordgo.InteractionCreate, mes *discordgo.Message) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    product.CommandPanelMinecraftAddServerName,
							Label:       translate.Message(i.Locale, "panel_minecraft_display_name"),
							Placeholder: translate.Message(i.Locale, "panel_minecraft_my_server"),
							Style:       discordgo.TextInputShort,
							MaxLength:   16,
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    product.CommandPanelMinecraftAddAddress,
							Label:       translate.Message(i.Locale, "panel_minecraft_address"),
							Placeholder: "example.com",
							Required:    true,
							MaxLength:   32,
							Style:       discordgo.TextInputShort,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    product.CommandPanelMinecraftAddPort,
							Label:       translate.Message(i.Locale, "panel_minecraft_port"),
							Placeholder: "25565",
							Style:       discordgo.TextInputShort,
							Value:       "25565",
							MinLength:   5,
							MaxLength:   5,
							Required:    true,
						},
					},
				},
			},
			CustomID: product.CommandPanelMinecraftAddModal + ":" + mes.ID,
			Title:    translate.Message(i.Locale, "panel_minecraft_add_server"),
			Flags:    discordgo.MessageFlagsEphemeral,
		},
	}))
}
