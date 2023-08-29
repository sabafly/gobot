package commands

import (
	"context"

	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
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
		Check: b.Self.CheckCommandPermission(b, "ticket.manage", discord.PermissionManageGuild),
		CommandHandlers: map[string]handler.CommandHandler{
			"new": ticketNewCommandHandler(b),
		},
	}
}

func ticketNewCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		modal := discord.NewModalCreateBuilder()
		modal.SetCustomID("handler:ticket:new")
		modal.SetTitle(translate.Message(event.Locale(), "ticket_new_modal_title"))
		modal.AddContainerComponents(
			discord.NewActionRow(
				discord.TextInputComponent{
					CustomID:  "subject",
					Style:     discord.TextInputStyleShort,
					Label:     translate.Message(event.Locale(), "ticket_new_modal_label_0"),
					Required:  true,
					MaxLength: 45,
				},
			),
			discord.NewActionRow(
				discord.TextInputComponent{
					CustomID:  "content",
					Style:     discord.TextInputStyleParagraph,
					Label:     translate.Message(event.Locale(), "ticket_new_modal_label_1"),
					Required:  false,
					MaxLength: 2048,
				},
			),
		)
		if err := event.CreateModal(modal.Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func TicketModal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "ticket",
		Handler: map[string]handler.ModalHandler{
			"new": ticketNewModalHandler(b),
		},
	}
}

func ticketNewModalHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		ticket := db.NewTicket(*event.GuildID(), event.User())
		ticket.SetSubject(event.ModalSubmitInteraction.Data.Text("subject"))
		ticket.SetContent(event.ModalSubmitInteraction.Data.Text("content"))
		if err := b.Self.DB.Ticket().Set(context.Background(), ticket.ID(), *ticket); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		message := discord.NewMessageCreateBuilder()
		message.SetContent("まだですな")
		message.SetFlags(discord.MessageFlagEphemeral)
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}
