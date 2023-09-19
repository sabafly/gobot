package commands

import (
	"fmt"
	"strings"

	"slices"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/handler/interactions"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Message(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "message",
			Description:  "message",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "pin",
					Description: "pin",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "create",
							Description:              "create pinned message",
							DescriptionLocalizations: translate.MessageMap("message_pin_create_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionBool{
									Name:                     "use-embed",
									Description:              "wither uses embed creator",
									DescriptionLocalizations: translate.MessageMap("message_pin_create_command_user_embed_option_description", false),
									Required:                 false,
								},
							},
						},
						{
							Name:                     "delete",
							Description:              "delete pinned message",
							DescriptionLocalizations: translate.MessageMap("message_pin_delete_command_description", false),
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "suffix",
					Description: "suffix",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:                     "set",
							Description:              "set user message suffix",
							DescriptionLocalizations: translate.MessageMap("message_suffix_set_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionUser{
									Name:                     "target",
									Description:              "target user",
									DescriptionLocalizations: translate.MessageMap("message_suffix_set_command_target_option_description", false),
									Required:                 true,
								},
								discord.ApplicationCommandOptionString{
									Name:                     "suffix",
									Description:              "suffix text",
									DescriptionLocalizations: translate.MessageMap("message_suffix_set_command_suffix_option_description", false),
									Required:                 true,
								},
								discord.ApplicationCommandOptionInt{
									Name:                     "rule-type",
									Description:              "force suffix rule type",
									DescriptionLocalizations: translate.MessageMap("message_suffix_set_command_rule_type_option_description", false),
									Required:                 true,
									Choices: []discord.ApplicationCommandOptionChoiceInt{
										{
											Name:              "send warning message",
											NameLocalizations: translate.MessageMap("message_suffix_set_command_rule_type_option_send_warn", false),
											Value:             db.MessageSuffixRuleTypeWarning,
										},
										{
											Name:              "delete message",
											NameLocalizations: translate.MessageMap("message_suffix_set_command_rule_type_option_delete", false),
											Value:             db.MessageSuffixRuleTypeDelete,
										},
										{
											Name:              "webhook replace",
											NameLocalizations: translate.MessageMap("message_suffix_set_command_rule_type_webhook", false),
											Value:             db.MessageSuffixRuleTypeWebhook,
										},
									},
								},
							},
						},
						{
							Name:                     "remove",
							Description:              "remove suffix rule from target",
							DescriptionLocalizations: translate.MessageMap("message_suffix_remove_command_description", false),
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionUser{
									Name:                     "target",
									Description:              "target user",
									DescriptionLocalizations: translate.MessageMap("message_suffix_remove_command_target_option_description", false),
									Required:                 true,
								},
							},
						},
					},
				},
			},
		},
		Check: b.Self.CheckCommandPermission(b, "message.manage", discord.PermissionManageChannels.Add(discord.PermissionManageMessages)),
		CommandHandlers: map[string]handler.CommandHandler{
			"pin/create":    messagePinCreateCommandHandler(b),
			"pin/delete":    messagePinDeleteCommandHandler(b),
			"suffix/set":    messageSuffixSetCommandHandler(b),
			"suffix/remove": messageSuffixRemoveCommandHandler(b),
		},
	}
}

func messagePinCreateCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if event.SlashCommandInteractionData().Bool("use-embed") {
			interaction_token := interactions.New(event.Token(), event.ID().Time())
			embed_dialog := db.NewEmbedDialog("message:p-e-create", interaction_token, event.Locale())
			if err := b.Self.DB.EmbedDialog().Set(embed_dialog.ID, *embed_dialog); err != nil {
				return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
			}
			embed_dialog.SetDescription("default message")
			if err := event.DeferCreateMessage(true); err != nil {
				return botlib.ReturnErr(event, err)
			}
			if _, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), embed_dialog.BaseMenu()); err != nil {
				return err
			}
			return nil
		} else {
			if err := event.CreateModal(discord.ModalCreate{
				Title:    translate.Message(event.Locale(), "command_message_pin_create_modal_title"),
				CustomID: "handler:message:pin-create",
				Components: []discord.ContainerComponent{
					discord.NewActionRow(
						discord.TextInputComponent{
							CustomID:    "content",
							Style:       discord.TextInputStyle(discord.TextInputStyleParagraph),
							Label:       translate.Message(event.Locale(), "command_message_pin_create_modal_action_row_0_label"),
							MaxLength:   2000,
							Placeholder: translate.Message(event.Locale(), "command_message_create_modal_action_row_0_placeholder"),
							Required:    true,
						},
					),
				},
			}); err != nil {
				return botlib.ReturnErr(event, err)
			}
			return nil
		}
	}
}

func messagePinDeleteCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if !b.Self.DB.MessagePin().Mu().TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy", botlib.WithEphemeral(true))
		}
		defer b.Self.DB.MessagePin().Mu().Unlock()
		m, ok := b.Self.MessagePin[*event.GuildID()]
		if !ok {
			return botlib.ReturnErrMessage(event, "error_not_found", botlib.WithEphemeral(true))
		}
		mp, ok := m.Pins[event.Channel().ID()]
		b.Logger.Debug(*event.GuildID(), event.Channel().ID())
		if !ok {
			return botlib.ReturnErrMessage(event, "error_not_found", botlib.WithEphemeral(true))
		}
		if mp.LastMessageID != nil {
			_ = event.Client().Rest().DeleteMessage(mp.ChannelID, *mp.LastMessageID)
		}
		delete(m.Pins, event.Channel().ID())
		if err := b.Self.DB.MessagePin().Set(*event.GuildID(), m); err != nil {
			return botlib.ReturnErr(event, err, botlib.WithEphemeral(true))
		}
		m.Pins[event.Channel().ID()] = mp
		b.Self.MessagePin[*event.GuildID()] = m
		embed := discord.NewEmbedBuilder()
		embed.SetDescription(translate.Message(event.Locale(), "message_pin_delete"))
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		return event.CreateMessage(message.SetFlags(discord.MessageFlagEphemeral).Build())
	}
}

func messageSuffixSetCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		target := event.SlashCommandInteractionData().User("target")
		if target.Bot || target.System {
			return botlib.ReturnErrMessage(event, "error_is_bot")
		}
		suffix_string := event.SlashCommandInteractionData().String("suffix")
		suffix_type := event.SlashCommandInteractionData().Int("rule-type")
		suffix := db.NewMessageSuffix(target.ID, suffix_string, db.MessageSuffixRuleType(suffix_type))
		gd.MessageSuffix[target.ID] = suffix
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		message.SetContentf("%sの語尾を「%s」に強制します", target.Mention(), suffix_string)
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func messageSuffixRemoveCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.DB.GuildData().Mu(*event.GuildID()).Lock()
		defer b.Self.DB.GuildData().Mu(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		target := event.SlashCommandInteractionData().User("target")
		if _, ok := gd.MessageSuffix[target.ID]; !ok {
			return botlib.ReturnErrMessage(event, "error_already_deleted")
		}
		delete(gd.MessageSuffix, target.ID)
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		message.SetContentf("%sの語尾を解除しました", target.Mention())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func MessageComponent(b *botlib.Bot[*client.Client]) handler.Component {
	return handler.Component{
		Name: "message",
		Handler: map[string]handler.ComponentHandler{
			"p-e-create": messageComponentPECreate(b),
		},
	}
}

func messageComponentPECreate(b *botlib.Bot[*client.Client]) handler.ComponentHandler {
	return func(event *events.ComponentInteractionCreate) error {
		args := strings.Split(event.Data.CustomID(), ":")
		ed_id, err := uuid.Parse(args[3])
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_invalid_id")
		}
		ed, err := b.Self.DB.EmbedDialog().Get(ed_id)
		if err != nil {
			return botlib.ReturnErrMessage(event, "error_timeout")
		}
		mp, err := b.Self.DB.MessagePin().Get(*event.GuildID())
		if err != nil {
			mp = db.NewMessagePin()
		}
		if token, err := ed.InteractionToken.Get(); err == nil {
			_ = event.Client().Rest().DeleteInteractionResponse(event.ApplicationID(), token)
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err)
		}
		wmc := discord.WebhookMessageCreate{
			Embeds:    []discord.Embed{ed.SetColor(botlib.Color).Build()},
			Username:  translate.Message(event.Locale(), "command_message_pin_create_pinned_message"),
			AvatarURL: b.Self.Config.MessagePinAvatarURL,
		}
		m, err := botlib.SendWebhook(event.Client(), event.Channel().ID(), wmc)
		if err != nil {
			return err
		}
		mp.Pins[event.Channel().ID()] = db.MessagePin{
			WebhookMessageCreate: wmc,
			ChannelID:            m.ChannelID,
			LastMessageID:        &m.ID,
		}
		if err := b.Self.DB.MessagePin().Set(*event.GuildID(), mp); err != nil {
			return err
		}
		b.Self.MessagePin[*event.GuildID()] = mp
		return nil
	}
}

func MessageModal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "message",
		Handler: map[string]handler.ModalHandler{
			"pin-create": messageModalPinCreate(b),
		},
	}
}

