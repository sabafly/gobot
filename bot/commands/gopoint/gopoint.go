package gopoint

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

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
	"gorm.io/gorm"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "gopoint",
		Schedulers: []components.Scheduler{
			{
				Duration: time.Minute,
				Worker: func(c *components.Components, client *bot.Client) error {
					return ProcessBackgroundTasks(c, client)
				},
			},
		},
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
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "tax-setup",
						Description:              "定期徴収の割合や間隔を設定します。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.tax-setup.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "tax-status",
						Description:              "現在の定期徴収の設定状況と徴収予定を表示します。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.tax-status.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "tax-force",
						Description:              "今すぐポイントの徴収（または徴収の予約計算）を強制実行します。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.tax-force.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "reset",
						Description:              "ギルド内の全員のGoPointsを一括リセットします。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.reset.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:        "points",
								Description: "リセット後のポイント数を指定します (デフォルト: 0)。",
								Required:    false,
								MinValue:    builtin.Ptr(0),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "season-start",
						Description:              "新しいシーズンを開始、またはスケジュールします。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.season-start.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "name",
								Description: "シーズンの名前を指定します。",
								Required:    true,
							},
							discord.ApplicationCommandOptionInt{
								Name:        "duration_days",
								Description: "シーズンの期間(日)を指定します (1-365)。",
								Required:    true,
								MinValue:    builtin.Ptr(1),
								MaxValue:    builtin.Ptr(365),
							},
							discord.ApplicationCommandOptionInt{
								Name:        "start_delay_hours",
								Description: "何時間後にシーズンを開始するか（スケジュール予約）を指定します。",
								Required:    false,
								MinValue:    builtin.Ptr(0),
							},
							discord.ApplicationCommandOptionString{
								Name:        "criteria",
								Description: "ランキングの基準を指定します (earned: 獲得ポイント, final: 最終ポイント)。",
								Required:    false,
								Choices: []discord.ApplicationCommandOptionChoiceString{
									{
										Name:  "獲得ポイント数",
										Value: "earned",
									},
									{
										Name:  "最終所持ポイント数",
										Value: "final",
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "season-end",
						Description:              "アクティブなシーズンを途中で終了し、表彰を行います。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.season-end.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "season-status",
						Description:              "現在のアクティブなシーズンの進捗と現在のランキングを表示します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.season-status.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "season-ranking",
						Description:              "アクティブなシーズン、または過去のシーズンのランキングを表示します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.gopoint.season-ranking.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "season_id",
								Description: "表示したい過去のシーズンのIDを指定します (省略時は現在のアクティブなシーズン)。",
								Required:    false,
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
			"/gopoint/tax-setup": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("gopoint.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return TaxSetupHandler(c, event)
				},
			},
			"/gopoint/tax-status": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("gopoint.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return TaxStatusHandler(c, event)
				},
			},
			"/gopoint/tax-force": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("gopoint.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return TaxForceHandler(c, event)
				},
			},
			"/gopoint/reset": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("gopoint.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return ResetPointsHandler(c, event)
				},
			},
			"/gopoint/season-start": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("gopoint.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonStartHandler(c, event)
				},
			},
			"/gopoint/season-end": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("gopoint.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonEndHandler(c, event)
				},
			},
			"/gopoint/season-status": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("gopoint.view"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonStatusHandler(c, event)
				},
			},
			"/gopoint/season-ranking": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("gopoint.view"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonRankingHandler(c, event)
				},
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"gopoint:tax_setup_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				return TaxSetupModalSubmitHandler(c, event)
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

func AddPointTx(tx *gorm.DB, userID snowflake.ID, guildID snowflake.ID, point int64) error {
	user, err := database.GetOrCreateUser(tx, userID)
	if err != nil {
		slog.Error("failed to find or create user", "error", err, "user_id", userID)
		return errors.NewError(err)
	}

	// Attempt atomic update first
	result := tx.Model(&models.GoPoint{}).
		Where("user_id = ? AND guild_id = ?", user.ID, guildID).
		Update("points", gorm.Expr("points + ?", point))
	if result.Error != nil {
		slog.Error("failed to update user point", "error", result.Error, "user_id", userID, "guild_id", guildID)
		return errors.NewError(result.Error)
	}

	// If no row existed, create one with the initial point value
	if point != 0 && result.RowsAffected == 0 {
		userPoint := models.GoPoint{
			UserID:  user.ID,
			GuildID: guildID,
			Points:  point,
		}
		if err := tx.Create(&userPoint).Error; err != nil {
			slog.Error("failed to create user point", "error", err, "user_id", userID, "guild_id", guildID)
			return errors.NewError(err)
		}
	}

	// Update active season points if this was a point gain (positive) or a point payment (negative)
	if point != 0 {
		var activeSeason models.GoPointSeason
		if err := tx.Where("guild_id = ? AND is_active = ?", guildID, true).First(&activeSeason).Error; err == nil {
			result := tx.Model(&models.GoPointSeasonUser{}).
				Where("season_id = ? AND user_id = ?", activeSeason.ID, user.ID).
				Update("points_earned", gorm.Expr("points_earned + ?", point))
			if result.Error != nil {
				slog.Error("failed to update season user points", "error", result.Error)
				return errors.NewError(result.Error)
			}
			if result.RowsAffected == 0 {
				seasonUser := models.GoPointSeasonUser{
					SeasonID:     activeSeason.ID,
					UserID:       user.ID,
					GuildID:      guildID,
					PointsEarned: point,
				}
				if err := tx.Create(&seasonUser).Error; err != nil {
					slog.Error("failed to create season user points", "error", err)
					return errors.NewError(err)
				}
			}
		}
	}

	return nil
}

func AddPoint(c *components.Components, userID snowflake.ID, guildID snowflake.ID, point int64) error {
	return c.GormDB().Transaction(func(tx *gorm.DB) error {
		return AddPointTx(tx, userID, guildID, point)
	})
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
