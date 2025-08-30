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

package discordutil

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/emoji"
)

func ParseCustomEmojis(str string) []discord.Emoji {
	return emoji.ParseCustomEmojis(str)
}

func ParseComponentEmoji(str string) discord.ComponentEmoji {
	return emoji.ParseComponentEmoji(str)
}

func FormatComponentEmoji(e discord.ComponentEmoji) string {
	return emoji.FormatComponentEmoji(e)
}

func ReactionComponentEmoji(e discord.ComponentEmoji) string {
	var zeroID snowflake.ID
	if e.ID == zeroID {
		return e.Name
	}
	return fmt.Sprintf("%s:%d", e.Name, e.ID)
}

// Number2Emoji は1から始まる
func Number2Emoji(n int) string {
	return Index2Emoji(n - 1)
}

// Index2Emoji は0から始まる
func Index2Emoji(n int) string {
	return string(rune('🇦' + n))
}
