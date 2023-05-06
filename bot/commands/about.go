package commands

import (
	"runtime"

	"github.com/sabafly/gobot/bot/db"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	lib "github.com/sabafly/sabafly-lib"
	botlib "github.com/sabafly/sabafly-lib/bot"
	"github.com/sabafly/sabafly-lib/handler"
)

func About(b *botlib.Bot[db.DB]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "about",
			Description:  "ボットの情報を表示します",
			DMPermission: &b.Config.DMPermission,
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"": aboutCommandHandler(b),
		},
	}
}

func aboutCommandHandler(b *botlib.Bot[db.DB]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		message := discord.NewMessageCreateBuilder()
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(botlib.BotName)
		embed.SetDescriptionf("**%s**\r- %s@%s\r**%s**\r- %s@%s\r**go version**\r- %s", lib.Name, lib.Module, lib.Version, disgo.Name, disgo.Module, disgo.Version, runtime.Version())
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message.SetEmbeds(embed.Build())
		return event.CreateMessage(message.Build())
	}
}
