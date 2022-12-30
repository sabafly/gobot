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
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/joho/godotenv"
)

var (
	BotToken       = flag.String("Token", "", "botアクセストークン")
	RemoveCommands = flag.Bool("rmcmd", true, "停止時にコマンドを登録解除するか")
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

	flag.Parse()

	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("無効なbotパラメータ: %v", err)
	}
	s.Identify.Intents = discordgo.IntentsAll

	var (
		dmPermission                 = false
		PermissionAdminMembers int64 = discordgo.PermissionManageServer
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
			Name:                     "panel",
			Description:              "manage or create panel",
			NameLocalizations:        translate.MessageMap("command_panel", true),
			DescriptionLocalizations: translate.MessageMap("command_panel_desc", false),
			GuildID:                  *SupportGuildID,
			DefaultMemberPermissions: &PermissionAdminMembers,
			DMPermission:             &dmPermission,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:                     "role",
					Description:              "manage role panel",
					NameLocalizations:        *translate.MessageMap("command_panel_option_role", true),
					DescriptionLocalizations: *translate.MessageMap("command_panel_option_desc_role", false),
					Type:                     discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:                     "create",
							Description:              "create role panel",
							NameLocalizations:        *translate.MessageMap("command_panel_option_role_option_create", true),
							DescriptionLocalizations: *translate.MessageMap("command_panel_option_role_option_desc_create", false),
							Type:                     discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:                     "name",
									Description:              "name of panel",
									NameLocalizations:        *translate.MessageMap("command_panel_option_role_option_create_option_name", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_role_option_create_option_desc_name", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
								},
								{
									Name:                     "description",
									Description:              "description of panel",
									NameLocalizations:        *translate.MessageMap("command_panel_option_role_option_create_option_desc", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_role_option_create_option_desc_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
								},
							},
						},
					},
				},
				{
					Name:                     "minecraft",
					Description:              "manage minecraft panel",
					NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft", true),
					DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_desc", false),
					Type:                     discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:                     "create",
							Description:              "create minecraft panel",
							NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create", true),
							DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_desc", false),
							Type:                     discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:                     "name",
									Description:              "name of panel",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_name", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_name_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
								},
								{
									Name:                     "servername",
									Description:              "name of server",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_servername", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_servername_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
								},
								{
									Name:                     "address",
									Description:              "address of server",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_address", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_address_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
								},
								{
									Name:                     "port",
									Description:              "port of server",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_port", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_port_desc", false),
									Type:                     discordgo.ApplicationCommandOptionInteger,
									Required:                 true,
								},
								{
									Name:                     "description",
									Description:              "description of panel",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_description", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_description_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
								},
								{
									Name:                     "showip",
									Description:              "show ip or not",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_showip", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_showip_desc", false),
									Type:                     discordgo.ApplicationCommandOptionBoolean,
								},
							},
						},
					},
				},
				{
					Name:                     "config",
					Description:              "test",
					NameLocalizations:        *translate.MessageMap("command_panel_option_config", true),
					DescriptionLocalizations: *translate.MessageMap("command_panel_option_config_desc", false),
					Type:                     discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:                     "emoji",
							Description:              "test",
							NameLocalizations:        *translate.MessageMap("command_panel_option_config_option_emoji", true),
							DescriptionLocalizations: *translate.MessageMap("command_panel_option_config_option_emoji_desc", false),
							Type:                     discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
			},
		},
		{
			Name:                     "tracker",
			Description:              "manage or create tracker",
			NameLocalizations:        translate.MessageMap("command_tracker", true),
			DescriptionLocalizations: translate.MessageMap("command_tracker_desc", false),
			DefaultMemberPermissions: &PermissionAdminMembers,
			DMPermission:             &dmPermission,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:                     "minecraft",
					Description:              "manage minecraft server tracker",
					NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft", true),
					DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_desc", false),
					Type:                     discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:                     "create",
							Description:              "create minecraft server tracker",
							NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_create", true),
							DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_create_desc", false),
							Type:                     discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:                     "name",
									Description:              "name of server",
									NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_create_option_name", true),
									DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_create_option_name_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
								},
								{
									Name:                     "address",
									Description:              "address of server",
									NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_create_option_address", true),
									DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_create_option_address_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
								},
								{
									Name:                     "port",
									Description:              "port of server",
									NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_create_option_port", true),
									DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_create_option_port_desc", false),
									Type:                     discordgo.ApplicationCommandOptionInteger,
									Required:                 true,
								},
								{
									Name:                     "role",
									Description:              "role to mention",
									NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_create_option_role", true),
									DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_create_option_role_desc", false),
									Type:                     discordgo.ApplicationCommandOptionRole,
								},
							},
						},
						{
							Name:                     "get",
							Description:              "test",
							NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_get", true),
							DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_get_desc", false),
							Type:                     discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:                     "remove",
							Description:              "test",
							NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_remove", true),
							DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_remove_desc", false),
							Type:                     discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:                     "name",
									Description:              "test",
									Type:                     discordgo.ApplicationCommandOptionString,
									NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_remove_option_name", true),
									DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_remove_option_name_desc", false),
									Required:                 true,
								},
							},
						},
					},
				},
			},
		},
		{
			Name:                     "role",
			Description:              "manage role",
			NameLocalizations:        translate.MessageMap("command_role", true),
			DescriptionLocalizations: translate.MessageMap("command_role_desc", false),
			DefaultMemberPermissions: &PermissionAdminMembers,
			DMPermission:             &dmPermission,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:                     "color",
					Description:              "create color role",
					NameLocalizations:        *translate.MessageMap("command_role_option_color", true),
					DescriptionLocalizations: *translate.MessageMap("command_role_option_color_desc", false),
					Type:                     discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:                     "rgb",
							Description:              "rgb color code",
							NameLocalizations:        *translate.MessageMap("command_role_option_color_option_rgb", true),
							DescriptionLocalizations: *translate.MessageMap("command_role_option_color_option_rgb_desc", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							Required:                 true,
						},
						{
							Name:                     "name",
							Description:              "name of role",
							NameLocalizations:        *translate.MessageMap("command_role_option_color_option_name", true),
							DescriptionLocalizations: *translate.MessageMap("command_role_option_color_option_name_desc", false),
							Type:                     discordgo.ApplicationCommandOptionString,
						},
					},
				},
			},
		},
		{
			Name:                     "modify",
			NameLocalizations:        translate.MessageMap("message_command_modify", true),
			Type:                     discordgo.MessageApplicationCommand,
			DMPermission:             &dmPermission,
			DefaultMemberPermissions: &PermissionAdminMembers,
		},
		{
			Name:              "info",
			NameLocalizations: translate.MessageMap("message_command_user_info", true),
			Type:              discordgo.UserApplicationCommand,
			DMPermission:      &dmPermission,
		},
		{
			Name:              "select",
			NameLocalizations: translate.MessageMap("message_command_select", true),
			Type:              discordgo.MessageApplicationCommand,
			DMPermission:      &dmPermission,
		},
	}
	var (
		commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
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
			"admin": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.Admin(s, i)
			},
			"panel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.Panel(s, i)
			},
			"tracker": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.Feed(s, i)
			},
			"role": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.Role(s, i)
			},
			"modify": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.MModify(s, i)
			},
			"select": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.MSelect(s, i)
			},
			"info": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
				command.UInfo(s, i)
			},
		}
	)

	messageComponentHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string){
		"gobot_panel_role": func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.MCpanelRole(s, i)
		},
		"gobot_panel_role_add": func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.MCpanelRoleAdd(s, i, sessionID)
		},
		"gobot_panel_role_create": func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.MCpanelRoleCreate(s, i, sessionID)
		},
		"gobot_panel_minecraft": func(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
			command.MCpanelMinecraft(s, i)
		},
	}

	modalSubmitHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, mid string){
		"gobot_panel_minecraft_add_modal": func(s *discordgo.Session, i *discordgo.InteractionCreate, mid string) {
			command.MSminecraftPanel(s, i, mid)
		},
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		p, err := s.State.UserChannelPermissions(s.State.User.ID, i.ChannelID)
		if err == nil && p&int64(discordgo.PermissionAdministrator) != 0 {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
				return
			case discordgo.InteractionMessageComponent:
				ids := strings.Split(i.MessageComponentData().CustomID, ":")
				var customID string
				var sessionID string
				for i2, v := range ids {
					switch i2 {
					case 0:
						customID = v
					case 1:
						sessionID = v
					}
				}
				if c, ok := messageComponentHandlers[customID]; ok {
					c(s, i, sessionID)
				}
				return
			case discordgo.InteractionModalSubmit:
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
				return
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: translate.Message(i.Locale, "error_unknown_command"),
				},
			})
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: translate.Message(i.Locale, "error_bot_does_not_have_permissions"),
				},
			})
		}
	})

	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		str, err := m.ContentWithMoreMentionsReplaced(s)
		if err != nil {
			str = m.Content
		}
		g, _ := s.Guild(m.GuildID)
		c, _ := s.Channel(m.ChannelID)
		log.Printf("[Message Created] : %v(%v) #%v(%v) <%v#%v>\n                 >> %v", g.Name, g.ID, c.Name, c.ID, m.Author.Username, m.Author.Discriminator, str)
		p, err := s.State.UserChannelPermissions(s.State.User.ID, m.ChannelID)
		if err == nil && p&int64(discordgo.PermissionAdministrator) == 0 {
			data, err := session.MessagePanelConfigEmojiLoad(m.Author.ID)
			if err != nil {
				log.Print(err)
				return
			} else {
				d := data.Data()
				data.Data().Handler(d, s, m)
			}
		}
	})

	return s, commands, RemoveCommands, GuildID
}

func GetSession() *discordgo.Session {
	return s
}
