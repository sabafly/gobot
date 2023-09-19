package commands

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
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
							Name:                     "on",
							Description:              "turn on",
							DescriptionLocalizations: translate.MessageMap("config_bump_on_command_description", false),
						},
						{
							Name:                     "off",
							Description:              "turn off",
							DescriptionLocalizations: translate.MessageMap("config_bump_off_command_description", false),
						},
						{
							Name:                     "message",
							Description:              "config message",
							DescriptionLocalizations: translate.MessageMap("config_bump_message_command_description", false),
						},
						{
							Name:                     "mention",
							Description:              "set mention role",
							DescriptionLocalizations: translate.MessageMap("config_bump_mention_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionRole{
									Name:                     "role",
									Description:              "target role",
									DescriptionLocalizations: translate.MessageMap("config_bump_mention_command_role_option_description", false),
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
							Name:                     "on",
							Description:              "turn on",
							DescriptionLocalizations: translate.MessageMap("config_up_on_command_description", false),
						},
						{
							Name:                     "off",
							Description:              "turn off",
							DescriptionLocalizations: translate.MessageMap("config_up_off_message_description", false),
						},
						{
							Name:                     "message",
							Description:              "config message",
							DescriptionLocalizations: translate.MessageMap("config_up_message_command_description", false),
						},
						{
							Name:                     "mention",
							Description:              "set mention role",
							DescriptionLocalizations: translate.MessageMap("config_up_mention_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionRole{
									Name:                     "role",
									Description:              "target role",
									DescriptionLocalizations: translate.MessageMap("config_up_mention_command_role_option_description", false),
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
							Name:                     "notice-message",
							Description:              "set level up message",
							DescriptionLocalizations: translate.MessageMap("config_level_notice_message_command_description", false),
						},
						{
							Name:                     "notice-channel",
							Description:              "set level up message channel to send",
							DescriptionLocalizations: translate.MessageMap("config_level_notice_channel_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionChannel{
									Name:                     "channel",
									Description:              "target channel",
									DescriptionLocalizations: translate.MessageMap("config_level_notice_channel_command_channel_option_description", false),
									ChannelTypes: []discord.ChannelType{
										discord.ChannelTypeGuildText,
									},
								},
							},
						},
						{
							Name:                     "exclude-add",
							Description:              "add exclude channel",
							DescriptionLocalizations: translate.MessageMap("config_level_exclude_add_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionChannel{
									Name:                     "channel",
									Description:              "target channel",
									DescriptionLocalizations: translate.MessageMap("config_level_exclude_add_command_channel_option_description", false),
									Required:                 true,
									ChannelTypes: []discord.ChannelType{
										discord.ChannelTypeGuildText,
										discord.ChannelTypeGuildPublicThread,
										discord.ChannelTypeGuildPrivateThread,
										discord.ChannelTypeGuildNewsThread,
									},
								},
							},
						},
						{
							Name:                     "exclude-remove",
							Description:              "remove exclude channel",
							DescriptionLocalizations: translate.MessageMap("config_level_exclude_remove_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionString{
									Name:                     "channel",
									Description:              "target channel",
									DescriptionLocalizations: translate.MessageMap("config_level_exclude_remove_command_channel_option_description", false),
									Required:                 true,
									Autocomplete:             true,
								},
							},
						},
						{
							Name:                     "import-mee6",
							Description:              "import level data from mee6 ⚠ all guild levels reset ⚠",
							DescriptionLocalizations: translate.MessageMap("config_level_import_mee6_command_description", false),
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
			"level/exclude-add":    configLevelExcludeAddCommandHandler(b),
			"level/exclude-remove": configLevelExcludeRemoveHandler(b),
			"level/import-mee6":    configLevelImportMee6CommandHandler(b),
		},
		AutocompleteCheck: func(ctx *events.AutocompleteInteractionCreate) bool {
			if b.CheckDev(ctx.User().ID) {
				return true
			}
			if ctx.Member() != nil && ctx.Member().Permissions.Has(discord.PermissionManageGuild) {
				return true
			}
			gd, err := b.Self.DB.GuildData().Get(*ctx.GuildID())
			if err == nil {
				if gd.UserPermissions[ctx.User().ID].Has("guild.config.manage") {
					return true
				}
				for _, id := range ctx.Member().RoleIDs {
					if gd.RolePermissions[id].Has("guild.config.manage") {
						return true
					}
				}
			}
			return false
		},
		AutocompleteHandlers: map[string]handler.AutocompleteHandler{
			"level/exclude-remove": configLevelExcludeAutocompleteHandler(b),
		},
	}
}

func configBumpOnCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
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
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_bump_on_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder().AddFlags(discord.MessageFlagEphemeral)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configBumpOffCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.BumpEnabled = false
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_bump_off_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder().AddFlags(discord.MessageFlagEphemeral)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configBumpMessageConfigHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
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
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		if role, ok := event.SlashCommandInteractionData().OptRole("role"); ok {
			gd.BumpStatus.BumpRole = json.Ptr(role.ID)
			embed.SetDescription(translate.Message(event.Locale(), "config_bump_up_mention_set", translate.WithTemplate(map[string]any{"Mention": discord.RoleMention(role.ID)})))
		} else if gd.BumpStatus.BumpRole != nil {
			embed.SetDescription(translate.Message(event.Locale(), "config_bump_up_mention_remove", translate.WithTemplate(map[string]any{"Mention": discord.RoleMention(*gd.BumpStatus.BumpRole)})))
			gd.BumpStatus.BumpRole = nil
		} else {
			gd.BumpStatus.BumpRole = nil
			embed.SetDescription(translate.Message(event.Locale(), "config_bump_up_mention_none"))
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder().AddFlags(discord.MessageFlagEphemeral)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configUpOnCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
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
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_up_on_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder().AddFlags(discord.MessageFlagEphemeral)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configUpOffCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.BumpStatus.UpEnabled = false
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_up_off_description"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder().AddFlags(discord.MessageFlagEphemeral)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configUpMessageConfigHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
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
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		if role, ok := event.SlashCommandInteractionData().OptRole("role"); ok {
			gd.BumpStatus.UpRole = json.Ptr(role.ID)
			embed.SetDescription(translate.Message(event.Locale(), "config_bump_up_mention_set", translate.WithTemplate(map[string]any{"Mention": discord.RoleMention(role.ID)})))
		} else if gd.BumpStatus.UpRole != nil {
			embed.SetDescription(translate.Message(event.Locale(), "config_bump_up_mention_remove", translate.WithTemplate(map[string]any{"Mention": discord.RoleMention(*gd.BumpStatus.UpRole)})))
			gd.BumpStatus.UpRole = nil
		} else {
			gd.BumpStatus.UpRole = nil
			embed.SetDescription(translate.Message(event.Locale(), "config_bump_up_mention_none"))
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder().AddFlags(discord.MessageFlagEphemeral)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func configLevelNoticeMessageCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
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
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		if channel, ok := event.SlashCommandInteractionData().OptChannel("channel"); ok {
			gd.Config.LevelUpMessageChannel = &channel.ID
			embed.SetDescription(translate.Message(event.Locale(), "config_level_notice_channel_set", translate.WithTemplate(map[string]any{"Mention": discord.ChannelMention(channel.ID)})))
		} else {
			gd.Config.LevelUpMessageChannel = nil
			embed.SetDescription(translate.Message(event.Locale(), "config_level_notice_channel_remove"))
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder().SetFlags(discord.MessageFlagEphemeral)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func configLevelExcludeAddCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		channel := event.SlashCommandInteractionData().Channel("channel")
		gd.UserLevelExcludeChannels[channel.ID] = channel.Name
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_level_exclude_add", translate.WithTemplate(map[string]any{"Mention": discord.ChannelMention(channel.ID)})))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		if err := event.CreateMessage(message.AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func configLevelExcludeRemoveHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		channel_id, err := snowflake.Parse(event.SlashCommandInteractionData().String("channel"))
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		delete(gd.UserLevelExcludeChannels, channel_id)
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_level_exclude_remove", translate.WithTemplate(map[string]any{"Mention": discord.ChannelMention(channel_id)})))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		if err := event.CreateMessage(message.AddFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func configLevelExcludeAutocompleteHandler(b *botlib.Bot[*client.Client]) handler.AutocompleteHandler {
	return func(event *events.AutocompleteInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return err
		}
		var choices []discord.AutocompleteChoice
		for i, v := range gd.UserLevelExcludeChannels {
			choices = append(choices, discord.AutocompleteChoiceString{
				Name:  fmt.Sprintf("%s (%s)", v, i.String()),
				Value: i.String(),
			})
		}
		if err := event.Result(choices); err != nil {
			return err
		}
		return nil
	}
}

func configLevelImportMee6CommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if err := event.DeferCreateMessage(false); err != nil {
			return botlib.ReturnErr(event, err)
		}
		url := fmt.Sprintf("https://mee6.xyz/api/plugins/levels/leaderboard/%s", event.GuildID().String())
		users := map[snowflake.ID]db.GuildDataUserLevel{}
		for page := 0; true; page++ {
			c, err := http.Get(fmt.Sprintf("%s?page=%d", url, page))
			if err != nil || c.StatusCode != http.StatusOK {
				switch c.StatusCode {
				case http.StatusUnauthorized:
					_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContent(fmt.Sprintf("# FAILED\r```| STATUS CODE | %d\r| RESPONSE | %v```%s", c.StatusCode, err, translate.Message(event.Locale(), "config_import_mee6_result_unauthorized", translate.WithTemplate(map[string]any{"GuildID": *event.GuildID()})))).Build())
				default:
					_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContent(fmt.Sprintf("# FAILED\r```| STATUS CODE | %d\r| RESPONSE | %v```", c.StatusCode, err)).Build())
				}
				return err
			}
			var leaderboard db.Mee6LeaderBoard
			if err := json.NewDecoder(c.Body).Decode(&leaderboard); err != nil {
				_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContent(fmt.Sprintf("# FAILED\r```| ERROR | %s```", err)).Build())
				return err
			}
			event.Client().Logger().Info(leaderboard)
			if len(leaderboard.Players) < 1 {
				break
			}
			for _, mp := range leaderboard.Players {
				u := db.GuildDataUserLevel{
					MessageCount: mp.MessageCount,
					UserDataLevel: db.UserDataLevel{
						Point: big.NewInt(mp.Xp),
					},
				}
				users[mp.ID] = u
			}
		}
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContent(fmt.Sprintf("# FAILED\r```| ERROR | %s```", err)).Build())
			return err
		}
		gd.UserLevels = users
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContent(fmt.Sprintf("# FAILED\r```| ERROR | %s```", err)).Build())
			return err
		}
		if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContentf("# DONE\r```%d users has been imported```", len(users)).Build()); err != nil {
			_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContent(fmt.Sprintf("# FAILED\r```| ERROR | %s```", err)).Build())
			return err
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
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
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
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_bump_message_changed"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.SetFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}

func configModalUpMessageHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
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
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_up_message_changed"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.SetFlags(discord.MessageFlagEphemeral).Build()); err != nil {
			return err
		}
		return nil
	}
}
