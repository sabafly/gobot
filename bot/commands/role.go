package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/db"
	"github.com/sabafly/gobot/lib/handler"
	"github.com/sabafly/gobot/lib/structs"
	"github.com/sabafly/gobot/lib/translate"
)

func Role(b *botlib.Bot) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "role",
			Description:  "require manage role permission",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "panel",
					Description: "role panel is the panel of role",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "create",
							Description: "creates a new role panel",
						},
						{
							Name:        "list",
							Description: "show list of role panels",
						},
						{
							Name:        "delete",
							Description: "deletes a role panel",
						},
					},
				},
			},
		},
		Check: func(ctx *events.ApplicationCommandInteractionCreate) bool {
			if b.CheckDev(ctx.User().ID) {
				return true
			}
			permission := discord.PermissionManageRoles
			if ctx.Member() != nil && ctx.Member().Permissions.Has(permission) {
				return true
			}
			_ = botlib.ReturnErrMessage(ctx, "error_no_permission", map[string]any{"Name": permission.String()})
			return false
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"panel/create": rolePanelCreateHandler(b),
			"panel/list":   rolePanelListHandler(b),
			"panel/delete": rolePanelDeleteHandler(b),
		},
	}
}

func rolePanelDeleteHandler(b *botlib.Bot) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		gData, err := b.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_has_no_data")
		}
		options := []discord.StringSelectMenuOption{}
		for u := range gData.RolePanel {
			rp, err := b.DB.RolePanel().Get(u)
			if err != nil {
				delete(gData.RolePanel, u)
			}
			options = append(options, discord.StringSelectMenuOption{
				Label:       rp.Name,
				Description: rp.Description,
				Value:       rp.UUID().String(),
			})
		}
		err = b.DB.GuildData().Set(gData.ID, gData)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		tokenID := uuid.New()
		err = b.DB.Interactions().Set(tokenID, event.Token())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := []discord.Embed{
			{
				Title: translate.Message(event.Locale(), "role_panel"),
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
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

func rolePanelListHandler(b *botlib.Bot) func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		gData, err := b.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_has_no_data")
		}
		fields := []discord.EmbedField{}
		for u, gdrp := range gData.RolePanel {
			rp, err := b.DB.RolePanel().Get(u)
			if err != nil {
				delete(gData.RolePanel, u)
			}
			var url string
			if !gdrp.OnList {
				mes, err := event.Client().Rest().GetMessage(rp.ChannelID, rp.MessageID)
				if err != nil {
					delete(gData.RolePanel, u)
					err := b.DB.RolePanel().Remove(u)
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
		err = b.DB.GuildData().Set(gData.ID, gData)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := []discord.Embed{
			{
				Title:  translate.Message(event.Locale(), "role_panel"),
				Fields: fields,
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
		err = event.CreateMessage(discord.MessageCreate{
			Embeds: embeds,
		})
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func rolePanelCreateHandler(b *botlib.Bot) func(event *events.ApplicationCommandInteractionCreate) error {
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

func RolePanelComponent(b *botlib.Bot) handler.Component {
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

func roleComponentCallHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		panelID := uuid.MustParse(event.StringSelectMenuInteractionData().Values[0])
		rp, err := b.DB.RolePanel().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		mes := rp.BuildMessage(botlib.SetEmbedProperties)
		mes.Flags = discord.MessageFlagEphemeral
		err = event.CreateMessage(mes)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func roleComponentDeleteHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(event.StringSelectMenuInteractionData().Values[0])
		token, err := b.DB.Interactions().Get(uuid.MustParse(args[3]))
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.RolePanel().Remove(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gData, err := b.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		delete(gData.RolePanel, panelID)
		err = b.DB.GuildData().Set(gData.ID, gData)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := []discord.Embed{
			{
				Title:       translate.Message(event.Locale(), "command_text_role_panel_delete_embed_title"),
				Description: translate.Message(event.Locale(), "command_text_role_panel_delete_embed_description"),
			},
		}
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentGetRole(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanel().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		member, err := event.Client().Rest().GetMember(*event.GuildID(), event.Member().User.ID)
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
					err := event.Client().Rest().AddMemberRole(*event.GuildID(), event.Member().User.ID, i, rest.WithReason(fmt.Sprintf("role panel %s", rp.UUID().String())))
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
			err := event.Client().Rest().RemoveMemberRole(*event.GuildID(), event.Member().User.ID, i, rest.WithReason(fmt.Sprintf("role panel %s", rp.UUID().String())))
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
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentUseHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanel().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		m, ok := event.Client().Caches().Member(*event.GuildID(), event.Member().User.ID)
		if !ok {
			member, err := event.Client().Rest().GetMember(*event.GuildID(), event.Member().User.ID)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			m = *member
		}
		mes := rp.UseMessage(botlib.SetEmbedProperties, m)
		b.Logger.Debugf("%+v", mes)
		err = event.CreateMessage(mes)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func roleComponentCreateHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
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
			embeds = botlib.SetEmbedProperties(embeds)
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

				gData, err := b.DB.GuildData().Get(*event.GuildID())
				if err != nil {
					gData = db.NewGuildData(*event.GuildID())
				}
				gData.RolePanel[r.UUID()] = db.GuildDataRolePanel{OnList: false}
				if gData.RolePanelLimit > 25 || len(gData.RolePanel) > gData.RolePanelLimit {
					return botlib.ReturnErrMessage(event, "error_guild_max_count_limit_has_reached")
				}
				err = b.DB.GuildData().Set(gData.ID, gData)
				if err != nil {
					return botlib.ReturnErr(event, err)
				}

				m, err := event.Client().Rest().CreateMessage(event.ChannelSelectMenuInteractionData().Values[0], r.BuildMessage(botlib.SetEmbedProperties))
				if err != nil {
					return botlib.ReturnErr(event, err)
				}
				r.MessageID = m.ID
				r.ChannelID = m.ChannelID
				r.GuildID = *event.GuildID()

				err = b.DB.RolePanel().Set(r)
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
			embeds = botlib.SetEmbedProperties(embeds)
			components := []discord.ContainerComponent{
				discord.ActionRowComponent{
					discord.ChannelSelectMenuComponent{
						CustomID: event.Data.CustomID(),
						ChannelTypes: []discord.ComponentType{
							discord.ComponentType(discord.ChannelTypeGuildText),
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

			gData, err := b.DB.GuildData().Get(*event.GuildID())
			if err != nil {
				gData = db.NewGuildData(*event.GuildID())
			}
			gData.RolePanel[r.UUID()] = db.GuildDataRolePanel{OnList: true}
			if gData.RolePanelLimit > 25 || len(gData.RolePanel) > gData.RolePanelLimit {
				return botlib.ReturnErrMessage(event, "error_guild_max_count_limit_has_reached")
			}
			err = b.DB.GuildData().Set(gData.ID, gData)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}

			err = b.DB.RolePanel().Set(r)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}

			embeds := []discord.Embed{
				{
					Title:       translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_add_list_embed_title"),
					Description: translate.Message(event.Locale(), "command_text_role_panel_create_edit_panel_create_add_list_embed_description"),
				},
			}
			embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentChangeSettingsHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
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

func roleComponentEditPanelSettingsHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.EditPanelSettingsEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentEditPanelInfoHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
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

func roleComponentEditRoleDeleteHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		rp.DeleteRole(roleID)
		err = b.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentEditRoleEmojiHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		role, _ := rp.GetRole(roleID)

		cancel := func(bt bot.Client) error {
			embeds := rp.EditRoleMenuEmbed(roleID)
			embeds = botlib.SetEmbedProperties(embeds)
			_, err := bt.Rest().UpdateInteractionResponse(event.ApplicationID(), token, discord.MessageUpdate{
				Embeds: &embeds,
				Components: &[]discord.ContainerComponent{
					rp.EditRoleMenuComponent(roleID),
				},
			})
			return err
		}

		var remove func()
		var removeButton func()
		author := event.Member()
		remove = b.Handler.AddMessage(handler.Message{
			UUID:      uuid.New(),
			ChannelID: event.ChannelID(),
			AuthorID:  &author.User.ID,
			Handler: func(event *events.MessageCreate) error {
				if event.Message.Author.ID != author.User.ID || !structs.Twemoji.MatchString(event.Message.Content) {
					return nil
				}
				matches := structs.Twemoji.FindAllString(event.Message.Content, -1)
				role.Emoji = botlib.ParseComponentEmoji(matches[0])
				remove()
				removeButton()

				rp.SetRole(role.Label, role.Description, roleID, &role.Emoji)
				err := b.DB.RolePanelCreate().Set(rp)
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
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentEditRoleInfoHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
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

func roleComponentAddRoleSelectMenuHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
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

		err = b.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentAddRoleHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(rp.UUID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.AddRoleMenuEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentBackMainMenuHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		var err error
		args := strings.Split(event.Data.CustomID(), ":")
		rp, err := b.DB.RolePanelCreate().Get(uuid.MustParse(args[3]))
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(rp.UUID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleComponentEditRoleHandler(b *botlib.Bot) func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		var err error
		args := strings.Split(event.Data.CustomID(), ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		roleID := snowflake.MustParse(event.StringSelectMenuInteractionData().Values[0])
		embeds := rp.EditRoleMenuEmbed(roleID)
		embeds = botlib.SetEmbedProperties(embeds)
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

func RolePanelModal(b *botlib.Bot) handler.Modal {
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

func roleModalChangeSettingsHandler(b *botlib.Bot) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
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
		err = b.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.EditPanelSettingsEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleModalEditPanelInfoHandler(b *botlib.Bot) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		panelID := uuid.MustParse(args[3])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		rp.Name = event.ModalSubmitInteraction.Data.Text("name")
		rp.Description = event.ModalSubmitInteraction.Data.Text("description")
		err = b.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleModalEditRoleInfoHandler(b *botlib.Bot) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		args := strings.Split(event.Data.CustomID, ":")
		panelID := uuid.MustParse(args[3])
		roleID := snowflake.MustParse(args[4])
		rp, err := b.DB.RolePanelCreate().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		token, err := b.DB.Interactions().Get(panelID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		role, _ := rp.GetRole(roleID)
		rp.SetRole(event.ModalSubmitInteraction.Data.Text("name"), event.ModalSubmitInteraction.Data.Text("description"), roleID, &role.Emoji)

		err = b.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}

		embeds := rp.EditRoleMenuEmbed(roleID)
		embeds = botlib.SetEmbedProperties(embeds)
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

func roleModalCreateHandler(b *botlib.Bot) func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		var err error
		rp := db.NewRolePanelCreate(event.ModalSubmitInteraction.Data.Text("name"), event.ModalSubmitInteraction.Data.Text("description"), event.Locale())
		err = b.DB.RolePanelCreate().Set(rp)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		err = b.DB.Interactions().Set(rp.UUID(), event.Token())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embeds := rp.BaseMenuEmbed()
		embeds = botlib.SetEmbedProperties(embeds)
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
