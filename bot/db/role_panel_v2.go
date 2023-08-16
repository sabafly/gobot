package db

import (
	"context"
	"fmt"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

type RolePanelV2DB interface {
	Get(id uuid.UUID) (*RolePanelV2, error)
	Set(id uuid.UUID, data RolePanelV2) error
	Del(id uuid.UUID) error
}

type rolePanelV2DBImpl struct {
	db *redis.Client
}

func (r *rolePanelV2DBImpl) Get(id uuid.UUID) (*RolePanelV2, error) {
	res := r.db.HGet(context.TODO(), "role-panel-v2", id.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	data := &RolePanelV2{}
	if err := json.Unmarshal([]byte(res.Val()), data); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *rolePanelV2DBImpl) Set(id uuid.UUID, data RolePanelV2) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := r.db.HSet(context.TODO(), "role-panel-v2", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (r *rolePanelV2DBImpl) Del(id uuid.UUID) error {
	res := r.db.HDel(context.TODO(), "role-panel-v2", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewRolePanelV2(name, description string) RolePanelV2 {
	return RolePanelV2{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Roles:       []RolePanelV2Role{},
	}
}

type RolePanelV2 struct {
	ID          uuid.UUID         `json:"uuid"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Roles       []RolePanelV2Role `json:"roles"`
}

type rolePanelV2MessageBuilder[T any] interface {
	AddEmbeds(...discord.Embed) T
	AddContainerComponents(...discord.ContainerComponent) T
}

func RolePanelV2EditMenuEmbed[T rolePanelV2MessageBuilder[T]](r *RolePanelV2, locale discord.Locale, edit *RolePanelV2Edit, message T) T {
	// 埋め込みを組み立てる
	embed := discord.NewEmbedBuilder()
	embed.AddFields(
		discord.EmbedField{
			Name:  translate.Message(locale, "rp_v2_edit_embed_field_title_0"),
			Value: r.Name,
		},
		discord.EmbedField{
			Name:  translate.Message(locale, "rp_v2_edit_embed_field_title_1"),
			Value: r.Description,
		},
	)

	disabled := false

	role_select_menu_option := make([]discord.StringSelectMenuOption, len(r.Roles))
	for i, rpvr := range r.Roles {
		role_select_menu_option[i] = discord.StringSelectMenuOption{
			Label:   rpvr.RoleName,
			Value:   rpvr.RoleID.String(),
			Emoji:   rpvr.Emoji,
			Default: edit.IsSelected(rpvr.RoleID),
		}
	}

	if len(r.Roles) == 0 {
		disabled = true
		role_select_menu_option = append(role_select_menu_option, discord.StringSelectMenuOption{
			Label: "disabled",
			Value: "disabled",
		})
	}

	// コンポーネントを組み立てる(クソだるい)
	role_select_menu := discord.StringSelectMenuComponent{
		CustomID:    fmt.Sprintf("handler:rp-v2:edit-rsm:%s", edit.ID.String()),
		Placeholder: translate.Message(locale, "rp_v2_edit_role_select_menu_placeholder"),
		MinValues:   json.Ptr(0),
		MaxValues:   1,
		Disabled:    disabled,
		Options:     role_select_menu_option,
	}

	panel_edit_buttons := []discord.InteractiveComponent{
		discord.ButtonComponent{
			Style:    discord.ButtonStylePrimary,
			Label:    translate.Message(locale, "rp_v2_edit_embed_edit_name_button"),
			CustomID: fmt.Sprintf("handler:rp-v2:edit_name:%s", edit.ID.String()),
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStylePrimary,
			Label:    translate.Message(locale, "rp_v2_edit_embed_edit_description_button"),
			CustomID: fmt.Sprintf("handler:rp-v2:edit_description:%s", edit.ID.String()),
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    translate.Message(locale, "rp_v2_edit_embed_edit_roles_button"),
			CustomID: fmt.Sprintf("handler:rp-v2:edit_roles:%s", edit.ID.String()),
		},
	}

	role_edit_buttons := []discord.InteractiveComponent{
		discord.ButtonComponent{
			Style:    discord.ButtonStyleDanger,
			Label:    translate.Message(locale, "rp_v2_edit_embed_edit_role_emoji_button"),
			CustomID: fmt.Sprintf("handler:rp-v2:edit_role_emoji:%s", edit.ID.String()),
			Disabled: !edit.HasSelectedRole(),
		},
		discord.ButtonComponent{
			Style:    discord.ButtonStyleSuccess,
			Label:    translate.Message(locale, "rp_v2_edit_embed_edit_role_name_button"),
			CustomID: fmt.Sprintf("handler:rp-v2:edit_role_name:%s", edit.ID.String()),
			Disabled: !edit.HasSelectedRole(),
		},
	}

	embed.Embed = botlib.SetEmbedProperties(embed.Embed)
	message.AddEmbeds(embed.Build())
	message.AddContainerComponents(
		discord.ActionRowComponent(panel_edit_buttons),
		discord.NewActionRow(
			role_select_menu,
		),
		discord.ActionRowComponent(role_edit_buttons),
	)

	return message
}

type RolePanelV2Role struct {
	RoleID   snowflake.ID            `json:"role_id"`
	RoleName string                  `json:"role_name"`
	Emoji    *discord.ComponentEmoji `json:"emoji"`
}
