/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package role

import (
	"fmt"
	"slices"
	"strings"

	"github.com/disgoorg/disgo/discord"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/emoji"
	"github.com/sabafly/gobot/internal/i18n"
)

func initialize(edit *models.RolePanelEdit, panel *models.RolePanel) {
	if edit.Roles == nil {
		edit.Roles = panel.Roles
	}
	if edit.Name == nil {
		edit.Name = &panel.Name
	}
	if edit.Description == nil {
		edit.Description = &panel.Description
	}
}

func rpEditBaseMessage(c *components.Components, panel *models.RolePanel, edit *models.RolePanelEdit, locale discord.Locale) (discordutil.MessageBuilder, error) {
	initialize(edit, panel)
	builder := discordutil.NewMessageBuilder()
	var roleField strings.Builder
	for i, r := range edit.Roles {
		emojiVal := r.Emoji
		if emojiVal == nil {
			emojiVal = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		fmt.Fprintf(&roleField, "%s: %s: %s\n", discordutil.FormatComponentEmoji(*emojiVal), r.Name, discord.RoleMention(r.ID))
	}

	embedList := []discord.Embed{
		discord.NewEmbed().
			WithTitle(i18n.TranslateText(locale, "components.role.panel.edit.menu.base.title", nil)).
			WithFields(
				discord.EmbedField{
					Name:   i18n.TranslateText(locale, "components.role.panel.edit.menu.base.field.name", nil),
					Value:  builtin.NonNil(edit.Name),
					Inline: new(true),
				},
				discord.EmbedField{
					Name:   i18n.TranslateText(locale, "components.role.panel.edit.menu.base.field.description", nil),
					Value:  builtin.Or(builtin.NonNil(edit.Description) != "", builtin.NonNil(edit.Description), fmt.Sprintf("`%s`", i18n.TranslateText(locale, "components.role.panel.edit.menu.base.field.value.empty", nil))),
					Inline: new(true),
				},
				discord.EmbedField{
					Name:  i18n.TranslateText(locale, "components.role.panel.edit.menu.base.field.roles", nil),
					Value: builtin.Or(roleField.String() != "", roleField.String(), fmt.Sprintf("`%s`", i18n.TranslateText(locale, "components.role.panel.edit.menu.base.field.value.empty", nil))),
				}).
			WithFooterTextf("id: %s", panel.ID),
	}
	builder.SetEmbeds(embeds.SetEmbedsProperties(embedList)...)

	var placeCount int64
	if err := c.GormDB().Model(&models.RolePanelPlaced{}).Where("role_panel_id = ?", panel.ID).Count(&placeCount).Error; err != nil {
		return builder, err
	}

	disabled := len(edit.Roles) < 1 || edit.SelectedRole == nil || !slices.ContainsFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole })
	builder.SetComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.change_name", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:change_name:%s", edit.ID),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.change_description", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:change_description:%s", edit.ID),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.modify_roles", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:modify_roles:%s", edit.ID),
			},
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.save_change", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:save_change:%s", edit.ID),
				Disabled: !edit.Modified,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleDanger,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.apply_change", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:apply_change:%s", edit.ID),
				Disabled: !panel.AppliedAt.Before(panel.UpdatedAt) || len(panel.Roles) < 1 || placeCount < 1,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.place", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:place:%s", edit.ID),
				Disabled: len(panel.Roles) < 1 || (panel.AppliedAt.Before(panel.UpdatedAt) && placeCount > 0),
			},
		),
		discord.NewActionRow(
			func() discord.StringSelectMenuComponent {
				options := make([]discord.StringSelectMenuOption, len(edit.Roles))
				for i, r := range edit.Roles {
					emojiVal := r.Emoji
					if emojiVal == nil {
						emojiVal = &discord.ComponentEmoji{
							Name: discordutil.Index2Emoji(i),
						}
					}
					options[i] = discord.StringSelectMenuOption{
						Label:   r.Name,
						Value:   r.ID.String(),
						Emoji:   emojiVal,
						Default: edit.SelectedRole != nil && *edit.SelectedRole == r.ID,
					}
				}
				if len(edit.Roles) < 1 {
					options = append(options, discord.NewStringSelectMenuOption("nil", "nil"))
				}
				return discord.StringSelectMenuComponent{
					CustomID:    fmt.Sprintf("role:panel_edit_component:select_role:%s", edit.ID),
					Placeholder: i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.select_role", nil),
					MinValues:   new(0),
					MaxValues:   1,
					Disabled:    len(edit.Roles) < 1,
					Options:     options,
				}
			}(),
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    "↑",
				CustomID: fmt.Sprintf("role:panel_edit_component:move_up:%s", edit.ID),
				Disabled: disabled || slices.IndexFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole }) == 0,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleDanger,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.delete", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:delete:%s", edit.ID),
				Disabled: disabled || len(edit.Roles) < 2,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    "↓",
				CustomID: fmt.Sprintf("role:panel_edit_component:move_down:%s", edit.ID),
				Disabled: disabled || slices.IndexFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole }) == len(edit.Roles)-1,
			},
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.set_emoji", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:set_emoji:%s", edit.ID),
				Disabled: disabled,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.set_display_name", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:set_display_name:%s", edit.ID),
				Disabled: disabled,
			},
		),
	)

	return builder, nil
}

