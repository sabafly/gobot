package components

import (
	"context"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/wordsuffix"
	"log/slog"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/user"
)

func (c *Components) OnGuildJoin() func(event *events.GuildJoin) {
	return func(event *events.GuildJoin) {
		slog.Info("ギルド参加", "id", event.Guild.ID, "member_count", event.Guild.MemberCount, "name", event.Guild.Name)
		member, err := event.Client().Rest().GetMember(event.GuildID, event.Guild.OwnerID)
		if err != nil {
			slog.Error("ギルド参加 オーナーの取得に失敗", "err", err)
			return
		}
		u, err := c.UserCreate(event, member.User)
		if err != nil {
			slog.Error("ギルド参加 オーナーの初期化に失敗", "err", err)
			return
		}

		if _, err := c.GuildCreate(event, u.ID, event.GenericGuild); err != nil {
			slog.Error("ギルドの作成に失敗", "err", err)
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

func (c *Components) GuildCreate(ctx context.Context, ownerID snowflake.ID, g *events.GenericGuild) (*ent.Guild, error) {
	ok := c.db.Guild.
		Query().
		Where(guild.ID(g.Guild.ID)).ExistX(ctx)
	if ok {
		return c.db.Guild.
			Query().
			Where(guild.ID(g.GuildID)).
			Only(ctx)
	}
	slog.Debug("新規ギルド作成", "gid", g.GuildID, "name", g.Guild.Name)
	return c.db.Guild.Create().
		SetID(g.GuildID).
		SetName(g.Guild.Name).
		SetOwnerID(ownerID).
		Save(ctx)
}

func (c *Components) GuildCreateID(ctx context.Context, gid snowflake.ID) (*ent.Guild, error) {
	return c.db.Guild.
		Query().
		Where(guild.ID(gid)).
		Only(ctx)
}

func (c *Components) GuildRequest(client bot.Client, gid snowflake.ID) (*discord.Guild, error) {
	if g, ok := client.Caches().Guild(gid); ok {
		return &g, nil
	}
	g, err := client.Rest().GetGuild(gid, true)
	if err != nil {
		return nil, err
	}
	return &g.Guild, nil
}
