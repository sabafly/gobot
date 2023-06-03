package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
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
									Name:        "use-embed",
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
		if event.SlashCommandInteractionData().Bool("use-embed") {
			if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("in developing").SetFlags(discord.MessageFlagEphemeral).Build()); err != nil {
				return botlib.ReturnErr(event, err)
			}
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
							MaxLength:   2000,
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
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err)
		}
		wmc := discord.WebhookMessageCreate{
			Content:   content,
			Username:  "Pinned Message",
			AvatarURL: b.Self.Config.MessagePinAvatarURL,
		}
		m, err := botlib.SendWebhook(event.Client(), event.Channel().ID(), wmc)
		if err != nil {
			return err
		}
		mp.Pins[event.Channel().ID()] = db.MessagePin{
			WebhookMessageCreate: wmc,
			ChannelID:            m.ChannelID,
			LastMessageID:        &m.ID,
		}
		if err := b.Self.DB.MessagePin().Set(*event.GuildID(), mp); err != nil {
			return err
		}
		b.Self.MessagePin[*event.GuildID()] = mp
		return nil
	}
}

func MessagePinMessageCreate(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		UUID: uuid.New(),
		Handler: func(event *events.MessageCreate) error {
			m, ok := b.Self.MessagePin[*event.GuildID]
			if !ok || !m.Enabled {
				return nil
			}
			mp, ok := m.Pins[event.ChannelID]
			if !ok {
				return nil
			}
			id, _, err := botlib.GetWebhook(event.Client(), event.ChannelID)
			if err != nil {
				return err
			}
			if event.Message.WebhookID != nil && id == *event.Message.WebhookID {
				return nil
			}
			if err := mp.Update(event.Client()); err != nil {
				return err
			}
			m.Pins[event.ChannelID] = mp
			if err := b.Self.DB.MessagePin().Set(*event.GuildID, m); err != nil {
				return err
			}
			return nil
		},
	}
}
