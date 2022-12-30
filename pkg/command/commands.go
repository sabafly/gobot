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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
	"github.com/joho/godotenv"
)

func Admin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := godotenv.Load(); err == nil || i.GuildID == os.Getenv("APPLICATION_ID") && os.Getenv("APPLICATION_ID") != "" {
		il := &discordgo.InteractionCreate{}
		util.DeepcopyJson(i, il)
		err := s.InteractionRespond(il.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Printf("例外: %v", err)
		}
		options := i.ApplicationCommandData().Options
		switch options[0].Name {
		case "ban":
			options = options[0].Options
			switch options[0].Name {
			case "add":
				options = options[0].Options
				var id string
				var reason string
				for _, v := range options {
					switch v.Name {
					case "target":
						id = v.StringValue()
					case "reason":
						reason = v.StringValue()
					}
				}
				resp, err := api.GetApi("/api/ban/create?id="+id+"&reason="+reason, http.NoBody)
				if err != nil {
					log.Printf("APIサーバーへのリクエストに失敗: %v", err)
					e := translate.ErrorEmbed(i.Locale, "error_failed_to_connect_api")
					m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &e,
					})
					log.Printf("message: %v", m.ID)
					if err != nil {
						log.Printf("例外: %v", err)
					}
					break
				}
				util.LogResp(resp)
				str := util.MessageResp(resp)
				m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &str,
				})
				log.Printf("message: %v", m.ID)
				if err != nil {
					log.Printf("例外: %v", err)
				}
			case "remove":
				options = options[0].Options
				var id string
				for _, v := range options {
					switch v.Name {
					case "target":
						id = v.StringValue()
					}
				}
				resp, err := api.GetApi("/api/ban/remove?id="+id, http.NoBody)
				if err != nil {
					log.Printf("APIサーバーへのリクエスト送信に失敗: %v", err)
					message := "Failed request to API server"
					m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: &message,
					})
					log.Printf("message: %v", m.ID)
					if err != nil {
						log.Printf("例外: %v", err)
					}
					break
				}
				util.LogResp(resp)
				defer resp.Body.Close()
				e := translate.ErrorEmbed(i.Locale, "error", map[string]interface{}{
					"Error": util.MessageResp(resp),
				})
				m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &e,
				})
				log.Printf("message: %v", m.ID)
				if err != nil {
					log.Printf("例外: %v", err)
				}
			case "get":
				resp, err := api.GetApi("/api/ban", http.NoBody)
				if err != nil {
					log.Printf("APIサーバーへのリクエスト送信に失敗: %v", err)
					e := translate.ErrorEmbed(i.Locale, "error_failed_to_connect_api")
					m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &e,
					})
					log.Printf("message: %v", m.ID)
					if err != nil {
						log.Printf("例外: %v", err)
					}
					break
				}
				util.LogResp(resp)
				defer resp.Body.Close()
				byteArray, _ := io.ReadAll(resp.Body)
				jsonBytes := ([]byte)(byteArray)
				data := &types.GlobalBan{}
				err = json.Unmarshal(jsonBytes, data)
				if err != nil {
					log.Printf("JSONデコードに失敗: %v", err)
					e := translate.ErrorEmbed(i.Locale, "error_failed_to_connect_api")
					m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &e,
					})
					log.Printf("message: %v", m.ID)
					if err != nil {
						log.Printf("例外: %v", err)
					}
					break
				}
				var str string
				for i, v := range data.Content {
					str += fmt.Sprintf("%3v: <@%v>\r	Reason: %v\r", i+1, v.ID, v.Reason)
				}
				message := fmt.Sprintf("OK %v %v \r%v", resp.Request.Method, resp.StatusCode, str)
				m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &message,
				})
				log.Printf("message: %v", m.ID)
				if err != nil {
					log.Printf("例外: %v", err)
				}
			}
		case "servers":
			options = options[0].Options
			switch options[0].Name {
			case "get":
				guilds := s.State.Guilds
				log.Print(len(guilds))
				var embeds []*discordgo.MessageEmbed
				for _, ug := range guilds {
					g, err := s.Guild(ug.ID)
					if err != nil {
						log.Print(err)
					}
					o, err := s.User(g.OwnerID)
					if err != nil {
						log.Print(err)
					}
					m, err := s.GuildMembers(ug.ID, "", 1000)
					if err != nil {
						log.Print(err)
					}
					for len(m)%1000 == 0 {
						mt, _ := s.GuildMembers(ug.ID, m[len(m)-1].User.ID, 1000)
						m = append(m, mt...)
					}
					c, err := s.GuildChannels(ug.ID)
					if err != nil {
						log.Print(err)
					}
					p, err := s.GuildMember(ug.ID, s.State.User.ID)
					if err != nil {
						log.Print(err)
					}
					var str string
					for _, gf := range ug.Features {
						str += string(gf) + "\r"
					}
					embeds = append(embeds, &discordgo.MessageEmbed{
						Title: ug.Name,
						Thumbnail: &discordgo.MessageEmbedThumbnail{
							URL:    g.IconURL(),
							Width:  512,
							Height: 512,
						},
						Fields: []*discordgo.MessageEmbedField{
							{
								Name:   "Owner",
								Value:  o.Username + "#" + o.Discriminator + "\r" + o.Mention(),
								Inline: true,
							},
							{
								Name:   "Description",
								Value:  "desc\r" + g.Description,
								Inline: true,
							},
							{
								Name:   "Locale",
								Value:  "loc\r" + g.PreferredLocale,
								Inline: true,
							},
							{
								Name:   "Boosts",
								Value:  strconv.Itoa(g.PremiumSubscriptionCount),
								Inline: true,
							},
							{
								Name:   "Members",
								Value:  strconv.Itoa(len(m)) + "/" + strconv.Itoa(g.MaxMembers),
								Inline: true,
							},
							{
								Name:   "Channels",
								Value:  strconv.Itoa(len(c)),
								Inline: true,
							},
							{
								Name:   "Roles",
								Value:  strconv.Itoa(len(g.Roles)),
								Inline: true,
							},
							{
								Name:   "Joined at",
								Value:  p.JoinedAt.Local().Format("2006-01-02 15:04:05 MST"),
								Inline: true,
							},
							{
								Name:   "Features",
								Value:  str,
								Inline: true,
							},
						},
					})
				}
				str := "OK"
				_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &str,
				})
				if err != nil {
					log.Print(err)
				}
				for 0 < len(embeds) {
					var mes []*discordgo.MessageEmbed
					if len(embeds) > 10 {
						mes = append(mes, embeds[:10]...)
						embeds = embeds[10:]
					} else {
						mes = append(mes, embeds...)
						embeds = []*discordgo.MessageEmbed{}
					}
					_, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
						Embeds: mes,
					})
					if err != nil {
						log.Print(err)
					}
					time.Sleep(time.Second)
				}
			}
		}
	} else {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You can't use this command!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("例外: %v", err)
		}
	}
}

