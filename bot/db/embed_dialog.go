package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/sabafly-disgo/discord"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler/interactions"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

type EmbedDialogDB interface {
	Set(id uuid.UUID, data EmbedDialog) error
	Get(id uuid.UUID) (*EmbedDialog, error)
	Del(id uuid.UUID) error
}

type embedDialogDBImpl struct {
	db *redis.Client
}

func (e *embedDialogDBImpl) Set(id uuid.UUID, data EmbedDialog) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := e.db.Set(context.TODO(), "embed-dialog"+id.String(), buf, time.Minute*5)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (e *embedDialogDBImpl) Get(id uuid.UUID) (*EmbedDialog, error) {
	res := e.db.Get(context.TODO(), "embed-dialog"+id.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	var data EmbedDialog
	if err := json.Unmarshal([]byte(res.Val()), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (e *embedDialogDBImpl) Del(id uuid.UUID) error {
	res := e.db.Del(context.TODO(), id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewEmbedDialog(handler_name string, interaction_token interactions.Token, locale discord.Locale) *EmbedDialog {
	return &EmbedDialog{
		ID:               uuid.New(),
		HandlerName:      handler_name,
		Locale:           locale,
		InteractionToken: interaction_token,
		EmbedBuilder:     discord.NewEmbedBuilder(),
	}
}

type EmbedDialog struct {
	ID                    uuid.UUID          `json:"id"`
	HandlerName           string             `json:"handler_name"`
	Locale                discord.Locale     `json:"locale"`
	InteractionToken      interactions.Token `json:"interaction_token"`
	*discord.EmbedBuilder `json:"embed"`
}

func (e EmbedDialog) BaseMenu() (mes discord.MessageUpdate) {
	if mes.Embeds == nil {
		mes.Embeds = &[]discord.Embed{}
	}
	*mes.Embeds = append(*mes.Embeds, e.Build())
	*mes.Embeds = append(*mes.Embeds,
		discord.Embed{
			Author: &discord.EmbedAuthor{
				Name: translate.Message(e.Locale, "embed_dialog_base_menu_title"),
			},
		},
	)
	*mes.Embeds = botlib.SetEmbedsProperties(*mes.Embeds)
	if mes.Components == nil {
		mes.Components = &[]discord.ContainerComponent{}
	}
	*mes.Components = append(*mes.Components,
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    translate.Message(e.Locale, "embed_dialog_button_to_title_description_menu"),
				CustomID: fmt.Sprintf("handler:ed:to-title-desc:%s", e.ID.String()),
			},
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    "✓", //TODO 絵文字にする
				CustomID: fmt.Sprintf("handler:%s:%s", e.HandlerName, e.ID.String()),
			},
		},
	)
	return
}

func (e EmbedDialog) TitleDescriptionMenu() (mes discord.MessageUpdate) {
	if mes.Embeds == nil {
		mes.Embeds = &[]discord.Embed{}
	}
	*mes.Embeds = append(*mes.Embeds, discord.Embed{
		Title: translate.Message(e.Locale, "embed_dialog_title_description_menu_title"),
		Fields: []discord.EmbedField{
			{
				Name:  translate.Message(e.Locale, "embed_dialog_title_description_menu_fields_0_title"),
				Value: e.Embed.Title,
			},
			{
				Name:  translate.Message(e.Locale, "embed_dialog_title_description_menu_fields_1_title"),
				Value: e.Embed.Description,
			},
		},
	})
	*mes.Embeds = botlib.SetEmbedsProperties(*mes.Embeds)
	if mes.Components == nil {
		mes.Components = &[]discord.ContainerComponent{}
	}
	*mes.Components = append(*mes.Components,
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    translate.Message(e.Locale, "embed_dialog_title_description_menu_button_set_title_label"),
				CustomID: fmt.Sprintf("handler:ed:set-title:%s", e.ID.String()),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    translate.Message(e.Locale, "embed_dialog_title_description_menu_button_set_description_label"),
				CustomID: fmt.Sprintf("handler:ed:set-desc:%s", e.ID.String()),
			},
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    "Back", //TODO: 絵文字にする
				CustomID: fmt.Sprintf("handler:ed:base-menu:%s", e.ID.String()),
			},
		},
	)
	return
}
