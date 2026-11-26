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
	"sync"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/internal/smap"
)

type GuildLocaleChangeHook func(ctx context.Context, client *bot.Client, guildID snowflake.ID, resolvedLocale discord.Locale)

func New(ctx context.Context, conf Config, gormDb *database.DB) *Components {
	return &Components{
		ctx:              ctx,
		commandsRegistry: make(map[string]Command),
		router:           handler.New(),
		config:           conf,
		gormDb:           gormDb,
	}
}

type Components struct {
	ctx    context.Context
	gormDb *database.DB

	config Config

	commandsRegistry map[string]Command
	router           *handler.Mux

	l smap.SyncedMap[string, *Mu]

	guildLocaleHooksMu sync.RWMutex
	guildLocaleHooks   []GuildLocaleChangeHook

	Version string
}

func (c *Components) RegisterGuildLocaleChangeHook(hook GuildLocaleChangeHook) {
	if c == nil || hook == nil {
		return
	}
	c.guildLocaleHooksMu.Lock()
	c.guildLocaleHooks = append(c.guildLocaleHooks, hook)
	c.guildLocaleHooksMu.Unlock()
}

func (c *Components) NotifyGuildLocaleChanged(ctx context.Context, client *bot.Client, guildID snowflake.ID, resolvedLocale discord.Locale) {
	if c == nil {
		return
	}
	c.guildLocaleHooksMu.RLock()
	hooks := append([]GuildLocaleChangeHook(nil), c.guildLocaleHooks...)
	c.guildLocaleHooksMu.RUnlock()
	for _, hook := range hooks {
		hook(ctx, client, guildID, resolvedLocale)
	}
}

func (c *Components) Ctx() context.Context { return c.ctx }

func (c *Components) Router() *handler.Mux { return c.router }

// DO NOT REUSE RETURN VALUE, MUST CALL EACH TIME TO GET NEW SESSION
func (c *Components) GormDB() *gorm.DB {
	if c.gormDb == nil || c.gormDb.DB == nil {
		return nil
	}
	return c.gormDb.DB.WithContext(c.ctx).
		Preload(clause.Associations)
}
