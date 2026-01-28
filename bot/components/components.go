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
	"context"

	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/internal/smap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func New(ctx context.Context, conf Config, gormDb *database.DB) *Components {
	return &Components{
		ctx:              ctx,
		commandsRegistry: make(map[string]Command),
		config:           conf,
		gormDb:           gormDb,
	}
}

type Components struct {
	ctx    context.Context
	gormDb *database.DB

	config Config

	commandsRegistry map[string]Command

	l smap.SyncedMap[string, *Mu]

	Version string
}

func (c *Components) Ctx() context.Context { return c.ctx }

// DO NOT REUSE RETURN VALUE, MUST CALL EACH TIME TO GET NEW SESSION
func (c *Components) GormDB() *gorm.DB {
	return c.gormDb.DB.WithContext(c.ctx).
		Preload(clause.Associations)
}
