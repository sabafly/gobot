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
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/emoji"
)

func ParseCustomEmojis(str string) []discord.Emoji {
	emojis := emoji.DiscordEmoji.FindAllString(str, -1)
	toReturn := make([]discord.Emoji, len(emojis))
	if len(emojis) < 1 {
		return toReturn
	}
	for i, em := range emojis {
		parts := strings.Split(em, ":")
		toReturn[i] = discord.Emoji{
			ID:       snowflake.MustParse(parts[2]),
			Name:     parts[1],
			Animated: strings.HasPrefix(em, "<a:"),
		}
	}
	return toReturn
}

func ParseComponentEmoji(str string) discord.ComponentEmoji {
	e := discord.ComponentEmoji{
		Name: str,
	}
	if !emoji.MatchString(str) {
		return e
	}
	emojis := ParseCustomEmojis(str)
	if len(emojis) < 1 {
		return e
	}
	e = discord.ComponentEmoji{
		ID:       emojis[0].ID,
		Name:     emojis[0].Name,
		Animated: emojis[0].Animated,
	}
	return e
}

func FormatComponentEmoji(e discord.ComponentEmoji) string {
	var zeroID snowflake.ID
	if e.ID == zeroID {
		return e.Name
	}
	if e.Animated {
		return fmt.Sprintf("<a:%s:%d>", e.Name, e.ID)
	}
	return fmt.Sprintf("<:%s:%d>", e.Name, e.ID)
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
