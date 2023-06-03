package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Message(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "message",
			Description:  "message",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "pin",
					Description: "pin",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "create",
							Description: "create pinned message",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionBool{
									Name:        "use embed",
									Description: "wither uses embed creator",
									Required:    false,
								},
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"pin/create": messagePinCreateCommandHandler(b),
		},
	}
}

func messagePinCreateCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if event.SlashCommandInteractionData().Bool("use embed") {
		} else {
			if err := event.CreateModal(discord.ModalCreate{
				Title:    translate.Message(event.Locale(), "command_message_pin_create_modal_title"),
				CustomID: "handler:message:pin-create",
				Components: []discord.ContainerComponent{
					discord.NewActionRow(
						discord.TextInputComponent{
							CustomID:    "content",
							Style:       discord.TextInputStyle(discord.TextInputStyleParagraph),
							Label:       translate.Message(event.Locale(), "command_message_pin_create_modal_action_row_0_label"),
							MaxLength:   4000,
							Placeholder: translate.Message(event.Locale(), "command_message_create_modal_action_row_0_placeholder"),
							Required:    true,
						},
					),
				},
			}); err != nil {
				return botlib.ReturnErr(event, err)
			}
			return nil
		}
		return nil
	}
}

func MessageModal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "message",
		Handler: map[string]handler.ModalHandler{
			"pin-create": messageModalPinCreate(b),
		},
	}
}

func messageModalPinCreate(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		content := event.ModalSubmitInteraction.Data.Text("content")
		mp, err := b.Self.DB.MessagePin().Get(*event.GuildID())
		if err != nil {
			mp = db.NewMessagePin()
		}
		mp.Pins[event.Channel().ID()] = discord.MessageCreate{Content: content}
		if err := b.Self.DB.MessagePin().Set(*event.GuildID(), mp); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}