func messageModalPinCreate(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		content := event.ModalSubmitInteraction.Data.Text("content")
		mp, err := b.Self.DB.MessagePin().Get(*event.GuildID())
		if err != nil {
			mp = db.NewMessagePin()
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return botlib.ReturnErr(event, err)
		}
		wmc := discord.WebhookMessageCreate{
			Content:   content,
			Username:  translate.Message(event.Locale(), "command_message_pin_create_pinned_message"),
			AvatarURL: b.Self.Config.MessagePinAvatarURL,
		}
		m, err := botlib.SendWebhook(event.Client(), event.Channel().ID(), wmc)
		if err != nil {
			return err
		}
		mp.Pins[event.Channel().ID()] = db.MessagePin{
			WebhookMessageCreate: wmc,
			ChannelID:            m.ChannelID,
			LastMessageID:        &m.ID,
		}
		if err := b.Self.DB.MessagePin().Set(*event.GuildID(), mp); err != nil {
			return err
		}
		b.Self.MessagePin[*event.GuildID()] = mp
		return nil
	}
}

func MessagePinMessageCreateHandler(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		Handler: func(event *events.GuildMessageCreate) error {
			if !b.Self.DB.MessagePin().Mu().TryLock() {
				return nil
			}
			defer b.Self.DB.MessagePin().Mu().Unlock()
			m, ok := b.Self.MessagePin[event.GuildID]
			if !ok || !m.Enabled {
				return nil
			}
			mp, ok := m.Pins[event.ChannelID]
			if !ok {
				return nil
			}
			if mp.CheckLimit() {
				id, _, err := botlib.GetWebhook(event.Client(), event.ChannelID)
				if err != nil {
					b.Logger.Error(err)
					return err
				}
				if event.Message.WebhookID != nil && id == *event.Message.WebhookID {
					return nil
				}
				if err := mp.Update(event.Client()); err != nil {
					return err
				}
			}
			m.Pins[event.ChannelID] = mp
			b.Self.MessagePin[event.GuildID] = m
			if err := b.Self.DB.MessagePin().Set(event.GuildID, m); err != nil {
				return err
			}
			return nil
		},
	}
}

func MessageSuffixMessageCreateHandler(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		Handler: func(event *events.GuildMessageCreate) error {
			if event.Message.Author.Bot || event.Message.Author.System || event.Message.Type.System() || !event.Message.Type.Deleteable() {
				return nil
			}
			if event.Message.Type != discord.MessageTypeDefault && event.Message.Type != discord.MessageTypeReply {
				return nil
			}
			b.Self.DB.GuildData().Mu(event.GuildID).Lock()
			defer b.Self.DB.GuildData().Mu(event.GuildID).Unlock()
			gd, err := b.Self.DB.GuildData().Get(event.GuildID)
			if err != nil {
				return err
			}
			suffix, ok := gd.MessageSuffix[event.Message.Author.ID]
			if !ok {
				return nil
			}
			has_suffix := strings.HasSuffix(event.Message.Content, suffix.Suffix)
			switch suffix.RuleType {
			case db.MessageSuffixRuleTypeWarning:
				if has_suffix {
					break
				}
				message := discord.NewMessageCreateBuilder()
				message.SetContent(fmt.Sprintf("語尾がついてないよ！\r「%s」を忘れないで(笑)", suffix.Suffix))
				message.SetAllowedMentions(&discord.AllowedMentions{
					RepliedUser: true,
				})
				message.SetMessageReferenceByID(event.MessageID)
				if _, err := event.Client().Rest().CreateMessage(event.ChannelID, message.Build()); err != nil {
					return err
				}
			case db.MessageSuffixRuleTypeDelete:
				if has_suffix {
					break
				}
				if err := event.Client().Rest().DeleteMessage(event.ChannelID, event.MessageID); err != nil {
					return err
				}
			case db.MessageSuffixRuleTypeWebhook:
				if !has_suffix {
					event.Message.Content += suffix.Suffix
				}
				if err := event.Client().Rest().DeleteMessage(event.ChannelID, event.MessageID); err != nil {
					return err
				}
				message := discord.NewWebhookMessageCreateBuilder()
				message.Content = event.Message.Content
				message.SetAvatarURL(event.Message.Member.EffectiveAvatarURL())
				message.SetUsername(event.Message.Author.EffectiveName())
				mention_users := make([]snowflake.ID, len(event.Message.Mentions))
				for i, u := range event.Message.Mentions {
					mention_users[i] = u.ID
				}
				replied_user := false
				if event.Message.MessageReference != nil && event.Message.MessageReference.ChannelID != nil && event.Message.MessageReference.MessageID != nil {
					reply_message, err := event.Client().Rest().GetMessage(*event.Message.MessageReference.ChannelID, *event.Message.MessageReference.MessageID)
					if err == nil {
						replied_user = slices.Index(mention_users, reply_message.Author.ID) != -1
					}
				}
				message.SetAllowedMentions(&discord.AllowedMentions{
					Users:       mention_users,
					Roles:       event.Message.MentionRoles,
					RepliedUser: replied_user,
				})

				// うーーん むりぽ¯\_(ツ)_/¯

				if _, err := botlib.SendWebhook(event.Client(), event.ChannelID, message.Build()); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
