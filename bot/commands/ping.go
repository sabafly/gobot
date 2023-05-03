package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/bot"
	"github.com/sabafly/sabafly-lib/handler"
	"github.com/sabafly/sabafly-lib/translate"
)

func Ping(b *botlib.Bot[db.DB]) handler.Command {
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

func pingHandler(b *botlib.Bot[db.DB]) handler.CommandHandler {
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
		embeds = botlib.SetEmbedsProperties(embeds)
		err := e.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
		})
		if err != nil {
			return err
		}
		return nil
	}
}
