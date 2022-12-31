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
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
)

func Panel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "role":
		options = options[0].Options
		switch options[0].Name {
		case "create":
			panelRoleCreate(s, i, options)
		}
	case "minecraft":
		options = options[0].Options
		switch options[0].Name {
		case "create":
			panelMinecraftCreate(s, i, options)
		}
	case "config":
		options = options[0].Options
		switch options[0].Name {
		case "emoji":
			panelConfigEmoji(s, i, options)
		}
	}
}

func panelRoleCreate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	options = options[0].Options
	var name string
	var description string
	for _, v := range options {
		switch v.Name {
		case "name":
			name = v.StringValue()
		case "description":
			description = v.StringValue()
		}
	}
	one := 1
	content := translate.Message(i.Locale, "message_modify_role_create_message")
	util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       name,
				Description: description,
			},
		},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:  discordgo.RoleSelectMenu,
						CustomID:  product.CommandPanelRoleCreate + ":" + session.InteractionSave(i),
						MinValues: &one,
						MaxValues: 25,
					},
				},
			},
		},
	}))
}

func panelMinecraftCreate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "OK",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}))
	options = options[0].Options
	var content2 discordgo.MessageSend
	cid := i.ChannelID
	var name string
	var description string
	var serverName string
	var address string
	var showIp bool
	port := 25565
	for _, o := range options {
		switch o.Name {
		case "name":
			name = o.StringValue()
		case "description":
			description = o.StringValue()
		case "servername":
			serverName = o.StringValue()
		case "address":
			address = o.StringValue()
		case "port":
			port = int(o.IntValue())
		case "showip":
			showIp = o.BoolValue()
		}
	}
	serverName = strings.ReplaceAll(serverName, ":", ";")
	address = strings.ReplaceAll(address, ":", ";")
	if port > 1<<16 || 1 > port {
		port = 25565
	}
	serverAddress := serverName + ":" + address + ":" + strconv.Itoa(port) + ":" + strconv.FormatBool(showIp)
	option := []discordgo.SelectMenuOption{}
	option = append(option, discordgo.SelectMenuOption{
		Label: serverName,
		Value: serverAddress,
	})
	zero := 0
	content2 = discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       name,
				Description: description,
			},
		},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    product.CommandPanelMinecraft,
						Options:     option,
						Placeholder: translate.Message(i.Locale, "command_panel_option_minecraft_placeholder"),
						MinValues:   &zero,
						MaxValues:   1,
					},
				},
			},
		},
	}
	_, err := util.ErrorCatch(s.ChannelMessageSendComplex(cid, &content2))
	if err != nil {
		str := fmt.Sprint(err)
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		}))
	} else {
		str := translate.Message(i.Locale, "command_panel_option_minecraft_message")
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		}))
	}
}

func panelConfigEmoji(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	uid := i.Member.User.ID
	mes, err := util.ErrorCatch(GetSelectingMessage(uid, i.GuildID))
	if err != nil {
		embed := translate.ErrorEmbed(i.Locale, "error", map[string]interface{}{
			"Error": err,
		})
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embed,
		})
		return
	}
	var data discordgo.SelectMenu
	for _, mc := range mes.Components {
		if mc.Type() == discordgo.ActionsRowComponent {

			var a discordgo.ActionsRow

			b, err := util.ErrorCatch(mc.MarshalJSON())
			if err != nil {
				continue
			}
			_, err = util.ErrorCatch("", json.Unmarshal(b, &a))
			if err != nil {
				continue
			}
			for _, smo := range a.Components {
				if smo.Type() == discordgo.SelectMenuComponent {
					b, err := util.ErrorCatch(smo.MarshalJSON())
					if err != nil {
						continue
					}
					_, err = util.ErrorCatch("", json.Unmarshal(b, &data))
					if err != nil {
						continue
					}
					break
				}
			}
			break
		}
	}
	switch data.CustomID {
	case product.CommandPanelMinecraft, product.CommandPanelRole:
		session.MessagePanelConfigEmojiRemove(uid)
		session.MessagePanelConfigEmojiSave(&types.MessageSessionData[types.MessagePanelConfigEmojiData]{
			Message: mes,
			Data: types.MessagePanelConfigEmojiData{
				UserID:     uid,
				Emojis:     []*discordgo.ComponentEmoji{},
				SelectMenu: data,
			},
			Handler: panelConfigEmojiHandler,
		}, uid)
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Description: translate.Message(i.Locale, "command_panel_option_config_option_emoji_message"),
				},
			},
		}))
		return
	}
	util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Description: "Error",
			},
		},
	}))
}

func panelConfigEmojiHandler(t types.MessageSessionData[types.MessagePanelConfigEmojiData], s *discordgo.Session, m *discordgo.MessageCreate) {
	mid := t.Data.UserID
	if t.Message.ChannelID != m.ChannelID {
		return
	}
	if m.Content == "cancel" {
		session.MessagePanelConfigEmojiRemove(mid)
		RemoveSelect(mid, t.Message.GuildID)
		return
	}
	emojiString := util.Regexp2FindAllString(types.Twemoji, m.Content)
	log.Print(emojiString)
	var e []*discordgo.ComponentEmoji
	for i, v := range emojiString {
		if discordgo.EmojiRegex.MatchString(v) {
			log.Printf("Custom  %v, %v", i, v)
			e = append(e, util.GetCustomEmojis(v)...)
		} else {
			log.Printf("Default %v, %v", i, v)
			e = append(e, &discordgo.ComponentEmoji{
				Name:     v,
				ID:       "",
				Animated: false,
			})
		}
	}
	t.Data.Emojis = append(t.Data.Emojis, e...)
	session.MessagePanelConfigEmojiSave(&t, mid)
	util.ErrorCatch("", s.ChannelMessageDelete(m.ChannelID, m.ID))
	if len(t.Data.Emojis) >= len(t.Data.SelectMenu.Options) {
		session.MessagePanelConfigEmojiRemove(mid)
		RemoveSelect(mid, t.Message.GuildID)
		if t.Data.SelectMenu.CustomID == product.CommandPanelRole {
			var value string
			str := strings.Split(t.Message.Embeds[0].Fields[0].Value, "\r")
			for i, v := range str {
				str1 := strings.Split(v, "|")
				log.Print(util.EmojiFormat(t.Data.Emojis[i]))
				str1[0] = util.EmojiFormat(t.Data.Emojis[i]) + " | "
				var str2 string
				for _, v1 := range str1 {
					str2 += v1
				}
				value += str2 + "\r"
			}
			t.Message.Embeds[0].Fields[0].Value = value
			log.Print(value)
		}
		go updateEmoji(s, t)
		return
	}
}

func updateEmoji(s *discordgo.Session, o types.MessageSessionData[types.MessagePanelConfigEmojiData]) {
	for i := range o.Data.SelectMenu.Options {
		o.Data.SelectMenu.Options[i].Emoji = *o.Data.Emojis[i]
		log.Print(*o.Data.Emojis[i])
	}
	e := discordgo.NewMessageEdit(o.Message.ChannelID, o.Message.ID)
	e.Components = append(e.Components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			o.Data.SelectMenu,
		},
	})
	e.Content = &o.Message.Content
	e.Embeds = o.Message.Embeds
	util.ErrorCatch(s.ChannelMessageEditComplex(e))
}
