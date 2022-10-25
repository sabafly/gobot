package command

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

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
				resp, err := api.GetApi("/api/ban/create?id=" + id + "&reason=" + reason)
				if err != nil {
					log.Printf("APIサーバーへのリクエストに失敗: %v", err)
					message := "Failed to create request"
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
				m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: util.MessageResp(resp),
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
				resp, err := api.GetApi("/api/ban/remove?id=" + id)
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
				m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: util.MessageResp(resp),
				})
				log.Printf("message: %v", m.ID)
				if err != nil {
					log.Printf("例外: %v", err)
				}
			case "get":
				resp, err := api.GetApi("/api/ban")
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
				byteArray, _ := io.ReadAll(resp.Body)
				jsonBytes := ([]byte)(byteArray)
				data := &GlobalBan{}
				err = json.Unmarshal(jsonBytes, data)
				if err != nil {
					log.Printf("JSONデコードに失敗: %v", err)
					message := "Failed unmarshal json objects"
					m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: &message,
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
				message := fmt.Sprintf("succeed %v %v \r%v", resp.Request.Method, resp.StatusCode, str)
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
