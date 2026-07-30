package currency

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
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

func GetCurrencyName(c *components.Components, guildID snowflake.ID) string {
	if c == nil || c.GormDB() == nil {
		return "GoPoint"
	}
	var cfg models.CurrencyConfig
	if err := c.GormDB().Where("guild_id = ?", guildID).First(&cfg).Error; err == nil && cfg.Name != "" {
		return cfg.Name
	}
	return "GoPoint"
}

func CurrencyNameConfigHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	guildID := *event.GuildID()
	data := event.SlashCommandInteractionData()
	newName, hasName := data.OptString("name")

	var cfg models.CurrencyConfig
	err := c.GormDB().Where("guild_id = ?", guildID).First(&cfg).Error

	if !hasName {
		currentName := GetCurrencyName(c, guildID)
		msg := i18n.TranslateText(event.Locale(), "components.currency.config.name_current", map[string]any{
			"name": currentName,
		})
		if msg == "components.currency.config.name_current" {
			msg = fmt.Sprintf("現在の通貨名設定: **%s**", currentName)
		}
		if errResp := event.RespondMessage(discord.NewMessageBuilder().
			SetEphemeral(true).
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(msg),
				).WithAccentColor(0x3498DB),
			),
		); errResp != nil {
			return errors.NewError(errResp)
		}
		return nil
	}

	newName = strings.TrimSpace(newName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cfg = models.CurrencyConfig{
				GuildID: guildID,
				Name:    newName,
			}
			if errCreate := c.GormDB().Create(&cfg).Error; errCreate != nil {
				return errors.NewError(errCreate)
			}
		} else {
			return errors.NewError(err)
		}
	} else {
		cfg.Name = newName
		if errSave := c.GormDB().Save(&cfg).Error; errSave != nil {
			return errors.NewError(errSave)
		}
	}

	msg := i18n.TranslateText(event.Locale(), "components.currency.config.name_success", map[string]any{
		"name": GetCurrencyName(c, guildID),
	})
	if msg == "components.currency.config.name_success" {
		msg = fmt.Sprintf("通貨名を **%s** に設定しました。", GetCurrencyName(c, guildID))
	}

	if errResp := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(true).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(msg),
			).WithAccentColor(0x2ECC71),
		),
	); errResp != nil {
		return errors.NewError(errResp)
	}
	return nil
}

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "currency",
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
				Name:                     "currency",
				Description:              "Currency(通貨)の情報を表示・管理します。",
				DescriptionLocalizations: i18n.TranslateTextMap("command.currency.description"),
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				IntegrationTypes: []discord.ApplicationIntegrationType{
					discord.ApplicationIntegrationTypeGuildInstall,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "view",
						Description:              "所持通貨とランキングを表示します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.currency.view.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "ranking",
						Description:              "通貨のランキングを表示します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.currency.ranking.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:                     "limit",
								Description:              "ランキングの表示件数を指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.ranking.limit.description"),
								Required:                 false,
								MinValue:                 builtin.Ptr(1),
								MaxValue:                 builtin.Ptr(100),
							},
							discord.ApplicationCommandOptionInt{
								Name:                     "skip",
								Description:              "ランキングのスキップ件数を指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.ranking.skip.description"),
								Required:                 false,
								MinValue:                 builtin.Ptr(0),
								MaxValue:                 builtin.Ptr(1000),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "give",
						Description:              "他のユーザーに通貨を渡します。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.currency.give.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionUser{
								Name:                     "target_user",
								Description:              "通貨を渡す対象のユーザーを指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.give.target_user.description"),
								Required:                 true,
							},
							discord.ApplicationCommandOptionInt{
								Name:                     "point",
								Description:              "渡す通貨の数を指定します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.give.point.description"),
								Required:                 true,
								MinValue:                 builtin.Ptr(1),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:                     "config",
						Description:              "通貨システムの設定を行います。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.currency.config.description"),
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "name",
								Description:              "ギルドの通貨名（表示名）を設定・表示します。(管理者のみ)",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.config-name.description"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:                     "name",
										Description:              "新しい通貨名（例: ゴールド, コイン, ptなど）を指定します。",
										DescriptionLocalizations: i18n.TranslateTextMap("command.currency.config-name.name.description"),
										Required:                 false,
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:                     "tax",
						Description:              "定期徴収の管理を行います。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.currency.tax.description"),
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "setup",
								Description:              "定期徴収の割合や間隔を設定します。(管理者のみ)",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.tax-setup.description"),
							},
							{
								Name:                     "status",
								Description:              "現在の定期徴収の設定状況と徴収予定を表示します。(管理者のみ)",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.tax-status.description"),
							},
							{
								Name:                     "force",
								Description:              "今すぐポイントの徴収（または徴収の予約計算）を強制実行します。(管理者のみ)",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.tax-force.description"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionBool{
										Name:                     "overwrite",
										Description:              "既存の徴収予定を上書き（再計算）するか指定します (デフォルト: false)。",
										DescriptionLocalizations: i18n.TranslateTextMap("command.currency.tax-force.overwrite.description"),
										Required:                 false,
									},
								},
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "reset",
						Description:              "ギルド内の全員の通貨を一括リセットします。(管理者のみ)",
						DescriptionLocalizations: i18n.TranslateTextMap("command.currency.reset.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:                     "points",
								Description:              "リセット後のポイント数を指定します (デフォルト: 0)。by_levelがtrueの場合は無視されます。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.reset.points.description"),
								Required:                 false,
								MinValue:                 builtin.Ptr(0),
							},
							discord.ApplicationCommandOptionBool{
								Name:                     "by_level",
								Description:              "各ユーザーのレベルの累積必要XPに応じたポイントでリセットするかどうか。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.reset.by_level.description"),
								Required:                 false,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommandGroup{
						Name:                     "season",
						Description:              "シーズンの管理・表示を行います。",
						DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season.description"),
						Options: []discord.ApplicationCommandOptionSubCommand{
							{
								Name:                     "start",
								Description:              "新しいシーズンを開始、またはスケジュールします。(管理者のみ)",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-start.description"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:                     "name",
										Description:              "シーズンの名前を指定します。",
										DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-start.name.description"),
										Required:                 true,
									},
									discord.ApplicationCommandOptionInt{
										Name:                     "duration_days",
										Description:              "シーズンの期間(日)を指定します (1-365)。",
										DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-start.duration_days.description"),
										Required:                 true,
										MinValue:                 builtin.Ptr(1),
										MaxValue:                 builtin.Ptr(365),
									},
									discord.ApplicationCommandOptionInt{
										Name:                     "start_delay_hours",
										Description:              "何時間後にシーズンを開始するか（スケジュール予約）を指定します。",
										DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-start.start_delay_hours.description"),
										Required:                 false,
										MinValue:                 builtin.Ptr(0),
									},
									discord.ApplicationCommandOptionString{
										Name:                     "criteria",
										Description:              "ランキングの基準を指定します (earned: 獲得ポイント, final: 最終ポイント)。",
										DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-start.criteria.description"),
										Required:                 false,
										Choices: []discord.ApplicationCommandOptionChoiceString{
											{
												Name:              i18n.TranslateText(discord.LocaleJapanese, "command.currency.season-start.criteria.choice.earned"),
												NameLocalizations: i18n.TranslateTextMap("command.currency.season-start.criteria.choice.earned"),
												Value:             "earned",
											},
											{
												Name:              i18n.TranslateText(discord.LocaleJapanese, "command.currency.season-start.criteria.choice.final"),
												NameLocalizations: i18n.TranslateTextMap("command.currency.season-start.criteria.choice.final"),
												Value:             "final",
											},
										},
									},
								},
							},
							{
								Name:                     "end",
								Description:              "アクティブなシーズンを途中で終了し、表彰を行います。(管理者のみ)",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-end.description"),
							},
							{
								Name:                     "status",
								Description:              "現在のアクティブなシーズンの進捗と現在のランキングを表示します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-status.description"),
							},
							{
								Name:                     "ranking",
								Description:              "アクティブなシーズン、または過去のシーズンのランキングを表示します。",
								DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-ranking.description"),
								Options: []discord.ApplicationCommandOption{
									discord.ApplicationCommandOptionString{
										Name:                     "season_id",
										Description:              "表示したい過去のシーズンのIDを指定します (省略時は現在のアクティブなシーズン)。",
										DescriptionLocalizations: i18n.TranslateTextMap("command.currency.season-ranking.season_id.description"),
										Required:                 false,
										Autocomplete:             true,
									},
								},
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/currency/view": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("currency.view"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.DeferCreateMessage(true); err != nil {
						return errors.NewError(err)
					}
					point, rank, err := GetCurrency(c, event.User().ID, *event.GuildID())
					if err != nil {
						slog.Error("failed to get Currency", "error", err, "user_id", event.User().ID, "guild_id", *event.GuildID())
						return errors.NewError(err)
					}
					slog.Info("Currency retrieved", "user_id", event.User().ID, "guild_id", *event.GuildID(), "point", point, "rank", rank)
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
						ranking = i18n.TranslateText(event.Locale(), "command.currency.ranking.no_data")
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
								WithText("currency_name", GetCurrencyName(c, *event.GuildID())).
								Translate(i18n.TranslateLayout(event.Locale(), "command.currency.view"))...,
						),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/currency/ranking": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("currency.ranking"),
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
								WithText("currency_name", GetCurrencyName(c, *event.GuildID())).
								Translate(i18n.TranslateLayout(event.Locale(), "command.currency.ranking"))...,
						),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/currency/give": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("currency.give"),
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
						return errors.NewError(errors.ErrorMessage("error.currency.give.self", event))
					}
					if targetUser.Bot || targetUser.System {
						return errors.NewError(errors.ErrorMessage("error.currency.give.invalid_target", event))
					}
					point, ok := event.SlashCommandInteractionData().OptInt("point")
					if !ok || point <= 0 {
						return errors.NewError(errors.ErrorMessage("error.currency.give.invalid_point", event))
					}
					currentPoint, _, err := GetCurrency(c, event.User().ID, *event.GuildID())
					if err != nil {
						slog.Error("failed to get Currency", "error", err, "user_id", event.User().ID, "guild_id", *event.GuildID())
						return errors.NewError(err)
					}
					currencyName := GetCurrencyName(c, *event.GuildID())
					if currentPoint < int64(point) {
						if err := event.CreateMessage(discord.NewMessageBuilder().
							SetEphemeral(true).
							SetIsComponentsV2(true).
							SetComponents(
								i18n.BuildContext().
									WithText("target_user_name", targetUserID.EffectiveName()).
									WithText("point", fmt.Sprintf("%d", point)).
									WithText("current_point", fmt.Sprintf("%d", currentPoint)).
									WithText("currency_name", currencyName).
									Translate(i18n.TranslateLayout(event.Locale(), "command.currency.give.not_enough_point"))...,
							).BuildCreate(),
						); err != nil {
							slog.Error("failed to create message for not enough points", "error", err, "user_id", event.User().ID, "guild_id", *event.GuildID())
							return errors.NewError(err)
						}
						return nil
					}
					if err := GiveCurrency(c, event.User().ID, targetUserID.ID, *event.GuildID(), int64(point)); err != nil {
						slog.Error("failed to give Currency", "error", err, "user_id", event.User().ID, "target_user_id", targetUserID.ID, "guild_id", *event.GuildID())
						return errors.NewError(err)
					}
					if err := event.CreateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(
							i18n.BuildContext().
								WithText("target_user_name", targetUserID.EffectiveName()).
								WithText("point", fmt.Sprintf("%d", point)).
								WithText("currency_name", currencyName).
								Translate(i18n.TranslateLayout(event.Locale(), "command.currency.give.success"))...,
						).BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/currency/config/name": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("currency.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return CurrencyNameConfigHandler(c, event)
				},
			},
			"/currency/tax/setup": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("currency.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return TaxSetupHandler(c, event)
				},
			},
			"/currency/tax/status": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("currency.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return TaxStatusHandler(c, event)
				},
			},
			"/currency/tax/force": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("currency.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return TaxForceHandler(c, event)
				},
			},
			"/currency/reset": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("currency.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return ResetPointsHandler(c, event)
				},
			},
			"/currency/season/start": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("currency.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonStartHandler(c, event)
				},
			},
			"/currency/season/end": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("currency.admin"),
				},
				DiscordPerm: discord.PermissionAdministrator,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonEndHandler(c, event)
				},
			},
			"/currency/season/status": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("currency.view"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonStatusHandler(c, event)
				},
			},
			"/currency/season/ranking": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("currency.view"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return SeasonRankingHandler(c, event)
				},
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"currency:tax_setup_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				return TaxSetupModalSubmitHandler(c, event)
			},
		},
		AutocompleteHandlers: map[string]generic.PermissionAutocompleteHandler{
			"/currency/season/ranking:season_id": generic.PAutocompleteHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("currency.view"),
				},
				AutocompleteHandler: seasonAutocomplete,
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
					if err := AddCurrency(c, e.Message.Author.ID, e.GuildID, 10); err != nil {
						slog.Error("failed to add Currency", "error", err, "user_id", e.Message.Author.ID, "guild_id", e.GuildID)
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
	}
	var rankQuery []struct {
		UserID snowflake.ID `gorm:"column:user_id"`
		Rank   int          `gorm:"column:rank"`
		Points int64        `gorm:"column:points"`
	}
	if err := c.GormDB().Model(&models.Currency{}).
		Select("user_id", "RANK() OVER (ORDER BY points DESC) AS rank", "points").
		Where("guild_id = ?", guildID).
		Limit(limit).
		Offset(skip).
		Scan(&rankQuery).Error; err != nil {
		return "", fmt.Errorf("failed to find ranking: %w", err)
	}
	if len(rankQuery) == 0 && skip == 0 {
		return i18n.TranslateText(locale, "command.currency.ranking.no_guild_data"), nil
	} else if len(rankQuery) == 0 {
		return i18n.TranslateText(locale, "command.currency.ranking.no_result"), nil
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
			ReplaceText(i18n.TranslateText(locale, "command.currency.ranking.entry"))
		if userID != nil && point.UserID == *userID {
			text += " " + i18n.TranslateText(locale, "command.currency.ranking.entry.you")
		}
		result.WriteString(text + "\n")
	}
	return result.String(), nil
}

