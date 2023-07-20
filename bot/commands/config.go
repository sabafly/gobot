package commands

import (
	"github.com/disgoorg/json"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Config(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "config",
			Description:  "config",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "bump",
					Description: "bump",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "on",
							Description: "turn on",
						},
						{
							Name:        "off",
							Description: "turn off",
						},
						{
							Name:        "message",
							Description: "config message",
						},
						{
							Name:        "mention",
							Description: "set mention role",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionRole{
									Name:        "role",
									Description: "target role",
									Required:    true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "up",
					Description: "up",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "on",
							Description: "turn on",
						},
						{
							Name:        "off",
							Description: "turn off",
						},
						{
							Name:        "message",
							Description: "config message",
						},
						{
							Name:        "mention",
							Description: "set mention role",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionRole{
									Name:        "role",
									Description: "target role",
									Required:    true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "level",
					Description: "level config",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "notice-message",
							Description: "set level up message",
						},
						{
							Name:        "notice-channel",
							Description: "set level up message channel to send",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionChannel{
									Name:        "channel",
									Description: "target channel",
									ChannelTypes: []discord.ChannelType{
										discord.ChannelTypeGuildText,
									},
								},
							},
						},
					},
				},
			},
		},
		Check: b.Self.CheckCommandPermission(b, "guild.config.manage", discord.PermissionManageGuild),
		CommandHandlers: map[string]handler.CommandHandler{
			"bump/on":              configBumpOnCommandHandler(b),
			"bump/off":             configBumpOffCommandHandler(b),
			"bump/message":         configBumpMessageConfigHandler(b),
			"bump/mention":         configBumpMentionCommandHandler(b),
			"up/on":                configUpOnCommandHandler(b),
			"up/off":               configUpOffCommandHandler(b),
			"up/message":           configUpMessageConfigHandler(b),
			"up/mention":           configUpMentionCommandHandler(b),
			"level/notice-message": configLevelNoticeMessageCommandHandler(b),
			"level/notice-channel": configLevelNoticeChannelCommandHandler(b),
		},
	}
}

func configBumpOnCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if gd.BumpStatus.BumpMessage == [2]string{} || gd.BumpStatus.BumpRemind == [2]string{} {
			return botlib.ReturnErrMessage(event, "error_message_not_configured")
		}
		gd.BumpStatus.BumpEnabled = true
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func configBumpOffCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.BumpEnabled = false
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func configBumpMessageConfigHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		modal := discord.NewModalCreateBuilder()
		modal.SetCustomID("handler:config:bump-message")
		modal.SetTitle("bump message")
		modal.AddContainerComponents(
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleShort,
				CustomID:  "message-title",
				Required:  true,
				MaxLength: 32,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_0_label"),
				Value:     gd.BumpStatus.BumpMessage[0],
			}),
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleParagraph,
				CustomID:  "message-body",
				Required:  true,
				MaxLength: 256,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_1_label"),
				Value:     gd.BumpStatus.BumpMessage[1],
			}),
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleShort,
				CustomID:  "remind-title",
				Required:  true,
				MaxLength: 32,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_2_label"),
				Value:     gd.BumpStatus.BumpRemind[0],
			}),
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleParagraph,
				CustomID:  "remind-body",
				Required:  true,
				MaxLength: 256,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_3_label"),
				Value:     gd.BumpStatus.BumpRemind[1],
			}),
		)
		if err := event.CreateModal(modal.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configBumpMentionCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.BumpRole = json.Ptr(event.SlashCommandInteractionData().Role("role").ID)
		if *gd.BumpStatus.BumpRole == 0 {
			gd.BumpStatus.BumpRole = nil
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func configUpOnCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if gd.BumpStatus.UpMessage == [2]string{} || gd.BumpStatus.UpRemind == [2]string{} {
			return botlib.ReturnErrMessage(event, "error_message_not_configured")
		}
		gd.BumpStatus.UpEnabled = true
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func configUpOffCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.UpEnabled = false
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func configUpMessageConfigHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		modal := discord.NewModalCreateBuilder()
		modal.SetCustomID("handler:config:up-message")
		modal.SetTitle("up message")
		modal.AddContainerComponents(
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleShort,
				CustomID:  "message-title",
				Required:  true,
				MaxLength: 32,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_0_label"),
				Value:     gd.BumpStatus.UpMessage[0],
			}),
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleParagraph,
				CustomID:  "message-body",
				Required:  true,
				MaxLength: 256,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_1_label"),
				Value:     gd.BumpStatus.UpMessage[1],
			}),
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleShort,
				CustomID:  "remind-title",
				Required:  true,
				MaxLength: 32,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_2_label"),
				Value:     gd.BumpStatus.UpRemind[0],
			}),
			discord.NewActionRow(discord.TextInputComponent{
				Style:     discord.TextInputStyleParagraph,
				CustomID:  "remind-body",
				Required:  true,
				MaxLength: 256,
				Label:     translate.Message(event.Locale(), "command_config_bump_message_modal_text_3_label"),
				Value:     gd.BumpStatus.UpRemind[1],
			}),
		)
		if err := event.CreateModal(modal.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configUpMentionCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.UpRole = json.Ptr(event.SlashCommandInteractionData().Role("role").ID)
		if *gd.BumpStatus.UpRole == 0 {
			gd.BumpStatus.UpRole = nil
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func configLevelNoticeMessageCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		modal := discord.NewModalCreateBuilder()
		modal.SetCustomID("handler:level:notice-message")
		modal.SetTitle(translate.Message(event.Locale(), "modal_level_notice_message_title"))
		modal.AddContainerComponents(
			discord.NewActionRow(
				discord.TextInputComponent{
					CustomID:    "message",
					Style:       discord.TextInputStyleParagraph,
					Label:       translate.Message(event.Locale(), "modal_level_notice_message_text_input_0_label"),
					MinLength:   json.Ptr(1),
					MaxLength:   512,
					Required:    true,
					Value:       gd.Config.LevelUpMessage,
					Placeholder: translate.Message(event.Locale(), "modal_level_notice_message_text_input_0_placeholder"),
				},
			),
		)
		if err := event.CreateModal(modal.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configLevelNoticeChannelCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		if channel, ok := event.SlashCommandInteractionData().OptChannel("channel"); ok {
			gd.Config.LevelUpMessageChannel = &channel.ID
		} else {
			gd.Config.LevelUpMessageChannel = nil
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.CreateMessage(discord.NewMessageCreateBuilder().SetContent("OK").SetFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func ConfigModal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "config",
		Handler: map[string]handler.ModalHandler{
			"bump-message": configModalBumpMessageHandler(b),
			"up-message":   configModalUpMessageHandler(b),
		},
	}
}

func configModalBumpMessageHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.BumpMessage = [2]string{
			event.ModalSubmitInteraction.Data.Text("message-title"),
			event.ModalSubmitInteraction.Data.Text("message-body"),
		}
		gd.BumpStatus.BumpRemind = [2]string{
			event.ModalSubmitInteraction.Data.Text("remind-title"),
			event.ModalSubmitInteraction.Data.Text("remind-body"),
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return err
		}
		return nil
	}
}

func configModalUpMessageHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.UpMessage = [2]string{
			event.ModalSubmitInteraction.Data.Text("message-title"),
			event.ModalSubmitInteraction.Data.Text("message-body"),
		}
		gd.BumpStatus.UpRemind = [2]string{
			event.ModalSubmitInteraction.Data.Text("remind-title"),
			event.ModalSubmitInteraction.Data.Text("remind-body"),
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return err
		}
		return nil
	}
}
