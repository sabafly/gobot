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
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/schema"
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
				Name:         "role",
				Description:  "role",
				DMPermission: builtin.Ptr(false),
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
							SetContainerComponents(
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "name",
										Style:     discord.TextInputStyleShort,
										Label:     translate.Message(event.Locale(), "components.role.panel.create.modal.input.1.label"),
										MinLength: builtin.Ptr(1),
										MaxLength: 32,
										Required:  true,
										Value:     translate.Message(event.Locale(), "components.role.panel.default_name"),
									},
								),
								discord.NewActionRow(
									discord.TextInputComponent{
										CustomID:  "description",
										Style:     discord.TextInputStyleParagraph,
										Label:     translate.Message(event.Locale(), "components.role.panel.create.modal.input.2.label"),
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

					if !g.QueryRolePanels().Where(rolepanel.ID(panelID)).ExistX(event) {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}

					rolePanel := g.QueryRolePanels().WithEdit().Where(rolepanel.ID(panelID)).FirstX(event)

					if rolePanel.QueryEdit().ExistX(event) {
						c.DB().RolePanelEdit.DeleteOneID(rolePanel.QueryEdit().FirstIDX(event)).ExecX(event)
					}

					var removeRoles []snowflake.ID
					var roles []discord.Role
					for _, r := range rolePanel.Roles {
						if roles == nil {
							roles, err = event.Client().Rest().GetRoles(*event.GuildID())
							if err != nil {
								return errors.NewError(err)
							}
						}
						if slices.ContainsFunc(roles, func(role discord.Role) bool { return role.ID == r.ID }) {
							continue
						}
						removeRoles = append(removeRoles, r.ID)
					}
					for _, id := range removeRoles {
						rolePanel.Roles = slices.DeleteFunc(rolePanel.Roles, func(r schema.Role) bool { return r.ID == id })
					}
					rolePanel =
						rolePanel.Update().
							SetUpdatedAt(time.Now()).
							SetRoles(rolePanel.Roles).
							SaveX(event)

					edit := c.DB().RolePanelEdit.Create().
						SetGuild(g).
						SetParent(rolePanel).
						SetChannelID(event.Channel().ID()).
						SaveX(event)

					if err := event.CreateMessage(
						rpEditBaseMessage(event, rolePanel, edit, event.Locale()).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
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

					if err := event.CreateMessage(
						rpPlaceBaseMenu(place, event.Locale()).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
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

					if !g.QueryRolePanels().Where(rolepanel.ID(panelID)).ExistX(event) {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}

					panel := g.QueryRolePanels().Where(rolepanel.ID(panelID)).FirstX(event)

					places := panel.QueryPlacements().AllX(event)
					for _, place := range places {
						if place.MessageID == nil {
							continue
						}
						_ = event.Client().Rest().DeleteMessage(place.ChannelID, *place.MessageID)
					}

					c.DB().RolePanelPlaced.Delete().
						Where(rolepanelplaced.HasRolePanelWith(rolepanel.ID(panel.ID))).
						ExecX(event)
					c.DB().RolePanelEdit.Delete().
						Where(rolepaneledit.HasParentWith(rolepanel.ID(panel.ID))).
						ExecX(event)

					c.DB().RolePanel.DeleteOne(panel).ExecX(event)

					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.role.panel.delete.message.embed.title")).
										SetDescription(translate.Message(event.Locale(), "components.role.panel.delete.message.embed.description", translate.WithTemplate(map[string]any{"RolePanel": panel.Name}))).
										Build(),
								),
							).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		AutocompleteHandlers: map[string]generic.PermissionAutocompleteHandler{
			"/role/panel/place:panel": generic.PAutocompleteHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.place"),
				},
				DiscordPerm:         discord.PermissionManageRoles,
				AutocompleteHandler: panelAutocomplete,
			},
			"/role/panel/edit:panel": generic.PAutocompleteHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.edit"),
				},
				DiscordPerm:         discord.PermissionManageRoles,
				AutocompleteHandler: panelAutocomplete,
			},
			"/role/panel/delete:panel": generic.PAutocompleteHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.delete"),
				},
				DiscordPerm:         discord.PermissionManageRoles,
				AutocompleteHandler: panelAutocomplete,
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"role:panel_create_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}

				rolePanel := c.DB().RolePanel.Create().
					SetName(event.Data.Text("name")).
					SetDescription(event.Data.Text("description")).
					SetGuild(g).
					SaveX(event)

				edit := c.DB().RolePanelEdit.Create().
					SetGuild(g).
					SetParent(rolePanel).
					SetChannelID(event.Channel().ID()).
					SaveX(event)

				initialize(edit, rolePanel)

				if err := event.CreateMessage(
					rpEditBaseMessage(event, rolePanel, edit, event.Locale()).
						SetFlags(discord.MessageFlagEphemeral).
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}

				return nil
			},
			"role:panel_edit_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				args := strings.Split(event.Data.CustomID, ":")
				action := args[2]
				editID, err := uuid.Parse(args[3])
				if err != nil {
					return errors.NewError(err)
				}
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				if !g.QueryRolePanelEdits().Where(rolepaneledit.ID(editID)).ExistX(event) {
					return errors.NewError(errors.ErrorMessage("errors.timeout", event))
				}

				edit := g.QueryRolePanelEdits().Where(rolepaneledit.ID(editID)).FirstX(event)
				panel := edit.QueryParent().OnlyX(event)

				initialize(edit, panel)

				switch action {
				case "change_name":
					edit = edit.Update().
						SetModified(true).
						SetName(event.Data.Text("name")).
						SaveX(event)
					if err := event.UpdateMessage(
						rpEditBaseMessage(event, panel, edit, event.Locale()).
							SetFlags(discord.MessageFlagEphemeral).
							BuildUpdate(),
					); err != nil {
						return errors.NewError(err)
					}
				case "change_description":
					edit = edit.Update().
						SetModified(true).
						SetDescription(event.Data.Text("description")).
						SaveX(event)
					if err := event.UpdateMessage(
						rpEditBaseMessage(event, panel, edit, event.Locale()).
							SetFlags(discord.MessageFlagEphemeral).
							BuildUpdate(),
					); err != nil {
						return errors.NewError(err)
					}
				case "set_display_name":
					if edit.SelectedRole != nil {
						edit.Roles[slices.IndexFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole })].Name = event.Data.Text("display_name")
						edit = edit.Update().
							SetModified(true).
							SetRoles(edit.Roles).
							SaveX(event)
					}

					if err := event.UpdateMessage(
						rpEditBaseMessage(event, panel, edit, event.Locale()).
							SetFlags(discord.MessageFlagEphemeral).
							BuildUpdate(),
					); err != nil {
						return errors.NewError(err)
					}
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
						return errors.NewError(err)
					}
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if !g.QueryRolePanelEdits().Where(rolepaneledit.ID(editID)).ExistX(event) {
						return errors.NewError(errors.ErrorMessage("errors.timeout", event))
					}
					edit := g.QueryRolePanelEdits().Where(rolepaneledit.ID(editID)).FirstX(event)
					panel := edit.QueryParent().OnlyX(event)

					initialize(edit, panel)

					switch action {
					case "change_name", "change_description":
						if err := event.Modal(
							discord.NewModalCreateBuilder().
								SetTitle(translate.Message(event.Locale(), fmt.Sprintf("components.role.panel.edit.action.%s.title", action))).
								SetCustomID(fmt.Sprintf("role:panel_edit_modal:%s:%s", action, edit.ID)).
								SetContainerComponents(
									builtin.Or(action == "change_name",
										discord.NewActionRow(
											discord.TextInputComponent{
												CustomID:  "name",
												Style:     discord.TextInputStyleShort,
												Label:     translate.Message(event.Locale(), "components.role.panel.create.modal.input.1.label"),
												MinLength: builtin.Ptr(1),
												MaxLength: 32,
												Required:  true,
												Value:     *edit.Name,
											},
										),
										discord.NewActionRow(
											discord.TextInputComponent{
												CustomID:  "description",
												Style:     discord.TextInputStyleParagraph,
												Label:     translate.Message(event.Locale(), "components.role.panel.create.modal.input.2.label"),
												MaxLength: 140,
												Value:     *edit.Description,
											},
										),
									),
								).
								Build(),
						); err != nil {
							return errors.NewError(err)
						}
					case "modify_roles":
						if err := event.UpdateMessage(
							rpEditModifyRolesMessage(edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "base_menu":
						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "add_role":
						selectedRoles := event.RoleSelectMenuInteractionData().Resolved.Roles
						self, valid := event.Client().Caches().SelfMember(*event.GuildID())
						if !valid {
							return errors.NewError(errors.ErrorMessage("errors.invalid.self", event))
						}
						var roles []discord.Role
						roleMap := map[snowflake.ID]discord.Role{}
						for _, id := range self.RoleIDs {
							role, err := event.Client().Rest().GetRole(*event.GuildID(), id)
							if err != nil {
								slog.Error("API ERROR GetRole", "error", err, "guild_id", *event.GuildID(), "id", id)
								continue
							}
							roleMap[id] = *role
							roles = append(roles, *role)
						}
						highestRole := discordutil.GetHighestRole(roles)
						if highestRole == nil {
							return errors.NewError(errors.ErrorMessage("errors.invalid.self", event))
						}
						var deletedRole []snowflake.ID

						for i, r := range selectedRoles {
							if slices.ContainsFunc(edit.Roles, func(r1 schema.Role) bool { return r1.ID == r.ID }) {
								continue
							}
							if r.Managed || r.Compare(*highestRole) != -1 {
								delete(selectedRoles, i)
								deletedRole = append(deletedRole, i)
								continue
							}
							edit.Roles = append(edit.Roles, schema.Role{
								ID:   r.ID,
								Name: r.Name,
							})
						}

						if len(deletedRole) > 0 {
							var deletedRoleString string
							for _, id := range deletedRole {
								deletedRoleString += fmt.Sprintf("- %s\r", discord.RoleMention(id))
							}
							if err := event.CreateMessage(
								discord.NewMessageBuilder().
									SetEmbeds(
										embeds.SetEmbedProperties(
											discord.NewEmbedBuilder().
												SetTitle(translate.Message(event.Locale(), "components.role.panel.edit.add_role.deleted_role.embed.title")).
												SetDescriptionf("%s\n"+deletedRoleString, translate.Message(event.Locale(), "components.role.panel.edit.add_role.deleted_role.embed.description")).
												Build(),
										),
									).
									SetFlags(discord.MessageFlagEphemeral).
									BuildCreate(),
							); err != nil {
								return errors.NewError(err)
							}
							return nil
						}
						edit.Roles = slices.DeleteFunc(edit.Roles, func(r schema.Role) bool { _, ok := selectedRoles[r.ID]; return !ok })

						edit = edit.Update().
							SetModified(true).
							SetRoles(edit.Roles).
							SaveX(event)

						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "select_role":
						var id *snowflake.ID
						if values := event.StringSelectMenuInteractionData().Values; len(values) > 0 {
							id = builtin.Ptr(snowflake.MustParse(values[0]))
						}
						if id == nil {
							edit = edit.Update().
								ClearSelectedRole().
								SaveX(event)
						} else {
							edit = edit.Update().
								SetNillableSelectedRole(id).
								SaveX(event)
						}

						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "delete":
						if edit.SelectedRole != nil {
							edit.Roles = slices.DeleteFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole })
							edit = edit.Update().
								SetModified(true).
								SetRoles(edit.Roles).
								SaveX(event)
						}

						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "move_up", "move_down":
						if edit.SelectedRole != nil {
							index := slices.IndexFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole })
							mv := builtin.Or(action == "move_up", -1, 1)
							edit.Roles[index+mv], edit.Roles[index] = edit.Roles[index], edit.Roles[index+mv]

							edit = edit.Update().
								SetModified(true).
								SetRoles(edit.Roles).
								SaveX(event)
						}

						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "set_display_name":
						if edit.SelectedRole != nil {
							if err := event.Modal(
								discord.NewModalCreateBuilder().
									SetTitle(translate.Message(event.Locale(), "components.role.panel.edit.set_display.name.modal.title")).
									SetCustomID(fmt.Sprintf("role:panel_edit_modal:set_display_name:%s", edit.ID)).
									SetContainerComponents(
										discord.NewActionRow(
											discord.TextInputComponent{
												CustomID:  "display_name",
												Style:     discord.TextInputStyleShort,
												Label:     translate.Message(event.Locale(), "components.role.panel.edit.set_display.name.modal.input.display_name.label"),
												MinLength: builtin.Ptr(1),
												MaxLength: 100,
												Required:  true,
												Value:     edit.Roles[slices.IndexFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole })].Name,
											},
										),
									).
									Build(),
							); err != nil {
								return errors.NewError(err)
							}
						}
					case "set_emoji":
						if edit.SelectedRole != nil {
							edit = edit.Update().
								SetEmojiAuthor(event.User().ID).
								SetToken(event.Token()).
								SaveX(event)
						}

						if err := event.UpdateMessage(
							rpEditSetEmojiMessage(edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "cancel_emoji", "reset_emoji":
						edit = edit.Update().
							ClearEmojiAuthor().
							ClearToken().
							SaveX(event)

						if action == "reset_emoji" {
							edit.Roles[slices.IndexFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole })].Emoji = nil
							edit = edit.Update().
								SetModified(true).
								SetRoles(edit.Roles).
								SaveX(event)
						}

						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "save_change":

						edit = edit.Update().
							SetModified(false).
							SaveX(event)

						update := panel.Update().
							SetUpdatedAt(time.Now()).
							SetNillableName(edit.Name).
							SetNillableDescription(edit.Description)
						if edit.Roles != nil {
							update.SetRoles(edit.Roles)
						}
						panel = update.SaveX(event)

						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "apply_change":
						var ok bool
						g.RolePanelEditTimes, ok = ratelimit.CheckLimit(g.RolePanelEditTimes, []ratelimit.Rule{
							{
								Limit: 3,
								Unit:  time.Minute * 10,
							},
							{
								Limit: 5,
								Unit:  time.Minute * 30,
							},
						})
						g.Update().
							SetRolePanelEditTimes(g.RolePanelEditTimes).
							SaveX(event)
						if !ok || len(panel.Roles) < 1 {
							return errors.NewError(errors.ErrorMessage("errors.ratelimited", event))
						}

						panel = panel.Update().
							SetAppliedAt(time.Now()).
							SaveX(event)

						c.DB().RolePanelPlaced.Delete().Where(rolepanelplaced.And(rolepanelplaced.Or(rolepanelplaced.MessageIDIsNil(), rolepanelplaced.TypeIsNil()), rolepanelplaced.HasGuildWith(guild.ID(g.ID)))).ExecX(event)
						go updateRolePanel(event, panel, event.Locale(), event.Client(), true)
						if err := event.UpdateMessage(
							rpEditBaseMessage(event, panel, edit, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					case "place":
						g, err := c.GuildCreateID(event, *event.GuildID())
						if err != nil {
							return errors.NewError(err)
						}

						place, err := createPanelPlace(event, c, panel.ID, event.Channel().ID(), g)
						if err != nil {
							return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
						}

						if err := event.UpdateMessage(
							rpPlaceBaseMenu(place, event.Locale()).
								SetFlags(discord.MessageFlagEphemeral).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
					default:
						slog.Warn("不明なcustom_id", "id", event.Data.CustomID())
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
					action := args[2]
					placeID, err := uuid.Parse(args[3])
					if err != nil {
						return errors.NewError(err)
					}
					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if !g.QueryRolePanelPlacements().Where(rolepanelplaced.ID(placeID)).ExistX(event) {
						return errors.NewError(errors.ErrorMessage("errors.timeout", event))
					}
					place := g.QueryRolePanelPlacements().Where(rolepanelplaced.ID(placeID)).FirstX(event)
					panel := place.QueryRolePanel().OnlyX(event)

					switch action {
					case "type":
						place = place.Update().
							SetType(rolepanelplaced.Type(event.StringSelectMenuInteractionData().Values[0])).
							SaveX(event)
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
						place = place.Update().
							SetButtonType(t).
							SaveX(event)
					case "show_name":
						place = place.Update().
							SetShowName(!place.ShowName).
							SaveX(event)
					case "folding_select_menu":
						place = place.Update().
							SetFoldingSelectMenu(!place.FoldingSelectMenu).
							SaveX(event)
					case "hide_notice":
						place = place.Update().
							SetHideNotice(!place.HideNotice).
							SaveX(event)
					case "use_display_name":
						place = place.Update().
							SetUseDisplayName(!place.UseDisplayName).
							SaveX(event)
					case "create":
						if len(panel.Roles) < 1 {
							return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
						}
						if err := rolePanelPlace(event, place, event.Locale(), event.Client(), true); err != nil {
							return errors.NewError(err)
						}

						updateMessage := discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.role.panel.create.message")).
										SetDescription(translate.Message(event.Locale(), "components.role.panel.create.description")).
										Build(),
								),
							).
							BuildUpdate()
						updateMessage.Components = &[]discord.ContainerComponent{}
						if err := event.UpdateMessage(
							updateMessage,
						); err != nil {
							return errors.NewError(err)
						}
						return nil
					}
					if err := event.UpdateMessage(
						rpPlaceBaseMenu(place, event.Locale()).
							SetFlags(discord.MessageFlagEphemeral).
							BuildUpdate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"role:panel_use": generic.ComponentHandler(func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
				args := strings.Split(event.Data.CustomID(), ":")
				action := args[2]
				placeID, err := uuid.Parse(args[3])
				if err != nil {
					return errors.NewError(err)
				}
				g, err := c.GuildCreateID(event, *event.GuildID())
				if err != nil {
					return errors.NewError(err)
				}
				if !g.QueryRolePanelPlacements().Where(rolepanelplaced.ID(placeID)).ExistX(event) {
					if err := event.Client().Rest().DeleteMessage(event.Channel().ID(), event.Message.ID); err != nil {
						return errors.NewError(err)
					}
					return errors.NewError(errors.ErrorMessage("errors.deleted", event))
				}
				place := g.QueryRolePanelPlacements().Where(rolepanelplaced.ID(placeID)).FirstX(event)

				switch action {
				case "button":
					roleID := snowflake.MustParse(args[4])
					if !slices.ContainsFunc(place.Roles, func(r schema.Role) bool { return r.ID == roleID }) {
						if err := event.UpdateMessage(
							rpPlacedMessage(place, event.Locale()).
								BuildUpdate(),
						); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					_, ok := event.Client().Caches().Role(*event.GuildID(), roleID)
					if !ok {
						if err := event.DeferUpdateMessage(); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					contain := slices.Contains(event.Member().RoleIDs, roleID)
					if contain {
						if err := event.Client().Rest().RemoveMemberRole(g.ID, event.User().ID, roleID, rest.WithReason(fmt.Sprintf("Role Panel \"%s\" (%s)", place.Name, place.ID))); err != nil {
							return errors.NewError(errors.ErrorMessage("errors.fail.role.panel", event))
						}
					} else {
						if err := event.Client().Rest().AddMemberRole(g.ID, event.User().ID, roleID, rest.WithReason(fmt.Sprintf("Role Panel \"%s\" (%s)", place.Name, place.ID))); err != nil {
							return errors.NewError(errors.ErrorMessage("errors.fail.role.panel", event))
						}
					}
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(event.Locale(), "components.role.panel.use."+builtin.Or(!contain, "added", "removed"))).
										SetDescription(translate.Message(event.Locale(), "components.role.panel.use."+builtin.Or(!contain, "added", "removed")+".description", translate.WithTemplate(map[string]any{"Role": discord.RoleMention(roleID)}))).
										Build(),
								),
							).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
				case "select_menu_fold":
					options := make([]discord.StringSelectMenuOption, len(place.Roles))
					for i, role := range place.Roles {
						if role.Emoji == nil {
							role.Emoji = &discord.ComponentEmoji{
								Name: discordutil.Index2Emoji(i),
							}
						}
						options[i] = discord.StringSelectMenuOption{
							Label:   role.Name,
							Value:   role.ID.String(),
							Emoji:   role.Emoji,
							Default: slices.Contains(event.Member().RoleIDs, role.ID),
						}
					}
					actionRow := discord.NewActionRow(
						discord.StringSelectMenuComponent{
							CustomID:    fmt.Sprintf("role:panel_use:select_menu:%s", place.ID.String()),
							Placeholder: translate.Message(event.Locale(), "components.role.panel.components.select_menu.placeholder"),
							MinValues:   builtin.Ptr(0),
							MaxValues:   len(place.Roles),
							Options:     options,
						},
					)
					if err := event.CreateMessage(
						discord.NewMessageBuilder().
							SetContainerComponents(actionRow).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
				case "select_menu":
					var selectedRoles []snowflake.ID
					for _, v := range event.StringSelectMenuInteractionData().Values {
						selectedRoles = append(selectedRoles, snowflake.MustParse(v))
					}

					var addRoles []snowflake.ID
					var removedRoles []snowflake.ID
					var unchangedRole []snowflake.ID
					for _, role := range place.Roles {
						if slices.Contains(selectedRoles, role.ID) {
							// 選ばれたとき
							if slices.Index(event.Member().RoleIDs, role.ID) != -1 {
								// 持ってたなら
								unchangedRole = append(unchangedRole, role.ID)
								continue
							} else {
								// 持ってないなら
								_, ok := event.Client().Caches().Role(*event.GuildID(), role.ID)
								if !ok {
									continue
								}
								addRoles = append(addRoles, role.ID)

								if err := event.Client().Rest().AddMemberRole(*event.GuildID(), event.User().ID, role.ID); err != nil {
									return errors.NewError(errors.ErrorMessage("errors.fail.role.panel", event))
								}
							}
						} else {
							// 選ばれてないとき
							if slices.Index(event.Member().RoleIDs, role.ID) != -1 {
								// 持ってたなら
								removedRoles = append(removedRoles, role.ID)

								if err := event.Client().Rest().RemoveMemberRole(*event.GuildID(), event.User().ID, role.ID); err != nil {
									return errors.NewError(errors.ErrorMessage("errors.fail.role.panel", event))
								}
							} else {
								// 持ってないなら
								continue
							}
						}
					}

					embed := discord.NewEmbedBuilder().
						SetTitle(translate.Message(event.Locale(), "components.role.panel.use.changed"))
					if len(addRoles) > 0 {
						var addRolesString string
						for _, id := range addRoles {
							addRolesString += fmt.Sprintf("%s\n", discord.RoleMention(id))
						}
						embed.AddFields(
							discord.EmbedField{
								Name:  translate.Message(event.Locale(), "components.role.panel.use.changed.add"),
								Value: addRolesString,
							},
						)
					}
					if len(unchangedRole) > 0 {
						var unchangedRoleString string
						for _, id := range unchangedRole {
							unchangedRoleString += fmt.Sprintf("%s\n", discord.RoleMention(id))
						}
						embed.AddFields(
							discord.EmbedField{
								Name:  translate.Message(event.Locale(), "components.role.panel.use.changed.unchanged"),
								Value: unchangedRoleString,
							},
						)
					}
					if len(removedRoles) > 0 {
						var removedRolesString string
						for _, id := range removedRoles {
							removedRolesString += fmt.Sprintf("%s\n", discord.RoleMention(id))
						}
						embed.AddFields(
							discord.EmbedField{
								Name:  translate.Message(event.Locale(), "components.role.panel.use.changed.remove"),
								Value: removedRolesString,
							},
						)
					}
					if err := event.RespondMessage(
						discord.NewMessageBuilder().
							SetEmbeds(embeds.SetEmbedProperties(embed.Build())).
							SetFlags(discord.MessageFlagEphemeral),
					); err != nil {
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
				g, err := c.GuildCreateID(event, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				u, err := c.UserCreate(event, event.Message.Author)
				if err != nil {
					return errors.NewError(err)
				}

				edits := g.QueryRolePanelEdits().Where(rolepaneledit.ChannelID(event.ChannelID)).AllX(event)
				for _, edit := range edits {
					if edit.EmojiAuthor == nil || *edit.EmojiAuthor != event.Message.Author.ID || edit.Token == nil {
						continue
					}
					token := *edit.Token
					emojis := emoji.FindAllString(event.Message.Content)
					if len(emojis) < 1 {
						continue
					}
					componentEmoji := discordutil.ParseComponentEmoji(emojis[0])
					panel := edit.QueryParent().OnlyX(event)

					initialize(edit, panel)

					edit.Roles[slices.IndexFunc(edit.Roles, func(r schema.Role) bool { return r.ID == *edit.SelectedRole })].Emoji = &componentEmoji
					edit = edit.Update().
						ClearEmojiAuthor().
						ClearToken().
						SetRoles(edit.Roles).
						SaveX(event)

					if err := event.Client().Rest().AddReaction(event.ChannelID, event.MessageID, "✅"); err != nil {
						return errors.NewError(err)
					}

					if _, err := event.Client().Rest().UpdateInteractionResponse(event.Client().ApplicationID(), token,
						rpEditBaseMessage(event, panel, edit, u.Locale).
							SetFlags(discord.MessageFlagEphemeral).
							BuildUpdate(),
					); err != nil {
						return errors.NewError(err)
					}
				}
			case *events.GuildMessageDelete:
				g, err := c.GuildCreateID(event, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}

				c.DB().RolePanelPlaced.Delete().
					Where(
						rolepanelplaced.And(
							rolepanelplaced.HasGuildWith(guild.ID(g.ID)),
							rolepanelplaced.ChannelID(event.ChannelID),
							rolepanelplaced.MessageID(event.MessageID),
						),
					).
					ExecX(event)
			case *events.GuildMessageReactionAdd:
				if event.Member.User.Bot || event.Member.User.System {
					return nil
				}
				g, err := c.GuildCreateID(event, event.GuildID)
				if err != nil {
					return errors.NewError(err)
				}
				u, err := c.UserCreate(event, event.Member.User)
				if err != nil {
					return errors.NewError(err)
				}

				if !g.QueryRolePanelPlacements().Where(rolepanelplaced.ChannelID(event.ChannelID), rolepanelplaced.MessageID(event.MessageID)).ExistX(event) {
					return nil
				}
				place := g.QueryRolePanelPlacements().Where(rolepanelplaced.ChannelID(event.ChannelID), rolepanelplaced.MessageID(event.MessageID)).FirstX(event)
				panel := place.QueryRolePanel().OnlyX(event)

				if err := event.Client().Rest().RemoveUserReaction(event.ChannelID, event.MessageID, event.Emoji.Reaction(), event.UserID); err != nil {
					return errors.NewError(err)
				}

				for i, role := range panel.Roles {
					if role.Emoji == nil {
						role.Emoji = &discord.ComponentEmoji{
							Name: discordutil.Index2Emoji(i),
						}
					}
					if event.Emoji.Reaction() != discordutil.ReactionComponentEmoji(*role.Emoji) {
						continue
					}
					_, ok := event.Client().Caches().Role(event.GuildID, role.ID)
					if !ok {
						return nil
					}
					contains := slices.Contains(event.Member.RoleIDs, role.ID)
					if contains {
						err = event.Client().Rest().RemoveMemberRole(event.GuildID, event.UserID, role.ID)
					} else {
						err = event.Client().Rest().AddMemberRole(event.GuildID, event.UserID, role.ID)
					}
					if err != nil {
						m, err := event.Client().Rest().CreateMessage(event.ChannelID,
							discord.NewMessageBuilder().
								SetEmbeds(
									embeds.SetEmbedProperties(
										discord.NewEmbedBuilder().
											SetTitlef("❗ %s", translate.Message(u.Locale, "errors.fail.role.panel")).
											SetDescription(translate.Message(u.Locale, "errors.fail.role.panel.description")).
											SetColor(0xff2121).
											Build(),
									),
								).
								SetFlags(discord.MessageFlagEphemeral).BuildCreate(),
						)
						if err != nil {
							return errors.NewError(err)
						}
						if err := discordutil.DeleteMessageAfter(event.Client(), event.ChannelID, m.ID, time.Second*10); err != nil {
							return errors.NewError(err)
						}
						return nil
					}
					if place.HideNotice {
						return nil
					}
					m, err := event.Client().Rest().CreateMessage(event.ChannelID,
						discord.NewMessageBuilder().
							SetContent(discord.UserMention(event.UserID)).
							SetEmbeds(
								embeds.SetEmbedProperties(
									discord.NewEmbedBuilder().
										SetTitle(translate.Message(u.Locale, "components.role.panel.use."+builtin.Or(!contains, "added", "removed"))).
										SetDescription(translate.Message(u.Locale, "components.role.panel.use."+builtin.Or(!contains, "added", "removed")+".description", translate.WithTemplate(map[string]any{"Role": discord.RoleMention(role.ID)}))).
										Build(),
								),
							).
							SetFlags(discord.MessageFlagEphemeral).BuildCreate(),
					)
					if err != nil {
						return errors.NewError(err)
					}
					if err := discordutil.DeleteMessageAfter(event.Client(), event.ChannelID, m.ID, time.Second*10); err != nil {
						return errors.NewError(err)
					}
				}
			}
			return nil
		},
	}).SetComponent(c)
}

func UpdateRolePanel(ctx context.Context, place *ent.RolePanelPlaced, locale discord.Locale, client bot.Client) {
	if err := rolePanelPlace(ctx, place, locale, client, true); err != nil {
		slog.Error("アップデートに失敗", "err", err)
	}
}

func updateRolePanel(ctx context.Context, panel *ent.RolePanel, locale discord.Locale, client bot.Client, react bool) {
	places := panel.QueryPlacements().AllX(ctx)
	for _, place := range places {
		place = place.Update().
			SetName(panel.Name).
			SetDescription(panel.Description).
			SetRoles(panel.Roles).
			SetUpdatedAt(time.Now()).
			SaveX(ctx)
		if err := rolePanelPlace(ctx, place, locale, client, react); err != nil {
			slog.Error("アップデートに失敗", "err", err)
		}
	}
}
