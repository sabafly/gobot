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

	"github.com/sabafly/gobot/database/models"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/gorm"
)

func (c *Components) OnGuildReady() func(event *events.GuildReady) {
	return func(event *events.GuildReady) {
		slog.Info("ギルド準備完了", "id", event.Guild.ID, "member_count", event.Guild.MemberCount, "name", event.Guild.Name)
		member, err := event.Client().Rest.GetMember(event.GuildID, event.Guild.OwnerID)
		if err != nil {
			slog.Error("ギルド準備完了 オーナーの取得に失敗", "err", err)
			return
		}
		u, err := c.UserCreate(event, member.User)
		if err != nil {
			slog.Error("ギルド準備完了 オーナーの初期化に失敗", "err", err)
			return
		}

		if _, err := c.GuildCreate(event, u.ID, &event.Guild.Guild); err != nil {
			slog.Error("ギルドの作成に失敗", "err", err)
			return
		}

		if err := c.InitializeGuild(event, event.Guild.Guild); err != nil {
			slog.Error("ギルドの初期化に失敗", "err", err, "guild_id", event.Guild.ID)
			return
		}

		var ownedGuilds []models.Guild
		c.GormDB().Where("owner_id = ?", u.ID).Find(&ownedGuilds)

		// For joined guilds, need a join query via Members
		var joinedMembers []models.Member
		c.GormDB().Preload("Guild").Where("user_id = ?", u.ID).Find(&joinedMembers)
		var joinedGuilds []models.Guild
		for _, m := range joinedMembers {
			joinedGuilds = append(joinedGuilds, m.Guild)
		}

		slog.Debug("ギルドオーナー情報", "id", u.ID, "name", u.Name, "own_guilds", ownedGuilds, "guilds", joinedGuilds)
	}
}

func (c *Components) OnGuildJoin() func(event *events.GuildJoin) {
	return func(event *events.GuildJoin) {
		slog.Info("ギルド参加", "id", event.Guild.ID, "member_count", event.Guild.MemberCount, "name", event.Guild.Name)
		member, err := event.Client().Rest.GetMember(event.GuildID, event.Guild.OwnerID)
		if err != nil {
			slog.Error("ギルド参加 オーナーの取得に失敗", "err", err)
			return
		}
		u, err := c.UserCreate(event, member.User)
		if err != nil {
			slog.Error("ギルド参加 オーナーの初期化に失敗", "err", err)
			return
		}

		if _, err := c.GuildCreate(event, u.ID, &event.Guild.Guild); err != nil {
			slog.Error("ギルドの作成に失敗", "err", err)
			return
		}

		if err := c.InitializeGuild(event, event.Guild.Guild); err != nil {
			slog.Error("ギルドの初期化に失敗", "err", err, "guild_id", event.Guild.ID)
			return
		}

		var ownedGuilds []models.Guild
		c.GormDB().Where("owner_id = ?", u.ID).Find(&ownedGuilds)

		var joinedMembers []models.Member
		c.GormDB().Preload("Guild").Where("user_id = ?", u.ID).Find(&joinedMembers)
		var joinedGuilds []models.Guild
		for _, m := range joinedMembers {
			joinedGuilds = append(joinedGuilds, m.Guild)
		}

		slog.Debug("ギルドオーナー情報", "id", u.ID, "name", u.Name, "own_guilds", ownedGuilds, "guilds", joinedGuilds)
	}
}

