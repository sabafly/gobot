package setup

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command"
	"github.com/ikafly144/gobot/pkg/util"
	"github.com/joho/godotenv"
)

var (
	GuildID        = flag.String("guild", "", "テストサーバーID")
	BotToken       = flag.String("Token", "", "botアクセストークン")
	RemoveCommands = flag.Bool("rmcmd", true, "停止時にコマンドを登録解除するか")
	ApplicationId  = flag.String("Application", "", "botのsnowflake")
)

func Setup() (s *discordgo.Session, commands []*discordgo.ApplicationCommand, RemoveCommands bool, GuildID string) {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env:%v", err)
	}
	*BotToken = os.Getenv("TOKEN")
	// *GuildID = os.Getenv("GUILD_ID")
	*ApplicationId = os.Getenv("APPLICATION_ID")

	flag.Parse()

	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("無効なbotパラメータ: %v", err)
	}

	var (
		// integerOptionMinValue          = 1.0
		dmPermission = false
		// defaultMemberPermissions int64 = discordgo.PermissionManageServer
		PermissionBanMembers  int64 = discordgo.PermissionBanMembers
		PermissionKickMembers int64 = discordgo.PermissionKickMembers
	)
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "pong!",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "ピング",
			},
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "ポング！",
			},
		},
		{
			Name:        "ban",
			Description: "ban the selected user",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "追放",
			},
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
		},
		{
			Name:        "unban",
			Description: "pardon the selected user",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "免罪",
			},
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
		},
		{
			Name:        "kick",
			Description: "kick the selected user",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "切断",
			},
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
		},
		{
			Name:        "admin",
			Description: "for only admins",
			GuildID:     "1005139879799291936",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "sudo",
					Description: "for only admins",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "ban",
							Description: "for only admins",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "target",
									Description: "for only admins",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
							},
						},
					},
				},
			},
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
				il := &discordgo.InteractionCreate{}
				util.DeepcopyJson(i, il)
				err := s.InteractionRespond(il.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "done",
					},
				})
				if err != nil {
					log.Printf("例外: %v", err)
				}
				options := i.ApplicationCommandData().Options
				var c []string
				switch options[0].Name {
				case "sudo":
					options = options[0].Options
					switch options[0].Name {
					case "ban":
						for _, g := range s.State.Guilds {
							err := s.GuildBanCreateWithReason(g.ID, options[0].Options[0].StringValue(), "GoBot Global Ban", 7)
							if err != nil {
								log.Printf("%v\n%v", i.ChannelID, fmt.Sprintf("failed: %v", err))
							}
							c = append(c, fmt.Sprintf("guildId: %v target: %v", g.ID, options[0].Options[0].StringValue()))
							time.Sleep(time.Second)
						}
					default:
					}
				}
				for _, d := range c {
					s.ChannelMessageSend(i.ChannelID, d)
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