func rpEditModifyRolesMessage(edit *models.RolePanelEdit, locale discord.Locale) discordutil.MessageBuilder {
	builder := discordutil.NewMessageBuilder()
	var roleField strings.Builder
	for i, r := range edit.Roles {
		emojiVal := r.Emoji
		if emojiVal == nil {
			emojiVal = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		fmt.Fprintf(&roleField, "%s: %s: %s\n", discordutil.FormatComponentEmoji(*emojiVal), r.Name, discord.RoleMention(r.ID))
	}
	embedList := []discord.Embed{
		discord.NewEmbed().
			WithTitle(i18n.TranslateText(locale, "components.role.panel.edit.menu.modify_roles.title", nil)).
			WithFields(
				discord.EmbedField{
					Name:  i18n.TranslateText(locale, "components.role.panel.edit.menu.modify_roles.field.roles", nil),
					Value: roleField.String(),
				},
			),
	}
	builder.SetEmbeds(embeds.SetEmbedsProperties(embedList)...)

	builder.SetComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.base.components.back_base_menu", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:base_menu:%s", edit.ID),
			},
		),
		discord.NewActionRow(
			discord.RoleSelectMenuComponent{
				CustomID:  fmt.Sprintf("role:panel_edit_component:add_role:%s", edit.ID),
				MinValues: new(1),
				MaxValues: 20,
				DefaultValues: func() []discord.SelectMenuDefaultValue {
					values := make([]discord.SelectMenuDefaultValue, len(edit.Roles))
					for i := range edit.Roles {
						values[i] = discord.NewSelectMenuDefaultRole(edit.Roles[i].ID)
					}
					return values
				}(),
			},
		),
	)
	return builder
}

func rpEditSetEmojiMessage(edit *models.RolePanelEdit, locale discord.Locale) discordutil.MessageBuilder {
	builder := discordutil.NewMessageBuilder()
	embed := discord.NewEmbed().
		WithTitle(i18n.TranslateText(locale, "components.role.panel.edit.menu.set_emoji.title", nil)).
		WithDescription(i18n.TranslateText(locale, "components.role.panel.edit.menu.set_emoji.description", nil))

	builder.SetEmbeds(embeds.SetEmbedProperties(embed))

	builder.SetComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.set_emoji.components.cancel", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:cancel_emoji:%s", edit.ID),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleDanger,
				Label:    i18n.TranslateText(locale, "components.role.panel.edit.menu.set_emoji.components.reset", nil),
				CustomID: fmt.Sprintf("role:panel_edit_component:reset_emoji:%s", edit.ID),
			},
		),
	)
	return builder
}

