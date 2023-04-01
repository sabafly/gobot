package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/handler"
)

func Util(b *botlib.Bot) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "util",
			Description:  "utilities",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "calc",
					Description: "in discord calculator",
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{},
	}
}

func UtilCommandCalcHandler(b *botlib.Bot) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		tokenID := uuid.New()
		err := b.DB.Interactions().Set(tokenID, event.Token())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		return nil
	}
}
