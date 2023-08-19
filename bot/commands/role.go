package commands

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/emoji"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/handler/interactions"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Role(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "role",
			Description:  "require manage role permission",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "panel-v2",
					Description: "panel version 2",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "create",
							Description: "create a new role panel",
						},
						// TODO: Listコマンドを実装する
						// {
						// 	Name:        "list",
						// 	Description: "show list of role panels",
						// },
						{
							Name:        "delete",
							Description: "deletes a role panel",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:         "panel",
									Description:  "panel identify",
									Autocomplete: true,
									Required:     true,
								},
							},
						},
						{
							Name:        "edit",
							Description: "edit a role panel",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:         "panel",
									Description:  "panel identify",
									Autocomplete: true,
									Required:     true,
								},
							},
						},
						{
							Name:        "place",
							Description: "place panel to the channel",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:         "panel",
									Description:  "panel identify",
									Autocomplete: true,
									Required:     true,
								},
							},
						},
					},
				},
			},
		},
		Check:             b.Self.CheckCommandPermission(b, "guild.role.manage", discord.PermissionManageRoles),
		AutocompleteCheck: b.Self.CheckAutoCompletePermission(b, "guild.role.manage", discord.PermissionManageRoles),
		AutocompleteHandlers: map[string]handler.AutocompleteHandler{
			"panel-v2/edit":   rolePanelV2PanelAutoCompleteHandler(b),
			"panel-v2/delete": rolePanelV2PanelAutoCompleteHandler(b),
			"panel-v2/place":  rolePanelV2PanelAutoCompleteHandler(b),
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"panel-v2/create": rolePanelV2Create(b),
			"panel-v2/edit":   rolePanelV2Edit(b),
			"panel-v2/place":  rolePanelV2Place(b),
			"panel-v2/delete": rolePanelV2Delete(b),
		},
	}
}

func rolePanelV2PanelAutoCompleteHandler(b *botlib.Bot[*client.Client]) handler.AutocompleteHandler {
	return func(event *events.AutocompleteInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return err
		}
		choices := []discord.AutocompleteChoice{}
		for u, v := range gd.RolePanelV2 {
			if !strings.HasPrefix(v, event.Data.String("panel")) {
				continue
			}
			if gd.RolePanelV2Name[v] > 1 {
				v = fmt.Sprintf("%s(%s)", v, u.String())
			}
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  v,
				Value: u.String(),
			})
		}
		if err := event.Result(choices); err != nil {
			return err
		}
		return nil
	}
}

func rolePanelV2Create(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		modal := discord.NewModalCreateBuilder()
		role := event.SlashCommandInteractionData().Role("role")
		self, valid := event.Client().Caches().SelfMember(*event.GuildID())
		if !valid {
			return botlib.ReturnErrMessage(event, "error_bot_member_not_found", botlib.WithEphemeral(true))
		}
		role_map := map[snowflake.ID]discord.Role{}
		for _, id := range self.RoleIDs {
			role, ok := event.Client().Caches().Role(*event.GuildID(), id)
			if !ok {
				continue
			}
			role_map[id] = role
		}
		hi, _ := botlib.GetHighestRolePosition(role_map)
		if role.Managed || role.Position >= hi {
			return botlib.ReturnErrMessage(event, "error_invalid_command_argument")
		}
		modal.SetCustomID(fmt.Sprintf("handler:rp-v2:create-modal:%s", role.ID.String()))
		modal.SetTitle(translate.Message(event.Locale(), "rp_v2_create_modal_title"))
		modal.AddActionRow(discord.TextInputComponent{
			CustomID:  "name",
			Style:     discord.TextInputStyleShort,
			Label:     translate.Message(event.Locale(), "rp_v2_create_modal_label_0"),
			Value:     translate.Message(event.Locale(), "rp_v2_default_panel_name"),
			MaxLength: 32,
			Required:  true,
		})
		modal.AddActionRow(discord.TextInputComponent{
			CustomID:  "description",
			Style:     discord.TextInputStyleParagraph,
			Label:     translate.Message(event.Locale(), "rp_v2_create_modal_label_1"),
			MaxLength: 140,
			Required:  false,
		})
		if err := event.CreateModal(modal.Build()); err != nil {
			return err
		}
		return nil
	}
}

