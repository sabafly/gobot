package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/sabafly-disgo/bot"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
)

func LogMessage(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		Handler: messageHandler(b),
	}
}

func messageHandler(b *botlib.Bot[*client.Client]) handler.MessageHandler {
	return func(event *events.GuildMessageCreate) error {
		err := b.Self.Logger.Message.Log(
			"message",
			fmt.Sprintf("%s at:%s/%s by:%s(%s) %s",
				event.Message.ID,
				event.GuildID,
				event.ChannelID,
				event.Message.Author.Tag(),
				event.Message.Author.ID,
				event.Message.Content,
			),
			event.Message.CreatedAt,
		)
		if err != nil {
			b.Logger.Errorf("error on message log: %s", err.Error())
		}

		if l, ok := b.Self.Logger.DebugChannel[event.ChannelID]; ok {
			raw, err := json.MarshalIndent(event.Message, "", "  ")
			if err != nil {
				b.Logger.Errorf("error on marshal json object: %s", err.Error())
				return err
			}
			if l.LogChannel != nil {
				if _, err := botlib.SendWebhook(event.Client(), *l.LogChannel, discord.WebhookMessageCreate{
					Content:    event.Message.Content,
					Username:   event.Message.Author.Tag(),
					AvatarURL:  event.Message.Author.EffectiveAvatarURL(),
					Embeds:     event.Message.Embeds,
					Components: event.Message.Components,
					Files: []*discord.File{
						{
							Name:   "message-" + event.Message.ID.String() + ".json",
							Reader: bytes.NewBuffer(raw),
						},
					},
				}); err != nil {
					b.Logger.Errorf("error on debug channel: %s", err.Error())
				}
			}
			if l.Logger != nil {
				if err := l.Logger.Log(
					"message",
					fmt.Sprintf("%s at:%s/%s by:%s(%s) %s",
						event.Message.ID,
						event.GuildID,
						event.ChannelID,
						event.Message.Author.Tag(),
						event.Message.Author.ID,
						event.Message.Content,
					),
					event.Message.CreatedAt,
				); err != nil {
					b.Logger.Errorf("error on logger log: %s", err)
				}
			}
		}

		return nil
	}
}

func LogEvent(b *botlib.Bot[*client.Client]) handler.Event {
	return handler.Event{
		Handler: eventHandler(b),
	}
}

func eventHandler(b *botlib.Bot[*client.Client]) handler.RawHandler {
	return func(event bot.Event) error {
		switch e := event.(type) {
		case *events.GuildAuditLogEntryCreate:
			if l, ok := b.Self.Logger.DebugGuild[e.GuildID]; ok {
				if l.LogChannel != nil {
					raw, err := json.MarshalIndent(e, "", "  ")
					if err != nil {
						b.Logger.Errorf("error on marshal json object: %s", err.Error())
						return err
					}
					if _, err := botlib.SendWebhook(event.Client(), *l.LogChannel, discord.WebhookMessageCreate{
						Files: []*discord.File{
							{
								Name:   "audit-log-" + e.GuildID.String() + ".json",
								Reader: bytes.NewBuffer(raw),
							},
						},
					}); err != nil {
						b.Logger.Errorf("error on debug channel: %s", err.Error())
					}
				}
				if l.Logger != nil {
					raw, err := json.Marshal(e)
					if err != nil {
						b.Logger.Errorf("error on marshal json object: %s", err.Error())
						return err
					}
					if err := l.Logger.Log(
						"audit-log",
						string(raw),
						e.AuditLogEntry.ID.Time(),
					); err != nil {
						b.Logger.Errorf("error on logger log: %s", err)
					}
				}
			}
		}
		return nil
	}
}
