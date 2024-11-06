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

package components

import (
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/smap"
)

func New(db *ent.Client, conf Config) *Components {
	return &Components{
		db:               db,
		commandsRegistry: make(map[string]Command),
		config:           conf,
	}
}

type Components struct {
	db *ent.Client

	config Config

	commandsRegistry map[string]Command

	l smap.SyncedMap[string, *Mu]

	Version string
}

func (c *Components) DB() *ent.Client { return c.db }
