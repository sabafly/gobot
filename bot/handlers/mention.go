package handlers

import (
	"fmt"
	"strings"

	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func MentionMessage(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		Handler: mentionMessageHandler(b),
	}
}

func mentionMessageHandler(b *botlib.Bot[*client.Client]) handler.MessageHandler {
	return func(event *events.GuildMessageCreate) error {
		if event.Message.Author.Bot || event.Message.Author.System {
			return nil
		}

		if !strings.Contains(event.Message.Content, fmt.Sprintf("<@%s>", event.Client().ApplicationID())) {
			return nil
		}

		message := discord.NewMessageCreateBuilder()
		message.SetMessageReferenceByID(event.MessageID)
		message.SetAllowedMentions(&discord.AllowedMentions{
			RepliedUser: true,
		})

		user, err := b.Self.DB.UserData().Get(event.Message.Author.ID)
		if err != nil {
			return nil
		}
		message.SetContent(translate.Message(user.Locale, "message_mention_help"))

		if _, err := event.Client().Rest().CreateMessage(event.ChannelID, message.Build()); err != nil {
			return err
		}
		return nil
	}
}
