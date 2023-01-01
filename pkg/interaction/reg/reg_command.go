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
package reg

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/env"
	"github.com/ikafly144/gobot/pkg/translate"
)

var (
	dmPermission                   = false
	PermissionAdminMembers   int64 = discordgo.PermissionManageServer
	PermissionManageMessages int64 = discordgo.PermissionManageMessages
	two                            = 2
	six                            = 6
	eight                          = 8
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "pong!",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Japanese: "ポング！",
			},
			Version:      "1",
			DMPermission: &dmPermission,
		},
		{
			Name:                     "panel",
			Description:              "manage or create panel",
			NameLocalizations:        translate.MessageMap("command_panel", true),
			DescriptionLocalizations: translate.MessageMap("command_panel_desc", false),
			GuildID:                  *env.SupportGuildID,
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
									MaxLength:                32,
								},
								{
									Name:                     "description",
									Description:              "description of panel",
									NameLocalizations:        *translate.MessageMap("command_panel_option_role_option_create_option_desc", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_role_option_create_option_desc_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									MaxLength:                256,
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
									MaxLength:                32,
								},
								{
									Name:                     "servername",
									Description:              "name of server",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_servername", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_servername_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
									MaxLength:                16,
								},
								{
									Name:                     "address",
									Description:              "address of server",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_address", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_address_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
									MaxLength:                32,
								},
								{
									Name:                     "port",
									Description:              "port of server",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_port", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_port_desc", false),
									Type:                     discordgo.ApplicationCommandOptionInteger,
									Required:                 true,
									MaxValue:                 1 << 16,
								},
								{
									Name:                     "description",
									Description:              "description of panel",
									NameLocalizations:        *translate.MessageMap("command_panel_option_minecraft_option_create_option_description", true),
									DescriptionLocalizations: *translate.MessageMap("command_panel_option_minecraft_option_create_option_description_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									MaxLength:                256,
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
									MaxLength:                16,
								},
								{
									Name:                     "address",
									Description:              "address of server",
									NameLocalizations:        *translate.MessageMap("command_tracker_option_minecraft_option_create_option_address", true),
									DescriptionLocalizations: *translate.MessageMap("command_tracker_option_minecraft_option_create_option_address_desc", false),
									Type:                     discordgo.ApplicationCommandOptionString,
									Required:                 true,
									MaxLength:                32,
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
									MaxLength:                16,
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
							MaxLength:                6,
						},
						{
							Name:                     "name",
							Description:              "name of role",
							NameLocalizations:        *translate.MessageMap("command_role_option_color_option_name", true),
							DescriptionLocalizations: *translate.MessageMap("command_role_option_color_option_name_desc", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MaxLength:                100,
						},
					},
				},
			},
		},
		{
			Name:                     "message",
			Description:              "test",
			NameLocalizations:        translate.MessageMap("command_message", true),
			DescriptionLocalizations: translate.MessageMap("command_message_description", false),
			DMPermission:             &dmPermission,
			DefaultMemberPermissions: &PermissionManageMessages,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:                     "embed",
					Description:              "create embed message with webhook",
					NameLocalizations:        *translate.MessageMap("command_message_embed", true),
					DescriptionLocalizations: *translate.MessageMap("command_message_embed_description", false),
					Type:                     discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:                     "embed_title",
							Description:              "title of message embed",
							NameLocalizations:        *translate.MessageMap("command_message_embed_embed_title", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_embed_title_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							Required:                 true,
							MaxLength:                256,
						},
						{
							Name:                     "embed_content",
							Description:              "content of message embed",
							NameLocalizations:        *translate.MessageMap("command_message_embed_embed_content", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_embed_content_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							Required:                 true,
							MaxLength:                4096,
						},
						{
							Name:                     "username",
							Description:              "name of message sender",
							NameLocalizations:        *translate.MessageMap("command_message_embed_username", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_username_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MinLength:                &two,
							MaxLength:                32,
						},
						{
							Name:                     "icon_url",
							Description:              "url of user avatar",
							NameLocalizations:        *translate.MessageMap("command_message_embed_icon_url", false),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_icon_url_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MinLength:                &eight,
							MaxLength:                512,
						},
						{
							Name:                     "content",
							Description:              "content of embed message",
							NameLocalizations:        *translate.MessageMap("command_message_embed_content", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_content_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MaxLength:                2000,
						},
						{
							Name:                     "color",
							Description:              "color of message embed",
							NameLocalizations:        *translate.MessageMap("command_message_embed_color", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_color_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MaxLength:                6,
							MinLength:                &six,
						},
					},
				},
				{
					Name:                     "webhook",
					Description:              "test",
					NameLocalizations:        *translate.MessageMap("command_message_webhook", true),
					DescriptionLocalizations: *translate.MessageMap("command_message_webhook_description", false),
					Type:                     discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:                     "content",
							Description:              "content of embed message",
							NameLocalizations:        *translate.MessageMap("command_message_embed_content", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_content_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							Required:                 true,
							MaxLength:                2000,
						},
						{
							Name:                     "username",
							Description:              "name of message sender",
							NameLocalizations:        *translate.MessageMap("command_message_embed_username", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_username_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MinLength:                &two,
							MaxLength:                32,
						},
						{
							Name:                     "icon_url",
							Description:              "url of user avatar",
							NameLocalizations:        *translate.MessageMap("command_message_embed_icon_url", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_icon_url_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MinLength:                &eight,
							MaxLength:                512,
						},
						{
							Name:                     "embed_title",
							Description:              "title of message embed",
							NameLocalizations:        *translate.MessageMap("command_message_embed_embed_title", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_embed_title_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MaxLength:                256,
						},
						{
							Name:                     "embed_content",
							Description:              "content of message embed",
							NameLocalizations:        *translate.MessageMap("command_message_embed_embed_content", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_embed_content_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MaxLength:                4096,
						},
						{
							Name:                     "color",
							Description:              "color of message embed",
							NameLocalizations:        *translate.MessageMap("command_message_embed_color", true),
							DescriptionLocalizations: *translate.MessageMap("command_message_embed_color_description", false),
							Type:                     discordgo.ApplicationCommandOptionString,
							MaxLength:                6,
							MinLength:                &six,
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
		{
			Name:                     "pin message",
			NameLocalizations:        translate.MessageMap("message_command_pin", false),
			Type:                     discordgo.MessageApplicationCommand,
			DMPermission:             &dmPermission,
			DefaultMemberPermissions: &PermissionManageMessages,
		},
	}
)

func Commands() []*discordgo.ApplicationCommand {
	return commands
}
