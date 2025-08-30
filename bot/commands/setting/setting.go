/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package setting

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
	"github.com/sabafly/gobot/internal/translate"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "setting",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "settings",
				Description:              "view settings",
				DescriptionLocalizations: translate.MessageMap("components.settings", false),
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
			},
			discord.SlashCommandCreate{
				Name:        "setting",
				Description: "setting",
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "bump",
						Description: "bump",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "toggle",
								Description:              "toggle",
								DescriptionLocalizations: translate.MessageMap("components.setting.bump.toggle", false),
							},
							{
								Name:                     "message",
								Description:              "set message",
								DescriptionLocalizations: translate.MessageMap("components.setting.bump.message", false),
							},
							{
								Name:                     "mention",
								Description:              "set mention target",
								DescriptionLocalizations: translate.MessageMap("components.setting.bump.mention", false),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionRole{
										Name:                     "target",
										Description:              "target role",
										DescriptionLocalizations: translate.MessageMap("components.setting.mention.target", false),
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "up",
						Description: "up",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "toggle",
								Description:              "toggle",
								DescriptionLocalizations: translate.MessageMap("components.setting.up.toggle", false),
							},
							{
								Name:                     "message",
								Description:              "set message",
								DescriptionLocalizations: translate.MessageMap("components.setting.up.message", false),
							},
							{
								Name:                     "mention",
								Description:              "set mention target",
								DescriptionLocalizations: translate.MessageMap("components.setting.up.mention", false),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionRole{
										Name:                     "target",
										Description:              "target role",
										DescriptionLocalizations: translate.MessageMap("components.setting.mention.target", false),
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "leveling",
						Description: "leveling",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "toggle",
								Description:              "toggle leveling",
								DescriptionLocalizations: translate.MessageMap("components.setting.leveling.toggle", false),
							},
						},
					},
					// discord.ApplicationCommandOptionSubCommandGroup{
					// 	Name:        "welcome",
					// 	Description: "welcome",
					// 	Options: []discord.ApplicationCommandOptionSubCommand{
					// 		{
					// 			Name:                     "set-message",
					// 			Description:              "set message",
					// 			DescriptionLocalizations: translate.MessageMap("components.setting.welcome.set-message", false),
					// 		},
					// 		{
					// 			Name:                     "set-channel",
					// 			Description:              "set channel",
					// 			DescriptionLocalizations: translate.MessageMap("components.setting.welcome.set-channel", false),
					// 			Options: []discord.ApplicationCommandOption{
					// 				discord.ApplicationCommandOptionChannel{
					// 					Name:                     "channel",
					// 					Description:              "channel",
					// 					DescriptionLocalizations: translate.MessageMap("components.setting.welcome.channel", false),
					// 				},
					// 			},
					// 		},
					// 	},
					// },
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/settings": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.view"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					l := i18n.TranslateLayout(event.Locale(), "command.settings.view")
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetIsComponentsV2(true).
							SetComponents(i18n.BuildContext().
								WithText("guild_name", g.Name).
								WithText("guild_id", g.ID.String()).
								WithText("bump_enabled", builtin.Or(g.BumpEnabled, i18n.TranslateText(event.Locale(), "general.state.enabled"), i18n.TranslateText(event.Locale(), "general.state.disabled"))).
								WithText("bump_mention", builtin.Or(g.BumpMention != nil, discord.RoleMention(builtin.NonNil(g.BumpMention)), i18n.TranslateText(event.Locale(), "general.value.none"))).
								WithText("bump_message_title", g.BumpMessageTitle).
								WithText("bump_message", g.BumpMessage).
								WithText("bump_remind_message_title", g.BumpRemindMessageTitle).
								WithText("bump_remind_message", g.BumpRemindMessage).
								WithText("up_enabled", builtin.Or(g.UpEnabled, i18n.TranslateText(event.Locale(), "general.state.enabled"), i18n.TranslateText(event.Locale(), "general.state.disabled"))).
								WithText("up_mention", builtin.Or(g.UpMention != nil, discord.RoleMention(builtin.NonNil(g.UpMention)), i18n.TranslateText(event.Locale(), "general.value.none"))).
								WithText("up_message_title", g.UpMessageTitle).
								WithText("up_message", g.UpMessage).
								WithText("up_remind_message_title", g.UpRemindMessageTitle).
								WithText("up_remind_message", g.UpRemindMessage).
								WithText("leveling_enabled", builtin.Or(!g.LevelingDisabled, i18n.TranslateText(event.Locale(), "general.state.enabled"), i18n.TranslateText(event.Locale(), "general.state.disabled"))).
								Translate(l)...).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/bump/toggle": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.bump.toggle"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g = g.Update().
						SetBumpEnabled(!g.BumpEnabled).
						SaveX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.setting.bump.toggle."+builtin.Or(g.BumpEnabled, "enabled", "disabled"))).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/up/toggle": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.up.toggle"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g = g.Update().
						SetUpEnabled(!g.UpEnabled).
						SaveX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.setting.up.toggle."+builtin.Or(g.UpEnabled, "enabled", "disabled"))).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/bump/mention": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.bump.mention"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					update := g.Update()
					if r, ok := event.SlashCommandInteractionData().OptRole("target"); ok {
						update.SetBumpMention(r.ID)
					} else {
						update.ClearBumpMention()
					}
					g = update.SaveX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.setting.bump.mention.used",
								translate.WithTemplate(map[string]any{
									"Role": builtin.Or(g.BumpMention != nil,
										discord.RoleMention(builtin.NonNil(g.BumpMention)),
										"`"+translate.Message(event.Locale(), "components.setting.mention.none")+"`",
									),
								}),
							)).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/up/mention": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.up.mention"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					update := g.Update()
					if r, ok := event.SlashCommandInteractionData().OptRole("target"); ok {
						update.SetUpMention(r.ID)
					} else {
						update.ClearUpMention()
					}
					g = update.SaveX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.setting.up.mention.used",
								translate.WithTemplate(map[string]any{
									"Role": builtin.Or(g.UpMention != nil,
										discord.RoleMention(builtin.NonNil(g.UpMention)),
										"`"+translate.Message(event.Locale(), "components.setting.mention.none")+"`",
									),
								}),
							)).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/bump/message": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.bump.message"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if err := event.Modal(
						discord.NewModalCreateBuilder().
							SetTitle(translate.Message(event.Locale(), "components.setting.bump.message.modal.title")).
							SetCustomID("setting:bump_message").
							SetComponents(
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "message_title",
										Style:     discord.TextInputStyleShort,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.message_title"),
										MinLength: builtin.Ptr(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.BumpMessageTitle,
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "message",
										Style:     discord.TextInputStyleParagraph,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.message"),
										MinLength: builtin.Ptr(1),
										MaxLength: 300,
										Required:  true,
										Value:     g.BumpMessage,
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "remind.message_title",
										Style:     discord.TextInputStyleShort,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.remind.message_title"),
										MinLength: builtin.Ptr(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.BumpRemindMessageTitle,
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "remind.message",
										Style:     discord.TextInputStyleParagraph,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.remind.message"),
										MinLength: builtin.Ptr(1),
										MaxLength: 300,
										Required:  true,
										Value:     g.BumpRemindMessage,
									},
								),
							).
							Build(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/up/message": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.up.message"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if err := event.Modal(
						discord.NewModalCreateBuilder().
							SetTitle(translate.Message(event.Locale(), "components.setting.up.message.modal.title")).
							SetCustomID("setting:up_message").
							SetComponents(
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "message_title",
										Style:     discord.TextInputStyleShort,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.message_title"),
										MinLength: builtin.Ptr(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.UpMessageTitle,
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "message",
										Style:     discord.TextInputStyleParagraph,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.message"),
										MinLength: builtin.Ptr(1),
										MaxLength: 300,
										Required:  true,
										Value:     g.UpMessage,
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "remind.message_title",
										Style:     discord.TextInputStyleShort,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.remind.message_title"),
										MinLength: builtin.Ptr(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.UpRemindMessageTitle,
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "remind.message",
										Style:     discord.TextInputStyleParagraph,
										Label:     translate.Message(event.Locale(), "components.setting.message.modal.remind.message"),
										MinLength: builtin.Ptr(1),
										MaxLength: 300,
										Required:  true,
										Value:     g.UpRemindMessage,
									},
								),
							).
							Build(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/setting/welcome/set-channel": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.welcome.set-channel"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {

					return nil
				},
			},
			"/setting/leveling/toggle": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("setting.leveling.toggle"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g = g.Update().
						SetLevelingDisabled(!g.LevelingDisabled).
						SaveX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.setting.leveling.enable."+builtin.Or(!g.LevelingDisabled, "enabled", "disabled"))).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"setting:bump_message": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				g.Update().
					SetBumpMessageTitle(event.ModalSubmitInteraction.Data.Text("message_title")).
					SetBumpMessage(event.ModalSubmitInteraction.Data.Text("message")).
					SetBumpRemindMessageTitle(event.ModalSubmitInteraction.Data.Text("remind.message_title")).
					SetBumpRemindMessage(event.ModalSubmitInteraction.Data.Text("remind.message")).
					ExecX(event)
				if err := event.DeferUpdateMessage(); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
			"setting:up_message": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				g.Update().
					SetUpMessageTitle(event.ModalSubmitInteraction.Data.Text("message_title")).
					SetUpMessage(event.ModalSubmitInteraction.Data.Text("message")).
					SetUpRemindMessageTitle(event.ModalSubmitInteraction.Data.Text("remind.message_title")).
					SetUpRemindMessage(event.ModalSubmitInteraction.Data.Text("remind.message")).
					ExecX(event)
				if err := event.DeferUpdateMessage(); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
		},

		EventHandler: func(c *components.Components, event bot.Event) errors.Error {
			if e, ok := event.(*events.GuildMessageUpdate); ok {
				event = &events.GuildMessageCreate{GenericGuildMessage: e.GenericGuildMessage}
			}
			switch event := event.(type) {
			case *events.GuildMessageCreate:
				if event.Message.Interaction == nil || event.Message.ApplicationID == nil {
					return nil
				}
				if event.Message.Author.ID != c.Config().BumpUserID && event.Message.Author.ID != c.Config().UpUserID {
					return nil
				}
				g, err := c.GuildCreateID(event, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				if g.BumpEnabled {
					if err := bumpHandler(c, g, event); err != nil {
						return errors.NewError(err)
					}
				}
				if g.UpEnabled {
					if err := upHandler(c, g, event); err != nil {
						return errors.NewError(err)
					}
				}
				return nil
			}
			return nil
		},

		Schedulers: []components.Scheduler{
			{
				Duration: time.Minute,
				Worker: func(c *components.Components, client *bot.Client) error {
					bumpLock.Lock()
					defer bumpLock.Unlock()
					for k, n := range bumpNotice {
						g, err := c.GuildCreateID(context.Background(), n.guildID)
						if err != nil {
							continue
						}
						if !g.BumpEnabled {
							continue
						}

						if time.Now().After(n.t.Add(-time.Minute * 2)) {
							go func() {
								time.Sleep(time.Until(n.t))
								createNotice(g.BumpRemindMessageTitle, g.BumpRemindMessage, n, client, builtin.Or(g.BumpMention != nil, discord.RoleMention(builtin.NonNil(g.BumpMention)), ""))
							}()
							delete(bumpNotice, k)
						}
					}
					upLock.Lock()
					defer upLock.Unlock()
					for k, n := range upNotice {
						g, err := c.GuildCreateID(context.Background(), n.guildID)
						if err != nil {
							continue
						}
						if !g.UpEnabled {
							continue
						}

						if time.Now().After(n.t.Add(-time.Minute * 2)) {
							go func() {
								time.Sleep(time.Until(n.t))
								createNotice(g.UpRemindMessageTitle, g.UpRemindMessage, n, client, builtin.Or(g.UpMention != nil, discord.RoleMention(builtin.NonNil(g.UpMention)), ""))
							}()
							delete(upNotice, k)
						}
					}

					return nil
				},
			},
		},
	}).SetComponent(c)
}

type notice struct {
	channelID snowflake.ID
	guildID   snowflake.ID
	t         time.Time
}

var bumpNotice = map[snowflake.ID]notice{}
var bumpLock sync.Mutex

func bumpHandler(c *components.Components, g *ent.Guild, event *events.GuildMessageCreate) error {
	bumpLock.Lock()
	defer bumpLock.Unlock()
	if event.Message.Interaction == nil || event.Message.Interaction.Name != "bump" {
		return nil
	}
	if len(event.Message.Embeds) < 1 || event.Message.Embeds[0].Image == nil || event.Message.Embeds[0].Image.URL != c.Config().BumpImage {
		return nil
	}
	if !g.BumpEnabled {
		return nil
	}
	n :=
		notice{
			channelID: event.ChannelID,
			guildID:   event.GuildID,
			t:         event.Message.CreatedAt.Add(time.Hour * 2),
		}
	bumpNotice[event.GuildID] = n
	createNotice(g.BumpMessageTitle, g.BumpMessage, n, event.Client(), "")
	return nil
}

var upNotice = map[snowflake.ID]notice{}
var upLock sync.Mutex

func upHandler(c *components.Components, g *ent.Guild, event *events.GuildMessageCreate) error {
	upLock.Lock()
	defer upLock.Unlock()
	if event.Message.Interaction == nil || event.Message.Interaction.Name != "dissoku up" {
		return nil
	}
	if len(event.Message.Embeds) < 1 || event.Message.Embeds[0].Color != c.Config().UpColor {
		return nil
	}
	if !g.UpEnabled {
		return nil
	}
	n :=
		notice{
			channelID: event.ChannelID,
			guildID:   event.GuildID,
			t:         event.Message.CreatedAt.Add(time.Hour * 1),
		}
	upNotice[event.GuildID] = n
	createNotice(g.UpMessageTitle, g.UpMessage, n, event.Client(), "")
	return nil
}

func createNotice(title, message string, n notice, client *bot.Client, content string) {
	if _, err := client.Rest.CreateMessage(n.channelID,
		discord.NewMessageBuilder().
			SetContent(content).
			SetEmbeds(
				embeds.SetEmbedProperties(
					discord.NewEmbedBuilder().
						SetTitle(title).
						SetDescription(message).
						Build(),
				),
			).
			BuildCreate(),
	); err != nil {
		slog.Error("通知作成に失敗", slog.Any("err", err))
		return
	}
}
