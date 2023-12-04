package ping

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func Command(c *components.Components) *generic.GenericCommand {
	return (&generic.GenericCommand{
		Namespace: "ping",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "ping",
				Description:              "pong!",
				DescriptionLocalizations: translate.MessageMap("components.ping.command.description", false),
				DMPermission:             builtin.Ptr(false),
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/ping": generic.CommandHandler(func(_ *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							embeds.SetEmbedProperties(discord.NewEmbedBuilder().
								SetTitlef("🏓 %s", translate.Message(event.Locale(), "components.ping.pong")).
								SetFields(
									discord.EmbedField{
										Name:  fmt.Sprintf("**Discord API(#%d)**", event.ShardID()),
										Value: event.Client().ShardManager().Shard(event.ShardID()).Latency().String(),
									},
								).
								Build()),
						).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
		},
	}).SetComponent(c)
}
