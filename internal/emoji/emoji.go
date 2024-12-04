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

package emoji

import (
	"regexp"

	"github.com/forPelevin/gomoji"
)

func MatchString(str string) bool {
	return DiscordEmoji.MatchString(str) || gomoji.ContainsEmoji(str)
}

func FindAllString(str string) []string {
	var s []string
	emojis := gomoji.CollectAll(str)
	for _, e := range emojis {
		s = append(s, e.Character)
	}
	discordEmojis := DiscordEmoji.FindAllString(str, -1)
	s = append(s, discordEmojis...)
	return s
}

var DiscordEmoji = regexp.MustCompile("<a?:[A-z0-9_~]+:[0-9]{18,20}>")
