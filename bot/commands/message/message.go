package message

import (
	"context"
	"fmt"
	"github.com/disgoorg/disgo/rest"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/wordsuffix"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/parse"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/sabafly/gobot/internal/webhookutil"
)

func Command(c *components.Components) *generic.Command {
	return (&generic.Command{
		Namespace: "message",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:         "message",
				Description:  "message",
				DMPermission: builtin.Ptr(false),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "suffix",
						Description: "suffix",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "set",
								Description:              "set member's suffix",
								DescriptionLocalizations: translate.MessageMap("components.message.suffix.set.command.description", false),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionUser{
										Name:                     "target",
										NameLocalizations:        translate.MessageMap("components.message.suffix.set.command.options.target.name", false),
										Description:              "target",
										DescriptionLocalizations: translate.MessageMap("components.message.suffix.set.command.options.target.description", false),
										Required:                 true,
									},
									discord.ApplicationCommandOptionString{
										Name:                     "suffix",
										NameLocalizations:        translate.MessageMap("components.message.suffix.set.command.options.suffix.name", false),
										Description:              "suffix",
										DescriptionLocalizations: translate.MessageMap("components.message.suffix.set.command.options.suffix.description", false),
										Required:                 true,
										MaxLength:                builtin.Ptr(512),
									},
									discord.ApplicationCommandOptionString{
										Name:                     "rule",
										NameLocalizations:        translate.MessageMap("components.message.suffix.set.command.options.rule.name", false),
										Description:              "rule",
										DescriptionLocalizations: translate.MessageMap("components.message.suffix.set.command.options.rule.description", false),
										Required:                 true,
										Choices: []discord.ApplicationCommandOptionChoiceString{
											{
												Name:              "webhook",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.rule.webhook", false),
												Value:             wordsuffix.RuleWebhook.String(),
											},
											{
												Name:              "warn",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.rule.warn", false),
												Value:             wordsuffix.RuleWarn.String(),
											},
											{
												Name:              "delete",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.rule.delete", false),
												Value:             wordsuffix.RuleDelete.String(),
											},
										},
									},
									discord.ApplicationCommandOptionInt{
										Name:                     "duration",
										NameLocalizations:        translate.MessageMap("components.message.suffix.set.command.options.duration.name", false),
										Description:              "duration",
										DescriptionLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.description", false),
										Required:                 false,
										Choices: []discord.ApplicationCommandOptionChoiceInt{
											{
												Name:              "1m",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.1m", false),
												Value:             0,
											},
											{
												Name:              "1h",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.1h", false),
												Value:             1,
											},
											{
												Name:              "3h",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.3h", false),
												Value:             2,
											},
											{
												Name:              "6h",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.6h", false),
												Value:             3,
											},
											{
												Name:              "1d",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.1d", false),
												Value:             4,
											},
											{
												Name:              "3d",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.3d", false),
												Value:             5,
											},
											{
												Name:              "1w",
												NameLocalizations: translate.MessageMap("components.message.suffix.set.command.options.duration.1w", false),
												Value:             6,
											},
										},
									},
								},
							},
							{
								Name:                     "remove",
								Description:              "remove member's suffix",
								DescriptionLocalizations: translate.MessageMap("components.message.suffix.remove.command.description", false),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionUser{
										Name:                     "target",
										NameLocalizations:        translate.MessageMap("components.message.suffix.remove.command.options.target.name", false),
										Description:              "target",
										DescriptionLocalizations: translate.MessageMap("components.message.suffix.remove.command.options.target.description", false),
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
										MinLength:   builtin.Ptr(1),
										MaxLength:   builtin.Ptr(16),
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

					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					u, err := c.UserCreate(event, event.SlashCommandInteractionData().User("target"))
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					var expired *time.Time
					if duration, ok := event.SlashCommandInteractionData().OptInt("duration"); ok {
						var d time.Duration
						switch duration {
						case 0:
							d = time.Minute
						case 1:
							d = time.Hour
						case 2:
							d = time.Hour * 3
						case 3:
							d = time.Hour * 6
						case 4:
							d = time.Hour * 24
						case 5:
							d = time.Hour * 24 * 3
						case 6:
							d = time.Hour * 24 * 7
						}
						expired = builtin.Or(d != 0, builtin.Ptr(time.Now().Add(d)), nil)
					}
					var w *ent.WordSuffix
					if u.QueryWordSuffix().Where(wordsuffix.GuildID(g.ID)).ExistX(event) {
						w = u.QueryWordSuffix().Where(wordsuffix.GuildID(g.ID)).OnlyX(event)
						w.Update().
							SetSuffix(event.SlashCommandInteractionData().String("suffix")).
							SetOwner(u).
							SetRule(wordsuffix.Rule(event.SlashCommandInteractionData().String("rule"))).
							SetNillableExpired(expired).
							SaveX(event)
					} else {
						w = c.DB().WordSuffix.
							Create().
							SetGuild(g).
							SetSuffix(event.SlashCommandInteractionData().String("suffix")).
							SetOwner(u).
							SetRule(wordsuffix.Rule(event.SlashCommandInteractionData().String("rule"))).
							SetNillableExpired(expired).
							SaveX(event)
					}
					var durationString string
					if expired != nil {
						durationString = discord.FormattedTimestampMention(expired.Unix(), discord.TimestampStyleRelative)
					} else {
						durationString = translate.Message(event.Locale(), "components.message.suffix.duration.none")
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContentf("%s\n%s",
								translate.Message(
									event.Locale(),
									"components.message.suffix.set.message",
									translate.WithTemplate(map[string]any{"User": discord.UserMention(u.ID), "Suffix": w.Suffix}),
								),
								translate.Message(
									event.Locale(),
									"components.message.suffix.duration.message",
									translate.WithTemplate(map[string]any{"Duration": durationString}),
								),
							).
							Create(),
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

					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					u, err := c.UserCreate(event, event.SlashCommandInteractionData().User("target"))
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}

					if !u.QueryWordSuffix().Where(wordsuffix.GuildID(g.ID)).ExistX(event) {
						if err := event.CreateMessage(
							discord.NewMessageBuilder().
								SetContent(translate.Message(event.Locale(), "components.message.suffix.remove.message.no_suffix", translate.WithTemplate(map[string]any{"User": discord.UserMention(u.ID)}))).
								SetAllowedMentions(&discord.AllowedMentions{}).
								SetFlags(discord.MessageFlagEphemeral).
								Create(),
						); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					c.DB().WordSuffix.DeleteOneID(u.QueryWordSuffix().Where(wordsuffix.GuildID(g.ID)).FirstIDX(event)).ExecX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.message.suffix.remove.message", translate.WithTemplate(map[string]any{"User": discord.UserMention(u.ID)}))).
							Create(),
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

					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}
					u, err := c.UserCreate(event, m.User)
					if err != nil {
						slog.Error("ユーザー取得に失敗", "err", err, "command", event.SlashCommandInteractionData().CommandPath())
						return errors.NewError(err)
					}

					messageStr := translate.Message(event.Locale(), "components.message.suffix.check.message.none",
						translate.WithTemplate(map[string]any{"User": discord.UserMention(u.ID)}),
					)
					if u.QueryWordSuffix().Where(
						wordsuffix.GuildID(g.ID),
					).ExistX(event) {
						w := u.QueryWordSuffix().Where(
							wordsuffix.GuildID(g.ID),
						).FirstX(event)
						messageStr = translate.Message(event.Locale(), "components.message.suffix.check.message",
							translate.WithTemplate(
								map[string]any{
									"Duration": builtin.Or(w.Expired != nil,
										discord.FormattedTimestampMention(builtin.NonNil(w.Expired).Unix(), discord.TimestampStyleRelative),
										translate.Message(event.Locale(), "components.message.suffix.duration.none"),
									),
									"User":   discord.UserMention(u.ID),
									"Suffix": w.Suffix,
									"Rule":   translate.Message(event.Locale(), "components.message.suffix.set.command.options.rule."+w.Rule.String()),
								},
							),
						)
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(messageStr).
							SetAllowedMentions(&discord.AllowedMentions{}).
							Create(),
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
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.Modal(
						discord.NewModalCreateBuilder().
							SetTitle(translate.Message(event.Locale(), "components.message.pin.create.modal.title")).
							SetCustomID("message:pin_create_modal").
							SetContainerComponents(
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "content",
										Style:     discord.TextInputStyleParagraph,
										Label:     translate.Message(event.Locale(), "components.message.pin.create.modal.input.1.label"),
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
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					if !g.QueryMessagePins().Where(messagepin.ChannelID(event.Channel().ID())).ExistX(event) {
						return errors.NewError(errors.ErrorMessage("errors.unavailable.message.pin", event))
					}
					if beforeID := g.QueryMessagePins().Where(messagepin.ChannelID(event.Channel().ID())).FirstX(event).BeforeID; beforeID != nil {
						_ = event.Client().Rest().DeleteMessage(event.Channel().ID(), *beforeID)
					}

					c.DB().MessagePin.Delete().Where(messagepin.ChannelID(event.Channel().ID())).ExecX(event)

					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.message.pin.delete.message")).
							SetFlags(discord.MessageFlagEphemeral).
							Create(),
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
					g, err := c.GuildCreateID(event, *event.GuildID())
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
						discord.NewModalCreateBuilder().
							SetTitle(translate.Message(event.Locale(), "components.message.remind.add.modal.title")).
							SetCustomID(fmt.Sprintf("message:remind_create_modal:%d", tm.Unix())).
							SetContainerComponents(
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "content",
										Style:     discord.TextInputStyleParagraph,
										Label:     translate.Message(event.Locale(), "components.message.remind.add.modal.input.content.label"),
										MinLength: builtin.Ptr(1),
										MaxLength: 1000,
										Required:  true,
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "name",
										Style:     discord.TextInputStyleShort,
										Label:     translate.Message(event.Locale(), "components.message.remind.add.modal.input.name.label"),
										MinLength: builtin.Ptr(1),
										MaxLength: 64,
										Required:  true,
										Value: fmt.Sprintf("%s#%d",
											translate.Message(event.Locale(), "components.message.remind.add.modal.input.name.value"),
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
					count := c.DB().MessageRemind.Delete().Where(
						messageremind.HasGuildWith(guild.ID(*event.GuildID())),
						messageremind.NameContains(event.SlashCommandInteractionData().String("remind")),
					).ExecX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.message.remind.cancel.message",
								translate.WithTemplate(map[string]any{
									"Count": strconv.Itoa(count),
								}),
							)).
							Create(),
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
					reminds := c.DB().MessageRemind.Query().Where(
						messageremind.HasGuildWith(guild.ID(*event.GuildID())),
						messageremind.NameContains(event.Data.String("remind")),
					).
						Limit(25).
						AllX(event)

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
				g, err := component.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}

				// もし既にあったら抹消する
				if g.QueryMessagePins().Where(messagepin.ChannelID(event.Channel().ID())).ExistX(event) {
					if beforeID := g.QueryMessagePins().Where(messagepin.ChannelID(event.Channel().ID())).FirstX(event).BeforeID; beforeID != nil {
						_ = event.Client().Rest().DeleteMessage(event.Channel().ID(), *beforeID)
					}

					component.DB().MessagePin.Delete().Where(messagepin.ChannelID(event.Channel().ID())).ExecX(event)
				}

				m := component.DB().MessagePin.Create().
					SetChannelID(event.Channel().ID()).
					SetContent(event.Data.Text("content")).
					SetGuild(g).
					SaveX(event)

				message, err := webhookutil.SendWebhook(event.Client(), m.ChannelID,
					discord.NewWebhookMessageCreateBuilder().
						SetAvatarURL(component.Config().Message.PinIconImage).
						SetUsername(translate.Message(g.Locale, "components.message.pin.username")).
						SetContent(m.Content).
						SetEmbeds(m.Embeds...).
						Build(),
				)
				if err != nil {
					return errors.NewError(err)
				}

				m.Update().SetBeforeID(message.ID).SaveX(event)

				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent(translate.Message(event.Locale(), "components.message.pin.create.message")).
						SetFlags(discord.MessageFlagEphemeral).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
			"message:remind_create_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				args := strings.Split(event.Data.CustomID, ":")
				tm := time.Unix(builtin.Must(strconv.ParseInt(args[2], 10, 64)), 0)
				if time.Now().After(tm) {
					return errors.NewError(errors.ErrorMessage("errors.invalid.time.before", event))
				}
				c.DB().MessageRemind.Create().
					SetGuild(g).
					SetTime(tm).
					SetContent(event.Data.Text("content")).
					SetChannelID(event.Channel().ID()).
					SetAuthorID(event.Member().User.ID).
					SetName(event.Data.Text("name")).
					ExecX(event)
				g.Update().AddRemindCount(1).ExecX(event)

				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent(translate.Message(event.Locale(), "components.message.remind.add.message",
							translate.WithTemplate(map[string]any{
								"Time": discord.FormattedTimestampMention(tm.Unix(), discord.TimestampStyleLongDateTime),
							}),
						)).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
		},

		Schedulers: []components.Scheduler{
			{
				Duration: time.Minute,
				Worker: func(c *components.Components, client bot.Client) error {
					reminds := c.DB().MessageRemind.Query().
						Where(
							messageremind.TimeLT(time.Now()),
						).
						AllX(context.Background())
					for _, remind := range reminds {
						if _, err := client.Rest().CreateMessage(remind.ChannelID,
							discord.NewMessageBuilder().
								SetContent(remind.Content).
								Create(),
						); err != nil {
							return err
						}
					}

					c.DB().MessageRemind.Delete().
						Where(
							messageremind.TimeLT(time.Now()),
						).
						ExecX(context.Background())
					return nil
				},
			},
		},

		EventHandler: func(c *components.Components, e bot.Event) errors.Error {
			switch e := e.(type) {
			case *events.GuildMessageCreate:

				// 語尾の処理
				if err := messageSuffixMessageCreateHandler(e, c); err != nil {
					return err
				}

				if ok := c.GetLock("message_pin").Mutex(e.ChannelID).TryLock(); !ok {
					return nil
				}
				defer c.GetLock("message_pin").Mutex(e.ChannelID).Unlock()

				// ピン留めメッセージの処理
				if err := func(event *events.GuildMessageCreate, c *components.Components) errors.Error {
					id, _, err := webhookutil.GetWebhook(event.Client(), event.ChannelID)
					if err != nil {
						err1 := rest.Error{}
						if errors.As(err, &err1) && err1.Response.StatusCode == http.StatusForbidden {
							// TODO: リファクタ
							return errors.NewError(event.Client().Rest().LeaveGuild(event.GuildID))
						}
						return errors.NewError(err)
					}
					if event.Message.WebhookID != nil && id == *event.Message.WebhookID {
						return nil
					}

					g, err := c.GuildCreateID(event, event.GuildID)
					if err != nil {
						return errors.NewError(err)
					}
					if !g.QueryMessagePins().Where(messagepin.ChannelID(event.ChannelID)).ExistX(event) {
						return nil
					}
					m := g.QueryMessagePins().Where(messagepin.ChannelID(event.ChannelID)).FirstX(event)

					if m.RateLimit.CheckLimit() {
						if m.BeforeID != nil {
							if err := event.Client().Rest().DeleteMessage(event.ChannelID, *m.BeforeID); err != nil {
								slog.Error("削除に失敗", "err", err)
								m.BeforeID = nil
							}
						}

						message, err := webhookutil.SendWebhook(event.Client(), m.ChannelID,
							discord.NewWebhookMessageCreateBuilder().
								SetAvatarURL(c.Config().Message.PinIconImage).
								SetUsername(translate.Message(g.Locale, "components.message.pin.username")).
								SetContent(m.Content).
								SetEmbeds(m.Embeds...).
								Build(),
						)
						if err != nil {
							return errors.NewError(err)
						}

						m.Update().SetBeforeID(message.ID).SetRateLimit(m.RateLimit).ExecX(event)
						slog.Info("ピン留め更新", "cid", event.ChannelID, "mid", event.MessageID)
					} else {
						m.Update().SetRateLimit(m.RateLimit).ExecX(event)
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

				id, _, err := webhookutil.GetWebhook(e.Client(), e.ChannelID)
				if err != nil {
					return errors.NewError(err)
				}
				if e.Message.WebhookID != nil && id != *e.Message.WebhookID {
					return nil
				}

				g, err := c.GuildCreateID(e, e.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				if !g.QueryMessagePins().Where(messagepin.ChannelID(e.ChannelID)).ExistX(e) {
					return nil
				}
				m := g.QueryMessagePins().Where(messagepin.ChannelID(e.ChannelID)).FirstX(e)

				if m.BeforeID != nil && *m.BeforeID == e.MessageID {
					slog.Info("ピン留め削除", "cid", e.ChannelID, "mid", e.MessageID)
					c.DB().MessagePin.DeleteOneID(m.ID).ExecX(e)
				}
			}
			return nil
		},
	}).SetComponent(c)
}

func messageSuffixMessageCreateHandler(e *events.GuildMessageCreate, c *components.Components) errors.Error {
	slog.Debug("メッセージ作成")
	if e.Message.Content == "" {
		return nil
	}
	if e.Message.Type.System() || e.Message.Author.System || e.Message.Author.Bot {
		return nil
	}
	if e.Message.Type != discord.MessageTypeDefault && e.Message.Type != discord.MessageTypeReply {
		return nil
	}

	u, err := c.UserCreate(e, e.Message.Author)
	if err != nil {
		slog.Error("メッセージ著者取得に失敗", "err", err, "uid", e.Message.Author.ID)
		return errors.NewError(err)
	}

	var w *ent.WordSuffix

	if u.QueryWordSuffix().Where(wordsuffix.GuildID(e.GuildID)).ExistX(e) {
		// Guild
		w = u.QueryWordSuffix().Where(wordsuffix.GuildID(e.GuildID)).FirstX(e)
	} else {
		// Global
		if !u.QueryWordSuffix().Where(wordsuffix.GuildIDIsNil()).ExistX(e) {
			slog.Debug("語尾が存在しません")
			return nil
		}
		w = u.QueryWordSuffix().Where(wordsuffix.GuildIDIsNil()).FirstX(e)
	}
	if w.Expired != nil && time.Now().Compare(*w.Expired) == 1 {
		c.DB().WordSuffix.DeleteOneID(w.ID).ExecX(e)
		return nil
	}
	switch w.Rule {
	case wordsuffix.RuleDelete:
		if strings.HasSuffix(e.Message.Content, w.Suffix) {
			return nil
		}
		if err := e.Client().Rest().DeleteMessage(e.ChannelID, e.MessageID); err != nil {
			slog.Error("メッセージを削除できません", "err", err)
			return errors.NewError(err)
		}
	case wordsuffix.RuleWarn:
		if strings.HasSuffix(e.Message.Content, w.Suffix) {
			return nil
		}
		if _, err := e.Client().Rest().CreateMessage(e.ChannelID,
			discord.NewMessageBuilder().
				SetContentf("%s\n%s",
					translate.Message(u.Locale, "components.message.suffix.warn.message.1"),
					translate.Message(u.Locale, "components.message.suffix.warn.message.2", translate.WithTemplate(map[string]any{"Suffix": w.Suffix})),
				).
				SetMessageReferenceByID(e.MessageID).
				Create(),
		); err != nil {
			slog.Error("メッセージを作成できません", "err", err)
			return errors.NewError(err)
		}
	case wordsuffix.RuleWebhook:
		content := e.Message.Content
		if !strings.HasSuffix(e.Message.Content, w.Suffix) {
			content += w.Suffix
		}
		if err := e.Client().Rest().DeleteMessage(e.ChannelID, e.MessageID); err != nil {
			return errors.NewError(err)
		}
		member, err := e.Client().Rest().GetMember(e.GuildID, e.Message.Author.ID)
		if err != nil {
			return errors.NewError(err)
		}
		mentionUsers := make([]snowflake.ID, len(e.Message.Mentions))
		for i, u := range e.Message.Mentions {
			mentionUsers[i] = u.ID
		}
		repliedUser := false
		if e.Message.MessageReference != nil && e.Message.MessageReference.ChannelID != nil && e.Message.MessageReference.MessageID != nil {
			replyMessage, err := e.Client().Rest().GetMessage(*e.Message.MessageReference.ChannelID, *e.Message.MessageReference.MessageID)
			if err == nil {
				repliedUser = slices.Index(mentionUsers, replyMessage.Author.ID) != -1
			}
		}
		if _, err := webhookutil.SendWebhook(e.Client(), e.ChannelID,
			discord.NewWebhookMessageCreateBuilder().
				SetAvatarURL(e.Message.Author.EffectiveAvatarURL()).
				SetUsername(member.EffectiveName()).
				SetContent(content).
				SetAllowedMentions(
					&discord.AllowedMentions{
						Users:       mentionUsers,
						Roles:       e.Message.MentionRoles,
						RepliedUser: repliedUser,
					},
				).
				Build(),
		); err != nil {
			return errors.NewError(err)
		}
	}
	return nil
}
