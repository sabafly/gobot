package commands

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/bot"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/disgo/rest"
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
			"panel/create":    rolePanelCreateHandler(b),
			"panel/list":      rolePanelListHandler(b),
			"panel/delete":    rolePanelDeleteHandler(b),
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
				delete(gd.RolePanelV2PlacedType, k)

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
			"edit-rsm":         rolePanelV2EditRoleSelectMenuHandler(b),
			"edit_name":        rolePanelV2EditPanelInfoHandler(b),
			"edit_description": rolePanelV2EditPanelInfoHandler(b),
			"edit_roles":       rolePanelV2EditRolesComponentHandler(b),
			"back_edit_roles":  rolePanelV2BackEditRolesComponentHandler(b),
			"edit_role_select": rolePanelV2EditRoleSelectComponentHandler(b),
			"edit_role_name":   rolePanelV2EditRoleNameComponentHandler(b),
			"edit_role_delete": rolePanelV2EditRoleDeleteComponent(b),
			"edit_role_emoji":  rolePanelV2EditRoleEmojiComponentHandler(b),
			"place_type":       rolePanelV2PlaceTypeComponentHandler(b),
			"place":            rolePanelV2PlaceComponentHandler(b),
			"use_select_menu":  rolePanelV2UseSelectMenuComponentHandler(b),
			"use_button":       rolePanelV2UseButtonComponentHandler(b),
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

				panel_type := gd.RolePanelV2PlacedType[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_type {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_type == db.RolePanelV2TypeReaction {
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

				panel_type := gd.RolePanelV2PlacedType[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_type {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_type == db.RolePanelV2TypeReaction {
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

		place.SelectedType = db.RolePanelV2Type(event.StringSelectMenuInteractionData().Values[0])

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
		switch place.SelectedType {
		case db.RolePanelV2TypeReaction:
			message_create = db.RolePanelV2MessageReaction(panel, event.Locale(), message_create)
		case db.RolePanelV2TypeSelectMenu:
			message_create = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_create)
		case db.RolePanelV2TypeButton:
			message_create = db.RolePanelV2MessageButton(panel, event.Locale(), message_create)
		}
		message, err := event.Client().Rest().CreateMessage(event.Channel().ID(), message_create.Build())
		if err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		key := fmt.Sprintf("%d/%d", event.Channel().ID(), message.ID)
		gd.RolePanelV2Placed[key] = panel.ID
		gd.RolePanelV2PlacedType[key] = place.SelectedType
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}

		if place.SelectedType == db.RolePanelV2TypeReaction {
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
		if !ok {
			return botlib.ReturnErrMessage(event, "error_not_found", botlib.WithEphemeral(true))
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

				panel_type := gd.RolePanelV2PlacedType[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_type {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_type == db.RolePanelV2TypeReaction {
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

				panel_type := gd.RolePanelV2PlacedType[k]

				message_update := discord.NewMessageUpdateBuilder()
				switch panel_type {
				case db.RolePanelV2TypeReaction:
					message_update = db.RolePanelV2MessageReaction(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeSelectMenu:
					message_update = db.RolePanelV2MessageSelectMenu(panel, event.Locale(), message_update)
				case db.RolePanelV2TypeButton:
					message_update = db.RolePanelV2MessageButton(panel, event.Locale(), message_update)
				}
				if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
					b.Logger.Errorf("error on update role panel message: %s", err.Error())
					return
				}

				if panel_type == db.RolePanelV2TypeReaction {
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

					panel_type := gd.RolePanelV2PlacedType[k]

					message_update := discord.NewMessageUpdateBuilder()
					switch panel_type {
					case db.RolePanelV2TypeReaction:
						message_update = db.RolePanelV2MessageReaction(panel, edit.EmojiLocale, message_update)
					case db.RolePanelV2TypeSelectMenu:
						message_update = db.RolePanelV2MessageSelectMenu(panel, edit.EmojiLocale, message_update)
					case db.RolePanelV2TypeButton:
						message_update = db.RolePanelV2MessageButton(panel, edit.EmojiLocale, message_update)
					}
					if _, err := event.Client().Rest().UpdateMessage(channel_id, message_id, message_update.Build()); err != nil {
						b.Logger.Errorf("error on update role panel message: %s", err.Error())
						return
					}

					if panel_type == db.RolePanelV2TypeReaction {
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
				delete(gd.RolePanelV2PlacedType, key)
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

// これより下V2完成後廃止予定

func rolePanelDeleteHandler(b *botlib.Bot[*client.Client]) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		mute := b.Self.GuildDataLock(*event.GuildID())
		if !mute.TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy")
		}
		defer mute.Unlock()
		gData, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_has_no_data")
		}
		options := []discord.StringSelectMenuOption{}
		for u := range gData.RolePanel {
			rp, err := b.Self.DB.RolePanel().Get(u)
			if err != nil {
				delete(gData.RolePanel, u)
			}
			options = append(options, discord.StringSelectMenuOption{
				Label:       rp.Name,
				Description: rp.Description,
				Value:       rp.UUID().String(),
			})
		}
		err = b.Self.DB.GuildData().Set(gData.ID, gData)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		tokenID := uuid.New()
		err = b.Self.DB.Interactions().Set(tokenID, event.Token())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := []discord.Embed{
			{
				Title: translate.Message(event.Locale(), "role_panel"),
			},
		}
		embeds = botlib.SetEmbedsProperties(embeds)
		err = event.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.StringSelectMenuComponent{
						CustomID: fmt.Sprintf("handler:rolepanel:delete:%s", tokenID.String()),
						Options:  options,
					},
				},
			},
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func rolePanelListHandler(b *botlib.Bot[*client.Client]) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		mute := b.Self.GuildDataLock(*event.GuildID())
		if !mute.TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy")
		}
		defer mute.Unlock()
		gData, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_has_no_data")
		}
		fields := []discord.EmbedField{}
		for u, gdrp := range gData.RolePanel {
			rp, err := b.Self.DB.RolePanel().Get(u)
			if err != nil {
				delete(gData.RolePanel, u)
			}
			var url string
			if !gdrp.OnList {
				mes, err := event.Client().Rest().GetMessage(rp.ChannelID, rp.MessageID)
				if err != nil {
					delete(gData.RolePanel, u)
					err := b.Self.DB.RolePanel().Del(u)
					if err != nil {
						return botlib.ReturnErr(event, err)
					}
				}
				url = fmt.Sprintf(" %s", mes.JumpURL())
			}
			fields = append(fields, discord.EmbedField{
				Name: func() string {
					if rp.Description == "" {
						return rp.Name
					}
					return rp.Name + url
				}(),
				Value: func() string {
					if rp.Description == "" {
						return url
					}
					return fmt.Sprintf("```\r%s```", rp.Description)
				}(),
			})
		}
		err = b.Self.DB.GuildData().Set(gData.ID, gData)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := []discord.Embed{
			{
				Title:  translate.Message(event.Locale(), "role_panel"),
				Fields: fields,
			},
		}
		embeds = botlib.SetEmbedsProperties(embeds)
		err = event.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func rolePanelCreateHandler(b *botlib.Bot[*client.Client]) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		err := event.CreateModal(discord.ModalCreate{
			Title:    translate.Message(event.Locale(), "command_text_role_panel_role_create_modal_title"),
			CustomID: "handler:rolepanel:create",
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:  "name",
						Style:     discord.TextInputStyle(discord.TextInputStyleShort),
						MaxLength: 100,
						Required:  true,
						Label:     translate.Message(event.Locale(), "command_text_role_panel_role_create_modal_component_name_label"),
					},
				},
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:  "description",
						Style:     discord.TextInputStyle(discord.TextInputStyleShort),
						MaxLength: 2048,
						Required:  false,
						Label:     translate.Message(event.Locale(), "command_text_role_panel_role_create_modal_component_description_label"),
					},
				},
			},
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func RolePanelComponent(b *botlib.Bot[*client.Client]) handler.Component {
	return handler.Component{
		Name: "rolepanel",
		Handler: map[string]handler.ComponentHandler{
			"editrole":          roleComponentEditRoleHandler(b),
			"backmainmenu":      roleComponentBackMainMenuHandler(b),
			"addrole":           roleComponentAddRoleHandler(b),
			"addroleselectmenu": roleComponentAddRoleSelectMenuHandler(b),
			"editroleinfo":      roleComponentEditRoleInfoHandler(b),
			"editroleemoji":     roleComponentEditRoleEmojiHandler(b),
			"editdeleterole":    roleComponentEditRoleDeleteHandler(b),
			"editpanelinfo":     roleComponentEditPanelInfoHandler(b),
			"editpanelsettings": roleComponentEditPanelSettingsHandler(b),
			"changesettings":    roleComponentChangeSettingsHandler(b),
			"create":            roleComponentCreateHandler(b),
			"use":               roleComponentUseHandler(b),
			"getrole":           roleComponentGetRole(b),
			"delete":            roleComponentDeleteHandler(b),
			"call":              roleComponentCallHandler(b),
		},
	}
}

func roleComponentCallHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		panelID := uuid.MustParse(event.StringSelectMenuInteractionData().Values[0])
		rp, err := b.Self.DB.RolePanel().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		mes := rp.BuildMessage(botlib.SetEmbedsProperties)
		mes.Flags = discord.MessageFlagEphemeral
		err = event.CreateMessage(mes)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func roleComponentDeleteHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		mute := b.Self.GuildDataLock(*event.GuildID())
		if !mute.TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy")
		}
		defer mute.Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(event.StringSelectMenuInteractionData().Values[0])
		token, err := b.Self.DB.Interactions().Get(uuid.MustParse(args[3]))
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.RolePanel().Del(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gData, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		delete(gData.RolePanel, panelID)
		err = b.Self.DB.GuildData().Set(gData.ID, gData)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := []discord.Embed{
			{
				Title:       translate.Message(event.Locale(), "command_text_role_panel_delete_embed_title"),
				Description: translate.Message(event.Locale(), "command_text_role_panel_delete_embed_description"),
			},
		}
		embeds = botlib.SetEmbedsProperties(embeds)
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &[]discord.ContainerComponent{},
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleComponentGetRole(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanel().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		member, err := event.Client().Rest().GetMember(*event.GuildID(), event.User().ID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		sMap := map[snowflake.ID]bool{}
		for _, v := range event.StringSelectMenuInteractionData().Values {
			id := snowflake.MustParse(v)
			sMap[id] = true
		}
		rMap := map[snowflake.ID]bool{}
		for _, i2 := range member.RoleIDs {
			rMap[i2] = true
		}
		roles := rp.GetRoles()
		var already, added, removed string
		for i := range roles {
			if !rMap[i] || sMap[i] {
				if rMap[i] {
					//選択されたがすでに持っている
					already += fmt.Sprintf("%s\r", discord.RoleMention(i))
					continue
				}
				if !rMap[i] && sMap[i] {
					//選択されたが持っていない
					err := event.Client().Rest().AddMemberRole(*event.GuildID(), event.User().ID, i, rest.WithReason(fmt.Sprintf("role panel %s", rp.UUID().String())))
					if err != nil {
						return botlib.ReturnErr(event, err)
					}
					added += fmt.Sprintf("%s\r", discord.RoleMention(i))
					continue
				}
				// 意味不明
				b.Logger.Debug("意味不明")
				continue
			}
			// 持っているし選ばれていない
			err := event.Client().Rest().RemoveMemberRole(*event.GuildID(), event.User().ID, i, rest.WithReason(fmt.Sprintf("role panel %s", rp.UUID().String())))
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			removed += fmt.Sprintf("%s\r", discord.RoleMention(i))
			continue
		}
		fields := []discord.EmbedField{}
		if added != "" {
			fields = append(fields, discord.EmbedField{
				Name:  translate.Message(event.Locale(), "command_text_role_panel_get_role_embed_field_added_name"),
				Value: added,
			})
		}
		if already != "" {
			fields = append(fields, discord.EmbedField{
				Name:  translate.Message(event.Locale(), "command_text_role_panel_get_role_embed_field_already_name"),
				Value: already,
			})
		}
		if removed != "" {
			fields = append(fields, discord.EmbedField{
				Name:  translate.Message(event.Locale(), "command_text_role_panel_get_role_embed_field_removed_name"),
				Value: removed,
			})
		}
		embeds := []discord.Embed{
			{
				Title:  translate.Message(event.Locale(), "role_panel"),
				Fields: fields,
			},
		}
		embeds = botlib.SetEmbedsProperties(embeds)
		err = event.CreateMessage(discord.MessageCreate{
			Flags:  discord.MessageFlagEphemeral,
			Embeds: embeds,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		go func() {
			time.Sleep(time.Second * 3)
			err := event.Client().Rest().DeleteInteractionResponse(event.ApplicationID(), event.Token())
			if err != nil {
				b.Logger.Error(err)
			}
		}()
		return nil
	}
}

func roleComponentUseHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanel().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		m, ok := event.Client().Caches().Member(*event.GuildID(), event.User().ID)
		if !ok {
			member, err := event.Client().Rest().GetMember(*event.GuildID(), event.User().ID)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			m = *member
		}
		mes := rp.UseMessage(botlib.SetEmbedsProperties, m)
		b.Logger.Debugf("%+v", mes)
		err = event.CreateMessage(mes)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func roleComponentCreateHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		mute := b.Self.GuildDataLock(*event.GuildID())
		if !mute.TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy")
		}
		defer mute.Unlock()
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if len(args) < 5 {
			embeds := []discord.Embed{
				{
					Title:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_embed_title"),
					Description: translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_embed_description"),
				},
			}
			embeds = botlib.SetEmbedsProperties(embeds)
			components := []discord.ContainerComponent{
				discord.ActionRowComponent{
					rp.BackMainMenuButton(),
					discord.ButtonComponent{
						CustomID: fmt.Sprintf("%s:%s", event.Data.CustomID(), "sendchannel"),
						Label:    translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_components_send_channel_label"),
						Style:    discord.ButtonStylePrimary,
					},
					discord.ButtonComponent{
						CustomID: fmt.Sprintf("%s:%s", event.Data.CustomID(), "addlist"),
						Label:    translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_components_add_list_label"),
						Style:    discord.ButtonStylePrimary,
					},
				},
			}
			_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &components,
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			return event.DeferUpdateMessage()
		}

		switch args[4] {
		case "sendchannel":
			b.Logger.Debug(event.Data.Type())
			if event.Data.Type() == discord.ComponentTypeChannelSelectMenu {
				r := db.NewRolePanel(rp)

				gData, err := b.Self.DB.GuildData().Get(*event.GuildID())
				if err != nil {
					gData = db.NewGuildData(*event.GuildID())
				}
				gData.RolePanel[r.UUID()] = db.GuildDataRolePanel{OnList: false}
				if gData.RolePanelLimit > 25 || len(gData.RolePanel) > gData.RolePanelLimit {
					return botlib.ReturnErrMessage(event, "error_guild_max_count_limit_has_reached")
				}
				err = b.Self.DB.GuildData().Set(gData.ID, gData)
				if err != nil {
					return botlib.ReturnErr(event, err)
				}

				m, err := event.Client().Rest().CreateMessage(event.ChannelSelectMenuInteractionData().Values[0], r.BuildMessage(botlib.SetEmbedsProperties))
				if err != nil {
					return botlib.ReturnErr(event, err)
				}
				r.MessageID = m.ID
				r.ChannelID = m.ChannelID
				r.GuildID = *event.GuildID()

				err = b.Self.DB.RolePanel().Set(r)
				if err != nil {
					return botlib.ReturnErr(event, err)
				}

				err = event.Client().Rest().DeleteInteractionResponse(event.ApplicationID(), token)
				if err != nil {
					return botlib.ReturnErr(event, err)
				}
				return event.DeferUpdateMessage()
			}
			embeds := []discord.Embed{
				{
					Title:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_send_channel_embed_title"),
					Description: translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_send_channel_embed_description"),
				},
			}
			embeds = botlib.SetEmbedsProperties(embeds)
			components := []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.ChannelSelectMenuComponent{
						CustomID: event.Data.CustomID(),
						ChannelTypes: []discord.ChannelType{
							discord.ChannelTypeGuildText,
						},
					},
				},
			}
			_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &components,
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			return event.DeferUpdateMessage()
		case "addlist":
			r := db.NewRolePanel(rp)
			r.GuildID = *event.GuildID()

			gData, err := b.Self.DB.GuildData().Get(*event.GuildID())
			if err != nil {
				gData = db.NewGuildData(*event.GuildID())
			}
			gData.RolePanel[r.UUID()] = db.GuildDataRolePanel{OnList: true}
			if gData.RolePanelLimit > 25 || len(gData.RolePanel) > gData.RolePanelLimit {
				return botlib.ReturnErrMessage(event, "error_guild_max_count_limit_has_reached")
			}
			err = b.Self.DB.GuildData().Set(gData.ID, gData)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}

			err = b.Self.DB.RolePanel().Set(r)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}

			embeds := []discord.Embed{
				{
					Title:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_add_list_embed_title"),
					Description: translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_add_list_embed_description"),
				},
			}
			embeds = botlib.SetEmbedsProperties(embeds)
			_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
				Embeds:     &embeds,
				Components: &[]discord.ContainerComponent{},
			})
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			return event.DeferUpdateMessage()
		}
		return event.DeferUpdateMessage()
	}
}

func roleComponentChangeSettingsHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		return event.CreateModal(discord.ModalCreate{
			CustomID: fmt.Sprintf("%s:%s", event.Data.CustomID(), event.StringSelectMenuInteractionData().Values[0]),
			Title:    translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_config_modal_title"),
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "value",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_config_modal_components_value_label"),
						MaxLength:   2,
						Required:    true,
						Placeholder: translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_config_modal_components_value_placeholder"),
					},
				},
			},
		})
	}
}

func roleComponentEditPanelSettingsHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.EditPanelSettingsEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.EditPanelSettingsComponent()
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleComponentEditPanelInfoHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.CreateModal(discord.ModalCreate{
			CustomID: event.Data.CustomID(),
			Title:    translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_info_modal_title"),
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "name",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_info_modal_component_name_label"),
						Placeholder: rp.Name,
						Value:       rp.Name,
						MaxLength:   32,
						Required:    true,
					},
				},
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "description",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_info_modal_component_description_label"),
						Placeholder: rp.Description,
						Value:       rp.Description,
						MaxLength:   100,
						Required:    false,
					},
				},
			},
		})
	}
}

func roleComponentEditRoleDeleteHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		rp.DeleteRole(roleID)
		err = b.Self.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.BaseMenuComponent()
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		return event.DeferUpdateMessage()
	}
}

func roleComponentEditRoleEmojiHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		role, _ := rp.GetRole(roleID)

		cancel := func(bt bot.Client) error {
			embeds := rp.EditRoleMenuEmbed(roleID)
			embeds = botlib.SetEmbedsProperties(embeds)
			_, err := bt.Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
				Embeds: &embeds,
				Components: &[]discord.ContainerComponent{
					rp.EditRoleMenuComponent(roleID),
				},
			})
			return err
		}

		channel := event.Channel()

		var remove func()
		var removeButton func()
		author := event.Member()
		remove = b.Handler.AddMessage(handler.Message{
			ChannelID: json.Ptr(channel.ID()),
			AuthorID:  &author.User.ID,
			Handler: func(event *events.GuildMessageCreate) error {
				if event.Message.Author.ID != author.User.ID || !emoji.MatchString(event.Message.Content) {
					return nil
				}
				matches := emoji.FindAllString(event.Message.Content)
				role.Emoji = botlib.ParseComponentEmoji(matches[0])
				remove()
				removeButton()

				rp.SetRole(role.Label, role.Description, roleID, &role.Emoji)
				err := b.Self.DB.RolePanelCreate().Set(rp)
				if err != nil {
					return err
				}
				_ = event.Client().Rest().DeleteMessage(event.ChannelID, event.Message.ID)
				return cancel(event.Client())
			},
		})
		customID := uuid.New()
		removeButton = b.Handler.AddComponent(handler.Component{
			Name: customID.String(),
			Handler: map[string]handler.ComponentHandler{
				"cancel": func(event *events.ComponentInteractionCreate) error {
					remove()
					removeButton()
					err := cancel(event.Client())
					if err != nil {
						return botlib.ReturnErr(event, err)
					}
					return event.DeferUpdateMessage()
				},
			},
		})

		embeds := []discord.Embed{
			{
				Title:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_role_emoji_embed_title"),
				Description: translate.Message(event.Locale(), "command_text_role_panel_create_edit_role_emoji_embed_description"),
			},
		}
		embeds = botlib.SetEmbedsProperties(embeds)
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds: &embeds,
			Components: &[]discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.ButtonComponent{
						Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
						CustomID: fmt.Sprintf("handler:%s:cancel", customID.String()),
						Emoji: &discord.ComponentEmoji{
							ID:   snowflake.ID(1082689149557014549),
							Name: "x_",
						},
					},
				},
			},
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		return event.DeferUpdateMessage()
	}
}

func roleComponentEditRoleInfoHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		role, _ := rp.GetRole(roleID)
		return event.CreateModal(discord.ModalCreate{
			CustomID: event.Data.CustomID(),
			Title:    translate.Message(event.Locale(), "command_text_role_panel_create_edit_role_info_modal_title"),
			Components: []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "name",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_role_info_modal_component_name_label"),
						Placeholder: role.Label,
						Value:       role.Label,
						MaxLength:   32,
						Required:    true,
					},
				},
				discord.ActionRowComponent{
					discord.TextInputComponent{
						CustomID:    "description",
						Style:       discord.TextInputStyle(discord.TextInputStyleShort),
						Label:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_role_info_modal_component_description_label"),
						Placeholder: role.Description,
						Value:       role.Description,
						MaxLength:   100,
						Required:    false,
					},
				},
			},
		})
	}
}

func roleComponentAddRoleSelectMenuHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		self, valid := event.Client().Caches().SelfMember(*event.GuildID())
		if !valid {
			return botlib.ReturnErrMessage(event, "error_bot_member_not_found")
		}
		roleMap := make(map[snowflake.ID]discord.Role)
		for _, i2 := range self.RoleIDs {
			role, valid := event.Client().Caches().Role(*event.GuildID(), i2)
			if !valid {
				continue
			}
			roleMap[i2] = role
		}
		hi, _ := botlib.GetHighestRolePosition(roleMap)
		roles := event.RoleSelectMenuInteractionData().Resolved.Roles
		for i, r := range roles {
			if r.Managed || r.Position >= hi {
				delete(roles, i)
				continue
			}
			var emoji *discord.ComponentEmoji = nil
			if r.Emoji != nil {
				b.Logger.Debug("has emoji")
				e := botlib.ParseComponentEmoji(*r.Emoji)
				emoji = &e
			}
			var description string
			if r.Description != nil {
				description = *r.Description
			}
			rp.SetRole(r.Name, description, r.ID, emoji)
		}

		err = b.Self.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.BaseMenuComponent()
		b.Logger.Debugf("%+v", components)
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleComponentAddRoleHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(rp.UUID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.AddRoleMenuEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.AddRoleMenuComponent()
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleComponentBackMainMenuHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		var err error
		args := strings.Split(event.Data.CustomID(), ":")
		rp, err := b.Self.DB.RolePanelCreate().Get(uuid.MustParse(args[3]))
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(rp.UUID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.BaseMenuComponent()
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleComponentEditRoleHandler(b *botlib.Bot[*client.Client]) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		var err error
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		roleID := snowflake.MustParse(event.StringSelectMenuInteractionData().Values[0])
		embeds := rp.EditRoleMenuEmbed(roleID)
		embeds = botlib.SetEmbedsProperties(embeds)
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds: &embeds,
			Components: &[]discord.ContainerComponent{
				rp.EditRoleMenuComponent(roleID),
			},
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func RolePanelModal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "rolepanel",
		Handler: map[string]handler.ModalHandler{
			"create":         roleModalCreateHandler(b),
			"editroleinfo":   roleModalEditRoleInfoHandler(b),
			"editpanelinfo":  roleModalEditPanelInfoHandler(b),
			"changesettings": roleModalChangeSettingsHandler(b),
		},
	}
}

