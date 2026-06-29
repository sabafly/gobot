package gopoint

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "gopoint",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "gopoint",
				Description: "GoPointの情報を表示します。",
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				IntegrationTypes: []discord.ApplicationIntegrationType{
					discord.ApplicationIntegrationTypeGuildInstall,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "view",
						Description:              "GoPointのランキングを表示します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.view.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "ranking",
						Description:              "GoPointのランキングを表示します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.ranking.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:                     "limit",
								Description:              "ランキングの表示件数を指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.ranking.limit.description"),
								Required:                 false,
								MinValue:                 builtin.Ptr(1),
								MaxValue:                 builtin.Ptr(100),
							},
							discord.ApplicationCommandOptionInt{
								Name:                     "skip",
								Description:              "ランキングのスキップ件数を指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.ranking.skip.description"),
								Required:                 false,
								MinValue:                 builtin.Ptr(0),
								MaxValue:                 builtin.Ptr(1000),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "give",
						Description:              "他のユーザーにGoPointを渡します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.give.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionUser{
								Name:                     "target_user",
								Description:              "GoPointを渡す対象のユーザーを指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.give.target_user.description"),
								Required:                 true,
							},
							discord.ApplicationCommandOptionInt{
								Name:                     "point",
								Description:              "渡すGoPointの数を指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.give.point.description"),
								Required:                 true,
								MinValue:                 builtin.Ptr(1),
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/gopoint/view": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("gopoint.view"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.DeferCreateMessage(true); err != nil {
						return errors.NewError(err)
					}
					point, rank, err := GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						slog.Error("failed to get GoPoint", "error", err, "user_id", event.User().ID, "guild_id", *event.GuildID())
						return errors.NewError(err)
					}
					slog.Info("GoPoint retrieved", "user_id", event.User().ID, "guild_id", *event.GuildID(), "point", point, "rank", rank)
					userID := event.User().ID
					const top = 10
					ranking, err := inGuildRanking(c, event.Locale(), event.Client(), *event.GuildID(), builtin.Or(rank < top, top, 3), 0, &userID)
					if err != nil {
						slog.Error("failed to get guild ranking", "error", err)
						return errors.NewError(err)
					}
					if rank > top {
						r, err := inGuildRanking(c, event.Locale(), event.Client(), *event.GuildID(), 7, rank-3, &userID)
						if err != nil {
							slog.Error("failed to get guild ranking for rank > 10", "error", err)
							return errors.NewError(err)
						}
						ranking += "`...`\n" + r
					} else if rank == 0 {
						ranking = i18n.TranslateText(event.Locale(), "command.gopoint.ranking.no_data")
					}
					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetEphemeral(true).
						SetIsComponentsV2(true).
						SetComponents(
							i18n.BuildContext().
								WithText("ranking", ranking).
								WithText("user_name", event.Member().EffectiveName()).
								WithURL("user_icon", event.Member().EffectiveAvatarURL()).
								WithText("point", fmt.Sprintf("%d", point)).
								WithText("rank", fmt.Sprintf("%d", rank)).
								Translate(i18n.TranslateLayout(event.Locale(), "command.gopoint.view"))...,
						),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/gopoint/ranking": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("gopoint.ranking"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.DeferCreateMessage(false); err != nil {
						return errors.NewError(err)
					}
					limit := 10
					skip := 0
					if option, ok := event.SlashCommandInteractionData().OptInt("limit"); ok {
						limit = option
					}
					if option, ok := event.SlashCommandInteractionData().OptInt("skip"); ok {
						skip = option
					}
					ranking, err := inGuildRanking(c, event.Locale(), event.Client(), *event.GuildID(), limit, skip, nil)
					if err != nil {
						slog.Error("failed to get guild ranking", "error", err)
						return errors.NewError(err)
					}
					if skip == 0 {
						ranking = "### " + ranking
					}
					guild, err := c.GuildRequest(event.Client(), *event.GuildID())
					if err != nil {
						slog.Error("failed to get guild", "error", err, "guild_id", *event.GuildID())
						return errors.NewError(err)
					}
					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(
							i18n.BuildContext().
								WithText("skip", strconv.Itoa(skip+1)).
								WithText("limit", strconv.Itoa(limit)).
								WithText("guild_name", guild.Name).
								WithText("ranking", ranking).
								Translate(i18n.TranslateLayout(event.Locale(), "command.gopoint.ranking"))...,
						),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/gopoint/give": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("gopoint.give"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					targetUserID, ok := event.SlashCommandInteractionData().OptUser("target_user")
					if !ok {
						return errors.NewError(fmt.Errorf("target user is required"))
					}
					targetUser, err := event.Client().Rest.GetUser(targetUserID.ID)
					if err != nil {
						slog.Error("failed to get target user", "error", err, "target_user_id", targetUserID.ID)
						return errors.NewError(err)
					}
					if targetUser.ID == event.User().ID {
						return errors.NewError(errors.ErrorMessage("error.gopoint.give.self", event))
					}
					if targetUser.Bot || targetUser.System {
						return errors.NewError(errors.ErrorMessage("error.gopoint.give.invalid_target", event))
					}
					point, ok := event.SlashCommandInteractionData().OptInt("point")
					if !ok || point <= 0 {
						return errors.NewError(errors.ErrorMessage("error.gopoint.give.invalid_point", event))
					}
					currentPoint, _, err := GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						slog.Error("failed to get GoPoint", "error", err, "user_id", event.User().ID, "guild_id", *event.GuildID())
						return errors.NewError(err)
					}
					if currentPoint < int64(point) {
						if err := event.CreateMessage(discord.NewMessageBuilder().
							SetEphemeral(true).
							SetIsComponentsV2(true).
							SetComponents(
								i18n.BuildContext().
									WithText("target_user_name", targetUserID.EffectiveName()).
									WithText("point", fmt.Sprintf("%d", point)).
									WithText("current_point", fmt.Sprintf("%d", currentPoint)).
									Translate(i18n.TranslateLayout(event.Locale(), "command.gopoint.give.not_enough_point"))...,
							).BuildCreate(),
						); err != nil {
							slog.Error("failed to create message for not enough points", "error", err, "user_id", event.User().ID, "guild_id", *event.GuildID())
							return errors.NewError(err)
						}
						return nil
					}
					if err := GivePoint(c, event.User().ID, targetUserID.ID, *event.GuildID(), int64(point)); err != nil {
						slog.Error("failed to give GoPoint", "error", err, "user_id", event.User().ID, "target_user_id", targetUserID.ID, "guild_id", *event.GuildID())
						return errors.NewError(err)
					}
					if err := event.CreateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(
							i18n.BuildContext().
								WithText("target_user_name", targetUserID.EffectiveName()).
								WithText("point", fmt.Sprintf("%d", point)).
								Translate(i18n.TranslateLayout(event.Locale(), "command.gopoint.give.success"))...,
						).BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		EventHandler: func(c *components.Components, event bot.Event) errors.Error {
			switch e := event.(type) {
			case *events.GuildMessageCreate:
				if e.Message.Author.Bot || e.Message.Author.System {
					return nil // Ignore bot messages
				}
				if e.Message.Type != discord.MessageTypeDefault &&
					e.Message.Type != discord.MessageTypeReply {
					return nil // Ignore non-default messages
				}
				if rand.Float64() < 0.1 {
					if err := AddPoint(c, e.Message.Author.ID, e.GuildID, 10); err != nil {
						slog.Error("failed to add GoPoint", "error", err, "user_id", e.Message.Author.ID, "guild_id", e.GuildID)
						return errors.NewError(err)
					}
				}
			}
			return nil
		},
	}).SetComponent(c)
}

