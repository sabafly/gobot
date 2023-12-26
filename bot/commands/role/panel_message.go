package role

import (
	"fmt"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/schema"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/emoji"
	"github.com/sabafly/gobot/internal/translate"
)

func rp_edit_base_message(panel *ent.RolePanel, edit *ent.RolePanelEdit, locale discord.Locale) discord.MessageBuilder {
	builder := discord.NewMessageBuilder()
	var roleField string
	for i, r := range edit.Roles {
		if r.Emoji == nil {
			r.Emoji = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		roleField += fmt.Sprintf("%s: %s: %s\n", discordutil.FormatComponentEmoji(*r.Emoji), r.Name, discord.RoleMention(r.ID))
	}
	if edit.Name == nil {
		edit.Name = &panel.Name
	}
	if edit.Description == nil {
		edit.Description = &panel.Description
	}
	if edit.Roles == nil {
		edit.Roles = panel.Roles
	}
	builder.SetEmbeds(
		embeds.SetEmbedsProperties(
			[]discord.Embed{
				discord.NewEmbedBuilder().
					SetTitle(translate.Message(locale, "components.role.panel.edit.menu.base.title")).
					SetFields(
						discord.EmbedField{
							Name:   translate.Message(locale, "components.role.panel.edit.menu.base.field.name"),
							Value:  builtin.NonNil(edit.Name),
							Inline: builtin.Ptr(true),
						},
						discord.EmbedField{
							Name:   translate.Message(locale, "components.role.panel.edit.menu.base.field.description"),
							Value:  builtin.Or(builtin.NonNil(edit.Description) != "", builtin.NonNil(edit.Description), fmt.Sprintf("`%s`", translate.Message(locale, "components.role.panel.edit.menu.base.field.value.empty"))),
							Inline: builtin.Ptr(true),
						},
						discord.EmbedField{
							Name:  translate.Message(locale, "components.role.panel.edit.menu.base.field.roles"),
							Value: builtin.Or(roleField != "", roleField, fmt.Sprintf("`%s`", translate.Message(locale, "components.role.panel.edit.menu.base.field.value.empty"))),
						},
					).
					SetFooterTextf("id: %s", edit.ID).
					Build(),
			},
		)...,
	)

	disabled := len(edit.Roles) < 1 || edit.SelectedRole == nil || !slices.ContainsFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole })
	builder.SetContainerComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.change_name"),
				CustomID: fmt.Sprintf("role:panel_edit_component:change_name:%s", edit.ID),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.change_description"),
				CustomID: fmt.Sprintf("role:panel_edit_component:change_description:%s", edit.ID),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.modify_roles"),
				CustomID: fmt.Sprintf("role:panel_edit_component:modify_roles:%s", edit.ID),
			},
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.save_change"),
				CustomID: fmt.Sprintf("role:panel_edit_component:save_change:%s", edit.ID),
				Disabled: !edit.Modified,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleDanger,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.apply_change"),
				CustomID: fmt.Sprintf("role:panel_edit_component:apply_change:%s", edit.ID),
				Disabled: !panel.AppliedAt.Before(panel.UpdatedAt),
			},
		),
		discord.NewActionRow(
			func() discord.StringSelectMenuComponent {
				menu := discord.StringSelectMenuComponent{
					CustomID:    fmt.Sprintf("role:panel_edit_component:select_role:%s", edit.ID),
					Placeholder: translate.Message(locale, "components.role.panel.edit.menu.base.components.select_role"),
					MinValues:   builtin.Ptr(0),
					MaxValues:   1,
					Disabled:    len(edit.Roles) < 1,
					Options: func() []discord.StringSelectMenuOption {
						options := make([]discord.StringSelectMenuOption, len(edit.Roles))
						for i, r := range edit.Roles {
							if r.Emoji == nil {
								r.Emoji = &discord.ComponentEmoji{
									Name: discordutil.Index2Emoji(i),
								}
							}
							options[i] = discord.StringSelectMenuOption{
								Label:   r.Name,
								Value:   r.ID.String(),
								Emoji:   r.Emoji,
								Default: edit.SelectedRole != nil && *edit.SelectedRole == r.ID,
							}
						}
						if len(edit.Roles) < 1 {
							options = append(options, discord.NewStringSelectMenuOption("nil", "nil"))
						}
						return options
					}(),
				}
				return menu
			}(),
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    "↑",
				CustomID: fmt.Sprintf("role:panel_edit_component:move_up:%s", edit.ID),
				Disabled: disabled || slices.IndexFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole }) == 0,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleDanger,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.delete"),
				CustomID: fmt.Sprintf("role:panel_edit_component:delete:%s", edit.ID),
				Disabled: disabled || len(edit.Roles) < 2,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStylePrimary,
				Label:    "↓",
				CustomID: fmt.Sprintf("role:panel_edit_component:move_down:%s", edit.ID),
				Disabled: disabled || slices.IndexFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole }) == len(edit.Roles)-1,
			},
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.set_emoji"),
				CustomID: fmt.Sprintf("role:panel_edit_component:set_emoji:%s", edit.ID),
				Disabled: disabled,
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.set_display_name"),
				CustomID: fmt.Sprintf("role:panel_edit_component:set_display_name:%s", edit.ID),
				Disabled: disabled,
			},
		),
	)

	return builder
}

