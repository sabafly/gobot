package commands

import (
	"fmt"

	"github.com/disgoorg/json"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

func Level(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:         "level",
			Description:  "level",
			DMPermission: &b.Config.DMPermission,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "user",
					Description: "user",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "move",
							Description: "set user level",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionUser{
									Name:        "target-from",
									Description: "target user move level from",
									Required:    true,
								},
								discord.ApplicationCommandOptionUser{
									Name:        "target-to",
									Description: "target user move level to",
									Required:    true,
								},
							},
						},
						{
							Name:        "reset",
							Description: "reset user level",
							Options: []discord.ApplicationCommandOption{
								discord.ApplicationCommandOptionUser{
									Name:        "target",
									Description: "target user",
									Required:    true,
								},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "rank",
					Description: "get the user rank",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionBool{
							Name:        "all-list",
							Description: "get all of rank list",
							Required:    false,
						},
					},
				},
				discord.ApplicationCommandOptionSubCommandGroup{
					Name:        "config",
					Description: "level config",
					Options: []discord.ApplicationCommandOptionSubCommand{
						{
							Name:        "notice-message",
							Description: "set level up message",
						},
					},
				},
			},
		},
		Checks: map[string]handler.Check[*events.ApplicationCommandInteractionCreate]{
			"user/move":             b.Self.CheckCommandPermission(b, "user.level.manage", discord.PermissionManageGuild),
			"user/reset":            b.Self.CheckCommandPermission(b, "user.level.manage", discord.PermissionManageGuild),
			"config/notice-message": b.Self.CheckCommandPermission(b, "guild.config.manage", discord.PermissionManageGuild),
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"user/move":             levelUserMoveCommandHandler(b),
			"user/reset":            levelUserResetCommandHandler(b),
			"rank":                  levelRankCommandHandler(b),
			"config/notice-message": levelConfigNoticeMessageCommandHandler(b),
		},
	}
}

func levelUserMoveCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		mute := b.Self.GuildDataLock(*event.GuildID())
		if !mute.TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy")
		}
		defer mute.Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		user_from := event.SlashCommandInteractionData().User("target-from")
		user_to := event.SlashCommandInteractionData().User("target-to")
		if user_from.Bot || user_from.System || user_to.Bot || user_to.System {
			return botlib.ReturnErrMessage(event, "error_is_bot")
		}
		gd.UserLevels[user_to.ID] = gd.UserLevels[user_from.ID]
		delete(gd.UserLevels, user_from.ID)
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "command_level_move_result_embed_title"))
		embed.SetDescriptionf(
			"%s```0 lvl 0 xp```-> %s```%s lvl %s xp```",
			user_from.Mention(),
			user_to.Mention(),
			gd.UserLevels[user_to.ID].Level(),
			gd.UserLevels[user_to.ID].Point,
		)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func levelUserResetCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		if !b.Self.GuildDataLock(*event.GuildID()).TryLock() {
			return botlib.ReturnErrMessage(event, "error_busy")
		}
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		target := event.SlashCommandInteractionData().User("target")
		if target.Bot || target.System {
			return botlib.ReturnErrMessage(event, "error_is_bot")
		}
		user_level := gd.UserLevels[target.ID]
		delete(gd.UserLevels, target.ID)
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "command_level_reset_result_embed_title"))
		embed.SetDescriptionf(
			"```%s: %s lvl %s xp -> 0 lvl 0 xp```",
			target.EffectiveName(),
			user_level.Level(),
			user_level.Point,
		)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func levelRankCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.UserDataLock(event.User().ID).Lock()
		defer b.Self.UserDataLock(event.User().ID).Unlock()
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		ud, err := b.Self.DB.UserData().Get(event.User().ID)
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		embed := discord.NewEmbedBuilder()
		embed.Author = &discord.EmbedAuthor{
			Name:    event.Member().EffectiveName(),
			IconURL: event.Member().EffectiveAvatarURL(),
		}
		embed.SetTitle(translate.Translate(event.Locale(), "command_level_rank_result_embed_title", map[string]any{"User": event.Member().EffectiveName()}))
		embed.SetDescriptionf("```%-6.6s:%16.v``````%-6.6s:%16.v/%v```",
			"Level", ud.GlobalLevel.Level(),
			"Point", ud.GlobalLevel.Point, ud.GlobalLevel.SumReqPoint(),
		)
		var guild discord.Guild
		var ok bool
		if guild, ok = event.Guild(); !ok {
			g, err := b.Client.Rest().GetGuild(*event.GuildID(), true)
			if err != nil {
				return botlib.ReturnErr(event, err)
			}
			guild = g.Guild
		}
		embed.AddFields(discord.EmbedField{
			Name: guild.Name,
			Value: fmt.Sprintf("```%-6.6s:%16.v``````%-6.6s:%16.v/%v```",
				"Level", gd.UserLevels[event.User().ID].Level(),
				"Point", gd.UserLevels[event.User().ID].Point, gd.UserLevels[event.User().ID].SumReqPoint(),
			),
		})
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return err
		}
		return nil
	}
}

func levelConfigNoticeMessageCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
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

func LevelModal(b *botlib.Bot[*client.Client]) handler.Modal {
	return handler.Modal{
		Name: "level",
		Handler: map[string]handler.ModalHandler{
			"notice-message": levelModalNoticeMessageHandler(b),
		},
	}
}

func levelModalNoticeMessageHandler(b *botlib.Bot[*client.Client]) handler.ModalHandler {
	return func(event *events.ModalSubmitInteractionCreate) error {
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		gd.Config.LevelUpMessage = event.Data.Text("message")
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		if err := event.DeferUpdateMessage(); err != nil {
			return err
		}
		return nil
	}
}
