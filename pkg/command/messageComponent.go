package command

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Tnze/go-mc/chat"
	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
	"github.com/millkhan/mcstatusgo/v2"
)

func MCpanelRole(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

func MCpanelRoleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	mid := i.Message.Embeds[0].Title
	gid := i.GuildID
	cid := i.ChannelID
	mes, _ := s.ChannelMessage(cid, mid)
	var unused string
	rv := i.Interaction.MessageComponentData().Values
	me, _ := s.GuildMember(i.GuildID, s.State.User.ID)
	var highestPosition int
	for _, v := range me.Roles {
		r, _ := s.State.Role(i.GuildID, v)
		if r.Position > highestPosition {
			highestPosition = r.Position
		}
	}
	roles := []discordgo.Role{}
	for _, v := range rv {
		role, _ := s.State.Role(gid, v)
		if role.Position < highestPosition && !role.Managed && role.ID != gid {
			roles = append(roles, *role)
		} else {
			unused += role.Mention() + " "
		}
	}
	if len(roles) == 0 {
		embeds := translate.ErrorEmbed(i.Locale, "error_invalid_roles")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embeds,
		})
		return
	}
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
	var embed []*discordgo.MessageEmbed
	if unused != "" {
		embed = append(embed, &discordgo.MessageEmbed{
			Title:       translate.Message(i.Locale, "error_cannot_use_roles"),
			Description: unused,
		})
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embed,
		})
	} else {
		s.InteractionResponseDelete(i.Interaction)
	}
}

func MCpanelRoleCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	title := i.Message.Embeds[0].Title
	description := i.Message.Embeds[0].Description
	gid := i.GuildID
	cid := i.ChannelID
	var unused string
	rv := i.Interaction.MessageComponentData().Values
	me, _ := s.GuildMember(i.GuildID, s.State.User.ID)
	var highestPosition int
	for _, v := range me.Roles {
		r, _ := s.State.Role(i.GuildID, v)
		if r.Position > highestPosition {
			highestPosition = r.Position
		}
	}
	roles := []discordgo.Role{}
	for _, v := range rv {
		role, _ := s.State.Role(gid, v)
		if role.Position < highestPosition && !role.Managed && role.ID != gid {
			roles = append(roles, *role)
		} else {
			unused += role.Mention() + " "
		}
	}
	if len(roles) == 0 {
		embeds := translate.ErrorEmbed(i.Locale, "error_invalid_roles")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embeds,
		})
		return
	}
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
						CustomID:  "gobot_panel_role",
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
	s.ChannelMessageSendComplex(cid, &content)
	str := translate.Message(i.Locale, "command_panel_option_role_message")
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
		Embeds:  &embed,
	})
}

func MCpanelMinecraft(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if fmt.Sprint(i.MessageComponentData().Values) == "[]" {
		s.InteractionResponseDelete(i.Interaction)
		return
	}
	initialTimeOut := time.Second * 10
	ioTimeOut := time.Second * 30
	data := i.MessageComponentData()
	addresses := strings.Split(data.Values[0], ":")
	name := addresses[0]
	address := addresses[1]
	port, err := strconv.ParseUint(addresses[2], 10, 16)
	if err != nil {
		log.Print(err)
		e := translate.ErrorEmbed(i.Locale, "error_invalid_port_value")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &e,
		})
		return
	}
	showIp, err := strconv.ParseBool(addresses[3])
	if err != nil {
		log.Print(err)
	}
	q, err := mcstatusgo.Status(address, uint16(port), initialTimeOut, ioTimeOut)
	if err != nil {
		log.Print(err)
		e := translate.ErrorEmbed(i.Locale, "error_failed_to_ping_server")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &e,
		})
		return
	}
	message := chat.Message{}
	message.UnmarshalJSON([]byte(q.Description))
	hash := sha256.New()
	thumb := strings.ReplaceAll(q.Favicon, "data:image/png;base64,", "")
	res, _ := base64.RawStdEncoding.DecodeString(thumb)
	io.WriteString(hash, thumb)
	str := hash.Sum(nil)
	code := hex.EncodeToString(str)
	bd := &types.ImagePngHash{
		Data: thumb,
		Hash: code,
	}
	b, _ := json.Marshal(bd)
	api.GetApi("/api/image/png/add", bytes.NewBuffer(b))
	color := 0x00ff00
	if q.Version.Protocol == 46 {
		color = 0xff0000
	}
	var player string
	for _, v := range q.Players.Sample {
		player += v["name"] + "\r"
	}
	if player != "" {
		player = "```" + player + "```"
	}
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       name,
			Description: "```ansi\r" + message.String() + "```",
			Color:       color,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "attachment://" + code + ".png",
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Footer:    &discordgo.MessageEmbedFooter{Text: "gobot"},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   translate.Message(i.Locale, "players"),
					Value:  "```" + strconv.Itoa(q.Players.Online) + "/" + strconv.Itoa(q.Players.Max) + "```" + player,
					Inline: true,
				},
				{
					Name:   translate.Message(i.Locale, "latency"),
					Value:  "```" + strconv.Itoa(int(q.Latency.Abs().Milliseconds())) + "ms" + "```",
					Inline: true,
				},
				{
					Name:   translate.Message(i.Locale, "version"),
					Value:  "```ansi\r" + chat.Text(q.Version.Name).String() + "```",
					Inline: true,
				},
			},
		},
	}
	if showIp {
		embeds[0].Fields = append(embeds[0].Fields, &discordgo.MessageEmbedField{
			Name:   translate.Message(i.Locale, "address"),
			Value:  "```" + address + "```",
			Inline: true,
		},
			&discordgo.MessageEmbedField{
				Name:   translate.Message(i.Locale, "port"),
				Value:  "```" + strconv.Itoa(int(q.Port)) + "```",
				Inline: true,
			})
	}
	s.ChannelMessageEditComplex(discordgo.NewMessageEdit(i.ChannelID, i.Message.ID))
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &embeds,
		Files: []*discordgo.File{
			{
				Name:        code + ".png",
				ContentType: "image/png",
				Reader:      bytes.NewReader(res),
			},
		},
	})
	if err != nil {
		log.Print(err)
	}
}
