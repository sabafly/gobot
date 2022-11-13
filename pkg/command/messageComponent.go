package command

import (
	"bytes"
	"crypto/sha256"
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
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "OK",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
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
	port, err := strconv.Atoi(addresses[2])
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
	io.WriteString(hash, thumb)
	str := hash.Sum(nil)
	code := hex.EncodeToString(str)
	bd := &types.ImagePngHash{
		Data: thumb,
		Hash: code,
	}
	b, _ := json.Marshal(bd)
	api.GetApi("/api/image/png/add", bytes.NewBuffer(b))
	log.Print(q.Version.Protocol)
	log.Print("https://sabafly.net/api/decode?s=" + code)
	color := 0x00ff00
	if q.Version.Protocol == 46 {
		color = 0xff0000
	}
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       name,
			Description: "```ansi\r" + message.String() + "```",
			Color:       color,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL:    "https://sabafly.net/api/decode?s=" + code,
				Width:  64,
				Height: 64,
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Footer:    &discordgo.MessageEmbedFooter{Text: "gobot"},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   translate.Message(i.Locale, "players"),
					Value:  "```" + strconv.Itoa(q.Players.Online) + "/" + strconv.Itoa(q.Players.Max) + "```",
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
	})
	if err != nil {
		log.Print(err)
	}
}
