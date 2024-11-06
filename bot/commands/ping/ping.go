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

package ping

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
)

func Command(c *components.Components) *generic.Command {
	return (&generic.Command{
		Namespace: "ping",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "ping",
				Description:              "pong!",
				DescriptionLocalizations: translate.MessageMap("components.ping.command.description", false),
				DMPermission:             builtin.Ptr(false),
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/ping": generic.CommandHandler(func(_ *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							embeds.SetEmbedProperties(discord.NewEmbedBuilder().
								SetTitlef("🏓 %s", translate.Message(event.Locale(), "components.ping.pong")).
								SetFields(
									discord.EmbedField{
										Name:  fmt.Sprintf("**Discord API(#%d)**", event.ShardID()),
										Value: event.Client().ShardManager().Shard(event.ShardID()).Latency().String(),
									},
								).
								Build()),
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
