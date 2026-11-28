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
 */

package components

import (
	"github.com/disgoorg/disgo/discord"
)

// CommandCategory groups commands into logical documentation sections.
type CommandCategory string

const (
	CommandCategoryGeneral    CommandCategory = "general"
	CommandCategoryModeration CommandCategory = "moderation"
	CommandCategoryEconomy    CommandCategory = "economy"
	CommandCategoryGames      CommandCategory = "games"
	CommandCategoryBBS        CommandCategory = "bbs"
	CommandCategoryAdmin      CommandCategory = "admin"
)

// CommandPermissionMeta captures the gobot permission nodes and Discord permissions required by a command path.
type CommandPermissionMeta struct {
	Permissions       []string
	DiscordPermission discord.Permissions
}

// CommandDefinition describes a command's static metadata independently of runtime dependencies (DB, Discord client, etc.).
type CommandDefinition struct {
	Name        string
	Category    CommandCategory
	Private     bool
	Commands    func() []discord.ApplicationCommandCreate
	Permissions map[string]CommandPermissionMeta
}
