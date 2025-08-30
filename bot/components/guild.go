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
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/wordsuffix"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/user"
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

		u = c.db.User.Query().Where(user.ID(u.ID)).OnlyX(event)
		slog.Debug("ギルドオーナー情報", "id", u.ID, "name", u.Name, "own_guilds", u.QueryOwnGuilds().AllX(event), "guilds", u.QueryGuilds().AllX(event))
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

		u = c.db.User.Query().Where(user.ID(u.ID)).OnlyX(event)
		slog.Debug("ギルドオーナー情報", "id", u.ID, "name", u.Name, "own_guilds", u.QueryOwnGuilds().AllX(event), "guilds", u.QueryGuilds().AllX(event))
	}
}

func (c *Components) OnGuildLeave() func(event *events.GuildLeave) {
	return func(event *events.GuildLeave) {
		slog.Info("ギルド脱退", "id", event.Guild.ID, "name", event.Guild.Name)
		c.db.Member.Delete().Where(member.HasGuildWith(guild.ID(event.Guild.ID))).ExecX(event)
		c.db.MessagePin.Delete().Where(messagepin.HasGuildWith(guild.ID(event.Guild.ID))).ExecX(event)
		c.db.MessageRemind.Delete().Where(messageremind.HasGuildWith(guild.ID(event.Guild.ID))).ExecX(event)
		c.db.RolePanelPlaced.Delete().Where(rolepanelplaced.HasGuildWith(guild.ID(event.Guild.ID))).ExecX(event)
		c.db.RolePanelEdit.Delete().Where(rolepaneledit.HasGuildWith(guild.ID(event.Guild.ID))).ExecX(event)
		c.db.RolePanel.Delete().Where(rolepanel.HasGuildWith(guild.ID(event.Guild.ID))).ExecX(event)
		c.db.WordSuffix.Delete().Where(wordsuffix.HasGuildWith(guild.ID(event.Guild.ID))).ExecX(event)
		c.db.Guild.DeleteOneID(event.Guild.ID).ExecX(event)
	}
}

func (c *Components) GuildCreate(ctx context.Context, ownerID snowflake.ID, g *discord.Guild) (*ent.Guild, error) {
	ok := c.db.Guild.
		Query().
		Where(guild.ID(g.ID)).ExistX(ctx)
	if ok {
		return c.db.Guild.
			Query().
			Where(guild.ID(g.ID)).
			Only(ctx)
	}
	slog.Debug("新規ギルド作成", "gid", g.ID)
	return c.db.Guild.Create().
		SetID(g.ID).
		SetName(g.Name).
		SetOwnerID(ownerID).
		Save(ctx)
}

func (c *Components) GuildCreateID(ctx context.Context, gid snowflake.ID) (*ent.Guild, error) {
	return c.db.Guild.
		Query().
		Where(guild.ID(gid)).
		Only(ctx)
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
		ID: guild.ID,
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
