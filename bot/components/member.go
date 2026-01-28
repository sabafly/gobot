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

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/database/models"
	"gorm.io/gorm"
)

func (c *Components) MemberCreate(ctx context.Context, u discord.User, gid snowflake.ID) (*models.Member, error) {
	_, err := c.UserCreate(ctx, u)
	if err != nil {
		return nil, err
	}

	var member models.Member
	err = c.GormDB().Where("guild_id = ? AND user_id = ?", gid, u.ID).First(&member).Error
	if err == nil {
		return &member, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	member = models.Member{
		GuildID: gid,
		UserID:  u.ID,
	}
	if err := c.GormDB().Create(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}
