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
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "setting",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "settings",
				Description:              "view settings",
				DescriptionLocalizations: i18n.TranslateTextMap("components.settings"),
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
								DescriptionLocalizations: i18n.TranslateTextMap("components.setting.bump.toggle"),
							},
							{
								Name:                     "message",
								Description:              "set message",
								DescriptionLocalizations: i18n.TranslateTextMap("components.setting.bump.message"),
							},
							{
								Name:                     "mention",
								Description:              "set mention target",
								DescriptionLocalizations: i18n.TranslateTextMap("components.setting.bump.mention"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionRole{
										Name:                     "target",
										Description:              "target role",
										DescriptionLocalizations: i18n.TranslateTextMap("components.setting.mention.target"),
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
								DescriptionLocalizations: i18n.TranslateTextMap("components.setting.up.toggle"),
							},
							{
								Name:                     "message",
								Description:              "set message",
								DescriptionLocalizations: i18n.TranslateTextMap("components.setting.up.message"),
							},
							{
								Name:                     "mention",
								Description:              "set mention target",
								DescriptionLocalizations: i18n.TranslateTextMap("components.setting.up.mention"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionRole{
										Name:                     "target",
										Description:              "target role",
										DescriptionLocalizations: i18n.TranslateTextMap("components.setting.mention.target"),
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
								DescriptionLocalizations: i18n.TranslateTextMap("components.setting.leveling.toggle"),
							},
						},
					},
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					l := i18n.TranslateLayout(event.Locale(), "command.settings.view")
					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g.BumpEnabled = !g.BumpEnabled
					if err := c.GormDB().Save(g).Error; err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(builtin.Or(g.BumpEnabled, i18n.TranslateText(event.Locale(), "components.setting.bump.toggle.enabled"), i18n.TranslateText(event.Locale(), "components.setting.bump.toggle.disabled"))).
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g.UpEnabled = !g.UpEnabled
					if err := c.GormDB().Save(g).Error; err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(builtin.Or(g.UpEnabled, i18n.TranslateText(event.Locale(), "components.setting.up.toggle.enabled"), i18n.TranslateText(event.Locale(), "components.setting.up.toggle.disabled"))).
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if r, ok := event.SlashCommandInteractionData().OptRole("target"); ok {
						g.BumpMention = &r.ID
					} else {
						g.BumpMention = nil
					}
					if err := c.GormDB().Save(g).Error; err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(i18n.TranslateText(event.Locale(), "components.setting.bump.mention.used", map[string]any{
								"role": builtin.Or(g.BumpMention != nil,
									discord.RoleMention(builtin.NonNil(g.BumpMention)),
									"`"+i18n.TranslateText(event.Locale(), "components.setting.mention.none")+"`",
								),
							})).
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if r, ok := event.SlashCommandInteractionData().OptRole("target"); ok {
						g.UpMention = &r.ID
					} else {
						g.UpMention = nil
					}
					if err := c.GormDB().Save(g).Error; err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(i18n.TranslateText(event.Locale(), "components.setting.up.mention.used", map[string]any{
								"role": builtin.Or(g.UpMention != nil,
									discord.RoleMention(builtin.NonNil(g.UpMention)),
									"`"+i18n.TranslateText(event.Locale(), "components.setting.mention.none")+"`",
								),
							})).
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if err := event.Modal(
						discordutil.NewModalCreateBuilder().
							SetTitle(i18n.TranslateText(event.Locale(), "components.setting.bump.message.modal.title")).
							SetCustomID("setting:bump_message").
							SetComponents(
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.message_title"),
									discord.TextInputComponent{
										CustomID:  "message_title",
										Style:     discord.TextInputStyleShort,
										MinLength: new(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.BumpMessageTitle,
									},
								),
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.message"),
									discord.TextInputComponent{
										CustomID:  "message",
										Style:     discord.TextInputStyleParagraph,
										MinLength: new(1),
										MaxLength: 300,
										Required:  true,
										Value:     g.BumpMessage,
									},
								),
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.remind.message_title"),
									discord.TextInputComponent{
										CustomID:  "remind.message_title",
										Style:     discord.TextInputStyleShort,
										MinLength: new(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.BumpRemindMessageTitle,
									},
								),
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.remind.message"),
									discord.TextInputComponent{
										CustomID:  "remind.message",
										Style:     discord.TextInputStyleParagraph,
										MinLength: new(1),
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if err := event.Modal(
						discordutil.NewModalCreateBuilder().
							SetTitle(i18n.TranslateText(event.Locale(), "components.setting.up.message.modal.title")).
							SetCustomID("setting:up_message").
							SetComponents(
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.message_title"),
									discord.TextInputComponent{
										CustomID:  "message_title",
										Style:     discord.TextInputStyleShort,
										MinLength: new(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.UpMessageTitle,
									},
								),
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.message"),
									discord.TextInputComponent{
										CustomID:  "message",
										Style:     discord.TextInputStyleParagraph,
										MinLength: new(1),
										MaxLength: 300,
										Required:  true,
										Value:     g.UpMessage,
									},
								),
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.remind.message_title"),
									discord.TextInputComponent{
										CustomID:  "remind.message_title",
										Style:     discord.TextInputStyleShort,
										MinLength: new(1),
										MaxLength: 30,
										Required:  true,
										Value:     g.UpRemindMessageTitle,
									},
								),
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.setting.message.modal.remind.message"),
									discord.TextInputComponent{
										CustomID:  "remind.message",
										Style:     discord.TextInputStyleParagraph,
										MinLength: new(1),
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
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g.LevelingDisabled = !g.LevelingDisabled
					if err := c.GormDB().Save(g).Error; err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(builtin.Or(!g.LevelingDisabled, i18n.TranslateText(event.Locale(), "components.setting.leveling.enable.enabled"), i18n.TranslateText(event.Locale(), "components.setting.leveling.enable.disabled"))).
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
				g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				g.BumpMessageTitle = event.ModalSubmitInteraction.Data.Text("message_title")
				g.BumpMessage = event.ModalSubmitInteraction.Data.Text("message")
				g.BumpRemindMessageTitle = event.ModalSubmitInteraction.Data.Text("remind.message_title")
				g.BumpRemindMessage = event.ModalSubmitInteraction.Data.Text("remind.message")
				if err := c.GormDB().Save(g).Error; err != nil {
					return errors.NewError(err)
				}

				if err := event.DeferUpdateMessage(); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
			"setting:up_message": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				g.UpMessageTitle = event.ModalSubmitInteraction.Data.Text("message_title")
				g.UpMessage = event.ModalSubmitInteraction.Data.Text("message")
				g.UpRemindMessageTitle = event.ModalSubmitInteraction.Data.Text("remind.message_title")
				g.UpRemindMessage = event.ModalSubmitInteraction.Data.Text("remind.message")
				if err := c.GormDB().Save(g).Error; err != nil {
					return errors.NewError(err)
				}

				if err := event.DeferUpdateMessage(); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
		},

		EventHandler: func(c *components.Components, event bot.Event) errors.Error {
			if e, ok := event.(*events.GuildMessageUpdate); ok {
				msg := e.Message
				if msg.Author.ID == 0 && e.OldMessage.Author.ID != 0 {
					msg.Author = e.OldMessage.Author
				}
				if msg.ApplicationID == nil && e.OldMessage.ApplicationID != nil {
					msg.ApplicationID = e.OldMessage.ApplicationID
				}
				//nolint:staticcheck // SA1019: Message.Interaction is deprecated in disgo
				if msg.Interaction == nil && e.OldMessage.Interaction != nil {
					msg.Interaction = e.OldMessage.Interaction
				}
				if msg.CreatedAt.IsZero() && !e.OldMessage.CreatedAt.IsZero() {
					msg.CreatedAt = e.OldMessage.CreatedAt
				}
				if len(msg.Embeds) == 0 && len(e.OldMessage.Embeds) > 0 {
					msg.Embeds = e.OldMessage.Embeds
				}
				event = &events.GuildMessageCreate{
					GenericGuildMessage: &events.GenericGuildMessage{
						GenericEvent: e.GenericEvent,
						MessageID:    e.MessageID,
						Message:      msg,
						ChannelID:    e.ChannelID,
						GuildID:      e.GuildID,
					},
				}
			}
			switch event := event.(type) {
			case *events.GuildMessageCreate:
				//nolint:staticcheck // SA1019: Message.Interaction is deprecated in disgo
				if event.Message.Interaction == nil || event.Message.ApplicationID == nil {
					return nil
				}
				if event.Message.Author.ID != c.Config().BumpUserID && event.Message.Author.ID != c.Config().UpUserID {
					return nil
				}
				g, err := c.GuildCreateID(c.Ctx(), event.GuildID)
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
				Duration: 10 * time.Second,
				Worker: func(c *components.Components, client *bot.Client) error {
					db := c.GormDB()
					if db == nil {
						return nil
					}
					var schedules []models.BumpSchedule
					now := time.Now()
					if err := db.Where("remind_at <= ?", now).Find(&schedules).Error; err != nil {
						return errors.NewError(err)
					}

					for _, s := range schedules {
						g, err := c.GuildCreateID(c.Ctx(), s.GuildID)
						if err != nil {
							_ = db.Delete(&s).Error
							continue
						}

						switch s.Type {
						case models.BumpTypeBump:
							if g.BumpEnabled {
								createNotice(g.BumpRemindMessageTitle, g.BumpRemindMessage, s.ChannelID, client,
									builtin.Or(g.BumpMention != nil, discord.RoleMention(builtin.NonNil(g.BumpMention)), ""))
							}
						case models.BumpTypeUp:
							if g.UpEnabled {
								createNotice(g.UpRemindMessageTitle, g.UpRemindMessage, s.ChannelID, client,
									builtin.Or(g.UpMention != nil, discord.RoleMention(builtin.NonNil(g.UpMention)), ""))
							}
						}

						_ = db.Delete(&s).Error
					}

					return nil
				},
			},
		},
	}).SetComponent(c)
}

func bumpHandler(c *components.Components, g *models.Guild, event *events.GuildMessageCreate) error {
	//nolint:staticcheck // SA1019: Message.Interaction is deprecated in disgo
	if event.Message.Interaction == nil || event.Message.Interaction.Name != "bump" {
		return nil
	}
	if len(event.Message.Embeds) < 1 || event.Message.Embeds[0].Image == nil || event.Message.Embeds[0].Image.URL != c.Config().BumpImage {
		return nil
	}
	if !g.BumpEnabled {
		return nil
	}

	db := c.GormDB()
	if db == nil {
		return nil
	}

	// Avoid duplicate notifications if the same message was already processed (e.g. from GuildMessageUpdate)
	var count int64
	if err := db.Model(&models.BumpSchedule{}).
		Where("guild_id = ? AND type = ? AND message_id = ?", event.GuildID, models.BumpTypeBump, event.Message.ID).
		Count(&count).Error; err == nil && count > 0 {
		return nil
	}

	msgTime := event.Message.CreatedAt
	if msgTime.IsZero() {
		msgTime = event.Message.ID.Time()
	}
	if msgTime.IsZero() {
		msgTime = time.Now()
	}

	// Remove any existing pending schedule for this guild and type
	_ = db.Where("guild_id = ? AND type = ?", event.GuildID, models.BumpTypeBump).Delete(&models.BumpSchedule{}).Error

	schedule := models.BumpSchedule{
		GuildID:   event.GuildID,
		Type:      models.BumpTypeBump,
		ChannelID: event.ChannelID,
		MessageID: event.Message.ID,
		RemindAt:  msgTime.Add(time.Hour * 2),
	}
	if err := db.Create(&schedule).Error; err != nil {
		return errors.NewError(err)
	}

	var client *bot.Client
	if event.GenericEvent != nil {
		client = event.Client()
	}
	createNotice(g.BumpMessageTitle, g.BumpMessage, event.ChannelID, client, "")
	return nil
}

func upHandler(c *components.Components, g *models.Guild, event *events.GuildMessageCreate) error {
	//nolint:staticcheck // SA1019: Message.Interaction is deprecated in disgo
	if event.Message.Interaction == nil || event.Message.Interaction.Name != "dissoku up" {
		return nil
	}
	if len(event.Message.Embeds) < 1 || event.Message.Embeds[0].Color != c.Config().UpColor {
		return nil
	}
	if !g.UpEnabled {
		return nil
	}

	db := c.GormDB()
	if db == nil {
		return nil
	}

	// Avoid duplicate notifications if the same message was already processed (e.g. from GuildMessageUpdate)
	var count int64
	if err := db.Model(&models.BumpSchedule{}).
		Where("guild_id = ? AND type = ? AND message_id = ?", event.GuildID, models.BumpTypeUp, event.Message.ID).
		Count(&count).Error; err == nil && count > 0 {
		return nil
	}

	msgTime := event.Message.CreatedAt
	if msgTime.IsZero() {
		msgTime = event.Message.ID.Time()
	}
	if msgTime.IsZero() {
		msgTime = time.Now()
	}

	// Remove any existing pending schedule for this guild and type
	_ = db.Where("guild_id = ? AND type = ?", event.GuildID, models.BumpTypeUp).Delete(&models.BumpSchedule{}).Error

	schedule := models.BumpSchedule{
		GuildID:   event.GuildID,
		Type:      models.BumpTypeUp,
		ChannelID: event.ChannelID,
		MessageID: event.Message.ID,
		RemindAt:  msgTime.Add(time.Hour * 1),
	}
	if err := db.Create(&schedule).Error; err != nil {
		return errors.NewError(err)
	}

	var client *bot.Client
	if event.GenericEvent != nil {
		client = event.Client()
	}
	createNotice(g.UpMessageTitle, g.UpMessage, event.ChannelID, client, "")
	return nil
}

func createNotice(title, message string, channelID snowflake.ID, client *bot.Client, content string) {
	if client == nil || client.Rest == nil {
		return
	}
	embed := embeds.SetEmbedProperties(
		discord.NewEmbed().
			WithTitle(title).
			WithDescription(message),
	)
	if _, err := client.Rest.CreateMessage(channelID,
		discordutil.NewMessageBuilder().
			SetContent(content).
			SetEmbeds(embed).
			BuildCreate(),
	); err != nil {
		slog.Error("通知作成に失敗", slog.Any("err", err))
		return
	}
}
