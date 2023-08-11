package db

import (
	"context"
	"fmt"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
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

func (r RolePanelV2) EditMenuEmbed(locale discord.Locale, edit RolePanelV2Edit) discord.MessageCreate {
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

	role_edit_buttons := []discord.ButtonComponent{
		{
			Style: discord.ButtonStylePr,
			Label: "名前を編集",
		},
	}
}

type RolePanelV2Role struct {
	RoleID   snowflake.ID            `json:"role_id"`
	RoleName string                  `json:"role_name"`
	Emoji    *discord.ComponentEmoji `json:"emoji"`
}
