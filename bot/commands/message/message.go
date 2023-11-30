package message

import (
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/wordsuffix"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/sabafly/gobot/internal/webhookutil"
)

func Command(c *components.Components) *generic.GenericCommand {
	return (&generic.GenericCommand{
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
				},
			},
		},

		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/message/suffix/set": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("message.manage.suffix", discord.PermissionManageMessages),
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
				PCommandHandler: generic.PermissionCommandCheck("message.manage.suffix", discord.PermissionManageMessages),
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
			"/message/pin/create": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("message.manage.pin", discord.PermissionManageChannels),
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
				PCommandHandler: generic.PermissionCommandCheck("message.manage.pin.delete"),
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
				if err := func(e *events.GuildMessageCreate, c *components.Components) errors.Error {
					id, _, err := webhookutil.GetWebhook(e.Client(), e.ChannelID)
					if err != nil {
						return errors.NewError(err)
					}
					if e.Message.WebhookID != nil && id == *e.Message.WebhookID {
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

					if m.RateLimit.CheckLimit() {
						if m.BeforeID != nil {
							if err := e.Client().Rest().DeleteMessage(e.ChannelID, *m.BeforeID); err != nil {
								slog.Error("削除に失敗", "err", err)
								m.BeforeID = nil
							}
						}

						message, err := webhookutil.SendWebhook(e.Client(), m.ChannelID,
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

						m.Update().SetBeforeID(message.ID).SetRateLimit(m.RateLimit).ExecX(e)
						slog.Info("ピン留め更新", "cid", e.ChannelID, "mid", e.MessageID)
					} else {
						m.Update().SetRateLimit(m.RateLimit).ExecX(e)
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
		mention_users := make([]snowflake.ID, len(e.Message.Mentions))
		for i, u := range e.Message.Mentions {
			mention_users[i] = u.ID
		}
		replied_user := false
		if e.Message.MessageReference != nil && e.Message.MessageReference.ChannelID != nil && e.Message.MessageReference.MessageID != nil {
			reply_message, err := e.Client().Rest().GetMessage(*e.Message.MessageReference.ChannelID, *e.Message.MessageReference.MessageID)
			if err == nil {
				replied_user = slices.Index(mention_users, reply_message.Author.ID) != -1
			}
		}
		if _, err := webhookutil.SendWebhook(e.Client(), e.ChannelID,
			discord.NewWebhookMessageCreateBuilder().
				SetAvatarURL(e.Message.Author.EffectiveAvatarURL()).
				SetUsername(member.EffectiveName()).
				SetContent(content).
				SetAllowedMentions(
					&discord.AllowedMentions{
						Users:       mention_users,
						Roles:       e.Message.MentionRoles,
						RepliedUser: replied_user,
					},
				).
				Build(),
		); err != nil {
			return errors.NewError(err)
		}
	}
	return nil
}
