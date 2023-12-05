package level

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log/slog"
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
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
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
									level_message(g, gl, m, index, target.Member, event),
								),
							).
							Create(),
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
						Order(
							member.ByXp(
								sql.OrderDesc(),
							),
						).
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
							Create(),
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
					before := toUser.Xp.Level()
					toUser.Xp.Add(movedXp)
					after := toUser.Xp.Level()
					fromUser.Xp = xppoint.XP(0)
					toUser = toUser.Update().SetXp(toUser.Xp).SaveX(event)
					fromUser = fromUser.Update().SetXp(fromUser.Xp).SaveX(event)
					if before != after {
						if err := level_up(g, after, event.Client(), *event.GuildID(), builtin.Or(g.LevelUpChannel != nil, builtin.NonNil(g.LevelUpChannel), event.Channel().ID()), toUser, to.User, before); err != nil {
							return errors.NewError(err)
						}
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
							Create(),
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
							Create(),
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
							Create(),
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
							Create(),
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

					members := []discord.Member{}
					member_count := 1000
					afterID := snowflake.ID(0)
					for member_count == 1000 {
						m, err := event.Client().Rest().GetMembers(*event.GuildID(), member_count, afterID)
						if err != nil {
							return errors.NewError(err)
						}
						member_count = len(m)
						members = append(members, m...)
						afterID = m[len(m)-1].User.ID
					}

					slog.Info("mee6インポート", slog.Any("gid", event.GuildID()), slog.Int("member_count", len(members)))

					member_count = 0
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
							member_count++
						}
					}

					g.Update().SetLevelMee6Imported(true).ExecX(event)

					if err := event.RespondMessage(
						discord.NewMessageBuilder().
							SetContent(fmt.Sprintf("# SUCCEED\n```| IMPORTED MEMBER COUNT | %d```", member_count)),
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
							Create(),
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
							Create(),
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
					role_map := map[snowflake.ID]discord.Role{}
					for _, id := range self.RoleIDs {
						role, ok := event.Client().Caches().Role(*event.GuildID(), id)
						if !ok {
							continue
						}
						role_map[id] = role
					}
					hi, _ := discordutil.GetHighestRolePosition(role_map)

					if role.Managed || role.Position >= hi || role.ID == *event.GuildID() {
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
							Create(),
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
					smap.MakeSortMap(g.LevelRole).Range(func(a, b int) int { return cmp.Compare(a, b) },
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
					if err := level_up(g, after, event.Client(), event.GuildID, builtin.Or(g.LevelUpChannel != nil, builtin.NonNil(g.LevelUpChannel), event.ChannelID), m, event.Message.Author, before); err != nil {
						return errors.NewError(err)
					}
				}
			}
			return nil
		},
	}).SetComponent(c)
}

func level_up(g *ent.Guild, after int64, client bot.Client, guildID, channelID snowflake.ID, m *ent.Member, user discord.User, before int64) error {
	// レベルロール
	r, ok := g.LevelRole[int(after)]
	if ok {
		if err := client.Rest().AddMemberRole(guildID, m.UserID, r); err != nil {
			slog.Error("レベルロール付与に失敗", slog.Any("err", err))
		}
	}

	// レベルアップ通知
	content := g.LevelUpMessage
	content = strings.ReplaceAll(content, "{user}", discord.UserMention(m.UserID))
	content = strings.ReplaceAll(content, "{username}", user.EffectiveName())
	content = strings.ReplaceAll(content, "{before_level}", strconv.FormatInt(before, 10))
	content = strings.ReplaceAll(content, "{after_level}", strconv.FormatInt(after, 10))
	content = strings.ReplaceAll(content, "{xp}", strconv.FormatUint(uint64(m.Xp), 10))
	if _, err := client.Rest().
		CreateMessage(
			builtin.Or(builtin.NonNil(g.LevelUpChannel) != 0, builtin.NonNil(g.LevelUpChannel), channelID),
			discord.NewMessageBuilder().
				SetContent(content).
				Create(),
		); err != nil {
		return err
	}
	return nil
}
