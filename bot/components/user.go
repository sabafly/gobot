package components

import (
	"context"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/user"
)

func (c *Components) UserCreate(ctx context.Context, u discord.User) (*ent.User, error) {
	if ok := c.db.User.
		Query().
		Where(user.ID(u.ID)).ExistX(ctx); ok {
		return c.db.User.
			Query().
			Where(user.ID(u.ID)).Only(ctx)
	} else {
		slog.Debug("新規ユーザー作成", "uid", u.ID, "uname", u.Username)
		return c.db.User.Create().
			SetID(u.ID).
			SetName(u.EffectiveName()).
			Save(ctx)
	}
}