func roleModalChangeSettingsHandler(b *botlib.Bot[*client.Client]) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		value, err := strconv.ParseInt(event.ModalSubmitInteraction.Data.Text("value"), 10, 64)
		if err != nil || value < 1 || value > 25 {
			return botlib.ReturnErrMessage(event, "error_out_of_range_select_menu")
		}
		switch args[4] {
		case "max":
			rp.Max = int(value)
		case "min":
			rp.Min = int(value)
		default:
			return botlib.ReturnErr(event, errors.New("invalid args"))
		}
		rp.Validate()
		err = b.Self.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.EditPanelSettingsEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.EditPanelSettingsComponent()
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleModalEditPanelInfoHandler(b *botlib.Bot[*client.Client]) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		rp.Name = event.ModalSubmitInteraction.Data.Text("name")
		rp.Description = event.ModalSubmitInteraction.Data.Text("description")
		err = b.Self.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.BaseMenuComponent()
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleModalEditRoleInfoHandler(b *botlib.Bot[*client.Client]) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.Self.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.Self.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		role, _ := rp.GetRole(roleID)
		rp.SetRole(event.ModalSubmitInteraction.Data.Text("name"), event.ModalSubmitInteraction.Data.Text("description"), roleID, &role.Emoji)

		err = b.Self.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		embeds := rp.EditRoleMenuEmbed(roleID)
		embeds = botlib.SetEmbedsProperties(embeds)
		_, err = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
			Embeds: &embeds,
			Components: &[]discord.ContainerComponent{
				rp.EditRoleMenuComponent(roleID),
			},
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return event.DeferUpdateMessage()
	}
}

func roleModalCreateHandler(b *botlib.Bot[*client.Client]) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		var err error
		rp := db.NewRolePanelCreate(event.ModalSubmitInteraction.Data.Text("name"), event.ModalSubmitInteraction.Data.Text("description"), event.Locale())
		err = b.Self.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.Self.DB.Interactions().Set(rp.UUID(), event.Token())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedsProperties(embeds)
		components := rp.BaseMenuComponent()
		err = event.CreateMessage(discord.MessageCreate{
			Flags:      discord.MessageFlagEphemeral,
			Embeds:     embeds,
			Components: components,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}