func inGuildRanking(c *components.Components, locale discord.Locale, client *bot.Client, guildID snowflake.ID, limit int, skip int, userID *snowflake.ID) (string, error) {
	var guild *models.Guild
	if err := c.GormDB().Where("id = ?", guildID).First(&guild).Error; err != nil {
		return "", fmt.Errorf("failed to find guild: %w", err)
	} // SELECT RANK() OVER (ORDER BY points DESC) AS rank
	var rankQuery []struct {
		UserID snowflake.ID `gorm:"column:user_id"`
		Rank   int          `gorm:"column:rank"`
		Points int64        `gorm:"column:points"`
	}
	if err := c.GormDB().Model(&models.GoPoint{}).
		Select("user_id", "RANK() OVER (ORDER BY points DESC) AS rank", "points").
		Where("guild_id = ?", guildID).
		Limit(limit).
		Offset(skip).
		Scan(&rankQuery).Error; err != nil {
		return "", fmt.Errorf("failed to find ranking: %w", err)
	}
	if len(rankQuery) == 0 && skip == 0 {
		return i18n.TranslateText(locale, "command.gopoint.ranking.no_guild_data"), nil
	} else if len(rankQuery) == 0 {
		return i18n.TranslateText(locale, "command.gopoint.ranking.no_result"), nil
	}
	var result strings.Builder
	for _, point := range rankQuery {
		user, err := client.Rest.GetMember(guildID, point.UserID)
		if err != nil {
			slog.Error("failed to get user for ranking", "user_id", point.UserID, "error", err)
			user = nil
		}
		name := fmt.Sprintf("<@%d>", point.UserID)
		if user != nil {
			name = fmt.Sprintf("**%s**", user.EffectiveName())
		}
		text := i18n.BuildContext().
			WithText("rank", fmt.Sprintf("%d", point.Rank)).
			WithText("name", name).
			WithText("points", fmt.Sprintf("%d", point.Points)).
			ReplaceText(i18n.TranslateText(locale, "command.gopoint.ranking.entry"))
		if userID != nil && point.UserID == *userID {
			text += " " + i18n.TranslateText(locale, "command.gopoint.ranking.entry.you")
		}
		result.WriteString(text + "\n")
	}
	return result.String(), nil
}

