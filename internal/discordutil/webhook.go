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

package discordutil

import (
	"errors"
	"log/slog"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

var (
	WebhookDefaultName                 = "gobot"
	WebhookDefaultAvatar *discord.Icon = nil
	ErrUnsupportedType                 = errors.New("unsupported channel type")
)

type Messenger interface {
	Send(message MessageBuilder) (*discord.Message, error)
	Update(target snowflake.ID, message MessageBuilder) (*discord.Message, error)
	Delete(message snowflake.ID) error
}

type WebhookMessenger interface {
	Messenger
	Webhook() discord.Webhook
	SendWebhook(message MessageBuilder, username, avatarURL, threadName string) (*discord.Message, error)
}

type WebhookManager interface {
	GetMessenger(channel discord.Channel) (WebhookMessenger, error)
	GetWebhook(channel discord.Channel) (discord.Webhook, error)
}

func NewWebhookManager(client *bot.Client, logger *slog.Logger) WebhookManager {
	return &webhookManagerImpl{
		client: client,
		logger: logger,
	}
}

type webhookManagerImpl struct {
	client *bot.Client
	logger *slog.Logger
}

func isThread(t discord.ChannelType) bool {
	return t == discord.ChannelTypeGuildNewsThread || t == discord.ChannelTypeGuildPublicThread || t == discord.ChannelTypeGuildPrivateThread
}

func (w *webhookManagerImpl) GetMessenger(channel discord.Channel) (WebhookMessenger, error) {
	if channel.Type() == discord.ChannelTypeDM || channel.Type() == discord.ChannelTypeGroupDM {
		return nil, ErrUnsupportedType
	}

	if isThread(channel.Type()) {
		var c discord.GuildThread
	interaction:
		switch ch := channel.(type) {
		case discord.GuildThread:
			c = ch
		case discord.MessageThread:
			c = ch.GuildThread
		case discord.InteractionChannel:
			channel = ch.MessageChannel
			goto interaction
		default:
			return nil, ErrUnsupportedType
		}
		return NewThreadWebhookMessenger(w.client, *c.ParentID(), c.ID())
	}

	return NewChannelWebhookMessenger(w.client, channel.ID())
}

func (w *webhookManagerImpl) GetWebhook(channel discord.Channel) (discord.Webhook, error) {
	m, err := w.GetMessenger(channel)
	if err != nil {
		return nil, err
	}
	return m.Webhook(), nil
}

func NewChannelWebhookMessenger(client *bot.Client, channelID snowflake.ID) (WebhookMessenger, error) {
	webhooks, err := client.Rest.GetWebhooks(channelID)
	if err != nil {
		return nil, err
	}

	for _, webhook := range webhooks {
		if webhook.Type() == discord.WebhookTypeIncoming && webhook.(discord.IncomingWebhook).User.ID == client.ApplicationID {
			return channelWebhookMessenger{channelID: channelID, webhook: webhook.(discord.IncomingWebhook), client: client}, nil
		}
	}

	webhook, err := client.Rest.CreateWebhook(channelID, discord.WebhookCreate{
		Name:   WebhookDefaultName,
		Avatar: WebhookDefaultAvatar,
	})
	if err != nil {
		return nil, err
	}
	return channelWebhookMessenger{channelID: channelID, webhook: *webhook, client: client}, nil
}

type channelWebhookMessenger struct {
	channelID snowflake.ID
	webhook   discord.IncomingWebhook
	client    *bot.Client
}

func (c channelWebhookMessenger) SendWebhook(message MessageBuilder, username, avatarURL, threadName string) (*discord.Message, error) {
	create := message.BuildWebhookCreate(username, avatarURL, threadName)
	return c.client.Rest.CreateWebhookMessage(
		c.webhook.ID(),
		c.webhook.Token,
		create,
		rest.CreateWebhookMessageParams{
			Wait:           true,
			WithComponents: create.Flags.Has(discord.MessageFlagIsComponentsV2),
		},
	)
}

func (c channelWebhookMessenger) Webhook() discord.Webhook {
	return c.webhook
}

func (c channelWebhookMessenger) Send(message MessageBuilder) (*discord.Message, error) {
	return c.SendWebhook(message, "", "", "")
}

func (c channelWebhookMessenger) Update(target snowflake.ID, message MessageBuilder) (*discord.Message, error) {
	update := message.BuildWebhookUpdate()
	return c.client.Rest.UpdateWebhookMessage(
		c.webhook.ID(),
		c.webhook.Token,
		target,
		update,
		rest.UpdateWebhookMessageParams{
			WithComponents: update.Flags.Has(discord.MessageFlagIsComponentsV2),
		},
	)
}

func (c channelWebhookMessenger) Delete(message snowflake.ID) error {
	return c.client.Rest.DeleteWebhookMessage(c.webhook.ID(), c.webhook.Token, message, 0)
}

func NewThreadWebhookMessenger(client *bot.Client, parentID, threadID snowflake.ID) (WebhookMessenger, error) {
	webhooks, err := client.Rest.GetWebhooks(parentID)
	if err != nil {
		return nil, err
	}

	for _, webhook := range webhooks {
		if webhook.Type() == discord.WebhookTypeIncoming && webhook.(discord.IncomingWebhook).User.ID == client.ApplicationID {
			return threadWebhookMessenger{channelID: parentID, threadID: threadID, webhook: webhook.(discord.IncomingWebhook), client: client}, nil
		}
	}

	webhook, err := client.Rest.CreateWebhook(parentID, discord.WebhookCreate{
		Name:   WebhookDefaultName,
		Avatar: WebhookDefaultAvatar,
	})
	if err != nil {
		return nil, err
	}
	return threadWebhookMessenger{channelID: parentID, threadID: threadID, webhook: *webhook, client: client}, nil
}

type threadWebhookMessenger struct {
	channelID snowflake.ID
	threadID  snowflake.ID
	webhook   discord.IncomingWebhook
	client    *bot.Client
}

func (t threadWebhookMessenger) SendWebhook(message MessageBuilder, username, avatarURL, threadName string) (*discord.Message, error) {
	create := message.BuildWebhookCreate(username, avatarURL, threadName)
	return t.client.Rest.CreateWebhookMessage(
		t.webhook.ID(),
		t.webhook.Token,
		create,
		rest.CreateWebhookMessageParams{
			Wait:           true,
			ThreadID:       t.threadID,
			WithComponents: create.Flags.Has(discord.MessageFlagIsComponentsV2),
		})
}

func (t threadWebhookMessenger) Webhook() discord.Webhook {
	return t.webhook
}

func (t threadWebhookMessenger) Send(message MessageBuilder) (*discord.Message, error) {
	return t.SendWebhook(message, "", "", "")
}

func (t threadWebhookMessenger) Update(target snowflake.ID, message MessageBuilder) (*discord.Message, error) {
	update := message.BuildWebhookUpdate()
	return t.client.Rest.UpdateWebhookMessage(
		t.webhook.ID(),
		t.webhook.Token,
		target,
		update,
		rest.UpdateWebhookMessageParams{
			ThreadID:       t.threadID,
			WithComponents: update.Flags.Has(discord.MessageFlagIsComponentsV2),
		})
}

func (t threadWebhookMessenger) Delete(message snowflake.ID) error {
	return t.client.Rest.DeleteWebhookMessage(t.webhook.ID(), t.webhook.Token, message, t.threadID)
}

func GetWebhookMessenger(client *bot.Client, channel discord.Channel) (WebhookMessenger, error) {
	wm := NewWebhookManager(client, slog.Default())
	return wm.GetMessenger(channel)
}
