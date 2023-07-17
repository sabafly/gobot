package commands

import (
	"github.com/disgoorg/json"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/permissions"
)

func Permission(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:        "permission",
			Description: "permission",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "add",
					Description: "add permission to target",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionMentionable{
							Name:        "target",
							Description: "target to add permission",
							Required:    true,
						},
						discord.ApplicationCommandOptionString{
							Name:         "permission",
							Description:  "permission string",
							Required:     true,
							Autocomplete: true,
							MaxLength:    json.Ptr(32),
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "remove",
					Description: "remove permission to target",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionMentionable{
							Name:        "target",
							Description: "target to add permission",
							Required:    true,
						},
						discord.ApplicationCommandOptionString{
							Name:         "permission",
							Description:  "permission string",
							Required:     true,
							Autocomplete: true,
							MaxLength:    json.Ptr(32),
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "list",
					Description: "show list os target permission",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionMentionable{
							Name:        "target",
							Description: "target to show permissions",
							Required:    true,
						},
					},
				},
			},
			DMPermission: &b.Config.DMPermission,
		},
		Checks: map[string]handler.Check[*events.ApplicationCommandInteractionCreate]{
			"add":    b.Self.CheckCommandPermission(b, "guild.permissions.manage", discord.PermissionManageGuild),
			"remove": b.Self.CheckCommandPermission(b, "guild.permissions.manage", discord.PermissionManageGuild),
			"list":   b.Self.CheckCommandPermission(b, "guild.permissions.manage", discord.PermissionManageGuild),
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"add":    permissionAddCommandHandler(b),
			"remove": permissionRemoveCommandHandler(b),
			"list":   permissionListCommandHandler(b),
		},
	}
}

func permissionAddCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		perm := event.SlashCommandInteractionData().String("permission")
		var mention string
		if user, ok := event.SlashCommandInteractionData().OptUser("target"); ok {
			if user.Bot || user.System {
				return botlib.ReturnErrMessage(event, "error_is_bot")
			}
			if gd.UserPermissions[user.ID] == nil {
				gd.UserPermissions[user.ID] = permissions.New()
			}
			gd.UserPermissions[user.ID].Add(perm)
			mention = discord.UserMention(user.ID)
		} else if role, ok := event.SlashCommandInteractionData().OptRole("target"); ok {
			if role.Managed {
				return botlib.ReturnErrMessage(event, "error_invalid_role")
			}
			if gd.RolePermissions[role.ID] == nil {
				gd.RolePermissions[role.ID] = permissions.New()
			}
			gd.RolePermissions[role.ID].Add(perm)
			mention = discord.RoleMention(role.ID)
		} else {
			return botlib.ReturnErrMessage(event, "error_invalid_command_argument")
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitlef("Permission added")
		embed.SetDescriptionf("To %s ```diff\r+ %s```", mention, perm)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func permissionRemoveCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		perm := event.SlashCommandInteractionData().String("permission")
		var mention string
		if user, ok := event.SlashCommandInteractionData().OptUser("target"); ok {
			if user.Bot || user.System {
				return botlib.ReturnErrMessage(event, "error_is_bot")
			}
			gd.UserPermissions[user.ID].Del(perm)
			mention = discord.UserMention(user.ID)
		} else if role, ok := event.SlashCommandInteractionData().OptRole("target"); ok {
			if role.Managed {
				return botlib.ReturnErrMessage(event, "error_invalid_role")
			}
			gd.RolePermissions[role.ID].Del(perm)
			mention = discord.RoleMention(role.ID)
		} else {
			return botlib.ReturnErrMessage(event, "error_invalid_command_argument")
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitlef("Permission removed")
		embed.SetDescriptionf("From %s ```diff\r- %s```", mention, perm)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func permissionListCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		var mention string
		var permissions []string
		if user, ok := event.SlashCommandInteractionData().OptUser("target"); ok {
			if user.Bot || user.System {
				return botlib.ReturnErrMessage(event, "error_is_bot")
			}
			permissions = gd.UserPermissions[user.ID].List()
			mention = user.EffectiveName()
		} else if role, ok := event.SlashCommandInteractionData().OptRole("target"); ok {
			if role.Managed {
				return botlib.ReturnErrMessage(event, "error_invalid_role")
			}
			permissions = gd.RolePermissions[role.ID].List()
			mention = role.Name
		} else {
			return botlib.ReturnErrMessage(event, "error_invalid_command_argument")
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitlef("%s's Permissions", mention)
		for _, v := range permissions {
			embed.Description += v + "\r"
		}
		embed.SetDescriptionf("```%s ```", embed.Description)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}