func Panel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
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
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
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
	})
}

func panelMinecraftCreate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "OK",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
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
	if port > 65535 || 1 > port {
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
	_, err := s.ChannelMessageSendComplex(cid, &content2)
	if err != nil {
		str := fmt.Sprint(err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
	} else {
		str := translate.Message(i.Locale, "command_panel_option_minecraft_message")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
	}
	util.DeferDeleteInteraction(s, i)
}

func panelConfigEmoji(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	uid := i.Member.User.ID
	mes, err := GetSelectingMessage(uid, i.GuildID)
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

			b, err := mc.MarshalJSON()
			if err != nil {
				log.Print(err)
				continue
			}
			err = json.Unmarshal(b, &a)
			if err != nil {
				log.Print(err)
				continue
			}
			for _, smo := range a.Components {
				if smo.Type() == discordgo.SelectMenuComponent {
					b, err := smo.MarshalJSON()
					if err != nil {
						log.Print(err)
						continue
					}
					err = json.Unmarshal(b, &data)
					if err != nil {
						log.Print(err)
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
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Description: translate.Message(i.Locale, "command_panel_option_config_option_emoji_message"),
				},
			},
		})
		return
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Description: "Error",
			},
		},
	})
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
	s.ChannelMessageDelete(m.ChannelID, m.ID)
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
	_, err := s.ChannelMessageEditComplex(e)
	if err != nil {
		log.Print(err)
	}
}

