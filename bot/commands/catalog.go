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

package commands

import (
	"github.com/sabafly/gobot/bot/commands/bbs"
	"github.com/sabafly/gobot/bot/commands/bet"
	"github.com/sabafly/gobot/bot/commands/currency"
	"github.com/sabafly/gobot/bot/commands/debug"
	"github.com/sabafly/gobot/bot/commands/level"
	"github.com/sabafly/gobot/bot/commands/message"
	"github.com/sabafly/gobot/bot/commands/permission"
	"github.com/sabafly/gobot/bot/commands/ping"
	"github.com/sabafly/gobot/bot/commands/play"
	"github.com/sabafly/gobot/bot/commands/role"
	"github.com/sabafly/gobot/bot/commands/setting"
	"github.com/sabafly/gobot/bot/commands/user"
	"github.com/sabafly/gobot/bot/components"
)

// AllDefinitions returns the single source of truth of static command specifications for all commands in gobot.
func AllDefinitions() []components.CommandDefinition {
	return []components.CommandDefinition{
		debug.Definition(),
		ping.Definition(),
		message.Definition(),
		role.Definition(),
		level.Definition(),
		permission.Definition(),
		setting.Definition(),
		role.ImportDefinition(),
		currency.Definition(),
		play.Definition(),
		bet.Definition(),
		user.Definition(),
		bbs.Definition(),
		bbs.AdminDefinition(),
	}
}

// PublicDefinitions returns static specifications for all non-private commands suitable for public documentation.
func PublicDefinitions() []components.CommandDefinition {
	all := AllDefinitions()
	out := make([]components.CommandDefinition, 0, len(all))
	for _, def := range all {
		if def.Private {
			continue
		}
		out = append(out, def)
	}
	return out
}

// BuildRuntimeCommands constructs runtime command instances for Discord registration.
// When bbsSys is nil, BBS commands are omitted.
func BuildRuntimeCommands(c *components.Components, bbsSys *bbs.System) []components.Command {
	cmds := []components.Command{
		debug.Command(c),
		ping.Command(c),
		message.Command(c),
		role.Command(c),
		level.Command(c),
		permission.Command(c),
		setting.Command(c),
		role.ImportCommand(c),
		currency.Command(c),
		play.Command(c),
		bet.Command(c),
		user.Command(c),
	}
	if bbsSys != nil {
		cmds = append(cmds, bbs.Command(bbsSys), bbs.AdminCommand(bbsSys))
	}
	return cmds
}
