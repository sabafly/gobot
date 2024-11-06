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

package embeds

import (
	"time"

	"github.com/disgoorg/disgo/discord"
)

var (
	Color   = 0x00AED9
	BotName = "gobot"
)

func SetEmbedProperties(embed discord.Embed) discord.Embed {
	now := time.Now()
	if embed.Color == 0 {
		embed.Color = Color
	}
	if embed.Footer == nil {
		embed.Footer = &discord.EmbedFooter{}
	}
	if embed.Footer.Text == "" {
		embed.Footer.Text = BotName
	}
	if embed.Timestamp == nil {
		embed.Timestamp = &now
	}
	return embed
}

func SetEmbedsProperties(embeds []discord.Embed) []discord.Embed {
	now := time.Now()
	for i := range embeds {
		if embeds[i].Color == 0 {
			embeds[i].Color = Color
		}
		if i == len(embeds)-1 {
			if embeds[i].Footer == nil {
				embeds[i].Footer = &discord.EmbedFooter{}
			}
			if embeds[i].Footer.Text == "" {
				embeds[i].Footer.Text = BotName
			}
			if embeds[i].Timestamp == nil {
				embeds[i].Timestamp = &now
			}
		}
	}
	return embeds
}