func rolePanelV2Edit(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		role_panel_id, err := uuid.Parse(event.SlashCommandInteractionData().String("panel"))
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}

		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		if _, ok := gd.RolePanelV2[role_panel_id]; !ok {
			return botlib.ReturnErrMessage(event, "error_unsearchable")
		}

		rp, err := b.Self.DB.RolePanelV2().Get(role_panel_id)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		edit := db.NewRolePanelV2Edit(rp.ID, *event.GuildID(), event.Channel().ID(), interactions.New(event.Token(), event.CreatedAt()))

		if id, ok := gd.RolePanelV2Editing[role_panel_id]; ok {
			old_edit, err := b.Self.DB.RolePanelV2Edit().Get(id)
			if err != nil {
				delete(gd.RolePanelV2Editing, role_panel_id)
				delete(gd.RolePanelV2EditingEmoji, role_panel_id)
			} else if old_edit.InteractionToken.IsValid() {
				token, _ := old_edit.InteractionToken.Get()
				message := discord.MessageUpdate{
					Content:    json.Ptr(translate.Message(event.Locale(), "rp_v2_expired_content")),
					Embeds:     &[]discord.Embed{},
					Components: &[]discord.ContainerComponent{},
				}
				_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message)
				if err := b.Self.DB.RolePanelV2Edit().Del(old_edit.ID); err != nil {
					return botlib.ReturnErr(event, err)
				}
			}
		}
		gd.RolePanelV2Editing[role_panel_id] = edit.ID

		mes := discord.NewMessageCreateBuilder()
		mes = db.RolePanelV2EditMenuEmbed(rp, event.Locale(), edit, mes)
		mes.SetFlags(discord.MessageFlagEphemeral)

		if err := b.Self.DB.RolePanelV2Edit().Set(edit.ID, edit); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}

		if err := event.CreateMessage(mes.Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func rolePanelV2Place(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		role_panel_id, err := uuid.Parse(event.SlashCommandInteractionData().String("panel"))
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}

		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if _, ok := gd.RolePanelV2[role_panel_id]; !ok {
			return botlib.ReturnErrMessage(event, "error_unsearchable", botlib.WithEphemeral(true))
		}

		rp, err := b.Self.DB.RolePanelV2().Get(role_panel_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		place := db.NewRolePanelV2Place(*event.GuildID(), rp.ID, interactions.New(event.Token(), event.CreatedAt()))
		if err := b.Self.DB.RolePanelV2Place().Set(place.ID, place); err != nil {
			return botlib.ReturnErr(event, err)
		}

		message := discord.NewMessageCreateBuilder()
		message = db.RolePanelV2PlaceMenuEmbed(rp, event.Locale(), place, message)
		message.SetFlags(discord.MessageFlagEphemeral)

		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2Delete(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		role_panel_id, err := uuid.Parse(event.SlashCommandInteractionData().String("panel"))
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}

		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		if _, ok := gd.RolePanelV2[role_panel_id]; !ok {
			return botlib.ReturnErrMessage(event, "error_unsearchable")
		}

		panel, err := b.Self.DB.RolePanelV2().Get(role_panel_id)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		edit_id, ok := gd.RolePanelV2Editing[panel.ID]
		if ok {
			delete(gd.RolePanelV2EditingEmoji, edit_id)
		}
		delete(gd.RolePanelV2Editing, panel.ID)
		gd.RolePanelV2Name[panel.Name]--
		delete(gd.RolePanelV2, panel.ID)

		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := b.Self.DB.RolePanelV2().Del(panel.ID); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		// パネルを更新する: ここから
		go func() {
			b.Self.GuildDataLock(*event.GuildID()).Lock()
			defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
			gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
			if err != nil {
				b.Logger.Errorf("error on delete role panel message: %s", err.Error())
				return
			}

			for k, u := range gd.RolePanelV2Placed {
				if u != panel.ID {
					continue
				}
				keys := strings.Split(k, "/")
				channel_id := snowflake.MustParse(keys[0])
				message_id := snowflake.MustParse(keys[1])

				delete(gd.RolePanelV2Placed, k)
				delete(gd.RolePanelV2PlacedConfig, k)

				if err := event.Client().Rest().DeleteMessage(channel_id, message_id); err != nil {
					b.Logger.Errorf("error on delete role panel message: %s", err.Error())
					return
				}
			}
		}()
		// パネルを更新する: ここまで

		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "rp_v2_delete_success_embed_title"))
		embed.SetDescription(translate.Message(event.Locale(), "rp_v2_delete_success_embed_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func RolePanelV2Component(b *botlib.Bot[*client.Client]) handler.Component {
	return handler.Component{
		Name: "rp-v2",
		Handler: map[string]handler.ComponentHandler{
			"edit-rsm":                 rolePanelV2EditRoleSelectMenuHandler(b),
			"edit_name":                rolePanelV2EditPanelInfoHandler(b),
			"edit_description":         rolePanelV2EditPanelInfoHandler(b),
			"edit_roles":               rolePanelV2EditRolesComponentHandler(b),
			"back_edit_roles":          rolePanelV2BackEditRolesComponentHandler(b),
			"edit_role_select":         rolePanelV2EditRoleSelectComponentHandler(b),
			"edit_role_name":           rolePanelV2EditRoleNameComponentHandler(b),
			"edit_role_delete":         rolePanelV2EditRoleDeleteComponent(b),
			"edit_role_emoji":          rolePanelV2EditRoleEmojiComponentHandler(b),
			"place_type":               rolePanelV2PlaceTypeComponentHandler(b),
			"place":                    rolePanelV2PlaceComponentHandler(b),
			"use_select_menu":          rolePanelV2UseSelectMenuComponentHandler(b),
			"use_button":               rolePanelV2UseButtonComponentHandler(b),
			"place_simple_select_menu": rolePanelV2PlaceSimpleSelectMenuComponentHandler(b),
			"place_button_show_name":   rolePanelV2PlaceButtonShowNameComponentHandler(b),
			"place_button_color":       rolePanelV2PlaceButtonColorComponentHandler(b),
			"call_select_menu":         rolePanelV2CallSelectMenuComponentHandler(b),
		},
	}
}

func rolePanelV2EditRoleSelectMenuHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}

		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		if len(event.StringSelectMenuInteractionData().Values) < 1 {
			edit.SelectedID = nil
		} else {
			selected_role, err := snowflake.Parse(event.StringSelectMenuInteractionData().Values[0])
			if err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
			edit.SelectedID = &selected_role
		}

		if err := b.Self.DB.RolePanelV2Edit().Set(edit.ID, edit); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		rp, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2EditMenuEmbed(rp, event.Locale(), edit, message)

		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2EditPanelInfoHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		modal := discord.NewModalCreateBuilder()
		switch args[2] {
		case "edit_name":
			modal.SetTitle(translate.Message(event.Locale(), "rp_v2_edit_embed_edit_name_modal_title"))
			modal.SetCustomID(event.Data.CustomID())
			modal.AddActionRow(
				discord.TextInputComponent{
					CustomID:  "value",
					Style:     discord.TextInputStyleShort,
					Label:     translate.Message(event.Locale(), "rp_v2_create_modal_label_0"),
					Value:     panel.Name,
					MaxLength: 32,
					Required:  true,
				},
			)
		case "edit_description":
			modal.SetTitle(translate.Message(event.Locale(), "rp_v2_edit_embed_edit_description_modal_title"))
			modal.SetCustomID(event.Data.CustomID())
			modal.AddActionRow(
				discord.TextInputComponent{
					CustomID:  "value",
					Style:     discord.TextInputStyleParagraph,
					Label:     translate.Message(event.Locale(), "rp_v2_create_modal_label_1"),
					Value:     panel.Description,
					MaxLength: 140,
					Required:  false,
				},
			)
		}
		if err := event.CreateModal(modal.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2EditRolesComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "rp_v2_edit_roles_embed_title"))
		embed.SetDescription(translate.Message(event.Locale(), "rp_v2_edit_roles_embed_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageUpdateBuilder()
		message.AddEmbeds(embed.Build())
		max_values := 20 - len(panel.Roles)
		disabled := false
		if max_values < 1 {
			max_values = 1
			disabled = true
		}
		message.AddContainerComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    translate.Message(event.Locale(), "rp_v2_edit_back_button"),
					CustomID: fmt.Sprintf("handler:rp-v2:back_edit_roles:%s", edit.ID.String()),
				},
			),
			discord.NewActionRow(
				discord.RoleSelectMenuComponent{
					CustomID:    fmt.Sprintf("handler:rp-v2:edit_role_select:%s", edit.ID.String()),
					Placeholder: translate.Message(event.Locale(), "rp_v2_edit_roles_select_menu_placeholder"),
					MaxValues:   max_values,
					MinValues:   json.Ptr(0),
					Disabled:    disabled,
				},
			),
		)

		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2BackEditRolesComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if edit.EmojiMode {
			gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
			if err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
			delete(gd.RolePanelV2EditingEmoji, panel.ID)
			edit.EmojiMode = false
			if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
		}

		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2EditMenuEmbed(panel, event.Locale(), edit, message)
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2EditRoleSelectComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		selected_role := event.RoleSelectMenuInteractionData().Resolved.Roles

		self, valid := event.Client().Caches().SelfMember(*event.GuildID())
		if !valid {
			return botlib.ReturnErrMessage(event, "error_bot_member_not_found", botlib.WithEphemeral(true))
		}
		role_map := map[snowflake.ID]discord.Role{}
		for _, id := range self.RoleIDs {
			role, ok := event.Client().Caches().Role(*event.GuildID(), id)
			if !ok {
				continue
			}
			role_map[id] = role
		}
		hi, _ := botlib.GetHighestRolePosition(role_map)
		deleted_role := []snowflake.ID{}
		for i, r := range selected_role {
			if r.Managed || r.Position >= hi {
				delete(selected_role, i)
				deleted_role = append(deleted_role, i)
				continue
			}
			var emoji *discord.ComponentEmoji
			if r.Emoji != nil {
				e := botlib.ParseComponentEmoji(*r.Emoji)
				emoji = &e
			}
			if !panel.AddRole(r.ID, r.Name, emoji) {
				delete(selected_role, i)
				deleted_role = append(deleted_role, i)
			}
		}

		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		message_update := discord.NewMessageUpdateBuilder()
		message_update = db.RolePanelV2EditMenuEmbed(panel, event.Locale(), edit, message_update)
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message_update.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if err := b.Self.DB.RolePanelV2().Set(panel.ID, panel); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		// パネルを更新する: ここから
		go func() {
			b.Logger.Debug("update called")
			b.Self.GuildDataLock(*event.GuildID()).Lock()
			defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
			gd, err := b.Self.DB.GuildData().Get(edit.GuildID)
			if err != nil {
				b.Logger.Errorf("error on update role panel message: %s", err.Error())
				return
			}

			for k, u := range gd.RolePanelV2Placed {
				if u != panel.ID {
					continue
				}
				keys := strings.Split(k, "/")
				channel_id := snowflake.MustParse(keys[0])
				message_id := snowflake.MustParse(keys[1])

				panel_config := gd.RolePanelV2PlacedConfig[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_config.PanelType {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update, panel_config)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update, panel_config)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_config.PanelType == db.RolePanelV2TypeReaction {
					if err := event.Client().Rest().RemoveAllReactions(channel_id, message_id); err != nil {
						b.Logger.Errorf("error on update role panel message: %s", err.Error())
						return
					}
					for _, role := range panel.Roles {
						if err = event.Client().Rest().AddReaction(channel_id, message_id, botlib.ReactionComponentEmoji(*role.Emoji)); err != nil {
							b.Logger.Errorf("error on update role panel message: %s", err.Error())
							return
						}
					}
				}
			}
		}()
		// パネルを更新する: ここまで

		if len(deleted_role) < 1 {
			if err := event.DeferUpdateMessage(); err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
			return nil
		}

		var deleted_role_string string
		for _, id := range deleted_role {
			deleted_role_string += fmt.Sprintf("- %s\r", discord.RoleMention(id))
		}

		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "rp_v2_edit_role_add_select_deleted_embed_title"))
		embed.SetDescriptionf("%s\r%s",
			translate.Message(event.Locale(), "rp_v2_edit_role_add_select_deleted_embed_description"),
			deleted_role_string,
		)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		message.SetFlags(discord.MessageFlagEphemeral)
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		return nil
	}
}

