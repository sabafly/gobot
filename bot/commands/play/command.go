package play

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

var (
	slotLocks   = make(map[uuid.UUID]*sync.Mutex)
	slotLocksMu sync.Mutex
)

func getSlotLock(id uuid.UUID) *sync.Mutex {
	slotLocksMu.Lock()
	defer slotLocksMu.Unlock()
	lock, ok := slotLocks[id]
	if !ok {
		lock = &sync.Mutex{}
		slotLocks[id] = lock
	}
	return lock
}

func deleteSlotLock(id uuid.UUID) {
	slotLocksMu.Lock()
	defer slotLocksMu.Unlock()
	delete(slotLocks, id)
}

func ptr[T any](v T) *T {
	return &v
}

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "play",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "play",
				Description: "Play some games with your friends!",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "high-and-low",
						Description:              "Play a high and low card game",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.high-and-low.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "slot",
						Description:              "Play a slot machine game",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.slot.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "fx",
						Description:              "Play an FX trading game with GoPoints",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.fx.description"),
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "portfolio",
						Description:              "Display a user's FX portfolio",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.portfolio.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionUser{
								Name:                     "user",
								Description:              "The user to display the portfolio for",
								DescriptionLocalizations: i18n.TranslateTextMap("command.play.portfolio.option.user.description"),
								Required:                 false,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "chinchiro",
						Description:              "チンチロリンをマルチプレイで開始します",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.chinchiro.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:                     "bet",
								Description:              "掛け金を指定します (デフォルト: 10)",
								DescriptionLocalizations: i18n.TranslateTextMap("command.play.chinchiro.option.bet.description"),
								Required:                 false,
								MinValue:                 ptr(1),
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "polymarket",
						Description:              "Bet GoPoints on Polymarket predictions",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.polymarket.description"),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:                     "query",
								Description:              "Search query to find active markets",
								DescriptionLocalizations: i18n.TranslateTextMap("command.play.polymarket.option.query.description"),
								Required:                 false,
							},
							discord.ApplicationCommandOptionBool{
								Name:                     "closed",
								Description:              "Include closed/finished predictions",
								DescriptionLocalizations: i18n.TranslateTextMap("command.play.polymarket.option.closed.description"),
								Required:                 false,
							},
						},
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/play/high-and-low": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					data := NewHALData(event.User().ID)

					hal_values.Set(data.id, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALStartMessage(*data, event.Locale(), point, data.startOptionIndex)...)); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/play/slot": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.slot"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.DeferCreateMessage(false); err != nil {
						return errors.NewError(err)
					}

					data := &SlotData{
						ID:      uuid.New(),
						UserID:  event.User().ID,
						GuildID: *event.GuildID(),
						Status:  SlotStatusNormal,
						LastReels: [3][3]string{
							{"🍇", "🔔", "🍒"},
							{"🔄", "7️⃣", "🍇"},
							{"🤡", "⬛", "🔄"},
						},
						GogoLit:     false,
						LastSpinWin: 0,
						TotalSpins:  0,
						TotalSpent:  0,
						TotalWon:    0,
					}

					reason, err := SlotPlay(c, data)
					if err != nil {
						return err
					}
					if reason == "insufficient_points" {
						const cost = 3
						if err := errors.ErrorMessage("error.play.slot.insufficient_points", event,
							errors.WithMapContext(i18n.BuildContext().WithText("point", strconv.FormatInt(cost, 10)))); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					slot_values.Set(data.ID, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(SlotMessage(c, data, event.Locale())...)); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/play/fx": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return FXPlayCommand(c, event)
				},
			},
			"/play/portfolio": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return FXPortfolioCommandHandler(c, event)
				},
			},
			"/play/chinchiro": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.chinchiro"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return ChinchiroPlayCommand(c, event)
				},
			},
			"/play/polymarket": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					return PolymarketPlayCommand(c, event)
				},
			},
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"play:hal_select_opt": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err1 := HALPrecondition(event)
					if err1 != nil {
						return err1
					}
					if data == nil {
						return nil
					}
					gopoint, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					selectedOption := 0
					if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
						selected, err := strconv.Atoi(data.Values[0])
						if err != nil {
							return errors.NewError(err)
						}
						if selected < 0 || selected >= len(HALStartOptions) {
							return errors.NewError(fmt.Errorf("invalid selected option"))
						}
						selectedOption = selected
					}

					data.SetStartOption(selectedOption, HALStartOptions[selectedOption])

					hal_values.Set(data.id, data)

					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALStartMessage(*data, event.Locale(), gopoint, selectedOption)...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_start": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err1 := HALPrecondition(event)
					if err1 != nil {
						return err1
					}
					if data == nil {
						return nil
					}

					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					if point < data.startOption.Cost {
						if err := errors.ErrorMessage("error.play.high-and-low.insufficient_points", event,
							errors.WithMapContext(i18n.BuildContext().WithText("point", strconv.FormatInt(data.startOption.Cost, 10)))); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					if err := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -int64(data.startOption.Cost)); err != nil {
						return errors.NewError(err)
					}

					data.Start()

					hal_values.Set(data.id, data)

					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_high": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err := HALPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					finish := HALPlay(data, HALResultHigh)
					if finish != HALFinishStateNone {
						return HALFinish(c, *data, finish, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_low": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err := HALPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					finish := HALPlay(data, HALResultLow)
					if finish != HALFinishStateNone {
						return HALFinish(c, *data, finish, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_same": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err := HALPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					finish := HALPlay(data, HALResultSame)
					if finish != HALFinishStateNone {
						return HALFinish(c, *data, finish, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_retry": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					if len(args) < 3 {
						return errors.NewError(fmt.Errorf("invalid custom ID"))
					}
					optionIndex, err := strconv.Atoi(args[2])
					if err != nil {
						return errors.NewError(err)
					}
					if optionIndex < 0 || optionIndex >= len(HALStartOptions) {
						optionIndex = 0 // default to first option if invalid
					}

					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					data := NewHALData(event.User().ID)
					data.SetStartOption(optionIndex, HALStartOptions[optionIndex])

					hal_values.Set(data.id, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALStartMessage(*data, event.Locale(), point, data.startOptionIndex)...)); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_cancel": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err := HALPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					return HALFinish(c, *data, HALFinishStatePayout, event.User().ID, *event.GuildID(), event)
				},
			},
			"play:slot_spin": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.slot"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					if len(args) < 3 {
						return errors.NewError(fmt.Errorf("invalid custom ID"))
					}
					id, errParse := uuid.Parse(args[2])
					if errParse != nil {
						return errors.NewError(errParse)
					}
					mu := getSlotLock(id)
					mu.Lock()
					defer mu.Unlock()

					data, err := SlotPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}

					reason, err := SlotPlay(c, data)
					if err != nil {
						return err
					}
					if reason == "insufficient_points" {
						var cost int64 = 3
						if data.Status == SlotStatusBB || data.Status == SlotStatusRB {
							cost = 1
						}
						if err := event.CreateMessage(discord.NewMessageCreateBuilder().
							SetContent(i18n.BuildContext().
								WithText("point", strconv.FormatInt(cost, 10)).
								ReplaceText(i18n.TranslateText(event.Locale(), "error.play.slot.insufficient_points.description"))).
							SetFlags(discord.MessageFlagEphemeral).
							Build()); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					slot_values.Set(data.ID, data)

					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(SlotMessage(c, data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:slot_quit": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.slot"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					if len(args) < 3 {
						return errors.NewError(fmt.Errorf("invalid custom ID"))
					}
					id, errParse := uuid.Parse(args[2])
					if errParse != nil {
						return errors.NewError(errParse)
					}
					mu := getSlotLock(id)
					mu.Lock()
					defer mu.Unlock()

					data, err := SlotPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					defer deleteSlotLock(id)
					return SlotFinish(c, data, event)
				},
			},
			"play:fx_symbol": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXSymbolHandler(c, event)
				},
			},
			"play:fx_margin_btn": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXMarginButtonHandler(c, event)
				},
			},
			"play:fx_leverage": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXLeverageHandler(c, event)
				},
			},
			"play:fx_buy": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXBuyHandler(c, event)
				},
			},
			"play:fx_sell": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXSellHandler(c, event)
				},
			},
			"play:fx_refresh": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXRefreshHandler(c, event)
				},
			},
			"play:fx_close": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXCloseHandler(c, event)
				},
			},
			"play:fx_add_margin_btn": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXAddMarginButtonHandler(c, event)
				},
			},
			"play:fx_quit": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXQuitHandler(c, event)
				},
			},
			"play:fx_switch_pos": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXSwitchPositionHandler(c, event)
				},
			},
			"play:fx_pending_order_btn": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXPendingOrderButtonHandler(c, event)
				},
			},
			"play:fx_cancel_order": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXCancelOrderHandler(c, event)
				},
			},
			"play:fx_set_tpsl_btn": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.fx"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return FXSetTPSLButtonHandler(c, event)
				},
			},
			"play:chinchiro_join": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.chinchiro"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return ChinchiroJoinHandler(c, event)
				},
			},
			"play:chinchiro_start": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.chinchiro"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return ChinchiroStartHandler(c, event)
				},
			},
			"play:chinchiro_cancel": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.chinchiro"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return ChinchiroCancelHandler(c, event)
				},
			},
			"play:chinchiro_roll": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.chinchiro"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return ChinchiroRollHandler(c, event)
				},
			},
			"play:pm_select_market": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketSelectMarketHandler(c, event)
				},
			},
			"play:pm_select_outcome": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketSelectOutcomeHandler(c, event)
				},
			},
			"play:pm_view_my_bets": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketViewMyBetsHandler(c, event)
				},
			},
			"play:pm_back_to_list": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketBackToListHandler(c, event)
				},
			},
			"play:pm_back_to_detail": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketBackToDetailHandler(c, event)
				},
			},
			"play:pm_quit": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketQuitHandler(c, event)
				},
			},
			"play:pm_bet_amount": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketBetAmountHandler(c, event)
				},
			},
			"play:pm_refresh_bets": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.polymarket"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					return PolymarketRefreshBetsHandler(c, event)
				},
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"play:fx_margin_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				return FXMarginModalHandler(c, event)
			},
			"play:fx_add_margin_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				return FXAddMarginModalHandler(c, event)
			},
			"play:fx_order_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				return FXOrderModalHandler(c, event)
			},
			"play:fx_tpsl_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				return FXTPSLModalHandler(c, event)
			},
			"play:pm_custom_bet_modal": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				return PolymarketCustomBetModalHandler(c, event)
			},
		},
		Schedulers: []components.Scheduler{
			{
				Duration: 1 * time.Second,
				Worker: func(c *components.Components, client *bot.Client) error {
					return CheckAllPositionsLiquidation(c, client)
				},
			},
		},
	}).SetComponent(c)
}
