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
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/user"
	"github.com/sabafly/gobot/internal/errors"
)

func (c *Components) UserCreate(ctx context.Context, u discord.User) (*ent.User, error) {
	if u.Bot || u.System {
		return nil, errors.New("bot cannot use to create user")
	}
	if ok := c.db.User.
		Query().
		Where(user.ID(u.ID)).ExistX(ctx); ok {
		return c.db.User.
			Query().
			Where(user.ID(u.ID)).Only(ctx)
	}
	slog.Debug("新規ユーザー作成", "uid", u.ID, "uname", u.Username)
	return c.db.User.Create().
		SetID(u.ID).
		SetName(u.EffectiveName()).
		Save(ctx)
}
