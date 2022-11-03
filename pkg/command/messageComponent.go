package command

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

func MCpanelRole(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				for _, m := range i.Member.Roles {
					if v.Value == m {
						for _, v2 := range i.MessageComponentData().Values {
							if v2 != v.Value {
								s.GuildMemberRoleRemove(gid, uid, v.Value)
								content += "はく奪 <@&" + v.Value + ">\r"

							}
						}
					}
				}
			}
			for _, r := range i.MessageComponentData().Values {
				s.GuildMemberRoleAdd(gid, uid, r)
				content += "付与 <@&" + r + ">\r"
			}
		}
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func MCpanelRoleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	mid := i.Message.Embeds[0].Title
	gid := i.GuildID
	cid := i.ChannelID
	mes, _ := s.ChannelMessage(cid, mid)
	rv := i.MessageComponentData().Values
	roles := []discordgo.Role{}
	for _, v := range rv {
		role, _ := s.State.Role(gid, v)
		roles = append(roles, *role)
	}
	options := []discordgo.SelectMenuOption{}
	for _, r := range roles {
		options = append(options, discordgo.SelectMenuOption{
			Label: r.Name,
			Value: r.ID,
		})
	}
	var fields string
	for _, r := range roles {
		fields += r.Mention() + "\r"
	}
	zero := 0
	content := discordgo.MessageEdit{
		ID:      mid,
		Channel: cid,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title: mes.Embeds[0].Title,
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
						CustomID:  "gobot_panel_role",
						MinValues: &zero,
						MaxValues: len(options),
						Options:   options,
					},
				},
			},
		},
	}
	s.ChannelMessageEditComplex(&content)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "OK",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
