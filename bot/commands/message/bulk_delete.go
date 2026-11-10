package message

import (
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"log/slog"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func BulkDeleteCommand(c *components.Components) *generic.Command {
	return (&generic.Command{
		Namespace: "bulk_delete",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "bulk_delete",
				Description: "bulk delete messages",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionInt{
						Name:        "count",
						Description: "number of messages to delete",
						Required:    true,
						MinValue:    2,
						MaxValue:    100,
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/bulk_delete": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("message.bulk_delete"),
				},
				DiscordPerm: discord.PermissionManageMessages,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					count := event.SlashCommandInteractionData().Int("count")
					messages, err := event.Client().Rest().GetMessages(event.Channel().ID(), count)
					if err != nil {
						return errors.NewError(err)
					}

					messageIDs := make([]snowflake.ID, len(messages))
					for i, message := range messages {
						messageIDs[i] = message.ID
					}

					if err := event.Client().Rest().BulkDeleteMessages(event.Channel().ID, messageIDs); err != nil {
						return errors.NewError(err)
					}

					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContent(translate.Message(event.Locale(), "components.message.bulk_delete.success", translate.WithTemplate(map[string]any{"Count": count}))).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}

					return nil
				},
			},
		},
	}).SetComponent(c)
}
