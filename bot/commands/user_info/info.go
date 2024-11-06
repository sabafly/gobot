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

package userinfo

import (
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
	"slices"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "userinfo",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.UserCommandCreate{
				Name:              "userinfo",
				NameLocalizations: translate.MessageMap("components.user.info.name", false),
				DMPermission:      builtin.Ptr(false),
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"u/userinfo": generic.CommandHandler(func(_ *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				var roleString string
				{ // ロールを取得する
					event.Member().RoleIDs = append(event.Member().RoleIDs, *event.GuildID())
					roles := event.Client().Caches().MemberRoles(event.Member().Member)
					slices.SortStableFunc(roles, func(a, b discord.Role) int {
						return a.Compare(b)
					})
					for i, r := range roles {
						roleString += fmt.Sprintf("%d %s\n", i+1, discord.RoleMention(r.ID))
					}
				}
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							embeds.SetEmbedProperties(
								discord.NewEmbedBuilder().
									SetFields(
										discord.EmbedField{
											Name:  translate.Message(event.Locale(), "userinfo.roles"),
											Value: roleString,
										},
									).
									Build(),
							),
						).
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
		},
	}).SetComponent(c)
}