func Feed(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "minecraft":
		options = options[0].Options
		switch options[0].Name {
		case "create":
			feedMinecraftCreate(s, i, options)
		case "get":
			feedMinecraftGet(s, i)
		case "remove":
			feedMinecraftRemove(s, i, options)
		}
	}
}

func feedMinecraftCreate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	gid := i.GuildID
	cid := i.ChannelID
	var name string
	var address string
	var port int
	var role discordgo.Role
	options = options[0].Options
	for _, v := range options {
		switch v.Name {
		case "name":
			name = v.StringValue()
		case "address":
			address = v.StringValue()
		case "port":
			port = int(v.IntValue())
		case "role":
			role = *v.RoleValue(s, gid)
		}
	}
	hash := sha256.New()
	io.WriteString(hash, address+":"+strconv.Itoa(port))
	st := hash.Sum(nil)
	code := hex.EncodeToString(st)
	data := &types.TransMCServer{
		Address: address,
		Port:    uint16(port),
		FeedMCServer: types.FeedMCServer{
			Hash:      code,
			Name:      name,
			GuildID:   gid,
			ChannelID: cid,
			RoleID:    role.ID,
			Locale:    i.Locale,
		},
	}
	log.Print(data.Address, data.Port, i.Locale)
	body, _ := json.Marshal(data)
	api.GetApi("/api/feed/mc/add", bytes.NewBuffer(body))
	str := "OK"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
	})
}

func feedMinecraftGet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	resp, err := api.GetApi("/api/feed/mc", http.NoBody)
	if err != nil {
		log.Print(err)
		embed := translate.ErrorEmbed(i.Locale, "error_failed_to_connect_api")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embed,
		})
		return
	}
	body, _ := io.ReadAll(resp.Body)
	content := types.Res{}
	data := types.FeedMCServers{}
	json.Unmarshal(body, &content)
	b, _ := json.Marshal(content.Content)
	json.Unmarshal(b, &data)
	array := []*discordgo.MessageEmbed{}
	var server types.FeedMCServers
	var locales []discordgo.Locale
	for _, v := range data {
		var locale discordgo.Locale
		if v.Locale == "" {
			locale = discordgo.Japanese
		}
		if l, ok := types.StL[string(v.Locale)]; ok {
			locale = l
		}
		if v.GuildID == i.GuildID {
			server = append(server, v)
			locales = append(locales, locale)
		}
	}
	resp2, err := api.GetApi("/api/feed/mc/hash", http.NoBody)
	if err != nil {
		log.Print(err)
		embed := translate.ErrorEmbed(i.Locale, "error_failed_to_connect_api")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embed,
		})
		return
	}
	body, _ = io.ReadAll(resp2.Body)
	content2 := types.Res{}
	json.Unmarshal(body, &content2)
	b, _ = json.Marshal(content2.Content)
	hash := types.MCServers{}
	json.Unmarshal(b, &hash)
	log.Printf("commands:538: %v | %v", len(server), len(hash))
	for n, v := range server {
		var address string
		var port uint16
		for _, v2 := range hash {
			if v2.Hash == v.Hash {
				address = v2.Address
				port = v2.Port
				break
			}
		}
		array = append(array, &discordgo.MessageEmbed{
			Title: v.Name,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   translate.Message(locales[n], "address"),
					Value:  address,
					Inline: true,
				},
				{
					Name:   translate.Message(locales[n], "port"),
					Value:  strconv.Itoa(int(port)),
					Inline: true,
				},
				{
					Name:  translate.Message(locales[n], "channel"),
					Value: "<#" + v.ChannelID + ">",
				},
			},
		})
	}
	if len(array) != 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &array,
		})
	} else {
		str := "no data"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
	}
}