func rolePanelV2EditRoleNameComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if edit.SelectedID == nil {
			return botlib.ReturnErrMessage(event, "error_unavailable", botlib.WithEphemeral(true))
		}
		role_index := slices.IndexFunc(panel.Roles, func(rpvr db.RolePanelV2Role) bool {
			return rpvr.RoleID == *edit.SelectedID
		})
		if role_index == -1 {
			return botlib.ReturnErrMessage(event, "error_unavailable", botlib.WithEphemeral(true))
		}

		modal := discord.NewModalCreateBuilder()
		modal.SetTitle(translate.Message(event.Locale(), "rp_v2_edit_role_name_modal_title"))
		modal.SetCustomID(fmt.Sprintf("handler:rp-v2:edit_role_name:%s", edit.ID))
		modal.AddActionRow(
			discord.TextInputComponent{
				CustomID:  "name",
				Style:     discord.TextInputStyleShort,
				Label:     translate.Message(event.Locale(), "rp_v2_edit_role_name_modal_label"),
				MaxLength: 32,
				Value:     panel.Roles[role_index].RoleName,
				Required:  true,
			},
		)

		if err := event.CreateModal(modal.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2EditRoleDeleteComponent(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if edit.SelectedID == nil {
			return botlib.ReturnErrMessage(event, "error_unavailable", botlib.WithEphemeral(true))
		}
		role_index := slices.IndexFunc(panel.Roles, func(rpvr db.RolePanelV2Role) bool {
			return rpvr.RoleID == *edit.SelectedID
		})
		if role_index == -1 {
			return botlib.ReturnErrMessage(event, "error_unavailable", botlib.WithEphemeral(true))
		}

		panel.Roles = slices.Delete(panel.Roles, role_index, role_index+1)
		edit.SelectedID = nil

		if err := b.Self.DB.RolePanelV2().Set(panel.ID, panel); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := b.Self.DB.RolePanelV2Edit().Set(edit.ID, edit); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		// パネルを更新する: ここから
		go func() {
			b.Self.GuildDataLock(*event.GuildID()).Lock()
			defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
			gd, err := b.Self.DB.GuildData().Get(edit.GuildID)
			if err != nil {
				b.Logger.Errorf("error on update role panel message: %s", err.Error())
				return
			}

			for k, u := range gd.RolePanelV2Placed {
				if u != panel.ID {
					continue
				}
				keys := strings.Split(k, "/")
				channel_id := snowflake.MustParse(keys[0])
				message_id := snowflake.MustParse(keys[1])

				panel_config := gd.RolePanelV2PlacedConfig[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_config.PanelType {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update, panel_config)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update, panel_config)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_config.PanelType == db.RolePanelV2TypeReaction {
					if err := event.Client().Rest().RemoveAllReactions(channel_id, message_id); err != nil {
						b.Logger.Errorf("error on update role panel message: %s", err.Error())
						return
					}
					for _, role := range panel.Roles {
						if err = event.Client().Rest().AddReaction(channel_id, message_id, botlib.ReactionComponentEmoji(*role.Emoji)); err != nil {
							b.Logger.Errorf("error on update role panel message: %s", err.Error())
							return
						}
					}
				}
			}
		}()
		// パネルを更新する: ここまで

		message := db.RolePanelV2EditMenuEmbed(panel, event.Locale(), edit, discord.NewMessageUpdateBuilder())
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		return nil
	}
}

