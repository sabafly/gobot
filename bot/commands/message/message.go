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

package message

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/gorm"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
	"github.com/sabafly/gobot/internal/parse"
	"github.com/sabafly/gobot/internal/uuidv7"
)

const (
	PinArgumentTypeDuration1m = iota
	PinArgumentTypeDuration1h
	PinArgumentTypeDuration3h
	PinArgumentTypeDuration6h
	PinArgumentTypeDuration1d
	PinArgumentTypeDuration3d
	PinArgumentTypeDuration1w
)

const (
	WordSuffixRuleWebhook = "webhook"
	WordSuffixRuleWarn    = "warn"
	WordSuffixRuleDelete  = "delete"
)

func Command(c *components.Components) *generic.Command {
	return (&generic.Command{
		Namespace: "message",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "message",
				Description: "message",
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "suffix",
						Description: "suffix",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "set",
								Description:              "set member's suffix",
								DescriptionLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.description"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionUser{
										Name:                     "target",
										NameLocalizations:        i18n.TranslateTextMap("components.message.suffix.set.command.options.target.name"),
										Description:              "target",
										DescriptionLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.target.description"),
										Required:                 true,
									},
									discord.ApplicationCommandOptionString{
										Name:                     "suffix",
										NameLocalizations:        i18n.TranslateTextMap("components.message.suffix.set.command.options.suffix.name"),
										Description:              "suffix",
										DescriptionLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.suffix.description"),
										Required:                 true,
										MaxLength:                new(512),
									},
									discord.ApplicationCommandOptionString{
										Name:                     "rule",
										NameLocalizations:        i18n.TranslateTextMap("components.message.suffix.set.command.options.rule.name"),
										Description:              "rule",
										DescriptionLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.rule.description"),
										Required:                 true,
										Choices: []discord.ApplicationCommandOptionChoiceString{
											{
												Name:              "webhook",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.rule.webhook"),
												Value:             WordSuffixRuleWebhook,
											},
											{
												Name:              "warn",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.rule.warn"),
												Value:             WordSuffixRuleWarn,
											},
											{
												Name:              "delete",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.rule.delete"),
												Value:             WordSuffixRuleDelete,
											},
										},
									},
									discord.ApplicationCommandOptionInt{
										Name:                     "duration",
										NameLocalizations:        i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.name"),
										Description:              "duration",
										DescriptionLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.description"),
										Required:                 false,
										Choices: []discord.ApplicationCommandOptionChoiceInt{
											{
												Name:              "1m",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.1m"),
												Value:             PinArgumentTypeDuration1m,
											},
											{
												Name:              "1h",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.1h"),
												Value:             PinArgumentTypeDuration1h,
											},
											{
												Name:              "3h",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.3h"),
												Value:             PinArgumentTypeDuration3h,
											},
											{
												Name:              "6h",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.6h"),
												Value:             PinArgumentTypeDuration6h,
											},
											{
												Name:              "1d",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.1d"),
												Value:             PinArgumentTypeDuration1d,
											},
											{
												Name:              "3d",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.3d"),
												Value:             PinArgumentTypeDuration3d,
											},
											{
												Name:              "1w",
												NameLocalizations: i18n.TranslateTextMap("components.message.suffix.set.command.options.duration.1w"),
												Value:             PinArgumentTypeDuration1w,
											},
										},
									},
								},
							},
							{
								Name:                     "remove",
								Description:              "remove member's suffix",
								DescriptionLocalizations: i18n.TranslateTextMap("components.message.suffix.remove.command.description"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionUser{
										Name:                     "target",
										NameLocalizations:        i18n.TranslateTextMap("components.message.suffix.remove.command.options.target.name"),
										Description:              "target",
										DescriptionLocalizations: i18n.TranslateTextMap("components.message.suffix.remove.command.options.target.description"),
										Required:                 true,
									},
								},
							},
							{
								Name:        "check",
								Description: "check member's suffix",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionUser{
										Name:        "target",
										Description: "target",
										Required:    false,
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "pin",
						Description: "pin",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "create",
								Description: "create pinned message",
							},
							{
								Name:        "delete",
								Description: "delete pinned message",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "remind",
						Description: "remind",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "set",
								Description: "set remind",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:        "time",
										Description: "format 2023-01-23 15:16",
										MinLength:   new(1),
										MaxLength:   new(16),
										Required:    true,
									},
								},
							},
							{
								Name:        "cancel",
								Description: "cancel remind",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:         "remind",
										Description:  "remind name",
										Autocomplete: true,
										Required:     true,
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "bulk_delete",
						Description: "bulk delete messages",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:        "count",
								Description: "number of messages to delete",
								Required:    true,
								MinValue:    2,
								MaxValue:    100,
							},
						},
					},
				},
			},
		},

		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/message/suffix/set": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.suffix.set"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if u := event.SlashCommandInteractionData().User("target"); u.Bot || u.System {
						return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
					}

					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					u, err := c.UserCreate(c.Ctx(), event.SlashCommandInteractionData().User("target"))
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					var expired *time.Time
					if duration, ok := event.SlashCommandInteractionData().OptInt("duration"); ok {
						var d time.Duration
						switch duration {
						case PinArgumentTypeDuration1m:
							d = time.Minute
						case PinArgumentTypeDuration1h:
							d = time.Hour
						case PinArgumentTypeDuration3h:
							d = time.Hour * 3
						case PinArgumentTypeDuration6h:
							d = time.Hour * 6
						case PinArgumentTypeDuration1d:
							d = time.Hour * 24
						case PinArgumentTypeDuration3d:
							d = time.Hour * 24 * 3
						case PinArgumentTypeDuration1w:
							d = time.Hour * 24 * 7
						}
						expired = builtin.Or(d != 0, new(time.Now().Add(d)), nil)
					}

					var w models.WordSuffix
					err = c.GormDB().Where("guild_id = ? AND owner_id = ?", g.ID, u.ID).First(&w).Error
					if err == nil {
						// Update
						w.Suffix = event.SlashCommandInteractionData().String("suffix")
						w.Rule = event.SlashCommandInteractionData().String("rule")
						w.Expired = expired
						if err := c.GormDB().Save(&w).Error; err != nil {
							return errors.NewError(err)
						}
					} else {
						if !errors.Is(err, gorm.ErrRecordNotFound) {
							return errors.NewError(err)
						}

						// Create
						w = models.WordSuffix{
							ID:      uuidv7.New(),
							GuildID: &g.ID,
							Suffix:  event.SlashCommandInteractionData().String("suffix"),
							OwnerID: u.ID,
							Rule:    event.SlashCommandInteractionData().String("rule"),
							Expired: expired,
						}
						if err := c.GormDB().Create(&w).Error; err != nil {
							return errors.NewError(err)
						}
					}

					var durationString string
					if expired != nil {
						durationString = discord.FormattedTimestampMention(expired.Unix(), discord.TimestampStyleRelative)
					} else {
						durationString = i18n.TranslateText(event.Locale(), "components.message.suffix.duration.none", nil)
					}
					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContentf("%s\n%s",
								i18n.TranslateText(
									event.Locale(),
									"components.message.suffix.set.message",
									map[string]any{"user": discord.UserMention(u.ID), "suffix": w.Suffix}),
								i18n.TranslateText(
									event.Locale(),
									"components.message.suffix.duration.message",
									map[string]any{"duration": durationString}),
							).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}

					return nil
				},
			},
			"/message/suffix/remove": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.suffix.remove"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if u := event.SlashCommandInteractionData().User("target"); u.Bot || u.System {
						return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
					}

					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					u, err := c.UserCreate(c.Ctx(), event.SlashCommandInteractionData().User("target"))
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}

					var w models.WordSuffix
					err = c.GormDB().Where("guild_id = ? AND owner_id = ?", g.ID, u.ID).First(&w).Error
					if err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							if err := event.CreateMessage(
								discordutil.NewMessageBuilder().
									SetContent(i18n.TranslateText(event.Locale(), "components.message.suffix.remove.message.no_suffix", map[string]any{"user": discord.UserMention(u.ID)})).
									SetAllowedMentions(&discord.AllowedMentions{}).
									SetFlags(discord.MessageFlagEphemeral).
									BuildCreate(),
							); err != nil {
								return errors.NewError(err)
							}
							return nil
						}
						return errors.NewError(err)
					}

					if err := c.GormDB().Delete(&w).Error; err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(i18n.TranslateText(event.Locale(), "components.message.suffix.remove.message", map[string]any{"user": discord.UserMention(u.ID)})).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}

					return nil
				},
			},
			"/message/suffix/check": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.suffix.check"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					m, ok := event.SlashCommandInteractionData().OptMember("target")
					if !ok {
						m = *event.Member()
					}
					if m.User.Bot || m.User.System {
						return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
					}

					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					u, err := c.UserCreate(c.Ctx(), m.User)
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}

					messageStr := i18n.TranslateText(event.Locale(), "components.message.suffix.check.message.none",
						map[string]any{"user": discord.UserMention(u.ID)},
					)

					var w models.WordSuffix
					err = c.GormDB().Where("guild_id = ? AND owner_id = ?", g.ID, u.ID).First(&w).Error
					if err != nil && err != gorm.ErrRecordNotFound {
						return errors.NewError(err)
					}
					if err == nil {
						var ruleKey string
						switch w.Rule {
						case WordSuffixRuleWebhook:
							ruleKey = "components.message.suffix.set.command.options.rule.webhook"
						case WordSuffixRuleWarn:
							ruleKey = "components.message.suffix.set.command.options.rule.warn"
						default:
							ruleKey = "components.message.suffix.set.command.options.rule.delete"
						}
						messageStr = i18n.TranslateText(event.Locale(), "components.message.suffix.check.message",
							map[string]any{
								"duration": builtin.Or(w.Expired != nil,
									discord.FormattedTimestampMention(builtin.NonNil(w.Expired).Unix(), discord.TimestampStyleRelative),
									i18n.TranslateText(event.Locale(), "components.message.suffix.duration.none", nil),
								),
								"user":   discord.UserMention(u.ID),
								"suffix": w.Suffix,
								"rule":   i18n.TranslateText(event.Locale(), ruleKey, nil),
							},
						)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(messageStr).
							SetAllowedMentions(&discord.AllowedMentions{}).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/message/pin/create": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.pin.create"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(_ *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.Modal(
						discordutil.NewModalCreateBuilder().
							SetTitle(i18n.TranslateText(event.Locale(), "components.message.pin.create.modal.title", nil)).
							SetCustomID("message:pin_create_modal").
							SetComponents(
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.message.pin.create.modal.input.1.label", nil),
									discord.TextInputComponent{
										CustomID:  "content",
										Style:     discord.TextInputStyleParagraph,
										MaxLength: 1000,
										Required:  true,
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
			"/message/pin/delete": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.pin.delete"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					_, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					var m models.MessagePin
					err = c.GormDB().Where("channel_id = ?", event.Channel().ID()).First(&m).Error
					if err != nil {
						if err == gorm.ErrRecordNotFound {
							return errors.NewError(errors.ErrorMessage("errors.unavailable.message.pin", event))
						}
						return errors.NewError(err)
					}

					if m.BeforeID != nil {
						_ = event.Client().Rest.DeleteMessage(event.Channel().ID(), *m.BeforeID)
					}

					if err := c.GormDB().Delete(&m).Error; err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(i18n.TranslateText(event.Locale(), "components.message.pin.delete.message", nil)).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}

					return nil
				},
			},
			"/message/remind/set": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.remind.set"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					tm := time.Now().Add(time.Hour)
					if timeStr, ok := event.SlashCommandInteractionData().OptString("time"); ok {
						t, err := parse.TimeFuture(timeStr)
						if err != nil {
							return errors.NewError(errors.ErrorMessage("errors.invalid.time.format", event))
						}
						if t.Before(time.Now()) {
							return errors.NewError(errors.ErrorMessage("errors.invalid.time.before", event))
						}
						tm = t
					}

					if err := event.Modal(
						discordutil.NewModalCreateBuilder().
							SetTitle(i18n.TranslateText(event.Locale(), "components.message.remind.add.modal.title", nil)).
							SetCustomID(fmt.Sprintf("message:remind_create_modal:%d", tm.Unix())).
							SetComponents(
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.message.remind.add.modal.input.content.label", nil),
									discord.TextInputComponent{
										CustomID:  "content",
										Style:     discord.TextInputStyleParagraph,
										MinLength: new(1),
										MaxLength: 1000,
										Required:  true,
									},
								),
								discord.NewLabel(i18n.TranslateText(event.Locale(), "components.message.remind.add.modal.input.name.label", nil),
									discord.TextInputComponent{
										CustomID:  "name",
										Style:     discord.TextInputStyleShort,
										MinLength: new(1),
										MaxLength: 64,
										Required:  true,
										Value: fmt.Sprintf("%s#%d",
											i18n.TranslateText(event.Locale(), "components.message.remind.add.modal.input.name.value", nil),
											g.RemindCount+1,
										),
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
			"/message/remind/cancel": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.remind.cancel"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					res := c.GormDB().Where("guild_id = ? AND name LIKE ?", *event.GuildID(), "%"+escapeLike(event.SlashCommandInteractionData().String("remind"))+"%").Delete(&models.MessageRemind{})
					if res.Error != nil {
						return errors.NewError(res.Error)
					}
					count := res.RowsAffected

					if err := event.CreateMessage(
						discordutil.NewMessageBuilder().
							SetContent(i18n.TranslateText(event.Locale(), "components.message.remind.cancel.message",
								map[string]any{
									"count": strconv.FormatInt(count, 10),
								},
							)).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/message/bulk_delete": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.bulk_delete"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					count := event.SlashCommandInteractionData().Int("count")
					messages, err := event.Client().Rest().GetMessages(event.Channel().ID(), count)
					if err != nil {
						return errors.NewError(err)
					}

					messageIDs := make([]snowflake.ID, len(messages))
					for i, message := range messages {
						messageIDs[i] = message.ID
					}

					if err := event.Client().Rest().BulkDeleteMessages(event.Channel().ID, messageIDs); err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.message.bulk_delete.success", translate.WithTemplate(map[string]any{"Count": count}))).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}

					return nil
				},
			},
		},

		AutocompleteHandlers: map[string]generic.PermissionAutocompleteHandler{
			"/message/remind/cancel:remind": generic.PAutocompleteHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.remind.cancel"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				AutocompleteHandler: func(c *components.Components, event *events.AutocompleteInteractionCreate) errors.Error {
					var reminds []models.MessageRemind
					if err := c.GormDB().Where("guild_id = ? AND name LIKE ?", *event.GuildID(), "%"+escapeLike(event.Data.String("remind"))+"%").Limit(25).Find(&reminds).Error; err != nil {
						return errors.NewError(err)
					}

					choices := make([]discord.AutocompleteChoice, len(reminds))
					for i, mr := range reminds {
						choices[i] = discord.AutocompleteChoiceString{
							Name:  fmt.Sprintf("%s - %s", mr.Name, mr.Time.Local().Format("2006-01-02 15:04 MST")),
							Value: mr.Name,
						}
					}
					if err := event.AutocompleteResult(choices); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},

		ModalHandlers: map[string]generic.ModalHandler{
			"message:pin_create_modal": func(component *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := component.GuildCreateID(component.Ctx(), *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}

				// もし既にあったら抹消する
				var oldPin models.MessagePin
				if err := component.GormDB().Where("channel_id = ?", event.Channel().ID()).First(&oldPin).Error; err == nil {
					if oldPin.BeforeID != nil {
						if err := event.Client().Rest.DeleteMessage(event.Channel().ID(), *oldPin.BeforeID); err != nil {
							return errors.NewError(err)
						}
					}
					if err := component.GormDB().Delete(&oldPin).Error; err != nil {
						return errors.NewError(err)
					}
				}

				m := models.MessagePin{
					ChannelID: event.Channel().ID(),
					Content:   event.Data.Text("content"),
					GuildID:   g.ID,
				}
				if err := component.GormDB().Create(&m).Error; err != nil {
					return errors.NewError(err)
				}

				channel, err := event.Client().Rest.GetChannel(m.ChannelID)
				if err != nil {
					return errors.NewError(err)
				}

				webhook, err := discordutil.GetWebhookMessenger(event.Client(), channel)
				if err != nil {
					return errors.NewError(err)
				}
				message, err := webhook.SendWebhook(
					discordutil.NewMessageBuilder().
						SetContent(m.Content).
						SetEmbeds(m.Embeds...),
					i18n.TranslateText(g.Locale, "components.message.pin.username", nil),
					component.Config().Message.PinIconImage,
					"",
				)
				if err != nil {
					return errors.NewError(err)
				}

				m.BeforeID = &message.ID
				if err := component.GormDB().Save(&m).Error; err != nil {
					return errors.NewError(err)
				}

				if err := event.CreateMessage(
					discordutil.NewMessageBuilder().
						SetContent(i18n.TranslateText(event.Locale(), "components.message.pin.create.message", nil)).
						SetFlags(discord.MessageFlagEphemeral).
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
			"message:remind_create_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(c.Ctx(), *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				args := strings.Split(event.Data.CustomID, ":")
				tm := time.Unix(builtin.Must(strconv.ParseInt(args[2], 10, 64)), 0)
				if time.Now().After(tm) {
					return errors.NewError(errors.ErrorMessage("errors.invalid.time.before", event))
				}

				remind := models.MessageRemind{
					GuildID:   g.ID,
					Time:      tm,
					Content:   event.Data.Text("content"),
					ChannelID: event.Channel().ID(),
					AuthorID:  event.Member().User.ID,
					Name:      event.Data.Text("name"),
				}
				if err := c.GormDB().Create(&remind).Error; err != nil {
					return errors.NewError(err)
				}

				g.RemindCount++
				if err := c.GormDB().Save(g).Error; err != nil {
					return errors.NewError(err)
				}

				if err := event.CreateMessage(
					discordutil.NewMessageBuilder().
						SetContent(i18n.TranslateText(event.Locale(), "components.message.remind.add.message",
							map[string]any{
								"time": discord.FormattedTimestampMention(tm.Unix(), discord.TimestampStyleLongDateTime),
							},
						)).
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
		},

		Schedulers: []components.Scheduler{
			{
				Duration: time.Minute,
				Worker: func(c *components.Components, client *bot.Client) error {
					var reminds []models.MessageRemind
					c.GormDB().Where("time < ?", time.Now()).Find(&reminds)

					for _, remind := range reminds {
						if _, err := client.Rest.CreateMessage(remind.ChannelID,
							discordutil.NewMessageBuilder().
								SetContent(remind.Content).
								BuildCreate(),
						); err != nil {
							return err
						}
					}

					c.GormDB().Where("time < ?", time.Now()).Delete(&models.MessageRemind{})
					return nil
				},
			},
		},

		EventHandler: func(c *components.Components, e bot.Event) errors.Error {
			switch e := e.(type) {
			case *events.GuildMessageCreate:

				err, shouldContinue := doTextCommand(c.Ctx(), e)
				if err != nil {
					return errors.NewError(err)
				}
				if !shouldContinue {
					return nil
				}

				// TODO: refactor
				// 変数が初期化されていないことが潜在的なバグの原因になりかねない

				// 語尾の処理
				var w models.WordSuffix
				var u *models.User

				if e.Message.Type.System() || e.Message.Author.System || e.Message.Author.Bot {
					goto messagePin
				}
				if e.Message.Type != discord.MessageTypeDefault && e.Message.Type != discord.MessageTypeReply {
					goto messagePin
				}

				if u, err = c.UserCreate(c.Ctx(), e.Message.Author); err != nil {
					slog.Error("メッセージ著者取得に失敗", "err", err, "uid", e.Message.Author.ID)
					return errors.NewError(err)
				}

				// Guild
				if err := c.GormDB().Where("owner_id = ? AND guild_id = ?", u.ID, e.GuildID).First(&w).Error; err == nil {
					// Found
				} else if errors.Is(err, gorm.ErrRecordNotFound) {
					// Global
					if err := c.GormDB().Where("owner_id = ? AND guild_id IS NULL", u.ID).First(&w).Error; err != nil {
						if !errors.Is(err, gorm.ErrRecordNotFound) {
							slog.Error("語尾取得エラー", "err", err)
						}
						// Not found
						slog.Debug("語尾が存在しません")
						goto messagePin
					}
				} else {
					slog.Error("ギルド語尾取得エラー", "err", err)
					goto messagePin
				}

				{
					webhookFlag := false
					if w.Rule == WordSuffixRuleWebhook {
						c.GetLock("message_pin").Mutex(e.ChannelID).Lock()
						webhookFlag = true
					}

					err = messageSuffixMessageCreateHandler(&w, u, e, c)

					if webhookFlag {
						c.GetLock("message_pin").Mutex(e.ChannelID).Unlock()
					}
				}

				if err != nil {
					return errors.NewError(err)
				}

				// ピン留めメッセージの処理
			messagePin:

				if err := func(event *events.GuildMessageCreate, c *components.Components) errors.Error {
					var channel discord.Channel
					channel, ok := event.Channel()
					if !ok {
						channel, err = event.Client().Rest.GetChannel(event.ChannelID)
						if err != nil {
							return errors.NewError(err)
						}
					}

					g, err := c.GuildCreateID(c.Ctx(), event.GuildID)
					if err != nil {
						return errors.NewError(err)
					}

					var m models.MessagePin
					if err := c.GormDB().Where("guild_id = ? AND channel_id = ?", g.ID, event.ChannelID).First(&m).Error; err != nil {
						if err == gorm.ErrRecordNotFound {
							return nil
						}
						return errors.NewError(err)
					}

					c.GetLock("message_pin").Mutex(e.ChannelID).Lock()
					defer c.GetLock("message_pin").Mutex(e.ChannelID).Unlock()

					// Re-fetch to be safe under lock? Or is it overkill?
					// The original code fetched it here.

					webhook, err := discordutil.GetWebhookMessenger(event.Client(), channel)
					if err != nil {
						err1 := &rest.Error{}
						if errors.As(err, &err1) && err1.Response.StatusCode == http.StatusForbidden {
							return errors.NewError(event.Client().Rest.LeaveGuild(event.GuildID))
						}
						return errors.NewError(err)
					}
					if event.Message.WebhookID != nil && webhook.Webhook().ID() == *event.Message.WebhookID {
						return nil
					}

					if m.RateLimit.CheckLimit() {
						if m.BeforeID != nil {
							if err := event.Client().Rest.DeleteMessage(event.ChannelID, *m.BeforeID); err != nil {
								slog.Error("削除に失敗", "err", err)
								m.BeforeID = nil
							}
						}

						var channel discord.Channel
						channel, ok := event.Channel()
						if !ok {
							channel, err = event.Client().Rest.GetChannel(event.ChannelID)
							if err != nil {
								return errors.NewError(err)
							}
						}
						webhook, err := discordutil.GetWebhookMessenger(event.Client(), channel)
						if err != nil {
							return errors.NewError(err)
						}

						message, err := webhook.SendWebhook(
							discordutil.NewMessageBuilder().
								SetContent(m.Content).
								SetEmbeds(m.Embeds...),
							i18n.TranslateText(g.Locale, "components.message.pin.username", nil),
							c.Config().Message.PinIconImage,
							"",
						)
						if err != nil {
							return errors.NewError(err)
						}

						m.BeforeID = &message.ID
						c.GormDB().Save(&m)
						slog.Info("ピン留め更新", "cid", event.ChannelID, "mid", event.MessageID)
					} else {
						c.GormDB().Save(&m)
					}
					return nil
				}(e, c); err != nil {
					return err
				}
			case *events.GuildMessageDelete:
				if ok := c.GetLock("message_pin").Mutex(e.ChannelID).TryLock(); !ok {
					return nil
				}
				defer c.GetLock("message_pin").Mutex(e.ChannelID).Unlock()
				if e.Message.WebhookID == nil {
					return nil
				}

				g, err := c.GuildCreateID(c.Ctx(), e.GuildID)
				if err != nil {
					return errors.NewError(err)
				}

				var m models.MessagePin
				if err := c.GormDB().Where("guild_id = ? AND channel_id = ?", g.ID, e.ChannelID).First(&m).Error; err != nil {
					return nil
				}

				if m.BeforeID != nil && *m.BeforeID == e.MessageID {
					slog.Info("ピン留め削除", "cid", e.ChannelID, "mid", e.MessageID)
					if err := c.GormDB().Delete(&m).Error; err != nil {
						slog.Error("ピン留め削除失敗", "err", err, "channel_id", e.ChannelID, "message_id", e.MessageID)
						return errors.NewError(err)
					}
				}
			}
			return nil
		},
	}).SetComponent(c)
}

func messageSuffixMessageCreateHandler(w *models.WordSuffix, u *models.User, e *events.GuildMessageCreate, c *components.Components) errors.Error {
	slog.Debug("メッセージ作成")
	if e.Message.Content == "" {
		return nil
	}

	if w.Expired != nil && time.Now().Compare(*w.Expired) == 1 {
		if err := c.GormDB().Delete(w).Error; err != nil {
			slog.Error("期限切れ語尾削除失敗", "err", err, "id", w.ID)
			return errors.NewError(err)
		}
		return nil
	}
	switch w.Rule {
	case WordSuffixRuleDelete:
		if strings.HasSuffix(e.Message.Content, w.Suffix) {
			return nil
		}
		if err := e.Client().Rest.DeleteMessage(e.ChannelID, e.MessageID); err != nil {
			slog.Error("メッセージを削除できません", "err", err)
			return errors.NewError(err)
		}
	case WordSuffixRuleWarn:
		if strings.HasSuffix(e.Message.Content, w.Suffix) {
			return nil
		}
		if _, err := e.Client().Rest.CreateMessage(e.ChannelID,
			discordutil.NewMessageBuilder().
				SetContentf("%s\n%s",
					i18n.TranslateText(u.Locale, "components.message.suffix.warn.message.1", nil),
					i18n.TranslateText(u.Locale, "components.message.suffix.warn.message.2", map[string]any{"suffix": w.Suffix}),
				).
				SetMessageReferenceByID(e.MessageID).
				BuildCreate(),
		); err != nil {
			slog.Error("メッセージを作成できません", "err", err)
			return errors.NewError(err)
		}
	case WordSuffixRuleWebhook:
		var content string
		for s := range strings.SplitSeq(e.Message.Content, "\n") {
			if s == "" {
				content += "\n"
				continue
			}
			// すでに語尾がある場合はそのまま通す
			if strings.HasSuffix(s, w.Suffix) {
				content += s + "\n"
				continue
			}
			// 末尾に文字列以外の文字がある場合(アルファベット、かな漢字以外のすべての文字)それらの前に語尾をなければ追加する
			// 例: "こんにちは→→"の場合、"こんにちは"の後ろに語尾を追加し、"→→"の後ろには追加しない
			runes := []rune(s)
			i := len(runes) - 1
			for i >= 0 {
				if (runes[i] < 'A' || runes[i] > 'Z') && (runes[i] < 'a' || runes[i] > 'z') && (runes[i] < '0' || runes[i] > '9') && !unicode.In(runes[i], unicode.Hiragana, unicode.Katakana, unicode.Han) {
					i--
				} else {
					break
				}
			}
			if i < 0 {
				// すべて文字列以外の文字の場合、そのまま通す
				content += s + "\n"
				continue
			}
			content += string(runes[:i+1]) + w.Suffix + string(runes[i+1:]) + "\n"
		}
		content = content[:len(content)-1] // 最後の改行を削除

		// メッセージを削除
		if err := e.Client().Rest.DeleteMessage(e.ChannelID, e.MessageID); err != nil {
			return errors.NewError(err)
		}
		member, err := e.Client().Rest.GetMember(e.GuildID, e.Message.Author.ID)
		if err != nil {
			return errors.NewError(err)
		}
		mentionUsers := make([]snowflake.ID, len(e.Message.Mentions))
		for i, u := range e.Message.Mentions {
			mentionUsers[i] = u.ID
		}
		repliedUser := false
		if e.Message.MessageReference != nil && e.Message.MessageReference.ChannelID != nil && e.Message.MessageReference.MessageID != nil {
			replyMessage, err := e.Client().Rest.GetMessage(*e.Message.MessageReference.ChannelID, *e.Message.MessageReference.MessageID)
			if err == nil {
				repliedUser = slices.Index(mentionUsers, replyMessage.Author.ID) != -1
			}
		}

		var channel discord.Channel
		channel, ok := e.Channel()
		if !ok {
			channel, err = e.Client().Rest.GetChannel(e.ChannelID)
			if err != nil {
				return errors.NewError(err)
			}
		}

		webhook, err := discordutil.GetWebhookMessenger(e.Client(), channel)
		if err != nil {
			return errors.NewError(err)
		}

		if _, err := webhook.SendWebhook(
			discordutil.NewMessageBuilder().
				SetContent(content).
				SetAllowedMentions(
					&discord.AllowedMentions{
						Users:       mentionUsers,
						Roles:       e.Message.MentionRoles,
						RepliedUser: repliedUser,
					},
				),
			member.EffectiveName(),
			e.Message.Author.EffectiveAvatarURL(),
			"",
		); err != nil {
			return errors.NewError(err)
		}
	}
	return nil
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
