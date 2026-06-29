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
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/emoji"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/ratelimit"
	"github.com/sabafly/gobot/internal/translate"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "role",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "role",
				Description: "role",
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:        "panel",
						Description: "panel",
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "create",
								Description:              "create role panel",
								DescriptionLocalizations: translate.MessageMap("components.role.panel.create.command.description", false),
							},
							{
								Name:                     "place",
								Description:              "place role panel",
								DescriptionLocalizations: translate.MessageMap("components.role.panel.place.command.description", false),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:                     "panel",
										Description:              "panel name or id",
										NameLocalizations:        translate.MessageMap("components.role.panel.place.command.options.panel.name", false),
										DescriptionLocalizations: translate.MessageMap("components.role.panel.place.command.options.panel.description", false),
										Required:                 true,
										Autocomplete:             true,
									},
								},
							},
							{
								Name:                     "edit",
								Description:              "edit role panel",
								DescriptionLocalizations: translate.MessageMap("components.role.panel.edit.command.description", false),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:                     "panel",
										Description:              "panel name or id",
										NameLocalizations:        translate.MessageMap("components.role.panel.edit.command.options.panel.name", false),
										DescriptionLocalizations: translate.MessageMap("components.role.panel.edit.command.options.panel.description", false),
										Required:                 true,
										Autocomplete:             true,
									},
								},
							},
							{
								Name:                     "delete",
								Description:              "delete role panel",
								DescriptionLocalizations: translate.MessageMap("components.role.panel.delete.command.description", false),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:                     "panel",
										Description:              "panel name or id",
										NameLocalizations:        translate.MessageMap("components.role.panel.delete.command.options.panel.name", false),
										DescriptionLocalizations: translate.MessageMap("components.role.panel.delete.command.options.panel.description", false),
										Required:                 true,
										Autocomplete:             true,
									},
								},
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/role/panel/create": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.create"),
				},
				DiscordPerm: discord.PermissionManageRoles,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.Modal(
						discord.NewModalCreateBuilder().
							SetTitle(translate.Message(event.Locale(), "components.role.panel.create.modal.title")).
							SetCustomID("role:panel_create_modal").
							SetComponents(
								discord.NewLabel(translate.Message(event.Locale(), "components.role.panel.create.modal.input.1.label"),
									discord.TextInputComponent{
										CustomID:  "name",
										Style:     discord.TextInputStyleShort,
										MinLength: builtin.Ptr(1),
										MaxLength: 32,
										Required:  true,
										Value:     translate.Message(event.Locale(), "components.role.panel.default_name"),
									}),
								discord.NewLabel(translate.Message(event.Locale(), "components.role.panel.create.modal.input.2.label"),
									discord.TextInputComponent{
										CustomID:  "description",
										Style:     discord.TextInputStyleParagraph,
										MaxLength: 140,
									},
								),
							).
							Build(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/role/panel/edit": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.edit"),
				},
				DiscordPerm: discord.PermissionManageRoles,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					panelID, err := uuid.Parse(event.SlashCommandInteractionData().String("panel"))
					if err != nil {
						return errors.NewError(err)
					}
					var rolePanel models.RolePanel
					if err := c.GormDB().Where("id = ? AND guild_id = ?", panelID, g.ID).First(&rolePanel).Error; err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					var oldEdit models.RolePanelEdit
					if err := c.GormDB().Where("parent_id = ?", rolePanel.ID).First(&oldEdit).Error; err == nil {
						c.GormDB().Delete(&oldEdit)
					}
					var removeRoles []snowflake.ID
					var discordRoles []discord.Role
					for _, r := range rolePanel.Roles {
						if discordRoles == nil {
							discordRoles, err = event.Client().Rest.GetRoles(*event.GuildID())
							if err != nil {
								return errors.NewError(err)
							}
						}
						if !slices.ContainsFunc(discordRoles, func(role discord.Role) bool { return role.ID == r.ID }) {
							removeRoles = append(removeRoles, r.ID)
						}
					}
					for _, id := range removeRoles {
						rolePanel.Roles = slices.DeleteFunc(rolePanel.Roles, func(r models.Role) bool { return r.ID == id })
					}
					rolePanel.UpdatedAt = time.Now()
					c.GormDB().Save(&rolePanel)
					edit := models.RolePanelEdit{ID: uuid.New(), GuildID: g.ID, ParentID: rolePanel.ID, ChannelID: event.Channel().ID()}
					if err := c.GormDB().Create(&edit).Error; err != nil {
						return errors.NewError(err)
					}
					edit.Parent = rolePanel
					builder, err := rpEditBaseMessage(c, &rolePanel, &edit, event.Locale())
					if err != nil {
						return errors.NewError(err)
					}
					builder.SetFlags(discord.MessageFlagEphemeral)
					if err := event.CreateMessage(builder.BuildCreate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/role/panel/place": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.place"),
				},
				DiscordPerm: discord.PermissionManageRoles,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					panelID, err := uuid.Parse(event.SlashCommandInteractionData().String("panel"))
					if err != nil {
						return errors.NewError(err)
					}
					place, err := createPanelPlace(event, c, panelID, event.Channel().ID(), g)
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					builder := rpPlaceBaseMenu(place, event.Locale())
					builder.SetFlags(discord.MessageFlagEphemeral)
					if err := event.CreateMessage(builder.BuildCreate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/role/panel/delete": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.delete"),
				},
				DiscordPerm: discord.PermissionManageRoles,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					panelID, err := uuid.Parse(event.SlashCommandInteractionData().String("panel"))
					if err != nil {
						return errors.NewError(err)
					}
					var panel models.RolePanel
					if err := c.GormDB().Where("id = ? AND guild_id = ?", panelID, g.ID).First(&panel).Error; err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}

					if err := c.GormDB().Transaction(func(tx *gorm.DB) error {
						var places []models.RolePanelPlaced
						if err := tx.Where("role_panel_id = ?", panel.ID).Find(&places).Error; err != nil {
							return err
						}

						for _, place := range places {
							if place.MessageID != nil {
								if err := event.Client().Rest.DeleteMessage(place.ChannelID, *place.MessageID); err != nil {
									slog.Error("Failed to delete message", "error", err, "panel_id", panel.ID, "channel_id", place.ChannelID, "message_id", *place.MessageID)
									return err
								}
							}
						}

						if err := tx.Where("role_panel_id = ?", panel.ID).Delete(&models.RolePanelPlaced{}).Error; err != nil {
							return err
						}
						if err := tx.Where("parent_id = ?", panel.ID).Delete(&models.RolePanelEdit{}).Error; err != nil {
							return err
						}
						if err := tx.Delete(&panel).Error; err != nil {
							return err
						}
						return nil
					}); err != nil {
						return errors.NewError(err)
					}

					builder := discord.NewMessageBuilder()
					builder.SetEmbeds(
						embeds.SetEmbedProperties(
							discord.NewEmbedBuilder().
								SetTitle(translate.Message(event.Locale(), "components.role.panel.delete.message.embed.title")).
								SetDescription(translate.Message(event.Locale(), "components.role.panel.delete.message.embed.description", translate.WithTemplate(map[string]any{"RolePanel": panel.Name}))).
								Build(),
						))
					if err := event.CreateMessage(builder.BuildCreate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		AutocompleteHandlers: map[string]generic.PermissionAutocompleteHandler{
			"/role/panel/place:panel":  generic.PAutocompleteHandler{Permission: []generic.Permission{generic.PermissionString("role.panel.place")}, DiscordPerm: discord.PermissionManageRoles, AutocompleteHandler: panelAutocomplete},
			"/role/panel/edit:panel":   generic.PAutocompleteHandler{Permission: []generic.Permission{generic.PermissionString("role.panel.edit")}, DiscordPerm: discord.PermissionManageRoles, AutocompleteHandler: panelAutocomplete},
			"/role/panel/delete:panel": generic.PAutocompleteHandler{Permission: []generic.Permission{generic.PermissionString("role.panel.delete")}, DiscordPerm: discord.PermissionManageRoles, AutocompleteHandler: panelAutocomplete},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"role:panel_create_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				rolePanel := models.RolePanel{ID: uuid.New(), Name: event.Data.Text("name"), Description: event.Data.Text("description"), GuildID: g.ID}
				if err := c.GormDB().Create(&rolePanel).Error; err != nil {
					return errors.NewError(err)
				}
				edit := models.RolePanelEdit{ID: uuid.New(), GuildID: g.ID, ParentID: rolePanel.ID, ChannelID: event.Channel().ID()}
				if err := c.GormDB().Create(&edit).Error; err != nil {
					return errors.NewError(err)
				}
				edit.Parent = rolePanel
				initialize(&edit, &rolePanel)
				builder, err := rpEditBaseMessage(c, &rolePanel, &edit, event.Locale())
				if err != nil {
					return errors.NewError(err)
				}
				builder.SetFlags(discord.MessageFlagEphemeral)
				if err := event.CreateMessage(builder.BuildCreate()); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
			"role:panel_edit_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				args := strings.Split(event.Data.CustomID, ":")
				action := args[2]
				editID, err := uuid.Parse(args[3])
				if err != nil {
					return errors.NewError(errors.ErrorMessage("errors.timeout", event))
				}
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				var edit models.RolePanelEdit
				if err := c.GormDB().Where("id = ? AND guild_id = ?", editID, g.ID).First(&edit).Error; err != nil {
					return errors.NewError(errors.ErrorMessage("errors.timeout", event))
				}
				var panel models.RolePanel
				c.GormDB().Where("id = ?", edit.ParentID).First(&panel)
				initialize(&edit, &panel)
				switch action {
				case "change_name":
					name := event.Data.Text("name")
					edit.Modified, edit.Name = true, &name
				case "change_description":
					desc := event.Data.Text("description")
					edit.Modified, edit.Description = true, &desc
				case "set_display_name":
					if edit.SelectedRole != nil {
						idx := slices.IndexFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole })
						if idx != -1 {
							edit.Roles[idx].Name, edit.Modified = event.Data.Text("display_name"), true
						}
					}
				}
				c.GormDB().Save(&edit)
				builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
				if err != nil {
					return errors.NewError(err)
				}
				builder.SetFlags(discord.MessageFlagEphemeral)
				if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
					return errors.NewError(err)
				}
				return nil
			},
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"role:panel_edit_component": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.edit"),
				},
				DiscordPerm: discord.PermissionManageRoles,
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					action := args[2]
					editID, err := uuid.Parse(args[3])
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.invalid_argument", event))
					}
					g, _ := c.GuildCreateID(event, *event.GuildID())
					var edit models.RolePanelEdit
					if err := c.GormDB().Where("id = ? AND guild_id = ?", editID, g.ID).First(&edit).Error; err != nil {
						return errors.NewError(errors.ErrorMessage("errors.timeout", event))
					}
					var panel models.RolePanel
					c.GormDB().Where("id = ?", edit.ParentID).First(&panel)
					initialize(&edit, &panel)
					switch action {
					case "change_name", "change_description":
						nameVal, descVal := "", ""
						if edit.Name != nil {
							nameVal = *edit.Name
						}
						if edit.Description != nil {
							descVal = *edit.Description
						}
						modal := discord.NewModalCreateBuilder().SetTitle(translate.Message(event.Locale(), fmt.Sprintf("components.role.panel.edit.action.%s.title", action))).SetCustomID(fmt.Sprintf("role:panel_edit_modal:%s:%s", action, edit.ID)).SetComponents(builtin.Or(action == "change_name", discord.NewLabel(translate.Message(event.Locale(), "components.role.panel.edit.change_name.modal.input.name.label"), discord.TextInputComponent{CustomID: "name", Style: discord.TextInputStyleShort, MinLength: builtin.Ptr(1), MaxLength: 32, Required: true, Value: nameVal}), discord.NewLabel(translate.Message(event.Locale(), "components.role.panel.edit.change_description.modal.input.description.label"), discord.TextInputComponent{CustomID: "description", Style: discord.TextInputStyleParagraph, MaxLength: 140, Value: descVal}))).Build()
						if err := event.Modal(modal); err != nil {
							return errors.NewError(err)
						}
					case "modify_roles":
						builder := rpEditModifyRolesMessage(&edit, event.Locale())
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "base_menu":
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "add_role":
						selectedRoles := event.RoleSelectMenuInteractionData().Resolved.Roles
						self, valid := event.Client().Caches.SelfMember(*event.GuildID())
						if !valid {
							return errors.NewError(errors.ErrorMessage("errors.invalid.self", event))
						}
						var roles []discord.Role
						for _, id := range self.RoleIDs {
							if r, err := event.Client().Rest.GetRole(*event.GuildID(), id); err == nil {
								roles = append(roles, *r)
							}
						}
						highestRole := discordutil.GetHighestRole(roles)
						if highestRole == nil {
							return errors.NewError(errors.ErrorMessage("errors.invalid.self", event))
						}
						var deletedRole []snowflake.ID
						for i, r := range selectedRoles {
							if slices.ContainsFunc(edit.Roles, func(r1 models.Role) bool { return r1.ID == r.ID }) {
								continue
							}
							if r.Managed || r.Compare(*highestRole) != -1 {
								delete(selectedRoles, i)
								deletedRole = append(deletedRole, i)
								continue
							}
							edit.Roles = append(edit.Roles, models.Role{ID: r.ID, Name: r.Name})
						}
						if len(deletedRole) > 0 {
							var s strings.Builder
							for _, id := range deletedRole {
								fmt.Fprintf(&s, "- %s\r", discord.RoleMention(id))
							}
							embed := discord.NewEmbedBuilder().SetTitle(translate.Message(event.Locale(), "components.role.panel.edit.add_role.deleted_role.embed.title")).SetDescriptionf("%s\n"+s.String(), translate.Message(event.Locale(), "components.role.panel.edit.add_role.deleted_role.embed.description")).Build()
							if err := event.CreateMessage(discord.NewMessageBuilder().
								SetEmbeds(embeds.SetEmbedProperties(embed)).
								SetFlags(discord.MessageFlagEphemeral).
								BuildCreate()); err != nil {
								return errors.NewError(err)
							}
							return nil
						}
						edit.Roles = slices.DeleteFunc(edit.Roles, func(r models.Role) bool { _, ok := selectedRoles[r.ID]; return !ok })
						edit.Modified = true
						c.GormDB().Save(&edit)
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "select_role":
						var id *snowflake.ID
						if values := event.StringSelectMenuInteractionData().Values; len(values) > 0 {
							id = builtin.Ptr(snowflake.MustParse(values[0]))
						}
						edit.SelectedRole = id
						c.GormDB().Save(&edit)
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "delete":
						if edit.SelectedRole != nil {
							edit.Roles = slices.DeleteFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole })
							edit.Modified = true
							c.GormDB().Save(&edit)
						}
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "move_up", "move_down":
						if edit.SelectedRole != nil {
							idx := slices.IndexFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole })
							mv := builtin.Or(action == "move_up", -1, 1)
							if idx+mv >= 0 && idx+mv < len(edit.Roles) {
								edit.Roles[idx+mv], edit.Roles[idx], edit.Modified = edit.Roles[idx], edit.Roles[idx+mv], true
								c.GormDB().Save(&edit)
							}
						}
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "set_display_name":
						if edit.SelectedRole != nil {
							idx := slices.IndexFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole })
							if idx != -1 {
								modal := discord.NewModalCreateBuilder().SetTitle(translate.Message(event.Locale(), "components.role.panel.edit.set_display.name.modal.title")).SetCustomID(fmt.Sprintf("role:panel_edit_modal:set_display_name:%s", edit.ID)).SetComponents(discord.NewLabel(translate.Message(event.Locale(), "components.role.panel.edit.set_display.name.modal.input.display_name.label"), discord.TextInputComponent{CustomID: "display_name", Style: discord.TextInputStyleShort, MinLength: builtin.Ptr(1), MaxLength: 100, Required: true, Value: edit.Roles[idx].Name})).Build()
								if err := event.Modal(modal); err != nil {
									return errors.NewError(err)
								}
							}
						}
					case "set_emoji":
						if edit.SelectedRole != nil {
							token := event.Token()
							edit.EmojiAuthor = builtin.Ptr(event.User().ID)
							edit.Token = &token
							c.GormDB().Save(&edit)
						}
						builder := rpEditSetEmojiMessage(&edit, event.Locale())
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "cancel_emoji", "reset_emoji":
						edit.EmojiAuthor, edit.Token = nil, nil
						if action == "reset_emoji" && edit.SelectedRole != nil {
							idx := slices.IndexFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole })
							if idx != -1 {
								edit.Roles[idx].Emoji, edit.Modified = nil, true
							}
						}
						if err := c.GormDB().Save(&edit).Error; err != nil {
							return errors.NewError(err)
						}
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "save_change":
						edit.Modified = false
						if err := c.GormDB().Save(&edit).Error; err != nil {
							return errors.NewError(err)
						}
						if edit.Name != nil {
							panel.Name = *edit.Name
						}
						if edit.Description != nil {
							panel.Description = *edit.Description
						}
						panel.UpdatedAt = time.Now()
						if edit.Roles != nil {
							panel.Roles = edit.Roles
						}
						if err := c.GormDB().Save(&panel).Error; err != nil {
							return errors.NewError(err)
						}
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "apply_change":
						var ok bool
						g.RolePanelEditTimes, ok = ratelimit.CheckLimit(g.RolePanelEditTimes, []ratelimit.Rule{{Limit: 3, Unit: time.Minute * 10}, {Limit: 5, Unit: time.Minute * 30}})
						c.GormDB().Save(g)
						if !ok || len(panel.Roles) < 1 {
							return errors.NewError(errors.ErrorMessage("errors.ratelimited", event))
						}
						panel.AppliedAt = time.Now()
						c.GormDB().Save(&panel)
						c.GormDB().Where("(message_id IS NULL OR type = '') AND guild_id = ?", g.ID).Delete(&models.RolePanelPlaced{})
						go updateRolePanel(context.Background(), &panel, event.Locale(), event.Client(), true, c)
						builder, err := rpEditBaseMessage(c, &panel, &edit, event.Locale())
						if err != nil {
							return errors.NewError(err)
						}
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					case "place":
						place, err := createPanelPlace(event, c, panel.ID, event.Channel().ID(), g)
						if err != nil {
							return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
						}
						builder := rpPlaceBaseMenu(place, event.Locale())
						builder.SetFlags(discord.MessageFlagEphemeral)
						if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
					}
					return nil
				},
			},
			"role:panel_place_component": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.place"),
				},
				DiscordPerm: discord.PermissionManageRoles,
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					action, placeID := args[2], uuid.MustParse(args[3])
					g, _ := c.GuildCreateID(event, *event.GuildID())
					var place models.RolePanelPlaced
					if err := c.GormDB().Where("id = ? AND guild_id = ?", placeID, g.ID).First(&place).Error; err != nil {
						return errors.NewError(errors.ErrorMessage("errors.timeout", event))
					}
					var panel models.RolePanel
					c.GormDB().Where("id = ?", place.RolePanelID).First(&panel)
					switch action {
					case "type":
						place.Type = event.StringSelectMenuInteractionData().Values[0]
					case "button_type":
						var t = discord.ButtonStylePrimary
						switch event.StringSelectMenuInteractionData().Values[0] {
						case "green":
							t = discord.ButtonStyleSuccess
						case "blue":
							t = discord.ButtonStylePrimary
						case "red":
							t = discord.ButtonStyleDanger
						case "gray":
							t = discord.ButtonStyleSecondary
						}
						place.ButtonType = t
					case "show_name":
						place.ShowName = !place.ShowName
					case "folding_select_menu":
						place.FoldingSelectMenu = !place.FoldingSelectMenu
					case "hide_notice":
						place.HideNotice = !place.HideNotice
					case "use_display_name":
						place.UseDisplayName = !place.UseDisplayName
					case "create":
						if len(panel.Roles) < 1 {
							return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
						}
						if err := rolePanelPlace(event, &place, event.Locale(), event.Client(), true, c); err != nil {
							return errors.NewError(err)
						}
						embed := discord.NewEmbedBuilder().SetTitle(translate.Message(event.Locale(), "components.role.panel.create.message")).SetDescription(translate.Message(event.Locale(), "components.role.panel.create.description")).Build()
						if err := event.UpdateMessage(discord.NewMessageBuilder().
							SetEmbeds(embeds.SetEmbedProperties(embed)).
							BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
						return nil
					}
					c.GormDB().Save(&place)
					builder := rpPlaceBaseMenu(&place, event.Locale())
					builder.SetFlags(discord.MessageFlagEphemeral)
					if err := event.UpdateMessage(builder.BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"role:panel_use": generic.ComponentHandler(func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
				args := strings.Split(event.Data.CustomID(), ":")
				action, placeID := args[2], uuid.MustParse(args[3])
				g, _ := c.GuildCreateID(event, *event.GuildID())
				var place models.RolePanelPlaced
				if err := c.GormDB().Where("id = ? AND guild_id = ?", placeID, g.ID).First(&place).Error; err != nil {
					_ = event.Client().Rest.DeleteMessage(event.Channel().ID(), event.Message.ID)
					return errors.NewError(errors.ErrorMessage("errors.deleted", event))
				}
				switch action {
				case "button":
					roleID := snowflake.MustParse(args[4])
					if !slices.ContainsFunc(place.Roles, func(r models.Role) bool { return r.ID == roleID }) {
						if err := event.UpdateMessage(rpPlacedMessage(&place, event.Locale()).BuildUpdate()); err != nil {
							return errors.NewError(err)
						}
						return nil
					}
					if _, ok := event.Client().Caches.Role(*event.GuildID(), roleID); !ok {
						_ = event.DeferUpdateMessage()
						return nil
					}
					contain := slices.Contains(event.Member().RoleIDs, roleID)
					reason := rest.WithReason(fmt.Sprintf("Role Panel \"%s\" (%s)", place.Name, place.ID))
					if contain {
						if err := event.Client().Rest.RemoveMemberRole(g.ID, event.User().ID, roleID, reason); err != nil {
							return errors.NewError(errors.ErrorMessage("errors.fail.role.panel", event))
						}
					} else {
						if err := event.Client().Rest.AddMemberRole(g.ID, event.User().ID, roleID, reason); err != nil {
							return errors.NewError(errors.ErrorMessage("errors.fail.role.panel", event))
						}
					}
					embed := discord.NewEmbedBuilder().SetTitle(translate.Message(event.Locale(), "components.role.panel.use."+builtin.Or(!contain, "added", "removed"))).SetDescription(translate.Message(event.Locale(), "components.role.panel.use."+builtin.Or(!contain, "added", "removed")+`.description`, translate.WithTemplate(map[string]any{"Role": discord.RoleMention(roleID)}))).Build()
					if err := event.CreateMessage(discord.NewMessageBuilder().
						SetEmbeds(embeds.SetEmbedProperties(embed)).
						SetFlags(discord.MessageFlagEphemeral).
						BuildCreate()); err != nil {
						return errors.NewError(err)
					}
				case "select_menu_fold":
					options := make([]discord.StringSelectMenuOption, len(place.Roles))
					for i, role := range place.Roles {
						emojiVal := role.Emoji
						if emojiVal == nil {
							emojiVal = &discord.ComponentEmoji{Name: discordutil.Index2Emoji(i)}
						}
						options[i] = discord.StringSelectMenuOption{Label: role.Name, Value: role.ID.String(), Emoji: emojiVal, Default: slices.Contains(event.Member().RoleIDs, role.ID)}
					}
					if err := event.CreateMessage(discord.NewMessageBuilder().SetComponents(discord.NewActionRow(discord.StringSelectMenuComponent{CustomID: fmt.Sprintf("role:panel_use:select_menu:%s", place.ID.String()), Placeholder: translate.Message(event.Locale(), "components.role.panel.components.select_menu.placeholder"), MinValues: builtin.Ptr(0), MaxValues: len(place.Roles), Options: options})).SetFlags(discord.MessageFlagEphemeral).BuildCreate()); err != nil {
						return errors.NewError(err)
					}
				case "select_menu":
					var selectedRoles []snowflake.ID
					for _, v := range event.StringSelectMenuInteractionData().Values {
						selectedRoles = append(selectedRoles, snowflake.MustParse(v))
					}
					var addRoles, removedRoles, unchangedRole []snowflake.ID
					for _, role := range place.Roles {
						if slices.Contains(selectedRoles, role.ID) {
							if slices.Index(event.Member().RoleIDs, role.ID) != -1 {
								unchangedRole = append(unchangedRole, role.ID)
							} else {
								if _, ok := event.Client().Caches.Role(*event.GuildID(), role.ID); ok {
									addRoles = append(addRoles, role.ID)
									_ = event.Client().Rest.AddMemberRole(*event.GuildID(), event.User().ID, role.ID)
								}
							}
						} else {
							if slices.Index(event.Member().RoleIDs, role.ID) != -1 {
								removedRoles = append(removedRoles, role.ID)
								_ = event.Client().Rest.RemoveMemberRole(*event.GuildID(), event.User().ID, role.ID)
							}
						}
					}
					embed := discord.NewEmbedBuilder().SetTitle(translate.Message(event.Locale(), "components.role.panel.use.changed"))
					if len(addRoles) > 0 {
						var s strings.Builder
						for _, id := range addRoles {
							fmt.Fprintf(&s, "%s\n", discord.RoleMention(id))
						}
						embed.AddFields(discord.EmbedField{Name: translate.Message(event.Locale(), "components.role.panel.use.changed.add"), Value: s.String()})
					}
					if len(unchangedRole) > 0 {
						var s strings.Builder
						for _, id := range unchangedRole {
							fmt.Fprintf(&s, "%s\n", discord.RoleMention(id))
						}
						embed.AddFields(discord.EmbedField{Name: translate.Message(event.Locale(), "components.role.panel.use.changed.unchanged"), Value: s.String()})
					}
					if len(removedRoles) > 0 {
						var s strings.Builder
						for _, id := range removedRoles {
							fmt.Fprintf(&s, "%s\n", discord.RoleMention(id))
						}
						embed.AddFields(discord.EmbedField{Name: translate.Message(event.Locale(), "components.role.panel.use.changed.remove"), Value: s.String()})
					}
					if err := event.RespondMessage(discord.NewMessageBuilder().SetEmbeds(embeds.SetEmbedProperties(embed.Build())).SetFlags(discord.MessageFlagEphemeral)); err != nil {
						return errors.NewError(err)
					}
				}
				return nil
			}),
		},
		EventHandler: func(c *components.Components, event bot.Event) errors.Error {
			switch event := event.(type) {
			case *events.GuildMessageCreate:
				if event.Message.Author.Bot || event.Message.Author.System {
					return nil
				}
				u, err := c.UserCreate(event, event.Message.Author)
				if err != nil {
					return errors.NewError(err)
				}
				var edits []models.RolePanelEdit
				c.GormDB().Where("channel_id = ?", event.ChannelID).Find(&edits)
				for _, edit := range edits {
					if edit.EmojiAuthor == nil || *edit.EmojiAuthor != event.Message.Author.ID || edit.Token == nil {
						continue
					}
					token, emojis := *edit.Token, emoji.FindAllString(event.Message.Content)
					if len(emojis) < 1 {
						continue
					}
					componentEmoji := discordutil.ParseComponentEmoji(emojis[0])
					var panel models.RolePanel
					if err := c.GormDB().Where("id = ?", edit.ParentID).First(&panel).Error; err != nil {
						continue
					}
					initialize(&edit, &panel)
					if edit.SelectedRole != nil {
						idx := slices.IndexFunc(edit.Roles, func(r models.Role) bool { return r.ID == *edit.SelectedRole })
						if idx != -1 {
							edit.Roles[idx].Emoji, edit.EmojiAuthor, edit.Token = &componentEmoji, nil, nil
							c.GormDB().Save(&edit)
						}
					}
					_ = event.Client().Rest.AddReaction(event.ChannelID, event.MessageID, "✅")
					builder, err := rpEditBaseMessage(c, &panel, &edit, u.Locale)
					if err == nil {
						_, _ = event.Client().Rest.UpdateInteractionResponse(event.Client().ApplicationID, token, builder.SetFlags(discord.MessageFlagEphemeral).BuildUpdate())
					}
				}
			case *events.GuildMessageDelete:
				g, _ := c.GuildCreateID(event, event.GuildID)
				c.GormDB().Where("guild_id = ? AND channel_id = ? AND message_id = ?", g.ID, event.ChannelID, event.MessageID).Delete(&models.RolePanelPlaced{})
			case *events.GuildMessageReactionAdd:
				if event.Member.User.Bot || event.Member.User.System {
					return nil
				}
				u, err := c.UserCreate(event, event.Member.User)
				if err != nil {
					return errors.NewError(err)
				}
				var place models.RolePanelPlaced
				if err := c.GormDB().Where("channel_id = ? AND message_id = ?", event.ChannelID, event.MessageID).First(&place).Error; err != nil {
					return nil
				}
				var panel models.RolePanel
				if err := c.GormDB().Where("id = ?", place.RolePanelID).First(&panel).Error; err != nil {
					return nil
				}
				_ = event.Client().Rest.RemoveUserReaction(event.ChannelID, event.MessageID, event.Emoji.Reaction(), event.UserID)
				for i, role := range panel.Roles {
					emojiVal := role.Emoji
					if emojiVal == nil {
						emojiVal = &discord.ComponentEmoji{Name: discordutil.Index2Emoji(i)}
					}
					if event.Emoji.Reaction() != discordutil.ReactionComponentEmoji(*emojiVal) {
						continue
					}
					if _, ok := event.Client().Caches.Role(event.GuildID, role.ID); !ok {
						return nil
					}
					contains := slices.Contains(event.Member.RoleIDs, role.ID)
					var err error
					if contains {
						err = event.Client().Rest.RemoveMemberRole(event.GuildID, event.UserID, role.ID)
					} else {
						err = event.Client().Rest.AddMemberRole(event.GuildID, event.UserID, role.ID)
					}
					if err != nil {
						embed := discord.NewEmbedBuilder().SetTitlef("❗ %s", translate.Message(u.Locale, "errors.fail.role.panel")).SetDescription(translate.Message(u.Locale, "errors.fail.role.panel.description")).SetColor(0xff2121).Build()
						m, err := event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageBuilder().
							SetEmbeds(embeds.SetEmbedProperties(embed)).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate())
						if err == nil {
							go func() {
								if err := discordutil.DeleteMessageAfter(event.Client(), event.ChannelID, m.ID, time.Second*10); err != nil {
									slog.Error("Failed to delete message", "err", err)
								}
							}()
						}
						return nil
					}
					if !place.HideNotice {
						embed := discord.NewEmbedBuilder().
							SetTitle(translate.Message(u.Locale, "components.role.panel.use."+builtin.Or(!contains, "added", "removed"))).
							SetDescription(translate.Message(u.Locale, "components.role.panel.use."+builtin.Or(!contains, "added", "removed")+`.description`, translate.WithTemplate(map[string]any{"Role": discord.RoleMention(role.ID)}))).
							Build()
						m, err := event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageBuilder().
							SetContent(discord.UserMention(event.UserID)).
							SetEmbeds(embeds.SetEmbedProperties(embed)).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate())
						if err == nil {
							go func() {
								if err := discordutil.DeleteMessageAfter(event.Client(), event.ChannelID, m.ID, time.Second*10); err != nil {
									slog.Error("Failed to delete message", "err", err)
								}
							}()
						}
					}
				}
			}
			return nil
		}}).SetComponent(c)
}

func UpdateRolePanel(ctx context.Context, place *models.RolePanelPlaced, locale discord.Locale, client *bot.Client, c *components.Components) {
	if err := rolePanelPlace(ctx, place, locale, client, true, c); err != nil {
		slog.Error("アップデートに失敗", "err", err)
	}
}

func updateRolePanel(ctx context.Context, panel *models.RolePanel, locale discord.Locale, client *bot.Client, react bool, c *components.Components) {
	var places []models.RolePanelPlaced
	c.GormDB().Where("role_panel_id = ?", panel.ID).Find(&places)
	for _, place := range places {
		place.Name, place.Description, place.Roles, place.UpdatedAt = panel.Name, panel.Description, panel.Roles, time.Now()
		c.GormDB().Save(&place)
		if err := rolePanelPlace(ctx, &place, locale, client, react, c); err != nil {
			slog.Error("アップデートに失敗", "err", err)
		}
	}
}
