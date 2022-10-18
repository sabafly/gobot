package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	GuildID        = flag.String("guild", "", "テストサーバーID")
	BotToken       = flag.String("Token", "", "botアクセストークン")
	RemoveCommands = flag.Bool("rmcmd", true, "停止時にコマンドを登録解除するか")
	ApplicationId  = flag.String("Application", "", "botのsnowflake")
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env:%v", err)
	}
	*BotToken = os.Getenv("TOKEN")
	*GuildID = os.Getenv("GUILD_ID")
	*ApplicationId = os.Getenv("APPLICATION_ID")
}

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("無効なbotパラメータ: %v", err)
	}
}

var (
	// integerOptionMinValue          = 1.0
	dmPermission = false
	// defaultMemberPermissions int64 = discordgo.PermissionManageServer
	PermissionBanMembers  int64 = discordgo.PermissionBanMembers
	PermissionKickMembers int64 = discordgo.PermissionKickMembers

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

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ban": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: commandBan(&i.Locale, i.ApplicationCommandData(), i.GuildID),
			})
			if err != nil {
				log.Printf("例外: %v", err)
			}
		},
		"unban": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: commandUnBan(&i.Locale, i.ApplicationCommandData(), i.GuildID),
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
				Data: commandKick(&i.Locale, i.ApplicationCommandData(), i.GuildID),
			})
			if err != nil {
				log.Printf("例外: %v", err)
			}
		},
		"admin": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			content := &discordgo.InteractionResponseData{}
			switch options[0].Name {
			case "sudo":
				options = options[0].Options
				switch options[0].Name {
				case "ban":
					for _, g := range s.State.Guilds {
						err := s.GuildBanCreateWithReason(g.ID, options[0].Options[0].StringValue(), "GoBot Global Ban", 7)
						if err != nil {
							content.Content = fmt.Sprintf("failed: %v", err)
						}
						s.ChannelMessageSend(i.ChannelID, fmt.Sprintf("guildId: %v target: %v", g.ID, options[0].Options[0].StringValue()))
						time.Sleep(time.Second)
					}
					content.Content = "done"
				default:
					content.Content = "Oops, something went wrong.\r" +
						"Hol' up, you aren't supposed to see this message."
				}
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: content,
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("%v#%v としてログインしました", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("セッションを開始できません: %v", err)
	}

	log.Println("コマンドを追加中...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("'%v'コマンドを追加できません: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	go updateStatus()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Ctrl + C で終了")
	<-stop

	if *RemoveCommands {
		log.Println("コマンドを登録解除中...")
		// // We need to fetch the commands, since deleting requires the command ID.
		// // We are doing this from the returned commands on line 375, because using
		// // this will delete all the commands, which might not be desirable, so we
		// // are deleting only the commands that we added.
		// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
		// if err != nil {
		// 	log.Fatalf("Could not fetch registered commands: %v", err)
		// }

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("'%v'コマンドを解除できません: %v", v.Name, err)
			}
		}
	}

	log.Println("正常にシャットダウンしました")
}

func updateStatus() {
	for {
		err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status: fmt.Sprintf("Shards %v / %v", s.ShardID+1, s.ShardCount),
			Activities: []*discordgo.Activity{
				{
					Name: fmt.Sprintf("Servers: %v,Shards: %v / %v", len(s.State.Guilds), s.ShardID+1, s.ShardCount),
					Type: discordgo.ActivityTypeWatching,
				},
			},
		})
		if err != nil {
			log.Printf("Error on update status: %v", err)
		}
		time.Sleep(time.Minute * 10)
	}
}