func (c *Components) OnGuildLeave() func(event *events.GuildLeave) {
	return func(event *events.GuildLeave) {
		slog.Info("ギルド脱退", "id", event.Guild.ID, "name", event.Guild.Name)

		// Use transaction for deletion
		if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("guild_id = ?", event.Guild.ID).Delete(&models.Member{}).Error; err != nil {
				slog.Error("ギルド脱退 メンバー削除に失敗", "err", err)
				return err
			}
			if err := tx.Where("guild_id = ?", event.Guild.ID).Delete(&models.MessagePin{}).Error; err != nil {
				slog.Error("ギルド脱退 メッセージピン削除に失敗", "err", err)
				return err
			}
			if err := tx.Where("guild_id = ?", event.Guild.ID).Delete(&models.MessageRemind{}).Error; err != nil {
				slog.Error("ギルド脱退 メッセージリマインド削除に失敗", "err", err)
				return err
			}
			if err := tx.Where("guild_id = ?", event.Guild.ID).Delete(&models.RolePanelPlaced{}).Error; err != nil {
				slog.Error("ギルド脱退 ロールパネル配置削除に失敗", "err", err)
				return err
			}
			if err := tx.Where("guild_id = ?", event.Guild.ID).Delete(&models.RolePanelEdit{}).Error; err != nil {
				slog.Error("ギルド脱退 ロールパネル編集削除に失敗", "err", err)
				return err
			}
			if err := tx.Where("guild_id = ?", event.Guild.ID).Delete(&models.RolePanel{}).Error; err != nil {
				slog.Error("ギルド脱退 ロールパネル削除に失敗", "err", err)
				return err
			}
			if err := tx.Where("guild_id = ?", event.Guild.ID).Delete(&models.WordSuffix{}).Error; err != nil {
				slog.Error("ギルド脱退 ワードサフィックス削除に失敗", "err", err)
				return err
			}
			if err := tx.Delete(&models.Guild{ID: event.Guild.ID}).Error; err != nil {
				slog.Error("ギルド脱退 ギルド削除に失敗", "err", err)
				return err
			}
			return nil
		}); err != nil {
			slog.Error("ギルド脱退 データベースからの削除に失敗", "err", err)
			return
		}
	}
}

func (c *Components) GuildCreate(ctx context.Context, ownerID snowflake.ID, g *discord.Guild) (*models.Guild, error) {
	var guild models.Guild
	err := c.GormDB().Where("id = ?", g.ID).First(&guild).Error
	if err == nil {
		return &guild, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	slog.Debug("新規ギルド作成", "gid", g.ID)
	guild = models.Guild{
		ID:      g.ID,
		Name:    g.Name,
		OwnerID: &ownerID,
	}
	if err := c.GormDB().Create(&guild).Error; err != nil {
		return nil, err
	}
	return &guild, nil
}

func (c *Components) GuildCreateID(ctx context.Context, gid snowflake.ID) (*models.Guild, error) {
	var guild models.Guild
	if err := c.GormDB().Where("id = ?", gid).First(&guild).Error; err != nil {
		return nil, err
	}
	return &guild, nil
}

func (c *Components) GuildRequest(client *bot.Client, gid snowflake.ID) (*discord.Guild, error) {
	if g, ok := client.Caches.Guild(gid); ok {
		return &g, nil
	}
	g, err := client.Rest.GetGuild(gid, true)
	if err != nil {
		return nil, err
	}
	return &g.Guild, nil
}

func (c *Components) InitializeGuild(ctx context.Context, guild discord.Guild) error {
	g := models.Guild{
		ID:      guild.ID,
		OwnerID: &guild.OwnerID,
	}
	if err := c.GormDB().Where(g).FirstOrCreate(&g).Error; err != nil {
		return err
	}
	return nil
}

func (c *Components) InitializeGuildMember(ctx context.Context, guildID snowflake.ID, members []discord.Member) error {
	for _, member := range members {
		if err := c.InitializeUser(ctx, member); err != nil {
			slog.Error("ユーザーの初期化に失敗", "error", err, "user_id", member.User.ID)
		}
	}
	return nil
}

func (c *Components) InitializeUser(ctx context.Context, member discord.Member) error {
	if member.User.Bot || member.User.System {
		return nil // Botやシステムユーザーは初期化しない
	}
	user := models.User{
		ID: member.User.ID,
	}
	if err := c.GormDB().Where(user).FirstOrCreate(&user).Error; err != nil {
		return err
	}
	return nil
}
