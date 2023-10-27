package db

import (
	"context"
	"fmt"
	"slices"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/internal"
	"github.com/sabafly/sabafly-disgo/discord"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

type RolePanelV2DB interface {
	Get(id uuid.UUID) (*RolePanelV2, error)
	Set(id uuid.UUID, data *RolePanelV2) error
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

func (r *rolePanelV2DBImpl) Set(id uuid.UUID, data *RolePanelV2) error {
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

func NewRolePanelV2(name, description string) *RolePanelV2 {
	return &RolePanelV2{
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

func (r *RolePanelV2) AddRole(id snowflake.ID, name string, emoji *discord.ComponentEmoji) bool {
	if slices.IndexFunc(r.Roles, func(rpvr RolePanelV2Role) bool {
		return rpvr.RoleID == id
	}) != -1 {
		return false
	}
	if emoji == nil {
		emoji = &discord.ComponentEmoji{
			Name: botlib.Number2Emoji(len(r.Roles) + 1),
		}
	}
	r.Roles = append(r.Roles, RolePanelV2Role{
		RoleID:   id,
		RoleName: name,
		Emoji:    emoji,
	})
	return true
}

type rolePanelV2MessageBuilder[T any] interface {
	AddEmbeds(...discord.Embed) T
	AddContainerComponents(...discord.ContainerComponent) T
}

func NewRolePanelV2Config() RolePanelV2Config {
	return RolePanelV2Config{
		PanelType:        RolePanelV2TypeNone,
		ButtonStyle:      discord.ButtonStyleSuccess,
		ButtonShowName:   false,
		SimpleSelectMenu: true,
	}
}

type RolePanelV2Config struct {
	PanelType        RolePanelV2Type     `json:"panel_type"`
	ButtonStyle      discord.ButtonStyle `json:"button_style"`
	ButtonShowName   bool                `json:"show_name"`
	SimpleSelectMenu bool                `json:"simple_select_menu"`
	SendNotice       bool                `json:"send_notice"`
	UseDisplayName   bool                `json:"use_display_name"`
}

func RolePanelV2MessageReaction[T rolePanelV2MessageBuilder[T]](r *RolePanelV2, locale discord.Locale, message T, config RolePanelV2Config) T {
	message.AddEmbeds(r.rolePanelV2Embed(locale, config))
	return message
}

func RolePanelV2MessageSelectMenu[T rolePanelV2MessageBuilder[T]](r *RolePanelV2, locale discord.Locale, message T, config RolePanelV2Config) T {
	message.AddEmbeds(r.rolePanelV2Embed(locale, config))
	if config.SimpleSelectMenu {
		options := make([]discord.StringSelectMenuOption, len(r.Roles))
		for i, rpvr := range r.Roles {
			options[i] = discord.StringSelectMenuOption{
				Label: rpvr.RoleName,
				Value: rpvr.RoleID.String(),
				Emoji: rpvr.Emoji,
			}
		}
		message.AddContainerComponents(
			discord.NewActionRow(
				discord.StringSelectMenuComponent{
					CustomID:    fmt.Sprintf("handler:rp-v2:use_select_menu:%s", r.ID.String()),
					Placeholder: translate.Message(locale, "rp_v2_select_menu_placeholder"),
					MinValues:   json.Ptr(0),
					MaxValues:   len(r.Roles),
					Options:     options,
				},
			),
		)
	} else {
		message.AddContainerComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSuccess,
					Label:    translate.Message(locale, "rp_v2_use_button_label"),
					CustomID: fmt.Sprintf("handler:rp-v2:call_select_menu:%s", r.ID.String()),
				},
			),
		)
	}
	return message
}

func RolePanelV2MessageButton[T rolePanelV2MessageBuilder[T]](r *RolePanelV2, locale discord.Locale, message T, config RolePanelV2Config) T {
	message.AddEmbeds(r.rolePanelV2Embed(locale, config))
	buttons := make([]discord.InteractiveComponent, len(r.Roles))
	for i, rpvr := range r.Roles {
		var label string
		if config.ButtonShowName {
			label = rpvr.RoleName
		}
		buttons[i] = discord.ButtonComponent{
			Style:    config.ButtonStyle,
			Emoji:    rpvr.Emoji,
			Label:    label,
			CustomID: fmt.Sprintf("handler:rp-v2:use_button:%s:%s", r.ID, rpvr.RoleID),
		}
	}
	components := make([]discord.ContainerComponent, (len(r.Roles)-1)/5+1)
	for i := range components {
		count := 5
		if len(buttons) < 5 {
			count = len(buttons)
		}
		components[i] = discord.NewActionRow(buttons[:count]...)
		buttons = buttons[count:]
	}
	message.AddContainerComponents(
		components...,
	)
	return message
}

func (r *RolePanelV2) rolePanelV2Embed(locale discord.Locale, config RolePanelV2Config) discord.Embed {
	embed := discord.NewEmbedBuilder()
	embed.SetTitle(r.Name)
	embed.SetDescription(r.Description)
	var role_string string
	for _, role := range r.Roles {
		role_string += fmt.Sprintf("%s| %s\r", botlib.FormatComponentEmoji(*role.Emoji), internal.Or(config.UseDisplayName, role.RoleName, discord.RoleMention(role.RoleID)))
	}
	embed.AddFields(
		discord.EmbedField{
			Name:  translate.Message(locale, "rp_v2_roles"),
			Value: role_string,
		},
	)
	embed.Embed = botlib.SetEmbedProperties(embed.Embed)
	return embed.Build()
}

func RolePanelV2PlaceMenuEmbed[T rolePanelV2MessageBuilder[T]](r *RolePanelV2, locale discord.Locale, place *RolePanelV2Place, message T) T {
	embed := discord.NewEmbedBuilder()
	embed.SetTitle(translate.Message(locale, "rp_v2_place_embed_title"))
	embed.SetDescription(translate.Message(locale, "rp_v2_place_embed_description"))
	embed.Embed = botlib.SetEmbedProperties(embed.Embed)

	message.AddEmbeds(embed.Build())

	panel_type_select_menu := discord.StringSelectMenuComponent{
		CustomID:    fmt.Sprintf("handler:rp-v2:place_type:%s", place.ID),
		MinValues:   json.Ptr(1),
		MaxValues:   1,
		Placeholder: translate.Message(locale, "rp_v2_place_type_select_menu_placeholder"),
		Options: []discord.StringSelectMenuOption{
			{
				Label:       translate.Message(locale, "rp_v2_place_type_reaction"),
				Value:       RolePanelV2TypeReaction,
				Description: translate.Message(locale, "rp_v2_place_type_reaction_description"),
				Emoji: &discord.ComponentEmoji{
					ID:   1141985795641716736,
					Name: "reaction",
				},
				Default: place.Config.PanelType == RolePanelV2TypeReaction,
			},
			{
				Label:       translate.Message(locale, "rp_v2_place_type_select_menu"),
				Value:       RolePanelV2TypeSelectMenu,
				Description: translate.Message(locale, "rp_v2_place_type_select_menu_description"),
				Emoji: &discord.ComponentEmoji{
					ID:   1141991243832901704,
					Name: "select_menu",
				},
				Default: place.Config.PanelType == RolePanelV2TypeSelectMenu,
			},
			{
				Label:       translate.Message(locale, "rp_v2_place_type_button"),
				Value:       RolePanelV2TypeButton,
				Description: translate.Message(locale, "rp_v2_place_type_button_description"),
				Emoji: &discord.ComponentEmoji{
					ID:   1141991285281001553,
					Name: "button",
				},
				Default: place.Config.PanelType == RolePanelV2TypeButton,
			},
		},
	}

	message.AddContainerComponents(
		discord.NewActionRow(panel_type_select_menu),
	)

	switch place.Config.PanelType {
	case RolePanelV2TypeButton:
		var emoji *discord.ComponentEmoji
		if place.Config.ButtonShowName {
			emoji = &discord.ComponentEmoji{
				ID:   1142095470227890279,
				Name: "on",
			}
		} else {
			emoji = &discord.ComponentEmoji{
				ID:   1142110196462788779,
				Name: "off",
			}
		}
		message.AddContainerComponents(
			discord.NewActionRow(
				discord.StringSelectMenuComponent{
					CustomID:    fmt.Sprintf("handler:rp-v2:place_button_color:%s", place.ID),
					Placeholder: translate.Message(locale, "rp_v2_place_button_color_select_menu_placeholder"),
					MinValues:   json.Ptr(1),
					MaxValues:   1,
					Options: []discord.StringSelectMenuOption{
						{
							Label: translate.Message(locale, "rp_v2_place_button_color_green"),
							Value: "green",
							Emoji: &discord.ComponentEmoji{
								ID:   1142333937180483687,
								Name: "green_button",
							},
							Default: place.Config.ButtonStyle == discord.ButtonStyleSuccess,
						},
						{
							Label: translate.Message(locale, "rp_v2_place_button_color_blue"),
							Value: "blue",
							Emoji: &discord.ComponentEmoji{
								ID:   1142333868490367037,
								Name: "blue_button",
							},
							Default: place.Config.ButtonStyle == discord.ButtonStylePrimary,
						},
						{
							Label: translate.Message(locale, "rp_v2_place_button_color_red"),
							Value: "red",
							Emoji: &discord.ComponentEmoji{
								ID:   1142334020403871745,
								Name: "red_button",
							},
							Default: place.Config.ButtonStyle == discord.ButtonStyleDanger,
						},
						{
							Label: translate.Message(locale, "rp_v2_place_button_color_gray"),
							Value: "gray",
							Emoji: &discord.ComponentEmoji{
								ID:   1142333913906298960,
								Name: "gray_button",
							},
							Default: place.Config.ButtonStyle == discord.ButtonStyleSecondary,
						},
					},
				},
			),
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    translate.Message(locale, "rp_v2_place_button_show_name_label"),
					CustomID: fmt.Sprintf("handler:rp-v2:place_button_show_name:%s", place.ID),
					Emoji:    emoji,
				},
			),
		)
	case RolePanelV2TypeSelectMenu:
		var emoji *discord.ComponentEmoji
		if !place.Config.SimpleSelectMenu {
			emoji = &discord.ComponentEmoji{
				ID:   1142095470227890279,
				Name: "on",
			}
		} else {
			emoji = &discord.ComponentEmoji{
				ID:   1142110196462788779,
				Name: "off",
			}
		}
		message.AddContainerComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    translate.Message(locale, "rp_v2_place_simple_select_menu_label"),
					CustomID: fmt.Sprintf("handler:rp-v2:place_simple_select_menu:%s", place.ID),
					Emoji:    emoji,
				},
			),
		)
	case RolePanelV2TypeReaction:
		var emoji *discord.ComponentEmoji
		if place.Config.SendNotice {
			emoji = &discord.ComponentEmoji{
				ID:   1142095470227890279,
				Name: "on",
			}
		} else {
			emoji = &discord.ComponentEmoji{
				ID:   1142110196462788779,
				Name: "off",
			}
		}
		message.AddContainerComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    translate.Message(locale, "rp_v2_place_reaction_send_notice_label"),
					CustomID: fmt.Sprintf("handler:rp-v2:place_reaction_send_notice:%s", place.ID),
					Emoji:    emoji,
				},
			),
		)
	}

	message.AddContainerComponents(
		discord.NewActionRow(discord.ButtonComponent{
			Style:    discord.ButtonStyleSuccess,
			CustomID: fmt.Sprintf("handler:rp-v2:place:%s", place.ID.String()),
			Label:    translate.Message(locale, "rp_v2_place_button_label"),
			Disabled: place.Config.PanelType == RolePanelV2TypeNone,
		}),
	)

	return message
}

func RolePanelV2EditMenuEmbed[T rolePanelV2MessageBuilder[T]](r *RolePanelV2, locale discord.Locale, edit *RolePanelV2Edit, message T) T {
	// 埋め込みを組み立てる
	var role_string string
	for _, rpvr := range r.Roles {
		role_string += fmt.Sprintf("%s| %s\r", botlib.FormatComponentEmoji(*rpvr.Emoji), discord.RoleMention(rpvr.RoleID))
	}
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
		discord.EmbedField{
			Name:  translate.Message(locale, "rp_v2_edit_embed_field_title_2"),
			Value: role_string,
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
			Style:    discord.ButtonStyleSuccess,
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
		discord.ButtonComponent{
			Style:    discord.ButtonStyleDanger,
			Label:    translate.Message(locale, "rp_v2_edit_embed_edit_role_delete_button"),
			CustomID: fmt.Sprintf("handler:rp-v2:edit_role_delete:%s", edit.ID.String()),
			Disabled: !edit.HasSelectedRole() || len(r.Roles) <= 1,
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
