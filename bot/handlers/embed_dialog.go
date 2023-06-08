package handlers

import (
	"strings"

	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func EmbedDialogComponent(b *botlib.Bot[*client.Client]) handler.Component {
	return handler.Component{
		Name: "ed",
		Handler: map[string]handler.ComponentHandler{
			"base-menu":     embedDialogComponentBaseMenuHandler(b),
			"to-title-desc": embedDialogComponentTitleDescHandler(b),
			"set-title":     embedDialogComponentSetTitleHandler(b),
			"set-desc":      embedDialogComponentSetDescHandler(b),
		},
	}
}

func embedDialogComponentBaseMenuHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
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
		token, err := ed.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, ed.BaseMenu()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func embedDialogComponentTitleDescHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
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
		token, err := ed.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, ed.TitleDescriptionMenu()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func embedDialogComponentSetTitleHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
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
		if err := event.CreateModal(discord.ModalCreate{
			CustomID: event.Data.CustomID(),
			Title:    translate.Message(event.Locale(), "embed_dialog_set_title_modal_title"),
			Components: []discord.ContainerComponent{
				discord.NewActionRow(
					discord.TextInputComponent{
						Style:     discord.TextInputStyleShort,
						CustomID:  "title",
						Label:     translate.Message(event.Locale(), "embed_dialog_set_title_modal_0_text_input_label"),
						Required:  true,
						MaxLength: 256,
						Value:     ed.Title,
					},
				),
			},
		}); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func embedDialogComponentSetDescHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
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
		if err := event.CreateModal(discord.ModalCreate{
			CustomID: event.Data.CustomID(),
			Title:    translate.Message(event.Locale(), "embed_dialog_set_desc_modal_title"),
			Components: []discord.ContainerComponent{
				discord.NewActionRow(
					discord.TextInputComponent{
						Style:     discord.TextInputStyleParagraph,
						CustomID:  "desc",
						Label:     translate.Message(event.Locale(), "embed_dialog_set_desc_modal_0_text_input_label"),
						Required:  true,
						MaxLength: 4000,
						Value:     ed.Description,
					},
				),
			},
		}); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func EmbedDialogModal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "ed",
		Handler: map[string]handler.ModalHandler{
			"set-title": embedDialogModalSetTitleHandler(b),
			"set-desc":  embedDialogModalSetDescHandler(b),
		},
	}
}

func embedDialogModalSetTitleHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		ed_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		ed, err := b.Self.DB.EmbedDialog().Get(ed_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		ed.SetTitle(event.ModalSubmitInteraction.Data.Text("title"))
		if err := b.Self.DB.EmbedDialog().Set(ed.ID, *ed); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		token, err := ed.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, ed.TitleDescriptionMenu()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return event.DeferUpdateMessage()
	}
}

func embedDialogModalSetDescHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		ed_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		ed, err := b.Self.DB.EmbedDialog().Get(ed_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		ed.SetDescription(event.ModalSubmitInteraction.Data.Text("desc"))
		if err := b.Self.DB.EmbedDialog().Set(ed.ID, *ed); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		token, err := ed.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, ed.TitleDescriptionMenu()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return event.DeferUpdateMessage()
	}
}
