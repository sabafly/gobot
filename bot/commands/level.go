package commands

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
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
					Name:        "leaderboard",
					Description: "show guild member leaderboard",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionInt{
							Name:        "page",
							Description: "page number",
							MinValue:    json.Ptr(1),
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "point",
					Description: "show yourself point",
				},
			},
		},
		Checks: map[string]handler.Check[*events.ApplicationCommandInteractionCreate]{
			"user/move":  b.Self.CheckCommandPermission(b, "user.level.manage", discord.PermissionManageGuild),
			"user/reset": b.Self.CheckCommandPermission(b, "user.level.manage", discord.PermissionManageGuild),
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"user/move":   levelUserMoveCommandHandler(b),
			"user/reset":  levelUserResetCommandHandler(b),
			"point":       levelPointCommandHandler(b),
			"leaderboard": levelLeaderBoard(b),
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
		tmp := gd.UserLevels[user_from.ID]
		tmp.Point = big.NewInt(0)
		gd.UserLevels[user_from.ID] = tmp
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "command_level_move_result_embed_title"))
		ul := gd.UserLevels[user_to.ID]
		embed.SetDescriptionf(
			"%s```0 lvl 0 xp```-> %s```%s lvl %s xp```",
			user_from.Mention(),
			user_to.Mention(),
			ul.Level().String(),
			ul.Point.String(),
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
		ul := gd.UserLevels[target.ID]
		ul.Point = big.NewInt(0)
		gd.UserLevels[target.ID] = ul
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return botlib.ReturnErr(event, err)
		}
		message := discord.NewMessageCreateBuilder()
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "command_level_reset_result_embed_title"))
		embed.SetDescriptionf(
			"```%s: %s lvl %s xp -> 0 lvl 0 xp```",
			target.EffectiveName(),
			user_level.Level().String(),
			user_level.Point.String(),
		)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			return botlib.ReturnErr(event, err)
		}
		return nil
	}
}

func levelPointCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
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
		embed.SetTitle(translate.Message(event.Locale(), "command_level_rank_result_embed_title", translate.WithTemplate(map[string]any{"User": event.Member().EffectiveName()})))
		embed.SetDescriptionf("```%-6.6s:%16s``````%-6.6s:%16s/%s```",
			"Level", ud.GlobalLevel.Level().String(),
			"Point", ud.GlobalLevel.Point.String(), ud.GlobalLevel.SumReqPoint().String(),
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
		ul, ok := gd.UserLevels[event.User().ID]
		if !ok {
			ul = db.NewGuildDataUserLevel()
		}
		embed.AddFields(discord.EmbedField{
			Name: guild.Name,
			Value: fmt.Sprintf("```%-6.6s:%16s``````%-6.6s:%16s/%v```",
				"Level", ul.Level().String(),
				"Point", ul.Point.String(), ul.SumReqPoint().String(),
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

func levelLeaderBoard(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		b.Self.UserDataLock(event.User().ID).Lock()
		defer b.Self.UserDataLock(event.User().ID).Unlock()
		b.Self.GuildDataLock(*event.GuildID()).Lock()
		defer b.Self.GuildDataLock(*event.GuildID()).Unlock()
		gd, err := b.Self.DB.GuildData().Get(*event.GuildID())
		if err != nil {
			return botlib.ReturnErr(event, err)
		}
		type sortLevel struct {
			user_id snowflake.ID
			level   db.UserDataLevel
		}
		sort_list := []sortLevel{}
		for id, level := range gd.UserLevels {
			sort_list = append(sort_list, sortLevel{
				user_id: id,
				level:   level.UserDataLevel,
			})
		}

		page_number, ok := event.SlashCommandInteractionData().OptInt("page")
		if !ok || page_number < 1 {
			page_number = 1
		}

		max_page := len(sort_list)/25 + 1

		if len(sort_list) < 1 || max_page < page_number {
			return botlib.ReturnErrMessage(event, "error_unavailable_page")
		}

		slices.SortFunc(sort_list, func(a, b sortLevel) int {
			return a.level.Point.Cmp(b.level.Point)
		})
		slices.Reverse(sort_list)

		sort_list = sort_list[25*(page_number-1) : min(25*page_number, len(sort_list))]

		var text_list_string string
		for i, sl := range sort_list {
			text_list_string += fmt.Sprintf(
				"**#%d | ** %s **XP:** `%s` **Level:** `%s`\r",
				(25*(page_number-1))+(i+1), discord.UserMention(sl.user_id), sl.level.Point.String(), sl.level.Level().String(),
			)
		}

		embed := discord.NewEmbedBuilder()
		embed.SetTitlef("💬%s(%d/%d)", translate.Message(event.Locale(), "level_leader_board_category_text"), page_number, max_page)
		embed.SetDescription(text_list_string)
		embed.SetAuthorNamef("🏆%s", translate.Message(event.Locale(), "level_leader_board_author_text"))
		if guild, ok := event.Guild(); ok && guild.Icon != nil {
			embed.SetAuthorIcon(*guild.IconURL())
		}
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message := discord.NewMessageCreateBuilder()
		message.AddEmbeds(embed.Build())
		if err := event.CreateMessage(message.Build()); err != nil {
			buf, _ := json.Marshal(message.Build())
			b.Logger.Debug(string(buf), len(sort_list))
			return botlib.ReturnErr(event, err)
		}
		b.Logger.Debug(len(sort_list))
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
		embed := discord.NewEmbedBuilder()
		embed.SetTitle(translate.Message(event.Locale(), "config_changed"))
		embed.SetDescription(translate.Message(event.Locale(), "config_level_notice_message_changed"))
		if err := event.DeferUpdateMessage(); err != nil {
			return err
		}
		return nil
	}
}
