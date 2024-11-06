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

package game

import (
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/translate"
)

func chinchiroEmbed(locale discord.Locale, players []*ent.ChinchiroPlayer) discord.Embed {
	str := ""
	for _, player := range players {
		str += fmt.Sprintf("- %s %dp (%dbet)\n", discord.UserMention(player.UserID), player.Point, builtin.NonNilOrDefault(player.Bet, 0))
	}
	return discord.NewEmbedBuilder().
		SetTitle(translate.Message(locale, "component.game.cinchiro.game.base.embed.title")).
		SetDescription(translate.Message(locale, "component.game.cinchiro.game.base.embed.description")).
		SetFields(
			discord.EmbedField{
				Name:  translate.Message(locale, "component.game.cinchiro.game.base.embed.field.players.name"),
				Value: str,
			},
		).
		Build()
}
