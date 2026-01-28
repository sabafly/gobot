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
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
)

func (c *Components) UserCreate(ctx context.Context, u discord.User) (*models.User, error) {
	if u.Bot || u.System {
		return nil, errors.New("bot cannot use to create user")
	}

	user := models.User{
		ID:   u.ID,
		Name: u.EffectiveName(),
	}

	if err := c.GormDB().FirstOrCreate(&user, models.User{ID: u.ID}).Error; err != nil {
		return nil, err
	}

	// If the user already existed, Name might be old. But FirstOrCreate won't update it if found.
	// The original code:
	// If exist, return existing.
	// If not exist, create new with ID and Name.
	// So FirstOrCreate matches the behavior of "if exist return, else create".
	// But wait, if I want to update name on every create call (idempotent upsert), I should use Clauses.
	// But the original code was: check exist -> return existing. Else create.
	// So existing user's name is NOT updated.
	// My Gorm code: FirstOrCreate(&user, models.User{ID: u.ID})
	// This will find by ID. If found, 'user' is populated from DB. If not found, created with 'user' struct content (ID and Name).
	// So this matches the behavior perfectly.

	if user.Name == "" && u.EffectiveName() != "" {
		// Just in case existing user has empty name?
		// But I'll stick to faithful migration.
	}

	if user.CreatedAt.IsZero() {
		// Just created?
		// Original code used .Save(ctx) which returns the created entity.
		// FirstOrCreate populates 'user'.
		slog.Debug("新規ユーザー作成", "uid", u.ID, "uname", u.Username)
	}

	return &user, nil
}