func rp_edit_modify_roles_message(p *ent.RolePanel, edit *ent.RolePanelEdit, locale discord.Locale) discord.MessageBuilder {
	builder := discord.NewMessageBuilder()
	var roleField string
	for i, r := range p.Roles {
		if r.Emoji == nil {
			r.Emoji = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		roleField += fmt.Sprintf("%s: %s: %s\n", discordutil.FormatComponentEmoji(*r.Emoji), r.Name, discord.RoleMention(r.ID))
	}
	builder.SetEmbeds(
		embeds.SetEmbedsProperties(
			[]discord.Embed{
				discord.NewEmbedBuilder().
					SetTitle(translate.Message(locale, "components.role.panel.edit.menu.modify_roles.title")).
					SetFields(
						discord.EmbedField{
							Name:  translate.Message(locale, "components.role.panel.edit.menu.modify_roles.field.roles"),
							Value: roleField,
						},
					).
					Build(),
			},
		)...,
	)

	builder.SetContainerComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.base.components.back_base_menu"),
				CustomID: fmt.Sprintf("role:panel_edit_component:base_menu:%s", edit.ID),
			},
		),
		discord.NewActionRow(
			discord.RoleSelectMenuComponent{
				CustomID:  fmt.Sprintf("role:panel_edit_component:add_role:%s", edit.ID),
				MinValues: builtin.Ptr(1),
				MaxValues: 20,
				DefaultValues: func() []discord.SelectMenuDefaultValue {
					values := make([]discord.SelectMenuDefaultValue, len(p.Roles))
					for i := range p.Roles {
						values[i] = discord.NewSelectMenuDefaultRole(p.Roles[i].ID)
					}
					return values
				}(),
			},
		),
	)
	return builder
}

func rp_edit_set_emoji_message(p *ent.RolePanel, edit *ent.RolePanelEdit, locale discord.Locale) discord.MessageBuilder {
	builder := discord.NewMessageBuilder()
	builder.SetEmbeds(
		embeds.SetEmbedProperties(
			discord.NewEmbedBuilder().
				SetTitle(translate.Message(locale, "components.role.panel.edit.menu.set_emoji.title")).
				SetDescription(translate.Message(locale, "components.role.panel.edit.menu.set_emoji.description")).
				Build(),
		),
	)

	builder.SetContainerComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.set_emoji.components.cancel"),
				CustomID: fmt.Sprintf("role:panel_edit_component:cancel_emoji:%s", edit.ID),
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyleDanger,
				Label:    translate.Message(locale, "components.role.panel.edit.menu.set_emoji.components.reset"),
				CustomID: fmt.Sprintf("role:panel_edit_component:reset_emoji:%s", edit.ID),
			},
		),
	)
	return builder
}