func rolePanelV2EditRoleEmojiComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		gd.RolePanelV2EditingEmoji[panel.ID] = [2]snowflake.ID{
			event.Channel().ID(),
			event.User().ID,
		}
		edit.EmojiMode = true
		edit.EmojiLocale = event.Locale()

		if err := b.Self.DB.RolePanelV2Edit().Set(edit.ID, edit); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "rp_v2_edit_role_emoji_embed_title"))
		embed.SetDescription(translate.Message(event.Locale(), "rp_v2_edit_role_emoji_embed_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)

		message := discord.NewMessageUpdateBuilder()
		message.AddEmbeds(embed.Build())

		message.AddContainerComponents(
			discord.NewActionRow(
				discord.ButtonComponent{
					Style:    discord.ButtonStyleSecondary,
					Label:    translate.Message(event.Locale(), "rp_v2_edit_back_button"),
					CustomID: fmt.Sprintf("handler:rp-v2:back_edit_roles:%s", edit.ID.String()),
				},
			),
		)

		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		return nil
	}
}

func rolePanelV2PlaceTypeComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		args := strings.Split(event.Data.CustomID(), ":")
		place_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		place, err := b.Self.DB.RolePanelV2Place().Get(place_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(place.PanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		place.Config.PanelType = db.RolePanelV2Type(event.StringSelectMenuInteractionData().Values[0])

		if err := b.Self.DB.RolePanelV2Place().Set(place.ID, place); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2PlaceMenuEmbed(panel, event.Locale(), place, message)

		token, err := place.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		return nil
	}
}

func rolePanelV2PlaceSimpleSelectMenuComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		args := strings.Split(event.Data.CustomID(), ":")
		place_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		place, err := b.Self.DB.RolePanelV2Place().Get(place_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(place.PanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		place.Config.SimpleSelectMenu = !place.Config.SimpleSelectMenu

		if err := b.Self.DB.RolePanelV2Place().Set(place.ID, place); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2PlaceMenuEmbed(panel, event.Locale(), place, message)

		token, err := place.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		return nil
	}
}

func rolePanelV2PlaceButtonShowNameComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		args := strings.Split(event.Data.CustomID(), ":")
		place_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		place, err := b.Self.DB.RolePanelV2Place().Get(place_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(place.PanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		place.Config.ButtonShowName = !place.Config.ButtonShowName

		if err := b.Self.DB.RolePanelV2Place().Set(place.ID, place); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2PlaceMenuEmbed(panel, event.Locale(), place, message)

		token, err := place.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		return nil
	}
}

func rolePanelV2PlaceButtonColorComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		args := strings.Split(event.Data.CustomID(), ":")
		place_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		place, err := b.Self.DB.RolePanelV2Place().Get(place_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(place.PanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		value := event.StringSelectMenuInteractionData().Values[0]

		switch value {
		case "red":
			place.Config.ButtonStyle = discord.ButtonStyleDanger
		case "green":
			place.Config.ButtonStyle = discord.ButtonStyleSuccess
		case "blue":
			place.Config.ButtonStyle = discord.ButtonStylePrimary
		case "gray":
			place.Config.ButtonStyle = discord.ButtonStyleSecondary
		}

		if err := b.Self.DB.RolePanelV2Place().Set(place.ID, place); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2PlaceMenuEmbed(panel, event.Locale(), place, message)

		token, err := place.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		return nil
	}
}

