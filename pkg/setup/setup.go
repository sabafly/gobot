package setup

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command"
	"github.com/ikafly144/gobot/pkg/util"
	"github.com/joho/godotenv"
)

var (
	BotToken       = flag.String("Token", "", "botアクセストークン")
	RemoveCommands = flag.Bool("rmcmd", true, "停止時にコマンドを登録解除するか")
	ApplicationId  = flag.String("Application", "", "botのsnowflake")
	SupportGuildID = flag.String("SupportServer", "", "サポートサーバーのID")
	APIServer      = flag.String("APIAddress", "", "APIサーバーのip")
)

func Setup() (s *discordgo.Session, commands []*discordgo.ApplicationCommand, RemoveCommands bool, GuildID string) {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env:%v", err)
	}
	*BotToken = os.Getenv("TOKEN")
	GuildID = os.Getenv("GUILD_ID")
	*SupportGuildID = os.Getenv("SUPPORT_ID")
	RemoveCommands, err = strconv.ParseBool(os.Getenv("REMOVE_COMMANDS"))
	*APIServer = os.Getenv("API_SERVER")
	if err != nil {
		RemoveCommands = true
	}
	*ApplicationId = os.Getenv("APPLICATION_ID")

	flag.Parse()

	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("無効なbotパラメータ: %v", err)
	}

	var (
		// integerOptionMinValue          = 1.0
		dmPermission = false
		// PermissionAll          int64 = discordgo.PermissionAll
		PermissionBanMembers   int64 = discordgo.PermissionBanMembers
		PermissionKickMembers  int64 = discordgo.PermissionKickMembers
		PermissionAdminMembers int64 = discordgo.PermissionAdministrator
	)
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "pong!",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "ポング！",
			},
			Version: "1",
		},
		{
			Name:        "ban",
			Description: "ban the selected user",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "選択したユーザーをbanする",
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "target",
					Description: "user to ban",
					NameLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "対象",
					},
					DescriptionLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "banするユーザー",
					},
					Type:     discordgo.ApplicationCommandOptionUser,
					Required: true,
				},
				{
					Name:        "reason",
					Description: "reason for ban",
					NameLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "理由",
					},
					DescriptionLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "banする理由",
					},
					Type: discordgo.ApplicationCommandOptionString,
				},
			},
			DefaultMemberPermissions: &PermissionBanMembers,
			DMPermission:             &dmPermission,
			Version:                  "1",
		},
		{
			Name:        "unban",
			Description: "pardon the selected user",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "指定したユーザーのbanを解除します",
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "target",
					Description: "user to pardon",
					NameLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "対象",
					},
					DescriptionLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "banを解除するユーザー",
					},
					Type:     discordgo.ApplicationCommandOptionUser,
					Required: true,
				},
			},
			DefaultMemberPermissions: &PermissionBanMembers,
			DMPermission:             &dmPermission,
			Version:                  "1",
		},
		{
			Name:        "kick",
			Description: "kick the selected user",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "指定したユーザーをキックする",
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "target",
					Description: "user to kick",
					NameLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "対象",
					},
					DescriptionLocalizations: map[discordgo.Locale]string{
						discordgo.Japanese: "キックするユーザー",
					},
					Type:     discordgo.ApplicationCommandOptionUser,
					Required: true,
				},
			},
			DefaultMemberPermissions: &PermissionKickMembers,
			DMPermission:             &dmPermission,
			Version:                  "1",
		},
		{
			Name:                     "admin",
			Description:              "only for bot admins",
			GuildID:                  *SupportGuildID,
			DMPermission:             &dmPermission,
			DefaultMemberPermissions: &PermissionAdminMembers,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "ban",
					Description: "only for bot admins",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "add",
							Description: "only for bot admins",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "target",
									Description: "only for bot admins",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "reason",
									Description: "only for bot admins",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    false,
								},
							},
						},
						{
							Name:        "remove",
							Description: "only for bot admins",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "target",
									Description: "only for bot admins",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
							},
						},
						{
							Name:        "get",
							Description: "only for admins",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
			},
			Version: "1",
		},
	}
	var (
		commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
			"ban": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: command.Ban(s, &i.Locale, i.ApplicationCommandData(), i.GuildID),
				})
				if err != nil {
					log.Printf("例外: %v", err)
				}
			},
			"unban": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: command.UnBan(s, &i.Locale, i.ApplicationCommandData(), i.GuildID),
				})
				if err != nil {
					log.Printf("例外: %v", err)
				}
			},
			"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				contents := map[discordgo.Locale]string{
					discordgo.Japanese: "ポング！",
				}
				content := "pong!"
				if c, ok := contents[i.Locale]; ok {
					content = c
				}

				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: content,
					},
				})
				if err != nil {
					log.Printf("例外: %v", err)
				}
			},
			"kick": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: command.Kick(s, &i.Locale, i.ApplicationCommandData(), i.GuildID),
				})
				if err != nil {
					log.Printf("例外: %v", err)
				}
			},
			"admin": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				if i.GuildID == *SupportGuildID {
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
							req, err := http.NewRequest("GET", "http://"+*APIServer+"/api/ban/create?id="+id+"&reason="+reason, http.NoBody)
							if err != nil {
								log.Printf("リクエスト作成に失敗: %v", err)
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
							client := http.Client{}
							resp, err := client.Do(req)
							if err != nil {
								log.Printf("APIサーバーへのリクエスト送信に失敗: %v", err)
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
							defer resp.Body.Close()
							byteArray, _ := io.ReadAll(resp.Body)
							jsonBytes := ([]byte)(byteArray)
							log.Printf("succeed %v %v %v", resp.Request.Method, resp.StatusCode, resp.Request.URL)
							message := fmt.Sprintf("succeed %v %v ```json\r%v```", resp.Request.Method, resp.StatusCode, string(jsonBytes))
							m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
								Content: &message,
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
							req, err := http.NewRequest("GET", "http://"+*APIServer+"/api/ban/remove?id="+id, http.NoBody)
							if err != nil {
								log.Printf("リクエスト作成に失敗: %v", err)
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
							client := http.Client{}
							resp, err := client.Do(req)
							if err != nil {
								log.Printf("APIサーバーへのリクエスト送信に失敗: %v", err)
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
							defer resp.Body.Close()
							byteArray, _ := io.ReadAll(resp.Body)
							jsonBytes := ([]byte)(byteArray)
							log.Printf("succeed %v %v %v", resp.Request.Method, resp.StatusCode, resp.Request.URL)
							message := fmt.Sprintf("succeed %v %v ```json\r%v```", resp.Request.Method, resp.StatusCode, string(jsonBytes))
							m, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
								Content: &message,
							})
							log.Printf("message: %v", m.ID)
							if err != nil {
								log.Printf("例外: %v", err)
							}
						case "get":
							req, err := http.NewRequest("GET", "http://"+*APIServer+"/api/ban", http.NoBody)
							if err != nil {
								log.Printf("リクエスト作成に失敗: %v", err)
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
							client := http.Client{}
							resp, err := client.Do(req)
							if err != nil {
								log.Printf("APIサーバーへのリクエスト送信に失敗: %v", err)
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
							defer resp.Body.Close()
							byteArray, _ := io.ReadAll(resp.Body)
							jsonBytes := ([]byte)(byteArray)
							log.Printf("succeed %v %v %v", resp.Request.Method, resp.StatusCode, resp.Request.URL)
							data := &globalBan{}
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
			},
		}
	)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
	return
}
