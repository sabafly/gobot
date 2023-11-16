package webhookutil

import (
	"fmt"
	"io"
	"net/http"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

var (
	webhookName = "gobot-webhook"
)

func SendWebhook(client bot.Client, channelID snowflake.ID, data discord.WebhookMessageCreate) (st *discord.Message, err error) {
	id, token, err := GetWebhook(client, channelID)
	if err != nil {
		return nil, fmt.Errorf("cannot get webhook: %w", err)
	}
	st, err = client.Rest().CreateWebhookMessage(id, token, data, true, snowflake.ID(0))
	if err != nil {
		return nil, fmt.Errorf("cannot send webhook: %w", err)
	}
	return st, nil
}

func GetWebhook(client bot.Client, channelID snowflake.ID) (id snowflake.ID, token string, err error) {
	webhooks, err := client.Rest().GetWebhooks(channelID)
	if err != nil {
		return 0, "", fmt.Errorf("cannot request webhook: %w", err)
	}
	me, ok := client.Caches().SelfUser()
	if !ok {
		return 0, "", fmt.Errorf("cannot cache self: %w", err)
	}
	var webhook discord.Webhook = nil
	for _, w := range webhooks {
		switch v := w.(type) {
		case discord.IncomingWebhook:
			if v.User.ID == me.User.ID {
				token = v.Token
				webhook = v
				return webhook.ID(), token, nil
			}
		}
	}
	if webhook == nil {
		var buf []byte
		if avatarURL := me.EffectiveAvatarURL(discord.WithFormat(discord.FileFormatPNG)); avatarURL != "" {
			resp, err := http.Get(avatarURL)
			if err != nil {
				return 0, "", fmt.Errorf("error on get: %w", err)
			}
			buf, err = io.ReadAll(resp.Body)
			if err != nil {
				return 0, "", fmt.Errorf("error on read all: %w", err)
			}
		}
		data, err := client.Rest().CreateWebhook(channelID, discord.WebhookCreate{
			Name:   webhookName,
			Avatar: discord.NewIconRaw(discord.IconTypePNG, buf),
		})
		if err != nil {
			return 0, "", fmt.Errorf("error on create webhook: %w", err)
		}
		token = data.Token
		webhook = data
	}
	return webhook.ID(), token, nil
}
