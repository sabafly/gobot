package level

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/smap"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/sabafly/gobot/internal/xppoint"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "level",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "level",
				Description: "level",
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:        "rank",
						Description: "view your level and points",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionUser{
								Name:        "target",
								Description: "target user",
							},
						},
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
							{
								Name:        "list",
								Description: "list exclude channels",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "import-mee6",
						Description: "import xp point from mee6",
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "reset",
						Description: "reset user xp",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionUser{
								Name:        "target",
								Description: "target user",
								Required:    true,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "role",
						Description: "role",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "set",
								Description: "set level role",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionInt{
										Name:        "level",
										Description: "level number",
										Required:    true,
										MinValue:    builtin.Ptr(1),
										MaxValue:    builtin.Ptr(1000),
									},
									discord.ApplicationCommandOptionRole{
										Name:        "role",
										Description: "role",
										Required:    true,
									},
								},
							},
							{
								Name:        "remove",
								Description: "remove level role",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionInt{
										Name:        "level",
										Description: "level number",
										Required:    true,
										MinValue:    builtin.Ptr(1),
										MaxValue:    builtin.Ptr(1000),
									},
								},
							},
							{
								Name:        "list",
								Description: "list level roles",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:        "required-point",
						Description: "required point",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:        "level",
								Description: "level number",
								MinValue:    builtin.Ptr(1),
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/level/required-point": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("level.required-point"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					mem, err := c.MemberCreate(event, event.User(), *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					level := uint64(0)
					l, ok := event.SlashCommandInteractionData().OptInt("level")
					level = uint64(l)
					if !ok {
						level = mem.Xp.Level() + 1
					}
					builder := discord.NewMessageBuilder()
					builder.SetEmbeds(
						embeds.SetEmbedProperties(discord.NewEmbedBuilder().
							SetTitle(translate.Message(event.Locale(), "components.level.required-point.embed.title", translate.WithTemplate(map[string]any{"Level": level}))).
							SetDescriptionf("# `%d`xp\n%s\n%s",
								xppoint.TotalPoint(level),
								translate.Message(event.Locale(), "components.level.required-point.embed.description", translate.WithTemplate(map[string]any{"User": event.Member().EffectiveName(), "Xp": mem.Xp})),
								translate.Message(event.Locale(), "components.level.required-point.embed.description.diff", translate.WithTemplate(map[string]any{"Xp": builtin.Or(xppoint.TotalPoint(level) > uint64(mem.Xp), xppoint.TotalPoint(level)-uint64(mem.Xp), 0)})),
							).
							Build()),
					)
					if err := event.CreateMessage(builder.BuildCreate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/rank": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("level.rank"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					target, ok := event.SlashCommandInteractionData().OptMember("target")
					if !ok {
						target = *event.Member()
					}
					if target.User.Bot || target.User.System {
						return errors.NewError(errors.ErrorMessage("errors.invalid.bot.target", event))
					}
					m, err := c.MemberCreate(event, target.User, *event.GuildID())
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
									levelMessage(g, gl, m, index, target.Member, event),
								),
							).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/leaderboard": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("level.leaderboard"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					const pageCount = 25
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
					if page > count/pageCount+1 {
						return errors.NewError(errors.ErrorMessage("errors.invalid.page", event))
					}
					members := g.QueryMembers().
						Order(member.ByXp(sql.OrderDesc())).
						Offset((page - 1) * pageCount).
						Limit(pageCount).
						AllX(event)
					var leaderboard string
					for i, m := range members {
						leaderboard += fmt.Sprintf("**#%d | %s XP: `%d` Level: `%d`**\n",
							i+1+((page-1)*pageCount),
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
											count/pageCount+1,
										).
										SetDescription(leaderboard).
										Build(),
								),
							).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/transfer": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.transfer"),
				},
				DiscordPerm: discord.PermissionManageGuild.Add(discord.PermissionModerateMembers),
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
					fromUser.Xp = xppoint.XP(0)
					fromUser = fromUser.Update().SetXp(fromUser.Xp).ClearLastNotifiedLevel().SaveX(event)
					if toUser, err = addXp(event, toUser.Update(), movedXp, event.Client(), toUser, g, event.Channel().ID(), to.EffectiveName(), true); err != nil {
						return errors.NewError(err)
					}
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
										levelMessage(g, gl, fromUser, fromIndex, from.Member, event),
										levelMessage(g, gl, toUser, toIndex, to.Member, event),
										discord.NewEmbedBuilder().
											SetTitlef("`%d`xp 移動しました", movedXp).
											Build(),
									},
								)...,
							).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/up/message": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.up.message"),
				},
				DiscordPerm: discord.PermissionManageGuild,
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
				Permission: []generic.Permission{
					generic.PermissionString("level.message-channel"),
				},
				DiscordPerm: discord.PermissionManageGuild,
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
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/exclude-channel/add": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.add"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					channel := event.SlashCommandInteractionData().Channel("channel")
					if slices.Contains(g.LevelUpExcludeChannel, channel.ID) {
						return errors.NewError(errors.ErrorMessage("errors.already_exist", event))
					}
					g.LevelUpExcludeChannel = append(g.LevelUpExcludeChannel, channel.ID)
					g.Update().
						SetLevelUpExcludeChannel(g.LevelUpExcludeChannel).
						ExecX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.level.exclude-channel.add.message",
								translate.WithTemplate(map[string]any{"Channel": discord.ChannelMention(channel.ID)}),
							)).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/exclude-channel/remove": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.remove"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					channel := event.SlashCommandInteractionData().Channel("channel")
					index := slices.Index(g.LevelUpExcludeChannel, channel.ID)
					if index == -1 {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					g.LevelUpExcludeChannel = slices.Delete(g.LevelUpExcludeChannel, index, index+1)
					g.Update().
						SetLevelUpExcludeChannel(g.LevelUpExcludeChannel).
						ExecX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.level.exclude-channel.remove.message",
								translate.WithTemplate(map[string]any{"Channel": discord.ChannelMention(channel.ID)}),
							)).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/exclude-channel/clear": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.clear"),
				},
				DiscordPerm: discord.PermissionManageGuild,
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
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/import-mee6": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.import-mee6"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					if g.LevelMee6Imported {
						return errors.NewError(errors.ErrorMessage("components.level.import-mee6.message.already", event))
					}

					var members []discord.Member
					memberCount := 1000
					afterID := snowflake.ID(0)
					for memberCount == 1000 {
						m, err := event.Client().Rest().GetMembers(*event.GuildID(), memberCount, afterID)
						if err != nil {
							return errors.NewError(err)
						}
						memberCount = len(m)
						members = append(members, m...)
						afterID = m[len(m)-1].User.ID
					}

					slog.Info("mee6インポート", slog.Any("gid", event.GuildID()), slog.Int("member_count", len(members)))

					memberCount = 0
					url := fmt.Sprintf("https://mee6.xyz/api/plugins/levels/leaderboard/%s", event.GuildID().String())
					for page := 0; true; page++ {
						response, err := http.Get(fmt.Sprintf("%s?page=%d", url, page))
						if err != nil || response.StatusCode != http.StatusOK {
							switch response.StatusCode {
							case http.StatusUnauthorized:
								if err := event.RespondMessage(
									discord.NewMessageBuilder().
										SetContent(
											fmt.Sprintf("# FAILED\n```| STATUS CODE | %d\n| RESPONSE | %v```%s",
												response.StatusCode,
												err,
												translate.Message(event.Locale(), "components.level.import-mee6.message.unauthorized",
													translate.WithTemplate(map[string]any{"GuildID": *event.GuildID()}),
												),
											),
										),
								); err != nil {
									return errors.NewError(err)
								}
								return nil
							default:
								if err := event.RespondMessage(
									discord.NewMessageBuilder().
										SetContent(fmt.Sprintf("# FAILED\n```| STATUS CODE | %d\n| RESPONSE | %v```", response.StatusCode, err)),
								); err != nil {
									return errors.NewError(err)
								}
								return nil
							}
						}
						var leaderboard mee6LeaderBoard
						if err := json.NewDecoder(response.Body).Decode(&leaderboard); err != nil {
							return errors.NewError(err)
						}
						_ = response.Body.Close()
						if len(leaderboard.Players) < 1 {
							break
						}
						for _, player := range leaderboard.Players {
							index := slices.IndexFunc(members, func(m discord.Member) bool { return m.User.ID == player.ID })
							if index == -1 {
								continue
							}
							slog.Info("mee6メンバーインポート", slog.Any("gid", event.GuildID()), slog.Any("member_id", player.ID))
							m, err := c.MemberCreate(event, members[index].User, *event.GuildID())
							if err != nil {
								return errors.NewError(err)
							}
							m.Update().SetXp(xppoint.XP(player.Xp)).ExecX(event)
							memberCount++
						}
					}

					g.Update().SetLevelMee6Imported(true).ExecX(event)

					if err := event.RespondMessage(
						discord.NewMessageBuilder().
							SetContent(fmt.Sprintf("# SUCCEED\n```| IMPORTED MEMBER COUNT | %d```", memberCount)),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/reset": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.reset"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					target := event.SlashCommandInteractionData().Member("target")
					m, err := c.MemberCreate(event, target.User, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					m.Update().SetXp(xppoint.XP(0)).ExecX(event)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.level.reset.message",
								translate.WithTemplate(map[string]any{"User": discord.UserMention(target.User.ID)}),
							)).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/exclude-channel/list": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.list"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					var listStr string
					for i, id := range g.LevelUpExcludeChannel {
						listStr += fmt.Sprintf("%d. %s\n", i+1, discord.ChannelMention(id))
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.level.exclude-channel.list.message")).
										SetDescription(
											builtin.Or(listStr != "",
												listStr,
												"- "+translate.Message(event.Locale(), "components.level.exclude-channel.list.message.none"),
											),
										).
										Build(),
								),
							).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/role/set": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.role.set"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if len(g.LevelRole) >= 20 {
						return errors.NewError(errors.ErrorMessage("errors.create.reach_max", event))
					}
					level := event.SlashCommandInteractionData().Int("level")
					role := event.SlashCommandInteractionData().Role("role")
					g.LevelRole = builtin.NonNilMap(g.LevelRole)
					g.LevelRole[level] = role.ID
					self, valid := event.Client().Caches().SelfMember(*event.GuildID())
					if !valid {
						return errors.NewError(errors.ErrorMessage("errors.invalid.self", event))
					}
					var roles []discord.Role
					for _, id := range self.RoleIDs {
						role, ok := event.Client().Caches().Role(*event.GuildID(), id)
						if !ok {
							continue
						}
						roles = append(roles, role)
					}
					highestRole := discordutil.GetHighestRole(roles)
					if highestRole == nil {
						return errors.NewError(errors.ErrorMessage("errors.invalid.self", event))
					}

					if role.Managed || role.Compare(*highestRole) != -1 || role.ID == *event.GuildID() {
						return errors.NewError(errors.ErrorMessage("errors.invalid.role", event))
					}

					g.Update().
						SetLevelRole(g.LevelRole).
						ExecX(event)

					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.level.role.set.message.embed.title")).
										SetDescription(
											translate.Message(event.Locale(), "components.level.role.set.message.embed.description",
												translate.WithTemplate(map[string]any{
													"Level": strconv.Itoa(level),
													"Role":  discord.RoleMention(role.ID),
												}),
											),
										).
										Build(),
								),
							).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/role/list": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.role.list"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g.LevelRole = builtin.NonNilMap(g.LevelRole)
					var listStr string
					smap.MakeSortMap(g.LevelRole).Range(cmp.Compare[int],
						func(k int, v snowflake.ID) {
							listStr += "- " + translate.Message(event.Locale(), "components.level.role.list.message",
								translate.WithTemplate(map[string]any{
									"Level": strconv.Itoa(k),
									"Role":  discord.RoleMention(v),
								}),
							) + "\n"
						},
					)
					if err := event.RespondMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.level.role.list.message.embed.title")).
										SetDescription(
											builtin.Or(listStr != "",
												listStr,
												translate.Message(event.Locale(), "components.level.role.list.message.none"),
											),
										).
										Build(),
								),
							),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/level/role/remove": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.role.remove"),
				},
				DiscordPerm: discord.PermissionManageGuild,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					g.LevelRole = builtin.NonNilMap(g.LevelRole)
					level := event.SlashCommandInteractionData().Int("level")
					r, ok := g.LevelRole[level]
					if !ok {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					delete(g.LevelRole, level)

					g.Update().
						SetLevelRole(g.LevelRole).
						ExecX(event)

					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.level.role.remove.message.embed.title")).
										SetDescription(
											translate.Message(event.Locale(), "components.level.role.remove.message.embed.description",
												translate.WithTemplate(map[string]any{
													"Level": strconv.Itoa(level),
													"Role":  discord.RoleMention(r),
												}),
											),
										).
										Build(),
								),
							).
							BuildCreate(),
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
						BuildCreate(),
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
				if slices.Contains(g.LevelUpExcludeChannel, event.ChannelID) {
					return nil
				}
				var channel discord.GuildChannel
				channel, ok := event.Channel()
				if !ok {
					c, err := event.Client().Rest().GetChannel(event.ChannelID)
					if err != nil {
						return errors.NewError(err)
					}
					channel, _ = c.(discord.GuildChannel)
				}
				if channel.ParentID() != nil && slices.Contains(g.LevelUpExcludeChannel, *channel.ParentID()) {
					return nil
				}
				m, err := c.MemberCreate(event, event.Message.Author, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				if _, err = addXp(event, m.Update(), rand.N[uint64](16)+15, event.Client(), m, g, event.ChannelID, event.Message.Author.EffectiveName(), false); err != nil {
					return errors.NewError(err)
				}
			}
			return nil
		},
	}).SetComponent(c)
}

