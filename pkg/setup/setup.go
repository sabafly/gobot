package setup

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/joho/godotenv"
)

var (
	BotToken       = flag.String("Token", "", "botアクセストークン")
	RemoveCommands = flag.Bool("rmcmd", true, "停止時にコマンドを登録解除するか")
	ApplicationId  = flag.String("Application", "", "botのsnowflake")
	SupportGuildID = flag.String("SupportServer", "", "サポートサーバーのID")
	APIServer      = flag.String("APIAddress", "", "APIサーバーのip")
	s              *discordgo.Session
)

func Setup() (*discordgo.Session, []*discordgo.ApplicationCommand, bool, string) {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env:%v", err)
	}
	*BotToken = os.Getenv("TOKEN")
	GuildID := os.Getenv("GUILD_ID")
	*SupportGuildID = os.Getenv("SUPPORT_ID")
	RemoveCommands, err := strconv.ParseBool(os.Getenv("REMOVE_COMMANDS"))
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
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "pong!",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "ポング！",
			},
			Version: "1",
		},
		{
			Name:                     "ban",
			Description:              "ban the selected user",
			DescriptionLocalizations: translate.MessageMap("command.ban.description"),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:                     "target",
					Description:              "user to ban",
					NameLocalizations:        *translate.MessageMap("command.ban.option.target"),
					DescriptionLocalizations: *translate.MessageMap("command.ban.option.description.target"),
					Type:                     discordgo.ApplicationCommandOptionUser,
					Required:                 true,
				},
				{
					Name:                     "reason",
					Description:              "reason for ban",
					NameLocalizations:        *translate.MessageMap("command.ban.option.reason"),
					DescriptionLocalizations: *translate.MessageMap("command.ban.option.description.reason"),
					Type:                     discordgo.ApplicationCommandOptionString,
				},
			},
			DefaultMemberPermissions: &PermissionBanMembers,
			DMPermission:             &dmPermission,
			Version:                  "1",
		},
		{
			Name:                     "unban",
			Description:              "pardon the selected user",
			DescriptionLocalizations: translate.MessageMap("command.unban.description"),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:                     "target",
					Description:              "user to pardon",
					NameLocalizations:        *translate.MessageMap("command.unban.option.target"),
					DescriptionLocalizations: *translate.MessageMap("command.unban.option.description.target"),
					Type:                     discordgo.ApplicationCommandOptionUser,
					Required:                 true,
				},
			},
			DefaultMemberPermissions: &PermissionBanMembers,
			DMPermission:             &dmPermission,
			Version:                  "1",
		},
		{
			Name:                     "kick",
			Description:              "kick the selected user",
			DescriptionLocalizations: translate.MessageMap("command.kick.description"),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:                     "target",
					Description:              "user to kick",
					NameLocalizations:        *translate.MessageMap("command.kick.option.target"),
					DescriptionLocalizations: *translate.MessageMap("command.kick.option.description.target"),
					Type:                     discordgo.ApplicationCommandOptionUser,
					Required:                 true,
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
		{
			Name:                     "panel",
			Description:              "manage or create panel",
			GuildID:                  *SupportGuildID,
			DefaultMemberPermissions: &PermissionAdminMembers,
			DMPermission:             &dmPermission,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "role",
					Description: "role panel",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "create",
							Description: "create role panel",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "name",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "role",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionRole,
									Required:    true,
								},
								{
									Name:        "description",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionString,
								},
							},
						},
					},
				},
				{
					Name:        "minecraft",
					Description: "test",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "create",
							Description: "test",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "name",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "servername",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "address",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "port",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
								{
									Name:        "description",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionString,
								},
								{
									Name:        "showip",
									Description: "test",
									Type:        discordgo.ApplicationCommandOptionBoolean,
								},
							},
						},
					},
				},
			},
		},
		{
			Name:                     "modify",
			Type:                     discordgo.MessageApplicationCommand,
			DMPermission:             &dmPermission,
			DefaultMemberPermissions: &PermissionAdminMembers,
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
					discordgo.Japanese: "ポング！\r" + s.HeartbeatLatency().String(),
				}
				content := "pong!\r" + s.HeartbeatLatency().String()
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
				command.Admin(s, i)
			},
			"panel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.Panel(s, i)
			},
			"modify": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.Mmodify(s, i)
			},
		}
	)

	messageComponentHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"gobot_panel_role": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.MCpanelRole(s, i)
		},
		"gobot_panel_role_add": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.MCpanelRoleAdd(s, i)
		},
		"gobot_panel_minecraft": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			command.MCpanelMinecraft(s, i)
		},
	}

	modalSubmitHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, mid string){
		"gobot_panel_minecraft_add_modal": func(s *discordgo.Session, i *discordgo.InteractionCreate, mid string) {
			command.MSminecraftPanel(s, i, mid)
		},
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		} else if i.Type == discordgo.InteractionMessageComponent {
			if c, ok := messageComponentHandlers[i.MessageComponentData().CustomID]; ok {
				c(s, i)
			}
		} else if i.Type == discordgo.InteractionModalSubmit {
			ids := strings.Split(i.ModalSubmitData().CustomID, ":")
			var customID string
			var mid string
			for i2, v := range ids {
				switch i2 {
				case 0:
					customID = v
				case 1:
					mid = v
				}
			}
			if m, ok := modalSubmitHandlers[customID]; ok {
				m(s, i, mid)
			}
		}
	})
	return s, commands, RemoveCommands, GuildID
}

func GetSession() *discordgo.Session {
	return s
}
