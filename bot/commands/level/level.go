package level

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/sabafly/gobot/internal/xppoint"
)

func Command(c *components.Components) components.Command {
	return (&generic.GenericCommand{
		Namespace: "level",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:         "level",
				Description:  "level",
				DMPermission: builtin.Ptr(false),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:        "rank",
						Description: "view your level and points",
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "leaderboard",
						Description: "view guild rank leaderboard",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:        "page",
								Description: "page number",
								Required:    false,
								MinValue:    builtin.Ptr(1),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "transfer",
						Description: "transfer xp to someone",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionUser{
								Name:        "to",
								Description: "who transfer xp to",
								Required:    true,
							},
							discord.ApplicationCommandOptionUser{
								Name:        "from",
								Description: "who transfer xp from",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "up",
						Description: "up",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "message",
								Description: "set level up message",
							},
							{
								Name:        "message-channel",
								Description: "set level up message channel",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionChannel{
										Name:        "channel",
										Description: "channel",
										Required:    false,
										ChannelTypes: []discord.ChannelType{
											discord.ChannelTypeGuildText,
											discord.ChannelTypeGuildNews,
										},
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "exclude-channel",
						Description: "exclude-channel",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "add",
								Description: "add exclude channel",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionChannel{
										Name:        "channel",
										Description: "channel",
										Required:    true,
										ChannelTypes: []discord.ChannelType{
											discord.ChannelTypeGuildText,
											discord.ChannelTypeGuildNews,
											discord.ChannelTypeGuildVoice,
											discord.ChannelTypeGuildForum,
											discord.ChannelTypeGuildStageVoice,
										},
									},
								},
							},
							{
								Name:        "remove",
								Description: "remove exclude channel",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionChannel{
										Name:        "channel",
										Description: "channel",
										Required:    true,
										ChannelTypes: []discord.ChannelType{
											discord.ChannelTypeGuildText,
											discord.ChannelTypeGuildNews,
											discord.ChannelTypeGuildVoice,
											discord.ChannelTypeGuildForum,
											discord.ChannelTypeGuildStageVoice,
										},
									},
								},
							},
							{
								Name:        "clear",
								Description: "clear exclude channels",
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/level/rank": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				m, err := c.MemberCreate(event, event.User(), *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				gl, err := c.GuildRequest(event.Client(), g.ID)
				if err != nil {
					return errors.NewError(err)
				}
				ids := g.QueryMembers().Order(
					member.ByXp(
						sql.OrderDesc(),
					),
				).IDsX(event)
				index := slices.Index(ids, m.ID)
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							embeds.SetEmbedProperties(
								level_message(g, gl, m, index, event.Member().Member, event),
							),
						).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/level/leaderboard": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				gl, err := c.GuildRequest(event.Client(), g.ID)
				if err != nil {
					return errors.NewError(err)
				}
				page := event.SlashCommandInteractionData().Int("page")
				page = builtin.Or(page > 0, page, 1)
				count := g.QueryMembers().CountX(event)
				if page > count/50+1 {
					return errors.NewError(errors.ErrorMessage("errors.invalid.page", event))
				}
				members := g.QueryMembers().
					Order(
						member.ByXp(
							sql.OrderDesc(),
						),
					).
					Offset((page - 1) * 50).
					Limit(50).
					AllX(event)
				var leaderboard string
				for i, m := range members {
					leaderboard += fmt.Sprintf("**#%d | %s XP: `%d` Level: `%d`**\n",
						i+1+((page-1)*50),
						discord.UserMention(m.UserID),
						m.Xp, m.Xp.Level(),
					)
				}
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							embeds.SetEmbedProperties(
								discord.NewEmbedBuilder().
									SetEmbedAuthor(
										&discord.EmbedAuthor{
											Name:    g.Name,
											IconURL: builtin.NonNil(gl.IconURL()),
										},
									).
									SetTitlef("🏆%s(%d/%d)",
										translate.Message(event.Locale(), "components.level.leaderboard.title"),
										page,
										count/50+1,
									).
									SetDescription(leaderboard).
									Build(),
							),
						).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/level/transfer": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("level.transfer"),
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					gl, err := c.GuildRequest(event.Client(), g.ID)
					if err != nil {
						return errors.NewError(err)
					}

					to := event.SlashCommandInteractionData().Member("to")
					from, ok := event.SlashCommandInteractionData().OptMember("from")
					if !ok {
						from = *event.Member()
					}
					if from.User.Bot || from.User.System || to.User.Bot || to.User.System {
						return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
					}
					if to.User.ID == from.User.ID {
						return errors.NewError(errors.ErrorMessage("errors.invalid.self.target", event))
					}

					fromUser, err := c.MemberCreate(event, from.User, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					toUser, err := c.MemberCreate(event, to.User, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					movedXp := uint64(fromUser.Xp)
					toUser.Xp.Add(movedXp)
					fromUser.Xp = xppoint.XP(0)
					toUser = toUser.Update().SetXp(toUser.Xp).SaveX(event)
					fromUser = fromUser.Update().SetXp(fromUser.Xp).SaveX(event)

					ids := g.QueryMembers().Order(
						member.ByXp(
							sql.OrderDesc(),
						),
					).IDsX(event)
					fromIndex := slices.Index(ids, fromUser.ID)
					toIndex := slices.Index(ids, toUser.ID)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedsProperties(
									[]discord.Embed{
										level_message(g, gl, fromUser, fromIndex, from.Member, event),
										level_message(g, gl, toUser, toIndex, to.Member, event),
										discord.NewEmbedBuilder().
											SetTitlef("`%d`xp 移動しました", movedXp).
											Build(),
									},
								)...,
							).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/up/message": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("level.up.message", discord.PermissionManageGuild),
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if err := event.Modal(
						discord.NewModalCreateBuilder().
							SetTitle(translate.Message(event.Locale(), "components.level.up.message.modal.title")).
							SetCustomID("level:up_message_modal").
							SetContainerComponents(
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:    "message",
										Style:       discord.TextInputStyleParagraph,
										Label:       translate.Message(event.Locale(), "components.level.up.message.modal.input.message"),
										MinLength:   builtin.Ptr(1),
										MaxLength:   140,
										Required:    true,
										Placeholder: translate.Message(event.Locale(), "components.level.up.message.modal.input.message.placeholder"),
										Value:       g.LevelUpMessage,
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
			"/level/up/message-channel": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("level.up.message-channel"),
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if channel, ok := event.SlashCommandInteractionData().OptChannel("channel"); ok {
						g = g.Update().
							SetLevelUpChannel(channel.ID).
							SaveX(event)
					} else {
						g = g.Update().
							ClearLevelUpChannel().
							SaveX(event)
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.level.up.message-channel.message",
								translate.WithTemplate(map[string]any{
									"Channel": builtin.Or(g.LevelUpChannel != nil,
										discord.ChannelMention(builtin.NonNil(g.LevelUpChannel)),
										translate.Message(event.Locale(), "components.level.up.message-channel.default"),
									),
								}),
							)).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/exclude-channel/add": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("level.exclude-channel.add"),
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					channel := event.SlashCommandInteractionData().Channel("channel")
					if !slices.Contains(g.LevelUpExcludeChannel, channel.ID) {
						g.LevelUpExcludeChannel = append(g.LevelUpExcludeChannel, channel.ID)
						g.Update().
							SetLevelUpExcludeChannel(g.LevelUpExcludeChannel).
							ExecX(event)
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.level.exclude-channel.add.message",
								translate.WithTemplate(map[string]any{"Channel": discord.ChannelMention(channel.ID)}),
							)).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/exclude-channel/remove": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("level.exclude-channel.add"),
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					channel := event.SlashCommandInteractionData().Channel("channel")
					if index := slices.Index(g.LevelUpExcludeChannel, channel.ID); index != -1 {
						g.LevelUpExcludeChannel = slices.Delete(g.LevelUpExcludeChannel, index, index+1)
						g.Update().
							SetLevelUpExcludeChannel(g.LevelUpExcludeChannel).
							ExecX(event)
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.level.exclude-channel.remove.message",
								translate.WithTemplate(map[string]any{"Channel": discord.ChannelMention(channel.ID)}),
							)).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/exclude-channel/clear": generic.PCommandHandler{
				PCommandHandler: generic.PermissionCommandCheck("level.exclude-channel.add"),
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g.Update().
						ClearLevelUpExcludeChannel().
						ExecX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.level.exclude-channel.clear.message")).
							Create(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"level:up_message_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				g = g.Update().
					SetLevelUpMessage(event.ModalSubmitInteraction.Data.Text("message")).
					SaveX(event)
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							embeds.SetEmbedProperties(
								discord.NewEmbedBuilder().
									SetTitle(translate.Message(event.Locale(), "components.level.up.message.message")).
									SetDescription(g.LevelUpMessage).
									Build(),
							),
						).
						Create(),
				); err != nil {
					return nil
				}
				return nil
			},
		},
		EventHandler: func(c *components.Components, event bot.Event) errors.Error {
			switch event := event.(type) {
			case *events.GuildMessageCreate:
				if event.Message.Author.Bot || event.Message.Author.System || event.Message.Type.System() {
					return nil
				}
				if event.Message.Type != discord.MessageTypeDefault && event.Message.Type != discord.MessageTypeReply {
					return nil
				}
				g, err := c.GuildCreateID(event, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				if err != nil {
					return errors.NewError(err)
				}
				if slices.Contains(g.LevelUpExcludeChannel, event.ChannelID) {
					return nil
				}
				m, err := c.MemberCreate(event, event.Message.Author, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				memberUpdate := m.Update()
				before := m.Xp.Level()
				if time.Now().After(m.LastXp.Add(time.Minute * 3)) {
					m.Xp.AddRandom()
					memberUpdate.
						SetXp(m.Xp).
						SetLastXp(time.Now())
				}
				after := m.Xp.Level()
				m = memberUpdate.
					SetMessageCount(m.MessageCount + 1).
					SaveX(event)
				if before != after {
					// レベルアップ通知
					content := g.LevelUpMessage
					content = strings.ReplaceAll(content, "{user}", discord.UserMention(m.UserID))
					content = strings.ReplaceAll(content, "{username}", event.Message.Author.EffectiveName())
					content = strings.ReplaceAll(content, "{before_level}", strconv.FormatInt(before, 10))
					content = strings.ReplaceAll(content, "{after_level}", strconv.FormatInt(after, 10))
					content = strings.ReplaceAll(content, "{xp}", strconv.FormatUint(uint64(m.Xp), 10))
					if _, err := event.Client().Rest().
						CreateMessage(
							builtin.Or(builtin.NonNil(g.LevelUpChannel) != 0, builtin.NonNil(g.LevelUpChannel), event.ChannelID),
							discord.NewMessageBuilder().
								SetContent(content).
								Create(),
						); err != nil {
						return errors.NewError(err)
					}
				}
			}
			return nil
		},
	}).SetComponent(c)
}
