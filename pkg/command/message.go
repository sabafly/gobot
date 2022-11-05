package command

import (
	"encoding/json"
	"log"

	"github.com/bwmarrin/discordgo"
)

func Mmodify(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := &discordgo.ApplicationCommandInteractionData{}
	byte, _ := json.Marshal(i.Interaction.Data)
	json.Unmarshal(byte, data)
	mes, err := s.ChannelMessage(i.ChannelID, data.TargetID)
	if err != nil {
		log.Print(err)
	}
	if mes.Author.ID == s.State.User.ID {
		if len(mes.Components) != 0 {
			for _, v := range mes.Components {
				if v.Type() == discordgo.ActionsRowComponent {
					byte, _ := v.MarshalJSON()
					data := &discordgo.ActionsRow{}
					json.Unmarshal(byte, data)
					for _, v := range data.Components {
						if v.Type() == discordgo.SelectMenuComponent {
							byte, _ := v.MarshalJSON()
							data := &discordgo.SelectMenu{}
							json.Unmarshal(byte, data)
							switch data.CustomID {
							case "gobot_panel_role":
								gobotPanelRole(s, i, *mes)
							case "gobot_panel_minecraft":
								err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
									Type: discordgo.InteractionResponseModal,
									Data: &discordgo.InteractionResponseData{
										Components: []discordgo.MessageComponent{
											discordgo.ActionsRow{
												Components: []discordgo.MessageComponent{
													discordgo.TextInput{
														CustomID:    "gobot_panel_minecraft_add_servername",
														Label:       "表示名",
														Placeholder: "マイサーバー",
														Style:       discordgo.TextInputShort,
														Required:    true,
													},
												},
											},
											discordgo.ActionsRow{
												Components: []discordgo.MessageComponent{
													discordgo.TextInput{
														CustomID:    "gobot_panel_minecraft_add_address",
														Label:       "アドレス",
														Placeholder: "example.com",
														Required:    true,
														Style:       discordgo.TextInputShort,
													},
												},
											},
											discordgo.ActionsRow{
												Components: []discordgo.MessageComponent{
													discordgo.TextInput{
														CustomID:    "gobot_panel_minecraft_add_port",
														Label:       "ポート",
														Placeholder: "25565",
														Style:       discordgo.TextInputShort,
														Value:       "25565",
														MaxLength:   5,
														Required:    true,
													},
												},
											},
										},
										CustomID: "gobot_panel_minecraft_add_modal:" + mes.ID,
										Title:    "サーバー追加",
									},
								})
								log.Print(err)
							}
						}
					}
				}
			}
		}
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "そのメッセージには使用できません",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func gobotPanelRole(s *discordgo.Session, i *discordgo.InteractionCreate, mes discordgo.Message) {
	roles, _ := s.GuildRoles(i.GuildID)
	options := []discordgo.SelectMenuOption{}
	me, _ := s.GuildMember(i.GuildID, s.State.User.ID)
	var highestPosition int
	for _, v := range me.Roles {
		r, _ := s.State.Role(i.GuildID, v)
		if r.Position > highestPosition {
			highestPosition = r.Position
		}
	}
	for _, v := range roles {
		if v.Position < highestPosition && !v.Managed && v.ID != i.GuildID {
			options = append(options, discordgo.SelectMenuOption{
				Label: v.Name,
				Value: v.ID,
			})
		}
	}
	zero := 0
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "追加するロールを選んでください",
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: mes.ID,
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:  "gobot_panel_role_add",
							MinValues: &zero,
							MaxValues: len(options),
							Options:   options,
						},
					},
				},
			},
		},
	})
}