func AddPoint(c *components.Components, userID snowflake.ID, guildID snowflake.ID, point int64) error {
	user, err := database.GetOrCreateUser(c.GormDB(), userID)
	if err != nil {
		slog.Error("failed to find or create user", "error", err, "user_id", userID)
		return errors.NewError(err)
	}
	userPoint := models.GoPoint{
		UserID:  user.ID,
		GuildID: guildID,
		Points:  0,
	}
	if err := c.GormDB().Where(userPoint).Find(&userPoint).Error; err != nil {
		slog.Error("failed to find or create user point", "error", err, "user_id", userID, "guild_id", guildID)
		return errors.NewError(err)
	}
	userPoint.Points += point // Increment points for each message
	if err := c.GormDB().Save(&userPoint).Error; err != nil {
		slog.Error("failed to update user point", "error", err, "user_id", userID, "guild_id", guildID)
		return errors.NewError(err)
	}
	return nil
}

func GivePoint(c *components.Components, userID snowflake.ID, targetUserID snowflake.ID, guildID snowflake.ID, point int64) error {
	if userID == targetUserID {
		return errors.NewError(fmt.Errorf("cannot give points to yourself"))
	}
	user, err := database.GetOrCreateUser(c.GormDB(), userID)
	if err != nil {
		slog.Error("failed to find or create user", "error", err, "user_id", userID)
		return errors.NewError(err)
	}
	targetUser, err := database.GetOrCreateUser(c.GormDB(), targetUserID)
	if err != nil {
		slog.Error("failed to find or create target user", "error", err, "target_user_id", targetUserID)
		return errors.NewError(err)
	}
	userPoint := models.GoPoint{
		UserID:  user.ID,
		GuildID: guildID,
	}
	if err := c.GormDB().Where(userPoint).FirstOrInit(&userPoint).Error; err != nil {
		slog.Error("failed to find user point", "error", err, "user_id", userID, "guild_id", guildID)
		return errors.NewError(err)
	}
	if userPoint.Points < point {
		slog.Error("not enough points to give", "user_id", userID, "target_user_id", targetUser.ID, "guild_id", guildID, "points", point)
		return errors.NewError(fmt.Errorf("not enough points to give"))
	}
	userPoint.Points -= point // Deduct points from the giver
	if err := c.GormDB().Save(&userPoint).Error; err != nil {
		slog.Error("failed to update user point", "error", err, "user_id", userID, "guild_id", guildID)
		return errors.NewError(err)
	}
	targetUserPoint := models.GoPoint{
		UserID:  targetUserID,
		GuildID: guildID,
	}
	if err := c.GormDB().Where(targetUserPoint).FirstOrInit(&targetUserPoint).Error; err != nil {
		slog.Error("failed to find target user point", "error", err, "target_user_id", targetUserID, "guild_id", guildID)
		return errors.NewError(err)
	}
	targetUserPoint.Points += point // Add points to the target user
	if err := c.GormDB().Save(&targetUserPoint).Error; err != nil {
		slog.Error("failed to update target user point", "error", err, "target_user_id", targetUserID, "guild_id", guildID)
		return errors.NewError(err)
	}
	slog.Info("points given successfully", "user_id", userID, "target_user_id", targetUserID, "guild_id", guildID, "points", point)
	return nil
}

func GetPoint(c *components.Components, userID snowflake.ID, guildID snowflake.ID) (point int64, rank int, err error) {
	user := models.User{
		ID: userID,
	}
	if err := c.GormDB().Where(user).FirstOrCreate(&user).Error; err != nil {
		slog.Error("failed to find or create user", "error", err, "user_id", userID)
		return 0, 0, err
	}
	userPoint := models.GoPoint{
		UserID:  userID,
		GuildID: guildID,
	}
	if err := c.GormDB().Where(userPoint).FirstOrInit(&userPoint).Error; err != nil {
		slog.Error("failed to find user point", "error", err, "user_id", userID, "guild_id", guildID)
		return 0, 0, err
	}
	point = userPoint.Points

	// SELECT RANK() OVER (ORDER BY points DESC) AS rank
	var rankQuery struct {
		UserID snowflake.ID `gorm:"column:user_id"`
		Rank   int          `gorm:"column:rank"`
		Points int64        `gorm:"column:points"`
	}
	if err := c.GormDB().Table("(?) as u", c.GormDB().Model(&models.GoPoint{}).
		Select("user_id", "RANK() OVER (ORDER BY points DESC) AS rank", "points").
		Where("guild_id = ?", guildID)).
		Where("user_id = ?", userID).
		Find(&rankQuery).Error; err != nil {
		slog.Error("failed to get user rank", "error", err, "user_id", userID, "guild_id", guildID)
		return 0, 0, err
	}
	return point, rankQuery.Rank, nil
}