func rp_place_base_menu(place *ent.RolePanelPlaced, locale discord.Locale) discord.MessageBuilder {
	builder := discord.NewMessageBuilder()
	var roleField string
	for i, r := range place.Roles {
		if r.Emoji == nil {
			r.Emoji = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		roleField += fmt.Sprintf("%s| %s\n", discordutil.FormatComponentEmoji(*r.Emoji), builtin.Or(place.UseDisplayName, r.Name, discord.RoleMention(r.ID)))
	}
	builder.SetEmbeds(
		embeds.SetEmbedsProperties(
			[]discord.Embed{
				discord.NewEmbedBuilder().
					SetAuthorName(translate.Message(locale, "components.role.panel.place.menu.author.text")).
					Build(),
				discord.NewEmbedBuilder().
					SetTitle(place.Name).
					SetDescription(place.Description).
					SetFields(
						discord.EmbedField{
							Name:  translate.Message(locale, "components.role.panel.embed.field.role"),
							Value: roleField,
						},
					).
					Build(),
			},
		)...,
	)

	builder.AddContainerComponents(
		discord.NewActionRow(
			discord.StringSelectMenuComponent{
				CustomID:    fmt.Sprintf("role:panel_place_component:type:%s", place.ID),
				MinValues:   builtin.Ptr(1),
				MaxValues:   1,
				Placeholder: translate.Message(locale, "components.role.panel.place.menu.select_type.placeholder"),
				Options: []discord.StringSelectMenuOption{
					{
						Label:       translate.Message(locale, "components.role.panel.type.reaction"),
						Value:       rolepanelplaced.TypeReaction.String(),
						Description: translate.Message(locale, "components.role.panel.type.reaction.description"),
						Emoji:       emoji.Reaction,
						Default:     place.Type == rolepanelplaced.TypeReaction,
					},
					{
						Label:       translate.Message(locale, "components.role.panel.type.select_menu"),
						Value:       rolepanelplaced.TypeSelectMenu.String(),
						Description: translate.Message(locale, "components.role.panel.type.select_menu.description"),
						Emoji:       emoji.SelectMenu,
						Default:     place.Type == rolepanelplaced.TypeSelectMenu,
					},
					{
						Label:       translate.Message(locale, "components.role.panel.type.button"),
						Value:       rolepanelplaced.TypeButton.String(),
						Description: translate.Message(locale, "components.role.panel.type.button.description"),
						Emoji:       emoji.Button,
						Default:     place.Type == rolepanelplaced.TypeButton,
					},
				},
			},
		),
	)

	switch place.Type {
	case rolepanelplaced.TypeButton:
		builder.AddContainerComponents(
			discord.NewActionRow(
				discord.StringSelectMenuComponent{
					CustomID:  fmt.Sprintf("role:panel_place_component:button_type:%s", place.ID),
					MinValues: builtin.Ptr(1),
					MaxValues: 1,
					Options: []discord.StringSelectMenuOption{
						{
							Label:   translate.Message(locale, "components.role.panel.button.color.green"),
							Value:   "green",
							Emoji:   emoji.GreenButton,
							Default: place.ButtonType == discord.ButtonStyleSuccess,
						},
						{
							Label:   translate.Message(locale, "components.role.panel.button.color.blue"),
							Value:   "blue",
							Emoji:   emoji.BlueButton,
							Default: place.ButtonType == discord.ButtonStylePrimary,
						},
						{
							Label:   translate.Message(locale, "components.role.panel.button.color.red"),
							Value:   "red",
							Emoji:   emoji.RedButton,
							Default: place.ButtonType == discord.ButtonStyleDanger,
						},
						{
							Label:   translate.Message(locale, "components.role.panel.button.color.gray"),
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
					Label:    translate.Message(locale, "components.role.panel.place.menu.button.show_name"),
					Emoji:    builtin.Or(place.ShowName, emoji.On, emoji.Off),
					CustomID: fmt.Sprintf("role:panel_place_component:show_name:%s", place.ID),
				},
			),
		)
	case rolepanelplaced.TypeSelectMenu:
		builder.AddContainerComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    translate.Message(locale, "components.role.panel.place.menu.select_menu.folding_select_menu"),
					Emoji:    builtin.Or(place.FoldingSelectMenu, emoji.On, emoji.Off),
					CustomID: fmt.Sprintf("role:panel_place_component:folding_select_menu:%s", place.ID),
				},
			),
		)
	case rolepanelplaced.TypeReaction:
		builder.AddContainerComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    translate.Message(locale, "components.role.panel.place.menu.reaction.hide_notice"),
					Emoji:    builtin.Or(place.HideNotice, emoji.On, emoji.Off),
					CustomID: fmt.Sprintf("role:panel_place_component:hide_notice:%s", place.ID),
				},
			),
		)
	}

	builder.AddContainerComponents(
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSecondary,
				Label:    translate.Message(locale, "components.role.panel.place.menu.generic.use_display_name"),
				Emoji:    builtin.Or(place.UseDisplayName, emoji.On, emoji.Off),
				CustomID: fmt.Sprintf("role:panel_place_component:use_display_name:%s", place.ID),
				Disabled: place.Type == "",
			},
		),
		discord.NewActionRow(
			discord.ButtonComponent{
				Style:    discord.ButtonStyleSuccess,
				Label:    translate.Message(locale, "components.role.panel.place.menu.generic.create"),
				CustomID: fmt.Sprintf("role:panel_place_component:create:%s", place.ID),
				Disabled: place.Type == "",
			},
		),
	)

	return builder
}