func AddCurrencyTx(tx *gorm.DB, userID snowflake.ID, guildID snowflake.ID, point int64) error {
	user, err := database.GetOrCreateUser(tx, userID)
	if err != nil {
		slog.Error("failed to find or create user", "error", err, "user_id", userID)
		return errors.NewError(err)
	}

	result := tx.Model(&models.Currency{}).
		Where("user_id = ? AND guild_id = ?", user.ID, guildID).
		Update("points", gorm.Expr("points + ?", point))
	if result.Error != nil {
		slog.Error("failed to update user point", "error", result.Error, "user_id", userID, "guild_id", guildID)
		return errors.NewError(result.Error)
	}

	if point != 0 && result.RowsAffected == 0 {
		userPoint := models.Currency{
			UserID:  user.ID,
			GuildID: guildID,
			Points:  point,
		}
		if err := tx.Create(&userPoint).Error; err != nil {
			slog.Error("failed to create user point", "error", err, "user_id", userID, "guild_id", guildID)
			return errors.NewError(err)
		}
	}

	if point != 0 {
		var activeSeason models.CurrencySeason
		err := tx.Where("guild_id = ? AND is_active = ?", guildID, true).First(&activeSeason).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("failed to find active season", "error", err, "guild_id", guildID)
			return errors.NewError(err)
		}
		if err == nil {
			result := tx.Model(&models.CurrencySeasonUser{}).
				Where("season_id = ? AND user_id = ?", activeSeason.ID, user.ID).
				Update("points_earned", gorm.Expr("points_earned + ?", point))
			if result.Error != nil {
				slog.Error("failed to update season user points", "error", result.Error)
				return errors.NewError(result.Error)
			}
			if result.RowsAffected == 0 {
				seasonUser := models.CurrencySeasonUser{
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

// AddPointTx is an alias for AddCurrencyTx for backward compatibility
func AddPointTx(tx *gorm.DB, userID snowflake.ID, guildID snowflake.ID, point int64) error {
	return AddCurrencyTx(tx, userID, guildID, point)
}

func AddCurrency(c *components.Components, userID snowflake.ID, guildID snowflake.ID, point int64) error {
	return c.GormDB().Transaction(func(tx *gorm.DB) error {
		return AddCurrencyTx(tx, userID, guildID, point)
	})
}

// AddPoint is an alias for AddCurrency for backward compatibility
func AddPoint(c *components.Components, userID snowflake.ID, guildID snowflake.ID, point int64) error {
	return AddCurrency(c, userID, guildID, point)
}

func GiveCurrency(c *components.Components, userID snowflake.ID, targetUserID snowflake.ID, guildID snowflake.ID, point int64) error {
	if userID == targetUserID {
		return errors.NewError(fmt.Errorf("cannot give currency to yourself"))
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
	userPoint := models.Currency{
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
	userPoint.Points -= point
	if err := c.GormDB().Save(&userPoint).Error; err != nil {
		slog.Error("failed to update user point", "error", err, "user_id", userID, "guild_id", guildID)
		return errors.NewError(err)
	}
	targetUserPoint := models.Currency{
		UserID:  targetUserID,
		GuildID: guildID,
	}
	if err := c.GormDB().Where(targetUserPoint).FirstOrInit(&targetUserPoint).Error; err != nil {
		slog.Error("failed to find target user point", "error", err, "target_user_id", targetUserID, "guild_id", guildID)
		return errors.NewError(err)
	}
	targetUserPoint.Points += point
	if err := c.GormDB().Save(&targetUserPoint).Error; err != nil {
		slog.Error("failed to update target user point", "error", err, "target_user_id", targetUserID, "guild_id", guildID)
		return errors.NewError(err)
	}
	slog.Info("points given successfully", "user_id", userID, "target_user_id", targetUserID, "guild_id", guildID, "points", point)
	return nil
}

// GivePoint is an alias for GiveCurrency for backward compatibility
func GivePoint(c *components.Components, userID snowflake.ID, targetUserID snowflake.ID, guildID snowflake.ID, point int64) error {
	return GiveCurrency(c, userID, targetUserID, guildID, point)
}

func GetCurrency(c *components.Components, userID snowflake.ID, guildID snowflake.ID) (point int64, rank int, err error) {
	user := models.User{
		ID: userID,
	}
	if err := c.GormDB().Where(user).FirstOrCreate(&user).Error; err != nil {
		slog.Error("failed to find or create user", "error", err, "user_id", userID)
		return 0, 0, err
	}
	userPoint := models.Currency{
		UserID:  userID,
		GuildID: guildID,
	}
	if err := c.GormDB().Where(userPoint).FirstOrInit(&userPoint).Error; err != nil {
		slog.Error("failed to find user point", "error", err, "user_id", userID, "guild_id", guildID)
		return 0, 0, err
	}
	point = userPoint.Points

	var rankQuery struct {
		UserID snowflake.ID `gorm:"column:user_id"`
		Rank   int          `gorm:"column:rank"`
		Points int64        `gorm:"column:points"`
	}
	if err := c.GormDB().Table("(?) as u", c.GormDB().Model(&models.Currency{}).
		Select("user_id", "RANK() OVER (ORDER BY points DESC) AS rank", "points").
		Where("guild_id = ?", guildID)).
		Where("user_id = ?", userID).
		Find(&rankQuery).Error; err != nil {
		slog.Error("failed to get user rank", "error", err, "user_id", userID, "guild_id", guildID)
		return 0, 0, err
	}
	return point, rankQuery.Rank, nil
}

// GetPoint is an alias for GetCurrency for backward compatibility
func GetPoint(c *components.Components, userID snowflake.ID, guildID snowflake.ID) (point int64, rank int, err error) {
	return GetCurrency(c, userID, guildID)
}
