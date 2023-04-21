/*
	Copyright (C) 2022-2023  ikafly144

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package botlib

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
	"github.com/disgoorg/log"
	"github.com/disgoorg/paginator"
	"github.com/sabafly/gobot/lib/db"
	"github.com/sabafly/gobot/lib/handler"
)

func New(logger log.Logger, version string, config Config) *Bot {
	return &Bot{
		Logger:    logger,
		Config:    config,
		Paginator: paginator.New(),
		Version:   version,
		Handler:   handler.New(logger),
	}
}

type Bot struct {
	Logger    log.Logger
	Client    bot.Client
	Paginator *paginator.Manager
	Config    Config
	Version   string
	Handler   *handler.Handler
	DB        db.DB
}

func (b *Bot) SetupBot(listeners ...bot.EventListener) {
	var err error
	b.DB, err = db.SetupDatabase(b.Config.DBConfig)
	if err != nil {
		b.Logger.Fatalf("botのセットアップに失敗 %s", err)
	}
	b.Client, err = disgo.New(b.Config.Token,
		bot.WithLogger(b.Logger),
		bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentsAll), gateway.WithAutoReconnect(true)),
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagsAll)),
		bot.WithShardManagerConfigOpts(sharding.WithAutoScaling(true), sharding.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentsAll), gateway.WithAutoReconnect(true))),
		bot.WithEventManagerConfigOpts(bot.WithAsyncEventsEnabled(), bot.WithListeners(b.Paginator), bot.WithListeners(listeners...)),
	)
	if err != nil {
		b.Logger.Fatalf("botのセットアップに失敗 %s", err)
	}
}

func (b *Bot) OnGuildJoin(g *events.GuildJoin) {
	b.Logger.Infof("[#%d]ギルド参加 %3dメンバー 作成 %s 名前 %s(%d)", g.ShardID(), g.Guild.MemberCount, g.Guild.CreatedAt().String(), g.Guild.Name, g.GuildID)
	go b.RefreshPresence()
}

func (b *Bot) OnGuildLeave(g *events.GuildLeave) {
	b.Logger.Infof("[#%d]ギルド脱退 %3dメンバー 作成 %s 名前 %s(%d)", g.ShardID(), g.Guild.MemberCount, g.Guild.CreatedAt().String(), g.Guild.Name, g.GuildID)
	b.Client.Caches().RemoveGuild(g.GuildID)
	b.Client.Caches().RemoveMembersByGuildID(g.GuildID)
	go b.RefreshPresence()
}

func (b *Bot) OnGuildMemberJoin(m *events.GuildMemberJoin) {
	if g, ok := m.Client().Caches().Guild(m.GuildID); ok {
		b.Logger.Infof("[#%d]ギルドメンバー参加 %32s#%s(%d) ギルド %s(%d) %3d メンバー", m.ShardID(), m.Member.User.Username, m.Member.User.Discriminator, m.Member.User.ID, g.Name, g.ID, g.MemberCount)
	}
	go b.RefreshPresence()
}

func (b *Bot) OnGuildMemberLeave(m *events.GuildMemberLeave) {
	if g, ok := m.Client().Caches().Guild(m.GuildID); ok {
		b.Logger.Infof("[#%d]ギルドメンバー脱退 %32s#%s(%d) ギルド %s(%d) %3d メンバー", m.ShardID(), m.Member.User.Username, m.Member.User.Discriminator, m.Member.User.ID, g.Name, g.ID, g.MemberCount)
	}
	b.Client.Caches().RemoveMember(m.GuildID, m.User.ID)
	go b.RefreshPresence()
}

func (b *Bot) RefreshPresence() {
	var (
		guilds int = b.Client.Caches().GuildsLen()
		users  int = b.Client.Caches().MembersAllLen()
	)
	shards := b.Client.ShardManager().Shards()
	for k := range shards {
		state := fmt.Sprintf("/help | %d Servers | %d Users | #%d", guilds, users, k)
		if err := b.Client.SetPresenceForShard(context.TODO(), k, gateway.WithOnlineStatus(discord.OnlineStatusOnline), gateway.WithPlayingActivity(state)); err != nil {
			b.Logger.Errorf("ステータス更新に失敗 %s", err)
		}
	}
	if len(shards) == 0 {
		state := fmt.Sprintf("/help | %d Servers | %d Users", guilds, users)
		err := b.Client.SetPresence(context.TODO(), gateway.WithPlayingActivity(state))
		if err != nil {
			b.Logger.Errorf("ステータス更新に失敗 %s", err)
		}
	}
}