func addXp(ctx context.Context, memberUpdate *ent.MemberUpdateOne, xp uint64, client bot.Client, m *ent.Member, g *ent.Guild, channelID snowflake.ID, username string, ignoreCooldown bool) (*ent.Member, error) {
	before := builtin.NonNilOrDefault(m.LastNotifiedLevel, m.Xp.Level())
	if ignoreCooldown || time.Now().After(m.LastXp.Add(time.Minute*3)) {
		m.Xp.Add(xp)
		memberUpdate.
			SetXp(m.Xp).
			SetLastXp(time.Now())
	}
	after := m.Xp.Level()
	m = memberUpdate.
		SetLastNotifiedLevel(after).
		SetMessageCount(m.MessageCount + 1).
		SaveX(ctx)
	if before < after {
		for i := range after - before {
			if err := levelUp(g, before+i+1, client, g.ID, m); err != nil {
				return m, err
			}
		}
		// レベルアップ通知
		content := g.LevelUpMessage
		content = strings.ReplaceAll(content, "{user}", discord.UserMention(m.UserID))
		content = strings.ReplaceAll(content, "{username}", username)
		content = strings.ReplaceAll(content, "{before_level}", strconv.FormatUint(before, 10))
		content = strings.ReplaceAll(content, "{after_level}", strconv.FormatUint(after, 10))
		content = strings.ReplaceAll(content, "{xp}", strconv.FormatUint(uint64(m.Xp), 10))
		if _, err := client.Rest().
			CreateMessage(
				builtin.Or(builtin.NonNil(g.LevelUpChannel) != 0, builtin.NonNil(g.LevelUpChannel), channelID),
				discord.NewMessageBuilder().
					SetContent(content).
					BuildCreate(),
			); err != nil {
			return m, err
		}
	}
	return m, nil
}

func levelUp(g *ent.Guild, after uint64, client bot.Client, guildID snowflake.ID, m *ent.Member) error {
	// レベルロール
	r, ok := g.LevelRole[int(after)]
	if ok {
		if err := client.Rest().AddMemberRole(guildID, m.UserID, r); err != nil {
			slog.Error("レベルロール付与に失敗", slog.Any("err", err))
		}
	}

	return nil
}
