package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
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
				discord.ApplicationCommandOptionSubCommand{
					Name:        "new",
					Description: "new role panel",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionBool{
							Name:        "without-thread",
							Description: "do not create thread",
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
		modal.SetCustomID(fmt.Sprintf("handler:ticket:new:%t", event.SlashCommandInteractionData().Bool("without-thread")))
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
		args := strings.Split(event.Data.CustomID, ":")
		without_thread, _ := strconv.ParseBool(args[3])

		ticket_data, err := b.Self.DB.GuildTicketData().Get(context.TODO(), *event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		defer ticket_data.Close()

		ticket := db.NewTicket(*event.GuildID(), event.User())
		ticket.SetSubject(event.ModalSubmitInteraction.Data.Text("subject"))
		ticket.SetContent(event.ModalSubmitInteraction.Data.Text("content"))
		ticket.SetHasThread(!without_thread)

		ticket_data.Value.Tickets = append(ticket_data.Value.Tickets, ticket.ID())
		if err := ticket_data.Set(context.TODO()); err != nil {
			return botlib.ReturnErr(event, err)
		}

		if err := b.Self.DB.Ticket().Set(context.Background(), ticket.ID(), *ticket); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if _, err := event.Client().Rest().CreateMessage(ticket_data.Value.ChannelID(), db.TicketMessage(discord.NewMessageCreateBuilder(), ticket).Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}

		message := discord.NewMessageBuilder()
		message.SetContent(translate.Message(event.Locale(), "ticket_new_created"))
		message.SetFlags(discord.MessageFlagEphemeral)
		if err := event.RespondMessage(message); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}
