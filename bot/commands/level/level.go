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

package level

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"gorm.io/gorm"
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
				CommandHandler: requiredPointHandler,
			},
			"/level/rank": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("level.rank"),
				},
				CommandHandler: rankHandler,
			},
			"/level/leaderboard": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("level.leaderboard"),
				},
				CommandHandler: leaderboardHandler,
			},
			"/level/transfer": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.transfer"),
				},
				DiscordPerm:    discord.PermissionManageGuild.Add(discord.PermissionModerateMembers),
				CommandHandler: transferHandler,
			},
			"/level/up/message": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.up.message"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: upMessageHandler,
			},
			"/level/up/message-channel": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.message-channel"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: upMessageChannelHandler,
			},
			"/level/exclude-channel/add": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.add"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: excludeChannelAddHandler,
			},
			"/level/exclude-channel/remove": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.remove"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: excludeChannelRemoveHandler,
			},
			"/level/exclude-channel/clear": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.clear"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: excludeChannelClearHandler,
			},
			"/level/import-mee6": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.import-mee6"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: importMee6Handler,
			},
			"/level/reset": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.reset"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: resetHandler,
			},
			"/level/exclude-channel/list": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.exclude-channel.list"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: excludeChannelListHandler,
			},
			"/level/role/set": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.role.set"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: roleSetHandler,
			},
			"/level/role/list": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.role.list"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: roleListHandler,
			},
			"/level/role/remove": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("level.role.remove"),
				},
				DiscordPerm:    discord.PermissionManageGuild,
				CommandHandler: roleRemoveHandler,
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"level:up_message_modal": upMessageModalHandler,
		},
		EventHandler: eventHandler,
	}).SetComponent(c)
}

func addXp(ctx context.Context, xp uint64, client *bot.Client, m *models.Member, g *models.Guild, channelID snowflake.ID, username string, ignoreCooldown bool, db *gorm.DB) (*models.Member, error) {
	before := builtin.NonNilOrDefault(m.LastNotifiedLevel, m.XP.Level())
	if ignoreCooldown || time.Now().After(m.LastXP.Add(time.Minute*3)) {
		m.XP.Add(xp)
		m.LastXP = time.Now()
	}
	after := m.XP.Level()
	m.LastNotifiedLevel = &after
	m.MessageCount++

	if err := db.Save(m).Error; err != nil {
		return m, err
	}

	if before < after {
		for i := uint64(0); i < after-before; i++ {
			_ = levelUp(g, before+i+1, client, g.ID, m)
		}
		content := g.LevelUpMessage
		content = strings.ReplaceAll(content, "{user}", discord.UserMention(m.UserID))
		content = strings.ReplaceAll(content, "{username}", username)
		content = strings.ReplaceAll(content, "{before_level}", strconv.FormatUint(before, 10))
		content = strings.ReplaceAll(content, "{after_level}", strconv.FormatUint(after, 10))
		content = strings.ReplaceAll(content, "{xp}", strconv.FormatUint(uint64(m.XP), 10))

		targetChannel := channelID
		if g.LevelUpChannel != nil && *g.LevelUpChannel != 0 {
			targetChannel = *g.LevelUpChannel
		}

		if _, err := client.Rest.CreateMessage(targetChannel, discord.NewMessageBuilder().SetContent(content).BuildCreate()); err != nil {
			return m, err
		}
	}
	return m, nil
}

func levelUp(g *models.Guild, after uint64, client *bot.Client, guildID snowflake.ID, m *models.Member) error {
	rID, ok := g.LevelRole[int(after)]
	if ok {
		if err := client.Rest.AddMemberRole(guildID, m.UserID, rID); err != nil {
			slog.Error("レベルロール付与に失敗", slog.Any("err", err))
		}
	}

	return nil
}
