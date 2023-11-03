package components

import (
	"context"
	"log/slog"

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
		u, err := c.UserCreate(context.Background(), member.User)
		if err != nil {
			slog.Error("ギルド参加 オーナーの初期化に失敗", "err", err)
			return
		}

		if _, err := c.GuildCreate(context.Background(), u.ID, event.GenericGuild); err != nil {
			slog.Error("ギルドの作成に失敗", "err", err)
			return
		}

		u = c.db.User.Query().Where(user.ID(u.ID)).OnlyX(context.Background())
		slog.Debug("ギルドオーナー情報", "id", u.ID, "name", u.Name, "own_guilds", u.QueryOwnGuilds().AllX(context.Background()), "guilds", u.QueryGuilds().AllX(context.Background()))
	}
}

func (c *Components) OnGuildLeave() func(event *events.GuildLeave) {
	return func(event *events.GuildLeave) {
		slog.Info("ギルド脱退", "id", event.Guild.ID, "name", event.Guild.Name)
		c.db.Guild.DeleteOneID(event.Guild.ID).ExecX(context.Background())
	}
}

func (c *Components) GuildCreate(ctx context.Context, owner_id snowflake.ID, g *events.GenericGuild) (*ent.Guild, error) {
	if ok := c.db.Guild.
		Query().
		Where(guild.ID(g.Guild.ID)).ExistX(ctx); ok {
		return c.db.Guild.
			Query().
			Where(guild.ID(g.GuildID)).
			Only(ctx)
	} else {
		slog.Debug("新規ギルド作成", "gid", g.GuildID, "name", g.Guild.Name)
		return c.db.Guild.Create().
			SetID(g.GuildID).
			SetName(g.Guild.Name).
			SetOwnerID(owner_id).
			Save(ctx)
	}
}

func (c *Components) GuildCreateID(ctx context.Context, gid snowflake.ID) (*ent.Guild, error) {
	return c.db.Guild.
		Query().
		Where(guild.ID(gid)).
		Only(ctx)
}
