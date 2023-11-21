package debug

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/json"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func Command(c *components.Components) *generic.GenericCommand {
	return (&generic.GenericCommand{
		Namespace: "debug",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:                     "debug",
				Description:              "debug",
				DMPermission:             builtin.Ptr(false),
				DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionAdministrator),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "translate",
						Description: "translate",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:        "get",
								Description: "get translate",
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:        "key",
										Description: "translate key",
										Required:    true,
									},
									discord.ApplicationCommandOptionString{
										Name:        "locale",
										Description: "locale",
										Required:    true,
									},
								},
							},
							{
								Name:        "reload",
								Description: "reload translate",
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/debug/translate/get": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				key := event.SlashCommandInteractionData().String("key")
				locale := discord.Locale(event.SlashCommandInteractionData().String("locale"))
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent(translate.Message(locale, key)).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
			"/debug/translate/reload": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				if _, err := translate.LoadDir(c.Config().TranslateDir); err != nil {
					slog.Error("翻訳ファイルを読み込めません", "err", err)
					return errors.NewError(err)
				}
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetContent("OK").
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
		},
	}).SetDB(c)
}
