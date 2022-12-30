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
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/util"
)

func ModalMinecraftPanel(s *discordgo.Session, i *discordgo.InteractionCreate, mid string) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	var name string
	var address string
	var port int
	for _, mc := range i.ModalSubmitData().Components {
		if mc.Type() == discordgo.ActionsRowComponent {
			bytes, _ := util.ErrorCatch(mc.MarshalJSON())
			data := &discordgo.ActionsRow{}
			util.ErrorCatch("", json.Unmarshal(bytes, data))
			bytes, _ = util.ErrorCatch(data.Components[0].MarshalJSON())
			text := &discordgo.TextInput{}
			util.ErrorCatch("", json.Unmarshal(bytes, text))
			switch text.CustomID {
			case product.CommandPanelMinecraftAddServerName:
				name = text.Value
			case product.CommandPanelMinecraftAddAddress:
				address = text.Value
			case product.CommandPanelMinecraftAddPort:
				i, err := util.ErrorCatch(strconv.Atoi(text.Value))
				if err != nil {
					port = 25565
				} else {
					port = i
				}
			}
		}
	}
	mes, _ := util.ErrorCatch(s.ChannelMessage(i.ChannelID, mid))
	bytes, _ := util.ErrorCatch(mes.Components[0].MarshalJSON())
	data := &discordgo.ActionsRow{}
	util.ErrorCatch("", json.Unmarshal(bytes, data))
	bytes, _ = util.ErrorCatch(data.Components[0].MarshalJSON())
	text := &discordgo.SelectMenu{}
	util.ErrorCatch("", json.Unmarshal(bytes, text))
	options := text.Options
	str := strings.Split(options[0].Value, ":")
	bl, _ := util.ErrorCatch(strconv.ParseBool(str[3]))
	name = strings.ReplaceAll(name, ":", ";")
	address = strings.ReplaceAll(address, ":", ";")
	options = append(options, discordgo.SelectMenuOption{
		Label: name,
		Value: name + ":" + address + ":" + strconv.Itoa(port) + ":" + strconv.FormatBool(bl),
	})
	if port > 1<<16 || 1 > port {
		port = 25565
	}
	zero := 0
	res := discordgo.MessageEdit{
		Channel: i.ChannelID,
		ID:      mid,
		Embeds:  mes.Embeds,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:  product.CommandPanelMinecraft,
						Options:   options,
						MinValues: &zero,
						MaxValues: 1,
					},
				},
			},
		},
	}
	_, err := util.ErrorCatch(s.ChannelMessageEditComplex(&res))
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprint(err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
	util.ErrorCatch("", s.InteractionResponseDelete(i.Interaction))
}
