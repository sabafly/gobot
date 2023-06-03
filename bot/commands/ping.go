package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Ping(b *botlib.Bot[*client.Client]) handler.Command {
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

func pingHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(e *events.ApplicationCommandInteractionCreate) error {
		embeds := []discord.Embed{}
		embeds = append(embeds, discord.Embed{
			Title: "🏓 " + translate.Message(e.Locale(), "command_text_ping_response_embed_title"),
			Fields: []discord.EmbedField{
				{
					Name:  fmt.Sprintf("DiscordAPI(#%d)", e.ShardID()),
					Value: e.Client().ShardManager().Shard(e.ShardID()).Latency().String(),
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