func rpPlaceBaseMenu(place *models.RolePanelPlaced, locale discord.Locale) discordutil.MessageBuilder {
	builder := discordutil.NewMessageBuilder()
	var roleField strings.Builder
	for i, r := range place.Roles {
		emojiVal := r.Emoji
		if emojiVal == nil {
			emojiVal = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		fmt.Fprintf(&roleField, "%s| %s\n", discordutil.FormatComponentEmoji(*emojiVal), builtin.Or(place.UseDisplayName, r.Name, discord.RoleMention(r.ID)))
	}
	embedList := []discord.Embed{
		discord.NewEmbed().
			WithAuthorName(i18n.TranslateText(locale, "components.role.panel.place.menu.author.text", nil)),
		discord.NewEmbed().
			WithTitle(place.Name).
			WithDescription(place.Description).
			WithFields(
				discord.EmbedField{
					Name:  i18n.TranslateText(locale, "components.role.panel.embed.field.role", nil),
					Value: roleField.String(),
				},
			),
	}
	builder.SetEmbeds(embeds.SetEmbedsProperties(embedList)...)

	builder.AddComponents(
		discord.NewActionRow(
			discord.StringSelectMenuComponent{
				CustomID:    fmt.Sprintf("role:panel_place_component:type:%s", place.ID),
				MinValues:   new(1),
				MaxValues:   1,
				Placeholder: i18n.TranslateText(locale, "components.role.panel.place.menu.select_type.placeholder", nil),
				Options: []discord.StringSelectMenuOption{
					{
						Label:       i18n.TranslateText(locale, "components.role.panel.type.reaction", nil),
						Value:       RolePanelPlacedTypeReaction,
						Description: i18n.TranslateText(locale, "components.role.panel.type.reaction.description", nil),
						Emoji:       emoji.Reaction,
						Default:     place.Type == RolePanelPlacedTypeReaction,
					},
					{
						Label:       i18n.TranslateText(locale, "components.role.panel.type.select_menu", nil),
						Value:       RolePanelPlacedTypeSelectMenu,
						Description: i18n.TranslateText(locale, "components.role.panel.type.select_menu.description", nil),
						Emoji:       emoji.SelectMenu,
						Default:     place.Type == RolePanelPlacedTypeSelectMenu,
					},
					{
						Label:       i18n.TranslateText(locale, "components.role.panel.type.button", nil),
						Value:       RolePanelPlacedTypeButton,
						Description: i18n.TranslateText(locale, "components.role.panel.type.button.description", nil),
						Emoji:       emoji.Button,
						Default:     place.Type == RolePanelPlacedTypeButton,
					},
				},
			},
		),
	)

	switch place.Type {
	case RolePanelPlacedTypeButton:
		builder.AddComponents(
			discord.NewActionRow(
				discord.StringSelectMenuComponent{
					CustomID:  fmt.Sprintf("role:panel_place_component:button_type:%s", place.ID),
					MinValues: new(1),
					MaxValues: 1,
					Options: []discord.StringSelectMenuOption{
						{
							Label:   i18n.TranslateText(locale, "components.role.panel.button.color.green", nil),
							Value:   "green",
							Emoji:   emoji.GreenButton,
							Default: place.ButtonType == discord.ButtonStyleSuccess,
						},
						{
							Label:   i18n.TranslateText(locale, "components.role.panel.button.color.blue", nil),
							Value:   "blue",
							Emoji:   emoji.BlueButton,
							Default: place.ButtonType == discord.ButtonStylePrimary,
						},
						{
							Label:   i18n.TranslateText(locale, "components.role.panel.button.color.red", nil),
							Value:   "red",
							Emoji:   emoji.RedButton,
							Default: place.ButtonType == discord.ButtonStyleDanger,
						},
						{
							Label:   i18n.TranslateText(locale, "components.role.panel.button.color.gray", nil),
							Value:   "gray",
							Emoji:   emoji.GrayButton,
							Default: place.ButtonType == discord.ButtonStyleSecondary,
						},
					},
				},
			),
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    i18n.TranslateText(locale, "components.role.panel.place.menu.button.show_name", nil),
					Emoji:    builtin.Or(place.ShowName, emoji.On, emoji.Off),
					CustomID: fmt.Sprintf("role:panel_place_component:show_name:%s", place.ID),
				},
			),
		)
	case RolePanelPlacedTypeSelectMenu:
		builder.AddComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    i18n.TranslateText(locale, "components.role.panel.place.menu.select_menu.folding_select_menu", nil),
					Emoji:    builtin.Or(place.FoldingSelectMenu, emoji.On, emoji.Off),
					CustomID: fmt.Sprintf("role:panel_place_component:folding_select_menu:%s", place.ID),
				},
			),
		)
	case RolePanelPlacedTypeReaction:
		builder.AddComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    i18n.TranslateText(locale, "components.role.panel.place.menu.reaction.hide_notice", nil),
					Emoji:    builtin.Or(place.HideNotice, emoji.On, emoji.Off),
					CustomID: fmt.Sprintf("role:panel_place_component:hide_notice:%s", place.ID),
				},
			),
		)
	}

	builder.AddComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    i18n.TranslateText(locale, "components.role.panel.place.menu.generic.use_display_name", nil),
				Emoji:    builtin.Or(place.UseDisplayName, emoji.On, emoji.Off),
				CustomID: fmt.Sprintf("role:panel_place_component:use_display_name:%s", place.ID),
				Disabled: place.Type == "",
			},
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    i18n.TranslateText(locale, "components.role.panel.place.menu.generic.create", nil),
				CustomID: fmt.Sprintf("role:panel_place_component:create:%s", place.ID),
				Disabled: place.Type == "",
			},
		),
	)

	return builder
}