func rp_placed_message(place *ent.RolePanelPlaced, locale discord.Locale) discord.MessageBuilder {
	builder := discord.NewMessageBuilder()
	var roleField string
	for i, r := range place.Roles {
		if r.Emoji == nil {
			r.Emoji = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		roleField += fmt.Sprintf("%s| %s\n", discordutil.FormatComponentEmoji(*r.Emoji), builtin.Or(place.UseDisplayName, r.Name, discord.RoleMention(r.ID)))
	}
	builder.SetEmbeds(
		embeds.SetEmbedsProperties(
			[]discord.Embed{
				discord.NewEmbedBuilder().
					SetTitle(place.Name).
					SetDescription(place.Description).
					SetFields(
						discord.EmbedField{
							Name:  translate.Message(locale, "components.role.panel.embed.field.role"),
							Value: roleField,
						},
					).
					Build(),
			},
		)...,
	)
	switch place.Type {
	case rolepanelplaced.TypeButton:
		buttons := make([]discord.InteractiveComponent, len(place.Roles))
		for i, role := range place.Roles {
			var label string
			if place.ShowName {
				label = role.Name
			}
			if role.Emoji == nil {
				role.Emoji = &discord.ComponentEmoji{
					Name: discordutil.Index2Emoji(i),
				}
			}
			buttons[i] = discord.ButtonComponent{
				Style:    place.ButtonType,
				Emoji:    role.Emoji,
				Label:    label,
				CustomID: fmt.Sprintf("role:panel_use:button:%s:%s", place.ID, role.ID),
			}
		}
		components := make([]discord.ContainerComponent, (len(place.Roles)-1)/5+1)
		for i := range components {
			count := 5
			if len(buttons) < 5 {
				count = len(buttons)
			}
			components[i] = discord.NewActionRow(buttons[:count]...)
			buttons = buttons[count:]
		}
		builder.AddContainerComponents(
			components...,
		)
	case rolepanelplaced.TypeSelectMenu:
		if place.FoldingSelectMenu {
			builder.AddContainerComponents(
				discord.NewActionRow(
					discord.ButtonComponent{
						Style:    discord.ButtonStyleSuccess,
						Label:    translate.Message(locale, "components.role.panel.components.use_button"),
						CustomID: fmt.Sprintf("role:panel_use:select_menu_fold:%s", place.ID),
					},
				),
			)
		} else {
			builder.AddContainerComponents(rp_placed_select_menu(place, locale))
		}
	}
	return builder
}

func rp_placed_select_menu(place *ent.RolePanelPlaced, locale discord.Locale) discord.ActionRowComponent {
	options := make([]discord.StringSelectMenuOption, len(place.Roles))
	for i, role := range place.Roles {
		if role.Emoji == nil {
			role.Emoji = &discord.ComponentEmoji{
				Name: discordutil.Index2Emoji(i),
			}
		}
		options[i] = discord.StringSelectMenuOption{
			Label: role.Name,
			Value: role.ID.String(),
			Emoji: role.Emoji,
		}
	}
	actionRow := discord.NewActionRow(
		discord.StringSelectMenuComponent{
			CustomID:    fmt.Sprintf("role:panel_use:select_menu:%s", place.ID.String()),
			Placeholder: translate.Message(locale, "components.role.panel.components.select_menu.placeholder"),
			MinValues:   builtin.Ptr(0),
			MaxValues:   len(place.Roles),
			Options:     options,
		},
	)
	return actionRow
}
