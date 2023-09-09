package gobot

import (
	"context"
	"fmt"

	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	"github.com/sabafly/sabafly-disgo/gateway"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
)

func onGuildJoin(b *botlib.Bot[*client.Client]) func(g *events.GuildJoin) {
	return func(g *events.GuildJoin) {
		b.Logger.Infof("[#%d]ギルド参加 %3dメンバー 作成 %s 名前 %s(%d)", g.ShardID(), g.Guild.MemberCount, g.Guild.CreatedAt().String(), g.Guild.Name, g.GuildID)
		go refreshPresence(b)
	}
}

func onGuildLeave(b *botlib.Bot[*client.Client]) func(g *events.GuildLeave) {
	return func(g *events.GuildLeave) {
		b.Logger.Infof("[#%d]ギルド脱退 %3dメンバー 作成 %s 名前 %s(%d)", g.ShardID(), g.Guild.MemberCount, g.Guild.CreatedAt().String(), g.Guild.Name, g.GuildID)
		b.Client.Caches().RemoveGuild(g.GuildID)
		b.Client.Caches().RemoveMembersByGuildID(g.GuildID)
		go refreshPresence(b)
	}
}

func onGuildMemberJoin(m *events.GuildMemberJoin, b *botlib.Bot[*client.Client]) {
	if g, ok := m.Client().Caches().Guild(m.GuildID); ok {
		b.Logger.Infof("[#%d]ギルドメンバー参加 %32s#%s(%d) ギルド %s(%d) %3d メンバー", m.ShardID(), m.Member.User.Username, m.Member.User.Discriminator, m.Member.User.ID, g.Name, g.ID, g.MemberCount)
	}
	go refreshPresence(b)
}

func onGuildMemberLeave(m *events.GuildMemberLeave, b *botlib.Bot[*client.Client]) {
	if g, ok := m.Client().Caches().Guild(m.GuildID); ok {
		b.Logger.Infof("[#%d]ギルドメンバー脱退 %32s#%s(%d) ギルド %s(%d) %3d メンバー", m.ShardID(), m.Member.User.Username, m.Member.User.Discriminator, m.Member.User.ID, g.Name, g.ID, g.MemberCount)
	}
	b.Client.Caches().RemoveMember(m.GuildID, m.User.ID)
	go refreshPresence(b)
}

func refreshPresence(b *botlib.Bot[*client.Client]) {
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