func rpPlacedMessage(place *models.RolePanelPlaced, locale discord.Locale) discordutil.MessageBuilder {
	builder := discordutil.NewMessageBuilder()
	var roleField strings.Builder
	for i, r := range place.Roles {
		emojiVal := r.Emoji
		if emojiVal == nil {
			emojiVal = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		fmt.Fprintf(&roleField, "%s| %s\n", discordutil.FormatComponentEmoji(*emojiVal), builtin.Or(place.UseDisplayName, r.Name, discord.RoleMention(r.ID)))
	}
	embedList := []discord.Embed{
		discord.NewEmbed().
			WithTitle(place.Name).
			WithDescription(place.Description).
			WithFields(
				discord.EmbedField{
					Name:  i18n.TranslateText(locale, "components.role.panel.embed.field.role", nil),
					Value: roleField.String(),
				},
			),
	}
	builder.SetEmbeds(embeds.SetEmbedsProperties(embedList)...)

	switch place.Type {
	case RolePanelPlacedTypeButton:
		buttons := make([]discord.InteractiveComponent, len(place.Roles))
		for i, role := range place.Roles {
			var label string
			if place.ShowName {
				label = role.Name
			}
			emojiVal := role.Emoji
			if emojiVal == nil {
				emojiVal = &discord.ComponentEmoji{
					Name: discordutil.Index2Emoji(i),
				}
			}
			buttons[i] = discord.ButtonComponent{
				Style:    place.ButtonType,
				Emoji:    emojiVal,
				Label:    label,
				CustomID: fmt.Sprintf("role:panel_use:button:%s:%s", place.ID, role.ID),
			}
		}
		components := make([]discord.LayoutComponent, (len(place.Roles)-1)/5+1)
		for i := range components {
			count := min(len(buttons), 5)
			components[i] = discord.NewActionRow(buttons[:count]...)
			buttons = buttons[count:]
		}
		builder.AddComponents(
			components...,
		)
	case RolePanelPlacedTypeSelectMenu:
		if place.FoldingSelectMenu {
			builder.AddComponents(
				discord.NewActionRow(
					discord.ButtonComponent{
						Style:    discord.ButtonStyleSuccess,
						Label:    i18n.TranslateText(locale, "components.role.panel.components.use_button", nil),
						CustomID: fmt.Sprintf("role:panel_use:select_menu_fold:%s", place.ID),
					},
				),
			)
		} else {
			builder.AddComponents(rpPlacedSelectMenu(place, locale))
		}
	}
	return builder
}

func rpPlacedSelectMenu(place *models.RolePanelPlaced, locale discord.Locale) discord.ActionRowComponent {
	options := make([]discord.StringSelectMenuOption, len(place.Roles))
	for i, role := range place.Roles {
		emojiVal := role.Emoji
		if emojiVal == nil {
			emojiVal = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		options[i] = discord.StringSelectMenuOption{
			Label: role.Name,
			Value: role.ID.String(),
			Emoji: emojiVal,
		}
	}
	actionRow := discord.NewActionRow(
		discord.StringSelectMenuComponent{
			CustomID:    fmt.Sprintf("role:panel_use:select_menu:%s", place.ID.String()),
			Placeholder: i18n.TranslateText(locale, "components.role.panel.components.select_menu.placeholder", nil),
			MinValues:   new(0),
			MaxValues:   len(place.Roles),
			Options:     options,
		},
	)
	return actionRow
}
