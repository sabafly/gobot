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
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
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
