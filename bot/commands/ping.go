package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/handler"
	"github.com/sabafly/gobot/lib/translate"
)

func Ping(b *botlib.Bot) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "ping",
			Description:  "pong!",
			DMPermission: &b.Config.DMPermission,
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"": pingHandler(b),
		},
	}
}

func pingHandler(b *botlib.Bot) handler.CommandHandler {
	return func(e *events.ApplicationCommandInteractionCreate) error {
		embeds := []discord.Embed{}
		embeds = append(embeds, discord.Embed{
			Title: "🏓 " + translate.Message(e.Locale(), "command_text_ping_response_embed_title"),
			Fields: []discord.EmbedField{
				{
					Name:  "DiscordAPI",
					Value: e.Client().Gateway().Latency().String(),
				},
			},
		})
		embeds = botlib.SetEmbedProperties(embeds)
		err := e.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
		})
		if err != nil {
			return err
		}
		return nil
	}
}