func rolePanelV2PlaceComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		args := strings.Split(event.Data.CustomID(), ":")
		place_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		place, err := b.Self.DB.RolePanelV2Place().Get(place_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(place.PanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if len(panel.Roles) < 1 {
			return botlib.ReturnErrMessage(event, "error_has_no_data", botlib.WithEphemeral(true))
		}

		token, err := place.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		if err := event.Client().Rest().DeleteInteractionResponse(event.ApplicationID(), token); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		message_create := discord.NewMessageCreateBuilder()
		switch place.Config.PanelType {
		case db.RolePanelV2TypeReaction:
			message_create = db.RolePanelV2MessageReaction(panel, event.Locale(), message_create)
		case db.RolePanelV2TypeSelectMenu:
			message_create = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_create, place.Config)
		case db.RolePanelV2TypeButton:
			message_create = db.RolePanelV2MessageButton(panel, event.Locale(), message_create, place.Config)
		}
		message, err := event.Client().Rest().CreateMessage(event.Channel().ID(), message_create.Build())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		key := fmt.Sprintf("%d/%d", event.Channel().ID(), message.ID)
		gd.RolePanelV2Placed[key] = panel.ID
		gd.RolePanelV2PlacedConfig[key] = place.Config
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if place.Config.PanelType == db.RolePanelV2TypeReaction {
			for _, role := range panel.Roles {
				if err = event.Client().Rest().AddReaction(event.Channel().ID(), message.ID, botlib.ReactionComponentEmoji(*role.Emoji)); err != nil {
					return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
				}
			}
		}

		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2UseSelectMenuComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		key := fmt.Sprintf("%s/%s", event.Message.ChannelID, event.Message.ID)
		panel_id, ok := gd.RolePanelV2Placed[key]
		if !ok && !event.Message.Flags.Has(discord.MessageFlagEphemeral) {
			return botlib.ReturnErrMessage(event, "error_not_found", botlib.WithEphemeral(true))
		}
		if event.Message.Flags.Has(discord.MessageFlagEphemeral) {
			panel_id = uuid.MustParse(strings.Split(event.Data.CustomID(), ":")[3])
		}
		panel, err := b.Self.DB.RolePanelV2().Get(panel_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		selected_roles := []snowflake.ID{}
		for _, v := range event.StringSelectMenuInteractionData().Values {
			selected_roles = append(selected_roles, snowflake.MustParse(v))
		}

		add_roles := []snowflake.ID{}
		removed_roles := []snowflake.ID{}
		unchanged_role := []snowflake.ID{}
		for _, role := range panel.Roles {
			if slices.Index(selected_roles, role.RoleID) != -1 {
				// 選ばれたとき
				if slices.Index(event.Member().RoleIDs, role.RoleID) != -1 {
					// 持ってたなら
					unchanged_role = append(unchanged_role, role.RoleID)
					continue
				} else {
					// 持ってないなら
					add_roles = append(add_roles, role.RoleID)

					if err := event.Client().Rest().AddMemberRole(*event.GuildID(), event.User().ID, role.RoleID); err != nil {
						embed := discord.NewEmbedBuilder().
							SetTitle(translate.Message(event.Locale(), "rp_v2_add_role_failed_embed_title")).
							SetDescription(translate.Message(event.Locale(), "rp_v2_add_role_failed_embed_description"))
						embed.Embed = botlib.SetEmbedProperties(embed.Embed)
						if err := event.CreateMessage(discord.MessageCreate{
							Embeds: []discord.Embed{
								embed.Build(),
							},
						}); err != nil {
							return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
						}
						return err
					}
				}
			} else {
				// 選ばれてないとき
				if slices.Index(event.Member().RoleIDs, role.RoleID) != -1 {
					// 持ってたなら
					removed_roles = append(removed_roles, role.RoleID)

					if err := event.Client().Rest().RemoveMemberRole(*event.GuildID(), event.User().ID, role.RoleID); err != nil {
						embed := discord.NewEmbedBuilder().
							SetTitle(translate.Message(event.Locale(), "rp_v2_remove_role_failed_embed_title")).
							SetDescription(translate.Message(event.Locale(), "rp_v2_remove_role_failed_embed_description"))
						embed.Embed = botlib.SetEmbedProperties(embed.Embed)
						if err := event.CreateMessage(discord.MessageCreate{
							Embeds: []discord.Embed{
								embed.Build(),
							},
						}); err != nil {
							return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
						}
						return err
					}
				} else {
					// 持ってないなら
					continue
				}
			}
		}

		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "rp_v2_select_menu_used"))
		if len(add_roles) > 0 {
			var add_roles_string string
			for _, id := range add_roles {
				add_roles_string += fmt.Sprintf("%s\r", discord.RoleMention(id))
			}
			embed.AddFields(
				discord.EmbedField{
					Name:  translate.Message(event.Locale(), "rp_v2_add_role"),
					Value: add_roles_string,
				},
			)
		}
		if len(unchanged_role) > 0 {
			var unchanged_role_string string
			for _, id := range unchanged_role {
				unchanged_role_string += fmt.Sprintf("%s\r", discord.RoleMention(id))
			}
			embed.AddFields(
				discord.EmbedField{
					Name:  translate.Message(event.Locale(), "rp_v2_unchanged_role"),
					Value: unchanged_role_string,
				},
			)
		}
		if len(removed_roles) > 0 {
			var removed_roles_string string
			for _, id := range removed_roles {
				removed_roles_string += fmt.Sprintf("%s\r", discord.RoleMention(id))
			}
			embed.AddFields(
				discord.EmbedField{
					Name:  translate.Message(event.Locale(), "rp_v2_removed_role"),
					Value: removed_roles_string,
				},
			)
		}
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		message.SetFlags(discord.MessageFlagEphemeral)
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2UseButtonComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()

		args := strings.Split(event.ButtonInteractionData().CustomID(), ":")

		role_id, err := snowflake.Parse(args[4])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}

		if slices.Index(event.Member().RoleIDs, role_id) == -1 {
			if err := event.Client().Rest().AddMemberRole(*event.GuildID(), event.User().ID, role_id); err != nil {
				embed := discord.NewEmbedBuilder().
					SetTitle(translate.Message(event.Locale(), "rp_v2_add_role_failed_embed_title")).
					SetDescription(translate.Message(event.Locale(), "rp_v2_add_role_failed_embed_description"))
				embed.Embed = botlib.SetEmbedProperties(embed.Embed)
				if err := event.CreateMessage(discord.MessageCreate{
					Embeds: []discord.Embed{
						embed.Build(),
					},
					Flags: discord.MessageFlagEphemeral,
				}); err != nil {
					return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
				}
				return err
			}
			embed := discord.NewEmbedBuilder().
				SetTitle(translate.Message(event.Locale(), "rp_v2_add_role_added_embed_title")).
				SetDescription(translate.Translate(event.Locale(), "rp_v2_add_role_added_embed_description", map[string]any{"Role": discord.RoleMention(role_id)}))
			embed.Embed = botlib.SetEmbedProperties(embed.Embed)
			if err := event.CreateMessage(discord.MessageCreate{
				Embeds: []discord.Embed{
					embed.Build(),
				},
				Flags: discord.MessageFlagEphemeral,
			}); err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
		} else {
			if err := event.Client().Rest().RemoveMemberRole(*event.GuildID(), event.User().ID, role_id); err != nil {
				embed := discord.NewEmbedBuilder().
					SetTitle(translate.Message(event.Locale(), "rp_v2_remove_role_failed_embed_title")).
					SetDescription(translate.Message(event.Locale(), "rp_v2_remove_role_failed_embed_description"))
				embed.Embed = botlib.SetEmbedProperties(embed.Embed)
				if err := event.CreateMessage(discord.MessageCreate{
					Embeds: []discord.Embed{
						embed.Build(),
					},
					Flags: discord.MessageFlagEphemeral,
				}); err != nil {
					return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
				}
				return err
			}
			embed := discord.NewEmbedBuilder().
				SetTitle(translate.Message(event.Locale(), "rp_v2_add_role_removed_embed_title")).
				SetDescription(translate.Translate(event.Locale(), "rp_v2_add_role_removed_embed_description", map[string]any{"Role": discord.RoleMention(role_id)}))
			embed.Embed = botlib.SetEmbedProperties(embed.Embed)
			if err := event.CreateMessage(discord.MessageCreate{
				Embeds: []discord.Embed{
					embed.Build(),
				},
				Flags: discord.MessageFlagEphemeral,
			}); err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
		}
		return nil
	}
}

