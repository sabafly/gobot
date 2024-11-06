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

import "github.com/disgoorg/disgo/discord"

var (
	Reaction = &discord.ComponentEmoji{
		ID:   1141985795641716736,
		Name: "reaction",
	}
	SelectMenu = &discord.ComponentEmoji{
		ID:   1141991243832901704,
		Name: "select_menu",
	}
	Button = &discord.ComponentEmoji{
		ID:   1141991285281001553,
		Name: "button",
	}
	GreenButton = &discord.ComponentEmoji{
		ID:   1142333937180483687,
		Name: "green_button",
	}
	BlueButton = &discord.ComponentEmoji{
		ID:   1142333868490367037,
		Name: "blue_button",
	}
	RedButton = &discord.ComponentEmoji{
		ID:   1142334020403871745,
		Name: "red_button",
	}
	GrayButton = &discord.ComponentEmoji{
		ID:   1142333913906298960,
		Name: "gray_button",
	}
	On = &discord.ComponentEmoji{
		ID:   1142095470227890279,
		Name: "on",
	}
	Off = &discord.ComponentEmoji{
		ID:   1142110196462788779,
		Name: "off",
	}
)
