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

package debug

import (
	"iter"
	"log/slog"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
	"github.com/sabafly/gobot/internal/translate"
)

func Command(c *components.Components, entClient *ent.Client) *generic.Command {
	return (&generic.Command{
		Namespace: "debug",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "debug",
				Description: "debug",
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				DefaultMemberPermissions: omit.NewPtr(discord.PermissionAdministrator),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "translate",
						Description: "translate",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "get",
								Description: "get translate",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:        "key",
										Description: "translate key",
										Required:    true,
									},
									discord.ApplicationCommandOptionString{
										Name:        "locale",
										Description: "locale",
										Required:    true,
									},
								},
							},
							{
								Name:        "reload",
								Description: "reload translate",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "guild",
						Description: "guild",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "leave",
								Description: "leave",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:        "guild",
										Description: "guild",
										Required:    true,
									},
								},
							},
							{
								Name:        "load_members",
								Description: "load members",
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "migration",
						Description: "database migration",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "ent_to_gorm",
								Description: "migrate from ent to gorm",
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/debug/translate/get": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				key := event.SlashCommandInteractionData().String("key")
				locale := discord.Locale(event.SlashCommandInteractionData().String("locale"))
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent(translate.Message(locale, key)).
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/debug/translate/reload": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				if _, err := translate.LoadDir(c.Config().TranslateDir); err != nil {
					slog.Error("翻訳ファイルを読み込めません", "err", err)
					return errors.NewError(err)
				}
				if err := i18n.LoadLocales(c.Config().LocaleDir); err != nil {
					slog.Error("ロケールファイルを読み込めません", "err", err)
					return errors.NewError(err)
				}
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent("OK").
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/debug/guild/leave": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				guildID := snowflake.MustParse(event.SlashCommandInteractionData().String("guild"))
				if err := event.Client().Rest.LeaveGuild(guildID); err != nil {
					return errors.NewError(err)
				}
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent("OK").
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/debug/guild/load_members": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				if err := event.DeferCreateMessage(false); err != nil {
					return errors.NewError(err)
				}
				for guild := range event.Client().Caches.Guilds() {
					members, err := event.Client().Rest.GetMembers(guild.ID, 1000, 0)
					if err != nil {
						slog.Error("ギルドメンバーの取得に失敗", "guild", guild.ID, "err", err)
						continue
					}
					if len(members) == 1000 {
						idFunc := func(members []discord.Member) iter.Seq[snowflake.ID] {
							return func(yield func(snowflake.ID) bool) {
								for _, m := range slices.All(members) {
									if !yield(m.User.ID) {
										return
									}
								}
							}
						}
						highest := slices.Max(slices.Collect(idFunc(members)))
						for {
							moreMembers, err := event.Client().Rest.GetMembers(guild.ID, 1000, highest)
							if err != nil {
								slog.Error("ギルドメンバーの取得に失敗", "guild", guild.ID, "err", err)
								break
							}
							if len(moreMembers) < 1000 {
								break
							}
							members = append(members, moreMembers...)
							highest = slices.Max(slices.Collect(idFunc(moreMembers)))
						}
					}
					if err := c.InitializeGuildMember(event, *event.GuildID(), members); err != nil {
						slog.Error("ギルドメンバーの初期化に失敗", "guild", guild.ID, "err", err)
						return errors.NewError(err)
					}
					slog.Info("ギルドメンバーの初期化に成功", "guild", guild.ID, "count", len(members))
				}
				if err := event.RespondMessage(discord.NewMessageBuilder().SetContent("OK")); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/debug/migration/ent_to_gorm": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				return migrateEntToGormHandler(c, entClient, event)
			}),
		},
	}).SetComponent(c)
}