func rolePanelV2CallSelectMenuComponentHandler(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		key := fmt.Sprintf("%s/%s", event.Message.ChannelID, event.Message.ID)
		panel_id, ok := gd.RolePanelV2Placed[key]
		if !ok {
			return botlib.ReturnErrMessage(event, "error_not_found", botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(panel_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "rp_v2_call_select_menu_embed_title"))
		embed.SetDescription(translate.Message(event.Locale(), "rp_v2_call_select_menu_embed_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		options := make([]discord.StringSelectMenuOption, len(panel.Roles))
		for i, rpvr := range panel.Roles {
			options[i] = discord.StringSelectMenuOption{
				Label:   rpvr.RoleName,
				Value:   rpvr.RoleID.String(),
				Emoji:   rpvr.Emoji,
				Default: slices.Contains(event.Member().RoleIDs, rpvr.RoleID),
			}
		}
		message.AddContainerComponents(
			discord.NewActionRow(
				discord.StringSelectMenuComponent{
					CustomID:    fmt.Sprintf("handler:rp-v2:use_select_menu:%s", panel.ID.String()),
					Placeholder: translate.Message(event.Locale(), "rp_v2_select_menu_placeholder"),
					MinValues:   json.Ptr(0),
					MaxValues:   len(panel.Roles),
					Options:     options,
				},
			),
		)
		message.SetFlags(discord.MessageFlagEphemeral)

		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func RolePanelV2Modal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "rp-v2",
		Handler: map[string]handler.ModalHandler{
			"create-modal":     rolePanelV2CreateModalHandler(b),
			"edit_name":        rolePanelV2EditPanelInfoModalHandler(b),
			"edit_description": rolePanelV2EditPanelInfoModalHandler(b),
			"edit_role_name":   rolePanelV2EditRoleNameModalHandler(b),
		},
	}
}

func rolePanelV2CreateModalHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		name := event.ModalSubmitInteraction.Data.Text("name")
		description := event.ModalSubmitInteraction.Data.Text("description")
		rp := db.NewRolePanelV2(name, description)
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		gd.RolePanelV2[rp.ID] = rp.Name
		gd.RolePanelV2Name[rp.Name]++
		if err := b.Self.DB.RolePanelV2().Set(rp.ID, rp); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		edit := db.NewRolePanelV2Edit(rp.ID, *event.GuildID(), event.Channel().ID(), interactions.New(event.Token(), event.CreatedAt()))
		if err := b.Self.DB.RolePanelV2Edit().Set(edit.ID, edit); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		gd.RolePanelV2Editing[rp.ID] = edit.ID

		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		message := discord.NewMessageCreateBuilder()
		message = db.RolePanelV2EditMenuEmbed(rp, event.Locale(), edit, message)
		message.SetFlags(discord.MessageFlagEphemeral)

		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2EditPanelInfoModalHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID, ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		value := event.ModalSubmitInteraction.Data.Text("value")
		switch args[2] {
		case "edit_name":
			gd, err := b.Self.DB.GuildData().Get(edit.GuildID)
			if err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}

			gd.RolePanelV2Name[panel.Name]--
			gd.RolePanelV2Name[value]++
			gd.RolePanelV2[panel.ID] = value

			if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}

			panel.Name = value
		case "edit_description":
			panel.Description = value
		}
		if err := b.Self.DB.RolePanelV2().Set(panel.ID, panel); err != nil {
			return nil
		}
		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2EditMenuEmbed(panel, event.Locale(), edit, message)
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		// パネルを更新する: ここから
		go func() {
			b.Self.GuildDataLock(*event.GuildID()).Lock()
			defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
			gd, err := b.Self.DB.GuildData().Get(edit.GuildID)
			if err != nil {
				b.Logger.Errorf("error on update role panel message: %s", err.Error())
				return
			}

			for k, u := range gd.RolePanelV2Placed {
				if u != panel.ID {
					continue
				}
				keys := strings.Split(k, "/")
				channel_id := snowflake.MustParse(keys[0])
				message_id := snowflake.MustParse(keys[1])

				panel_config := gd.RolePanelV2PlacedConfig[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_config.PanelType {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update, panel_config)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update, panel_config)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_config.PanelType == db.RolePanelV2TypeReaction {
					if err := event.Client().Rest().RemoveAllReactions(channel_id, message_id); err != nil {
						b.Logger.Errorf("error on update role panel message: %s", err.Error())
						return
					}
					for _, role := range panel.Roles {
						if err = event.Client().Rest().AddReaction(channel_id, message_id, botlib.ReactionComponentEmoji(*role.Emoji)); err != nil {
							b.Logger.Errorf("error on update role panel message: %s", err.Error())
							return
						}
					}
				}
			}
		}()
		// パネルを更新する: ここまで

		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func rolePanelV2EditRoleNameModalHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		args := strings.Split(event.Data.CustomID, ":")
		edit_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id", botlib.WithEphemeral(true))
		}
		edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if edit.SelectedID == nil {
			return botlib.ReturnErrMessage(event, "error_unavailable", botlib.WithEphemeral(true))
		}
		role_index := slices.IndexFunc(panel.Roles, func(rpvr db.RolePanelV2Role) bool {
			return rpvr.RoleID == *edit.SelectedID
		})
		if role_index == -1 {
			return botlib.ReturnErrMessage(event, "error_unavailable", botlib.WithEphemeral(true))
		}

		role_name := event.ModalSubmitInteraction.Data.Text("name")
		panel.Roles[role_index].RoleName = role_name

		if err := b.Self.DB.RolePanelV2().Set(panel.ID, panel); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		token, err := edit.InteractionToken.Get()
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout", botlib.WithEphemeral(true))
		}
		message := discord.NewMessageUpdateBuilder()
		message = db.RolePanelV2EditMenuEmbed(panel, event.Locale(), edit, message)
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, message.Build()); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		// パネルを更新する: ここから
		go func() {
			b.Self.GuildDataLock(*event.GuildID()).Lock()
			defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
			gd, err := b.Self.DB.GuildData().Get(edit.GuildID)
			if err != nil {
				b.Logger.Errorf("error on update role panel message: %s", err.Error())
				return
			}

			for k, u := range gd.RolePanelV2Placed {
				if u != panel.ID {
					continue
				}
				keys := strings.Split(k, "/")
				channel_id := snowflake.MustParse(keys[0])
				message_id := snowflake.MustParse(keys[1])

				panel_config := gd.RolePanelV2PlacedConfig[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_config.PanelType {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update, panel_config)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update, panel_config)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_config.PanelType == db.RolePanelV2TypeReaction {
					if err := event.Client().Rest().RemoveAllReactions(channel_id, message_id); err != nil {
						b.Logger.Errorf("error on update role panel message: %s", err.Error())
						return
					}
					for _, role := range panel.Roles {
						if err = event.Client().Rest().AddReaction(channel_id, message_id, botlib.ReactionComponentEmoji(*role.Emoji)); err != nil {
							b.Logger.Errorf("error on update role panel message: %s", err.Error())
							return
						}
					}
				}
			}
		}()
		// パネルを更新する: ここまで

		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		return nil
	}
}

