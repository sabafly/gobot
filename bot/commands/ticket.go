package commands

import (
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
)

func Ticket(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "ticket",
			Description:  "ticket",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "panel",
					Description: "panel",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "create",
							Description: "create a new ticket panel",
						},
						{
							Name:        "edit",
							Description: "edit the ticket panel",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:         "panel",
									Description:  "target panel",
									Required:     true,
									Autocomplete: true,
								},
							},
						},
						{
							Name:        "delete",
							Description: "delete the ticket panel",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:         "panel",
									Description:  "target panel",
									Required:     true,
									Autocomplete: true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "new",
					Description: "new role panel",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionChannel{
							Name:        "target",
							Description: "the targe channel",
							Required:    true,
							ChannelTypes: []discord.ChannelType{
								discord.ChannelTypeGuildText,
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{},
	}
}

func ticketNewCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {

	}
}
