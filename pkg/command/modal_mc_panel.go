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
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/product"
)

func ModalMinecraftPanel(s *discordgo.Session, i *discordgo.InteractionCreate, mid string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	var name string
	var address string
	var port int
	for _, mc := range i.ModalSubmitData().Components {
		if mc.Type() == discordgo.ActionsRowComponent {
			bytes, _ := mc.MarshalJSON()
			data := &discordgo.ActionsRow{}
			json.Unmarshal(bytes, data)
			bytes, _ = data.Components[0].MarshalJSON()
			text := &discordgo.TextInput{}
			json.Unmarshal(bytes, text)
			switch text.CustomID {
			case product.CommandPanelMinecraftAddServerName:
				name = text.Value
			case product.CommandPanelMinecraftAddAddress:
				address = text.Value
			case product.CommandPanelMinecraftAddPort:
				i, _ := strconv.Atoi(text.Value)
				port = i
			}
		}
	}
	mes, _ := s.ChannelMessage(i.ChannelID, mid)
	bytes, _ := mes.Components[0].MarshalJSON()
	data := &discordgo.ActionsRow{}
	json.Unmarshal(bytes, data)
	bytes, _ = data.Components[0].MarshalJSON()
	text := &discordgo.SelectMenu{}
	json.Unmarshal(bytes, text)
	options := text.Options
	str := strings.Split(options[0].Value, ":")
	bl, _ := strconv.ParseBool(str[3])
	options = append(options, discordgo.SelectMenuOption{
		Label: name,
		Value: name + ":" + address + ":" + strconv.Itoa(port) + ":" + strconv.FormatBool(bl),
	})
	name = strings.ReplaceAll(name, ":", ";")
	address = strings.ReplaceAll(address, ":", ";")
	if port > 65535 || 1 > port {
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
	_, err := s.ChannelMessageEditComplex(&res)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprint(err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
	s.InteractionResponseDelete(i.Interaction)
	log.Print(name + address + strconv.Itoa(port))
}