func RolePanelV2Message(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		Handler: rolePanelV2MessageCreate(b),
	}
}

func rolePanelV2MessageCreate(b *botlib.Bot[*client.Client]) handler.MessageHandler {
	return func(event *events.GuildMessageCreate) error {
		if event.Message.Type != discord.MessageTypeDefault {
			return nil
		}
		b.Self.GuildDataLock(event.GuildID).Lock()
		defer b.Self.GuildDataLock(event.GuildID).Unlock()

		gd, err := b.Self.DB.GuildData().Get(event.GuildID)
		if err != nil {
			return err
		}
		for u, v := range gd.RolePanelV2EditingEmoji {
			if v[0] != event.ChannelID || v[1] != event.Message.Author.ID {
				continue
			}

			if !emoji.MatchString(event.Message.Content) {
				continue
			}
			raw_emoji := emoji.FindAllString(event.Message.Content)
			if len(raw_emoji) < 1 {
				continue
			}
			emoji := botlib.ParseComponentEmoji(raw_emoji[0])

			edit_id, ok := gd.RolePanelV2Editing[u]
			if !ok {
				return nil
			}
			edit, err := b.Self.DB.RolePanelV2Edit().Get(edit_id)
			if err != nil {
				return nil
			}
			panel, err := b.Self.DB.RolePanelV2().Get(edit.RolePanelID)
			if err != nil {
				return nil
			}

			if edit.SelectedID == nil {
				return nil
			}
			role_index := slices.IndexFunc(panel.Roles, func(rpvr db.RolePanelV2Role) bool {
				return rpvr.RoleID == *edit.SelectedID
			})
			if role_index == -1 {
				return nil
			}

			panel.Roles[role_index].Emoji = &emoji
			edit.EmojiMode = false
			delete(gd.RolePanelV2EditingEmoji, panel.ID)

			if err := b.Self.DB.RolePanelV2().Set(panel.ID, panel); err != nil {
				return err
			}
			if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
				return err
			}
			if err := b.Self.DB.RolePanelV2Edit().Set(edit.ID, edit); err != nil {
				return err
			}

			token, err := edit.InteractionToken.Get()
			if err != nil {
				return nil
			}
			message := discord.NewMessageUpdateBuilder()
			message = db.RolePanelV2EditMenuEmbed(panel, edit.EmojiLocale, edit, message)
			if _, err := event.Client().Rest().UpdateInteractionResponse(event.Client().ApplicationID(), token, message.Build()); err != nil {
				return err
			}

			// パネルを更新する: ここから
			go func() {
				b.Self.GuildDataLock(event.GuildID).Lock()
				defer b.Self.GuildDataLock(event.GuildID).Unlock()
				gd, err := b.Self.DB.GuildData().Get(edit.GuildID)
				if err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				for k, u := range gd.RolePanelV2Placed {
					if u != panel.ID {
						continue
					}
					keys := strings.Split(k, "/")
					channel_id := snowflake.MustParse(keys[0])
					message_id := snowflake.MustParse(keys[1])

					panel_config := gd.RolePanelV2PlacedConfig[k]

					message_update := discord.NewMessageUpdateBuilder()
					switch panel_config.PanelType {
					case db.RolePanelV2TypeReaction:
						message_update = db.RolePanelV2MessageReaction(panel, edit.EmojiLocale, message_update)
					case db.RolePanelV2TypeSelectMenu:
						message_update = db.RolePanelV2MessageSelectMenu(panel, edit.EmojiLocale, message_update, panel_config)
					case db.RolePanelV2TypeButton:
						message_update = db.RolePanelV2MessageButton(panel, edit.EmojiLocale, message_update, panel_config)
					}
					if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
						b.Logger.Errorf("error on update role panel message: %s", err.Error())
						return
					}

					if panel_config.PanelType == db.RolePanelV2TypeReaction {
						if err := event.Client().Rest().RemoveAllReactions(channel_id, message_id); err != nil {
							b.Logger.Errorf("error on update role panel message: %s", err.Error())
							return
						}
						for _, role := range panel.Roles {
							if err = event.Client().Rest().AddReaction(channel_id, message_id, botlib.ReactionComponentEmoji(*role.Emoji)); err != nil {
								b.Logger.Errorf("error on update role panel message: %s", err.Error())
								return
							}
						}
					}
				}
			}()
			// パネルを更新する: ここまで

			if err := event.Client().Rest().AddReaction(event.ChannelID, event.MessageID, "✅"); err != nil {
				return err
			}

			return nil
		}
		return nil
	}
}