func feedMinecraftRemove(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name string
	options = options[0].Options
	for _, v := range options {
		switch v.Name {
		case "name":
			name = v.StringValue()
		}
		data := &types.FeedMCServer{
			Name:    name,
			GuildID: i.GuildID,
		}
		body, _ := json.Marshal(data)
		api.GetApi("/api/feed/mc/remove", bytes.NewBuffer(body))
		str := "OK"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})

	}
}

func Role(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "color":
		roleColor(s, i, options)
	}
}

func roleColor(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	options = options[0].Options
	var raw string
	var name string
	for _, o := range options {
		switch o.Name {
		case "rgb":
			raw = "0x" + o.StringValue()
		case "name":
			name = o.StringValue()
		}
	}
	col, err := strconv.ParseInt(raw, 0, 32)
	if name == "" {
		name = strconv.FormatInt(col, 16)
	}
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		})
		return
	}
	colInt := int(col)
	r, err := s.GuildRoleCreate(i.GuildID, &discordgo.RoleParams{
		Name:  name,
		Color: &colInt,
	})
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		})
		return
	}
	s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, r.ID)
	str := "OK " + r.Mention()
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
	})
}

func UInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	gid := i.GuildID
	uid := i.ApplicationCommandData().TargetID
	m, err := s.State.Member(i.GuildID, uid)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		})
		return
	}
	var status string
	p, err := s.State.Presence(gid, uid)
	if err != nil {
		status = "Status: " + util.StatusString(discordgo.StatusOffline)
	} else {
		if p.Status != discordgo.StatusOffline {
			if str := util.StatusString(p.ClientStatus.Web); str != "" {
				status += translate.Message(i.Locale, "client_web") + ": " + str + "\r"
			}
			if str := util.StatusString(p.ClientStatus.Desktop); str != "" {
				status += translate.Message(i.Locale, "client_desktop") + ": " + str + "\r"
			}
			if str := util.StatusString(p.ClientStatus.Mobile); str != "" {
				status += translate.Message(i.Locale, "client_mobile") + ": " + str + "\r"
			}
		}
		if status == "" {
			status += translate.Message(i.Locale, "online_status") + ": " + util.StatusString(p.Status) + "\r"
		}
	}
	u, err := s.User(uid)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		})
		return
	}
	var roles string
	var color int = 0x000000
	role, _ := s.GuildRoles(i.GuildID)
	me, _ := s.GuildMember(i.GuildID, uid)
	var highestPosition int
	for _, v := range me.Roles {
		r, _ := s.State.Role(i.GuildID, v)
		if r.Position >= highestPosition {
			highestPosition = r.Position
		}
	}
	var r []*discordgo.Role
	for _, r2 := range role {
		for _, v := range me.Roles {
			if r2.ID == v {
				r = append(r, r2)
			}
		}
	}
	sort.Slice(r, func(i2, j int) bool {
		return r[i2].Position < r[j].Position
	})
	for i2, j := 0, len(r)-1; i2 < j; i2, j = i2+1, j-1 {
		r[i2], r[j] = r[j], r[i2]
	}

	for _, v := range r {
		roles += v.Mention()
	}
	if roles == "" {
		roles = "`" + translate.Message(i.Locale, "message_command_user_info_none") + "`"
	}
	for _, v := range r {
		if v.Color != 0x000000 {
			color = v.Color
		}
	}
	sColor := strconv.FormatInt(int64(color), 16)
	for utf8.RuneCountInString(sColor) < 6 {
		sColor = "0" + sColor
	}

	if m.Nick == "" {
		m.Nick = "`" + translate.Message(i.Locale, "message_command_user_info_none") + "`"
	}
	embed := discordgo.MessageEmbed{
		Title: u.Username + "#" + u.Discriminator,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:    m.AvatarURL("512"),
			Width:  512,
			Height: 512,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_nick"),
				Value:  m.Nick,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_id"),
				Value:  m.User.ID,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_roles"),
				Value:  roles,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_joined_at"),
				Value:  "<t:" + strconv.FormatInt(m.JoinedAt.Unix(), 10) + ":F>",
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_color_code"),
				Value:  sColor,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_status"),
				Value:  status,
				Inline: true,
			},
		},
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{&embed},
	})
	if err != nil {
		log.Print(err)
	}
}
