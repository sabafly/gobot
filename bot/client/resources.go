package client

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/sabafly-disgo/bot"
	"github.com/sabafly/sabafly-disgo/discord"
)

func NewResourceManager(b bot.Client, cl *Client) *ResourceManager {
	return &ResourceManager{
		bot:    b,
		client: cl,
	}
}

type ResourceManager struct {
	bot    bot.Client
	client *Client
}

func (r *ResourceManager) Channel(id snowflake.ID) (discord.Channel, error) {
	channel, ok := r.bot.Caches().Channel(id)
	if !ok {
		channel, err := r.bot.Rest().GetChannel(id)
		if err != nil {
			return nil, err
		}
		return channel, nil
	}
	return channel, nil
}

func (r *ResourceManager) Guild(id snowflake.ID) (*discord.Guild, error) {
	guild, ok := r.bot.Caches().Guild(id)
	if !ok {
		guild, err := r.bot.Rest().GetGuild(id, true)
		if err != nil {
			return nil, err
		}
		return &guild.Guild, err
	}
	return &guild, nil
}

func (r *ResourceManager) Member(guildID, userID snowflake.ID) (*discord.Member, error) {
	member, ok := r.bot.Caches().Member(guildID, userID)
	if !ok {
		member, err := r.bot.Rest().GetMember(guildID, userID)
		if err != nil {
			return nil, err
		}
		return member, nil
	}
	return &member, nil
}

func (r *ResourceManager) Role(guildID, roleID snowflake.ID) (*discord.Role, error) {
	role, ok := r.bot.Caches().Role(guildID, roleID)
	if !ok {
		role, err := r.bot.Rest().GetRole(guildID, roleID)
		if err != nil {
			return nil, err
		}
		return role, nil
	}
	return &role, nil
}
