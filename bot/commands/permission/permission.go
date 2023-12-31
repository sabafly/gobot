package permission

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "permission",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:         "permission",
				Description:  "permission",
				DMPermission: builtin.Ptr(false),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:        "set",
						Description: "set member/role permission",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionMentionable{
								Name:        "target",
								Description: "target member/role",
								Required:    true,
							},
							discord.ApplicationCommandOptionString{
								Name:        "permission",
								Description: "permission text",
								Required:    true,
								MinLength:   builtin.Ptr(1),
								MaxLength:   builtin.Ptr(100),
							},
							discord.ApplicationCommandOptionBool{
								Name:        "value",
								Description: "the value of permission default is true",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "unset",
						Description: "unset member/role permission",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionMentionable{
								Name:        "target",
								Description: "target member/role",
								Required:    true,
							},
							discord.ApplicationCommandOptionString{
								Name:        "permission",
								Description: "permission text",
								Required:    true,
								MinLength:   builtin.Ptr(1),
								MaxLength:   builtin.Ptr(100),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "check",
						Description: "check member/role permission",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionMentionable{
								Name:        "target",
								Description: "target member/role",
								Required:    true,
							},
							discord.ApplicationCommandOptionString{
								Name:        "permission",
								Description: "permission text",
								Required:    true,
								MinLength:   builtin.Ptr(1),
								MaxLength:   builtin.Ptr(100),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "list",
						Description: "list member/role permission",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionMentionable{
								Name:        "target",
								Description: "target member/role",
								Required:    true,
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/permission/set": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("permission.set"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					perm := event.SlashCommandInteractionData().String("permission")
					value, ok := event.SlashCommandInteractionData().OptBool("value")
					if !ok {
						value = true
					}
					var mention string
					if target, ok := event.SlashCommandInteractionData().OptMember("target"); ok {
						if target.User.Bot || target.User.System {
							return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
						}
						t, err := c.MemberCreate(event, target.User, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}
						t.Permission.Set(perm, value)
						t = t.Update().
							SetPermission(t.Permission).
							SaveX(event)
						mention = discord.UserMention(t.UserID)
					} else {
						role := event.SlashCommandInteractionData().Role("target")
						g, err := c.GuildCreateID(event, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}
						p := g.Permissions[role.ID]
						p.Set(perm, value)
						g.Permissions[role.ID] = p
						g.Update().SetPermissions(g.Permissions).ExecX(event)
						mention = discord.RoleMention(role.ID)
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.permission.set.message.embed.title")).
										SetDescriptionf("%s```yaml\n%s: %t\n```",
											mention, perm, value,
										).
										Build(),
								),
							).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/permission/unset": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("permission.unset"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					var mention string
					perm := event.SlashCommandInteractionData().String("permission")
					if target, ok := event.SlashCommandInteractionData().OptMember("target"); ok {
						if target.User.Bot || target.User.System {
							return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
						}
						t, err := c.MemberCreate(event, target.User, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}
						t.Permission.UnSet(perm)
						t = t.Update().
							SetPermission(t.Permission).
							SaveX(event)
						mention = discord.UserMention(t.UserID)
					} else {
						role := event.SlashCommandInteractionData().Role("target")
						g, err := c.GuildCreateID(event, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}
						p := g.Permissions[role.ID]
						p.UnSet(perm)
						g.Permissions[role.ID] = p
						g.Update().SetPermissions(g.Permissions).ExecX(event)
						mention = discord.RoleMention(role.ID)
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.permission.unset.message.embed.title")).
										SetDescriptionf("%s```yaml\n%s\n```",
											mention, perm,
										).
										Build(),
								),
							).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/permission/check": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("permission.check"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					var mention string
					var p bool
					perm := event.SlashCommandInteractionData().String("permission")
					if target, ok := event.SlashCommandInteractionData().OptMember("target"); ok {
						if target.User.Bot || target.User.System {
							return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
						}
						t, err := c.MemberCreate(event, target.User, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}
						mention = discord.UserMention(t.UserID)
						p = generic.PermissionCheck(event, c, g, event.Client(), target, *event.GuildID(), []generic.Permission{generic.PermissionString(perm)})
					} else {
						role := event.SlashCommandInteractionData().Role("target")
						mention = discord.RoleMention(role.ID)
						p = generic.RolePermissionCheck(g, *event.GuildID(), event.Client(), []snowflake.ID{role.ID}, []generic.Permission{generic.PermissionString(perm)})
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.permission.check.message.embed.title")).
										SetDescriptionf("%s```yaml\n%s: %t\n```",
											mention, perm, p,
										).
										Build(),
								),
							).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/permission/list": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("permission.check"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					var str, mention string
					if target, ok := event.SlashCommandInteractionData().OptMember("target"); ok {
						if target.User.Bot || target.User.System {
							return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
						}
						t, err := c.MemberCreate(event, target.User, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}
						mention, str = discord.UserMention(t.UserID), t.Permission.String()
					} else {
						role := event.SlashCommandInteractionData().Role("target")
						g, err := c.GuildCreateID(event, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}
						mention, str = discord.RoleMention(role.ID), g.Permissions[role.ID].String()
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.permission.list.message.embed.title")).
										SetDescriptionf(
											fmt.Sprintf("%s```yaml\n%s```",
												mention,
												builtin.Or(str != "", str, "Empty"),
											),
										).
										Build(),
								),
							).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
	}).SetComponent(c)
}
