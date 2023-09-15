package client

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/sabafly-disgo/bot"
	"github.com/sabafly/sabafly-disgo/discord"
)

func NewResourceManager(client bot.Client) *ResourceManager {
	return &ResourceManager{
		client: client,
	}
}

type ResourceManager struct {
	client bot.Client
}

func (r *ResourceManager) Channel(id snowflake.ID) (discord.Channel, error) {
	channel, ok := r.client.Caches().Channel(id)
	if !ok {
		channel, err := r.client.Rest().GetChannel(id)
		if err != nil {
			return nil, err
		}
		return channel, nil
	}
	return channel, nil
}

func (r *ResourceManager) Guild(id snowflake.ID) (*discord.Guild, error) {
	guild, ok := r.client.Caches().Guild(id)
	if !ok {
		guild, err := r.client.Rest().GetGuild(id, true)
		if err != nil {
			return nil, err
		}
		return &guild.Guild, err
	}
	return &guild, nil
}

func (r *ResourceManager) Member(guildID, userID snowflake.ID) (*discord.Member, error) {
	member, ok := r.client.Caches().Member(guildID, userID)
	if !ok {
		member, err := r.client.Rest().GetMember(guildID, userID)
		if err != nil {
			return nil, err
		}
		return member, nil
	}
	return &member, nil
}

func (r *ResourceManager) Role(guildID, roleID snowflake.ID) (*discord.Role, error) {
	role, ok := r.client.Caches().Role(guildID, roleID)
	if !ok {
		role, err := r.client.Rest().GetRole(guildID, roleID)
		if err != nil {
			return nil, err
		}
		return role, nil
	}
	return &role, nil
}
