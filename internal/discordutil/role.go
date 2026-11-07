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
	"slices"

	"github.com/disgoorg/disgo/discord"
)

func CompareRoles(a, b discord.Role) int {
	if a.Position < b.Position {
		return -1
	}
	if a.Position > b.Position {
		return 1
	}
	if a.ID < b.ID {
		return -1
	}
	if a.ID > b.ID {
		return 1
	}
	return 0
}

func GetHighestRole(roles []discord.Role) *discord.Role {
	slices.SortStableFunc(roles, func(a, b discord.Role) int {
		return CompareRoles(b, a)
	})
	if len(roles) < 1 {
		return nil
	}
	return &roles[0]
}
