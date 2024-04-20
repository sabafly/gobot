package components

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/user"
)

func (c *Components) MemberCreate(ctx context.Context, u discord.User, gid snowflake.ID) (*ent.Member, error) {
	eu, err := c.UserCreate(ctx, u)
	if err != nil {
		return nil, err
	}
	ok := c.db.Member.
		Query().
		Where(member.HasUserWith(user.ID(u.ID)), member.HasGuildWith(guild.ID(gid))).ExistX(ctx)
	if ok {
		return c.db.Member.
			Query().
			Where(member.HasUserWith(user.ID(u.ID)), member.HasGuildWith(guild.ID(gid))).Only(ctx)
	}
	return c.db.Member.Create().
		SetUser(eu).
		SetGuildID(gid).
		Save(ctx)
}
