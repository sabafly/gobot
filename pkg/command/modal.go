package command

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func MSminecraftPanel(s *discordgo.Session, i *discordgo.InteractionCreate, mid string) {
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
			case "gobot_panel_minecraft_add_servername":
				name = text.Value
			case "gobot_panel_minecraft_add_address":
				address = text.Value
			case "gobot_panel_minecraft_add_port":
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
						CustomID:  "gobot_panel_minecraft",
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