func RolePanelV2MessageDelete(b *botlib.Bot[*client.Client]) handler.MessageDelete {
	return handler.MessageDelete{
		Handler: func(event *events.GuildMessageDelete) error {
			b.Self.GuildDataLock(event.GuildID).Lock()
			defer b.Self.GuildDataLock(event.GuildID).Unlock()

			gd, err := b.Self.DB.GuildData().Get(event.GuildID)
			if err != nil {
				return err
			}
			key := fmt.Sprintf("%s/%s", event.ChannelID, event.MessageID)
			if _, ok := gd.RolePanelV2Placed[key]; ok {
				delete(gd.RolePanelV2PlacedConfig, key)
				delete(gd.RolePanelV2Placed, key)
				if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
					b.Logger.Errorf("error on delete role panel message handling: %s", err.Error())
					return err
				}
			}
			return nil
		},
	}
}

func RolePanelV2MessageReaction(b *botlib.Bot[*client.Client]) handler.Generics[events.GuildMessageReactionAdd] {
	return handler.Generics[events.GuildMessageReactionAdd]{
		Handler: func(event *events.GuildMessageReactionAdd) error {
			if event.Member.User.Bot || event.Member.User.System {
				return nil
			}
			b.Logger.Debug("reaction added")
			b.Self.GuildDataLock(event.GuildID).Lock()
			defer b.Self.GuildDataLock(event.GuildID).Unlock()

			gd, err := b.Self.DB.GuildData().Get(event.GuildID)
			if err != nil {
				return err
			}
			key := fmt.Sprintf("%s/%s", event.ChannelID, event.MessageID)
			if panel_id, ok := gd.RolePanelV2Placed[key]; ok {
				panel, err := b.Self.DB.RolePanelV2().Get(panel_id)
				if err != nil {
					b.Logger.Errorf("error on handling add reaction role panel: %s", err.Error())
					return err
				}
				user, err := b.Self.DB.UserData().Get(event.UserID)
				if err != nil {
					return err
				}
				_ = event.Client().Rest().RemoveUserReaction(event.ChannelID, event.MessageID, event.Emoji.Reaction(), event.UserID)
				for _, rpvr := range panel.Roles {
					if event.Emoji.Reaction() != botlib.ReactionComponentEmoji(*rpvr.Emoji) {
						continue
					}
					if slices.Index(event.Member.RoleIDs, rpvr.RoleID) == -1 {
						if err := event.Client().Rest().AddMemberRole(event.GuildID, event.UserID, rpvr.RoleID); err != nil {
							embed := discord.NewEmbedBuilder().
								SetTitle(translate.Message(user.Locale, "rp_v2_add_role_failed_embed_title")).
								SetDescription(translate.Message(user.Locale, "rp_v2_add_role_failed_embed_description"))
							embed.Embed = botlib.SetEmbedProperties(embed.Embed)
							message, err := event.Client().Rest().CreateMessage(event.ChannelID, discord.MessageCreate{
								Content: discord.UserMention(event.UserID),
								Embeds: []discord.Embed{
									embed.Build(),
								},
							})
							if err == nil {
								go RolePanelV2DeferDeleteMessage(b, event.ChannelID, message.ID)
							}
							return err
						}
						embed := discord.NewEmbedBuilder().
							SetTitle(translate.Message(user.Locale, "rp_v2_add_role_added_embed_title")).
							SetDescription(translate.Translate(user.Locale, "rp_v2_add_role_added_embed_description", map[string]any{"Role": discord.RoleMention(rpvr.RoleID)}))
						embed.Embed = botlib.SetEmbedProperties(embed.Embed)
						message, err := event.Client().Rest().CreateMessage(event.ChannelID, discord.MessageCreate{
							Content: discord.UserMention(event.UserID),
							Embeds: []discord.Embed{
								embed.Build(),
							},
						})
						if err == nil {
							go RolePanelV2DeferDeleteMessage(b, event.ChannelID, message.ID)
						}
					} else {
						if err := event.Client().Rest().RemoveMemberRole(event.GuildID, event.UserID, rpvr.RoleID); err != nil {
							embed := discord.NewEmbedBuilder().
								SetTitle(translate.Message(user.Locale, "rp_v2_remove_role_failed_embed_title")).
								SetDescription(translate.Message(user.Locale, "rp_v2_remove_role_failed_embed_description"))
							embed.Embed = botlib.SetEmbedProperties(embed.Embed)
							message, err := event.Client().Rest().CreateMessage(event.ChannelID, discord.MessageCreate{
								Content: discord.UserMention(event.UserID),
								Embeds: []discord.Embed{
									embed.Build(),
								},
							})
							if err == nil {
								go RolePanelV2DeferDeleteMessage(b, event.ChannelID, message.ID)
							}
							return err
						}
						embed := discord.NewEmbedBuilder().
							SetTitle(translate.Message(user.Locale, "rp_v2_add_role_removed_embed_title")).
							SetDescription(translate.Translate(user.Locale, "rp_v2_add_role_removed_embed_description", map[string]any{"Role": discord.RoleMention(rpvr.RoleID)}))
						embed.Embed = botlib.SetEmbedProperties(embed.Embed)
						message, err := event.Client().Rest().CreateMessage(event.ChannelID, discord.MessageCreate{
							Content: discord.UserMention(event.UserID),
							Embeds: []discord.Embed{
								embed.Build(),
							},
						})
						if err == nil {
							go RolePanelV2DeferDeleteMessage(b, event.ChannelID, message.ID)
						}
					}
				}
			}
			return nil
		},
	}
}

func RolePanelV2DeferDeleteMessage(b *botlib.Bot[*client.Client], channel_id, message_id snowflake.ID) {
	time.Sleep(time.Second * 5)
	if err := b.Client.Rest().DeleteMessage(channel_id, message_id); err != nil {
		b.Logger.Errorf("error on role panel defer delete message: %s", err.Error())
		return
	}
}
