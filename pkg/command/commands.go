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
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/util"
	"github.com/joho/godotenv"
)

func Ban(s *discordgo.Session, locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	var banId string
	var banReason string
	for _, d := range option.Options {
		if d.Name == "target" {
			banId = d.UserValue(s).ID
		} else if d.Name == "reason" {
			banReason = translate.Translates(*locale, "command.ban.reason", map[string]interface{}{"Reason": d.StringValue()}, 1)
		}
	}

	// メッセージ&banの処理
	if banId != s.State.User.ID {
		res.Content = translate.Translate(*locale, "command.ban.message", map[string]interface{}{
			"Target": "<@" + banId + ">",
		})
		if banReason != "" {
			res.Content += "\r" + banReason
			err := s.GuildBanCreateWithReason(gid, banId, banReason, 7)
			if err != nil {
				res = util.ErrorMessage(*locale, err)
			}
		} else {
			err := s.GuildBanCreate(gid, banId, 7)
			if err != nil {
				res = util.ErrorMessage(*locale, err)
			}
		}
	} else {
		res.Content = translate.Message(*locale, "error.TargetIsBot")
		res.Flags = discordgo.MessageFlagsEphemeral
	}
	return
}

func UnBan(s *discordgo.Session, locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	var kickId string
	for _, d := range option.Options {
		if d.Name == "target" {
			kickId = d.UserValue(s).ID
		}
	}

	if kickId != s.State.User.ID {
		res.Content = translate.Translate(*locale, "command.unban.message", map[string]interface{}{
			"Target": "<@" + kickId + ">",
		})
		err := s.GuildBanDelete(gid, kickId)
		if err != nil {
			res = util.ErrorMessage(*locale, err)
		}
	} else {
		res.Content = translate.Message(*locale, "error.TargetIsBot")
		res.Flags = discordgo.MessageFlagsEphemeral
	}
	return
}

func Kick(s *discordgo.Session, locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{Content: "ERR"}
	var kickId string
	for _, d := range option.Options {
		if d.Name == "target" {
			kickId = d.UserValue(s).ID
		}
	}

	if kickId != s.State.User.ID {
		res.Content = translate.Translate(*locale, "command.kick.message", map[string]interface{}{
			"Target": "<@" + kickId + ">",
		})
		err := s.GuildMemberDelete(gid, kickId)
		if err != nil {
			res = util.ErrorMessage(*locale, err)
		}
	} else {
		res.Content = translate.Message(*locale, "error.TargetIsBot")
		res.Flags = discordgo.MessageFlagsEphemeral
	}
	return
}

func Admin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := godotenv.Load(); err == nil || i.GuildID == os.Getenv("APPLICATION_ID") && os.Getenv("APPLICATION_ID") != "" {
		il := &discordgo.InteractionCreate{}
		util.DeepcopyJson(i, il)
		err := s.InteractionRespond(il.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "OK",
			},
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
				data := &GlobalBan{}
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
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "OK",
			Flags:   discordgo.MessageFlagsEphemeral,
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
	}
}

func panelRoleCreate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	options = options[0].Options
	var content discordgo.MessageSend
	gid := i.GuildID
	cid := i.ChannelID
	var name string
	var description string
	var role *discordgo.Role
	for _, v := range options {
		switch v.Name {
		case "name":
			name = v.StringValue()
		case "description":
			description = v.StringValue()
		case "role":
			role = v.RoleValue(s, gid)
		}
	}
	zero := 0
	content = discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       name,
				Description: description,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "roles",
						Value: role.Mention(),
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
						Options: []discordgo.SelectMenuOption{
							{
								Label: role.Name,
								Value: role.ID,
							},
						},
					},
				},
			},
		},
	}
	_, err := s.ChannelMessageSendComplex(cid, &content)
	if err != nil {
		str := fmt.Sprint(err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
	} else {
		str := "ロールを追加するにはメッセージを右クリックまたは長押しして「アプリ」から「編集」を押してください"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
	}
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
	var content discordgo.MessageSend
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
	content = discordgo.MessageSend{
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
						CustomID:    "gobot_panel_minecraft",
						Options:     option,
						Placeholder: "サーバーを選択",
						MinValues:   &zero,
						MaxValues:   1,
					},
				},
			},
		},
	}
	_, err := s.ChannelMessageSendComplex(cid, &content)
	if err != nil {
		str := fmt.Sprint(err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
	} else {
		str := "サーバーを追加するにはメッセージを右クリックまたは長押しして「アプリ」から「編集」を押してください"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
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
	data := &api.TransMCServer{
		Address: address,
		Port:    uint16(port),
		FeedMCServer: api.FeedMCServer{
			Hash:      code,
			Name:      name,
			GuildID:   gid,
			ChannelID: cid,
			RoleID:    role.ID,
		},
	}
	log.Print(data.Address, data.Port)
	body, _ := json.Marshal(data)
	api.GetApi("/api/feed/mc/add", bytes.NewBuffer(body))
	str := "OK"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
	})
}
