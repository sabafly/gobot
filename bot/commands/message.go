package commands

import (
	"strings"

	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/handler/interactions"
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
						{
							Name:        "delete",
							Description: "delete pinned message",
						},
					},
				},
			},
		},
		Check: func(ctx *events.ApplicationCommandInteractionCreate) bool {
			return ctx.Member().Permissions.Has(discord.PermissionManageChannels)
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"pin/create": messagePinCreateCommandHandler(b),
			"pin/delete": messagePinDeleteCommandHandler(b),
		},
	}
}

func messagePinCreateCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if event.SlashCommandInteractionData().Bool("use-embed") {
			interaction_token := interactions.New(event.Token(), event.ID().Time())
			embed_dialog := db.NewEmbedDialog("message:p-e-create", interaction_token, event.Locale())
			if err := b.Self.DB.EmbedDialog().Set(embed_dialog.ID, *embed_dialog); err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
			embed_dialog.SetDescription("default message")
			if err := event.DeferCreateMessage(true); err != nil {
				return botlib.ReturnErr(event, err)
			}
			if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), embed_dialog.BaseMenu()); err != nil {
				return err
			}
			return nil
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
	}
}

func messagePinDeleteCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if !b.Self.MessagePinSync.TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy", botlib.WithEphemeral(true))
		}
		defer b.Self.MessagePinSync.Unlock()
		m, ok := b.Self.MessagePin[*event.GuildID()]
		if !ok {
			return botlib.ReturnErrMessage(event, "error_has_no_data", botlib.WithEphemeral(true))
		}
		mp, ok := m.Pins[event.Channel().ID()]
		b.Logger.Debug(*event.GuildID(), event.Channel().ID())
		if !ok {
			return botlib.ReturnErrMessage(event, "error_has_no_data", botlib.WithEphemeral(true))
		}
		if mp.LastMessageID != nil {
			_ = event.Client().Rest().DeleteMessage(mp.ChannelID, *mp.LastMessageID)
		}
		delete(m.Pins, event.Channel().ID())
		if err := b.Self.DB.MessagePin().Set(*event.GuildID(), m); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		m.Pins[event.Channel().ID()] = mp
		b.Self.MessagePin[*event.GuildID()] = m
		return event.CreateMessage(discord.MessageCreate{Content: "OK", Flags: discord.MessageFlagEphemeral})
	}
}

func MessageComponent(b *botlib.Bot[*client.Client]) handler.Component {
	return handler.Component{
		Name: "message",
		Handler: map[string]handler.ComponentHandler{
			"p-e-create": messageComponentPECreate(b),
		},
	}
}

func messageComponentPECreate(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		ed_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		ed, err := b.Self.DB.EmbedDialog().Get(ed_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		mp, err := b.Self.DB.MessagePin().Get(*event.GuildID())
		if err != nil {
			mp = db.NewMessagePin()
		}
		if token, err := ed.InteractionToken.Get(); err == nil {
			_ = event.Client().Rest().DeleteInteractionResponse(event.ApplicationID(), token)
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err)
		}
		wmc := discord.WebhookMessageCreate{
			Embeds:    []discord.Embed{ed.SetColor(botlib.Color).Build()},
			Username:  translate.Message(event.Locale(), "command_message_pin_create_pinned_message"),
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
			Username:  translate.Message(event.Locale(), "command_message_pin_create_pinned_message"),
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
		Handler: func(event *events.GuildMessageCreate) error {
			if !b.Self.MessagePinSync.TryLock() {
				return nil
			}
			defer b.Self.MessagePinSync.Unlock()
			m, ok := b.Self.MessagePin[event.GuildID]
			if !ok || !m.Enabled {
				return nil
			}
			mp, ok := m.Pins[event.ChannelID]
			if !ok {
				return nil
			}
			id, _, err := botlib.GetWebhook(event.Client(), event.ChannelID)
			if err != nil {
				b.Logger.Error(err)
				return err
			}
			if event.Message.WebhookID != nil && id == *event.Message.WebhookID {
				return nil
			}
			if err := mp.Update(event.Client()); err != nil {
				return err
			}
			m.Pins[event.ChannelID] = mp
			b.Self.MessagePin[event.GuildID] = m
			if err := b.Self.DB.MessagePin().Set(event.GuildID, m); err != nil {
				return err
			}
			return nil
		},
	}
}
