package play

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"gorm.io/gorm"
)

var (
	fx_sessions = database.NewMemoryValues[uuid.UUID, *FXSession](time.Minute * 15)
)

type FXSession struct {
	ID               uuid.UUID
	UserID           snowflake.ID
	GuildID          snowflake.ID
	SelectedSymbol   string
	SelectedMargin   int64
	SelectedLeverage int
}

func (s *FXSession) OnDelete() error {
	return nil
}

var (
	fxSymbols   = []string{"USD_JPY", "EUR_JPY", "GBP_JPY", "AUD_JPY", "NZD_JPY", "CAD_JPY", "CHF_JPY"}
	fxMargins   = []int64{10, 50, 100, 500, 1000, 5000, 10000}
	fxLeverages = []int{1, 5, 10, 25}
)

type TickerResponse struct {
	Status       int          `json:"status"`
	ResponseTime string       `json:"responsetime"`
	Data         []TickerData `json:"data"`
}

type TickerData struct {
	Symbol    string `json:"symbol"`
	Ask       string `json:"ask"`
	Bid       string `json:"bid"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
}

func (t *TickerResponse) GetSymbolData(symbol string) (TickerData, bool) {
	for _, d := range t.Data {
		if d.Symbol == symbol {
			return d, true
		}
	}
	return TickerData{}, false
}

var (
	tickerCache   *TickerResponse
	lastFetchTime time.Time
	tickerCacheMu sync.Mutex
)

func fetchTickerData() (*TickerResponse, error) {
	tickerCacheMu.Lock()
	defer tickerCacheMu.Unlock()

	// Rate limit: GMO Coin FX API has a rate limit of 1 request per second.
	// Reuse cached response if last fetch was less than 1.1 seconds ago.
	if tickerCache != nil && time.Since(lastFetchTime) < 1100*time.Millisecond {
		return tickerCache, nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://forex-api.coin.z.com/public/v1/ticker")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tickerResp TickerResponse
	if err := json.NewDecoder(resp.Body).Decode(&tickerResp); err != nil {
		return nil, err
	}

	tickerCache = &tickerResp
	lastFetchTime = time.Now()

	return &tickerResp, nil
}

func getFXPosition(c *components.Components, userID snowflake.ID, guildID snowflake.ID) (*models.FXPosition, error) {
	var pos models.FXPosition
	err := c.GormDB().Where("user_id = ? AND guild_id = ?", userID, guildID).First(&pos).Error
	if err != nil {
		return nil, err
	}
	return &pos, nil
}

func getLiquidationPrice(pos *models.FXPosition) float64 {
	if pos.Direction == "BUY" {
		return pos.EntryPrice * (1.0 - 1.0/float64(pos.Leverage))
	} else {
		return pos.EntryPrice * (1.0 + 1.0/float64(pos.Leverage))
	}
}

func checkLiquidation(c *components.Components, pos *models.FXPosition, ticker *TickerResponse) (bool, float64, float64, error) {
	symbolData, ok := ticker.GetSymbolData(pos.Symbol)
	if !ok {
		return false, 0, 0, fmt.Errorf("symbol not found in ticker data")
	}

	ask, err := strconv.ParseFloat(symbolData.Ask, 64)
	if err != nil {
		return false, 0, 0, err
	}
	bid, err := strconv.ParseFloat(symbolData.Bid, 64)
	if err != nil {
		return false, 0, 0, err
	}

	var currentPrice float64
	var pnl float64
	if pos.Direction == "BUY" {
		currentPrice = bid
		pnl = float64(pos.Margin) * float64(pos.Leverage) * ((currentPrice / pos.EntryPrice) - 1.0)
	} else {
		currentPrice = ask
		pnl = float64(pos.Margin) * float64(pos.Leverage) * (1.0 - (currentPrice / pos.EntryPrice))
	}

	if pnl <= -float64(pos.Margin) {
		if err := c.GormDB().Delete(pos).Error; err != nil {
			return true, currentPrice, pnl, err
		}
		return true, currentPrice, pnl, nil
	}

	return false, currentPrice, pnl, nil
}

func FXMessage(c *components.Components, session *FXSession, position *models.FXPosition, ticker *TickerResponse, points int64) []discord.LayoutComponent {
	var layoutComponents []discord.LayoutComponent

	container := discord.NewContainer().WithAccentColor(0x3498DB) // Sleek blue

	container = container.AddComponents(
		discord.NewTextDisplay("📈 **FX TRADING SIMULATOR** 📉"),
		discord.NewLargeSeparator(),
	)

	var ratesText []string
	for _, sym := range fxSymbols {
		if data, ok := ticker.GetSymbolData(sym); ok {
			ask, _ := strconv.ParseFloat(data.Ask, 64)
			bid, _ := strconv.ParseFloat(data.Bid, 64)
			spread := (ask - bid) * 100.0 // spread in pips (0.01 JPY = 1 pip)
			ratesText = append(ratesText, fmt.Sprintf("• **%s**: Bid `%.3f` | Ask `%.3f` (スプレッド: `%.1f` pips)", strings.Replace(sym, "_", "/", 1), bid, ask, spread))
		}
	}

	container = container.AddComponents(
		discord.NewTextDisplay("### 💱 現在の為替レート"),
		discord.NewTextDisplay(strings.Join(ratesText, "\n")),
		discord.NewLargeSeparator(),
		discord.NewTextDisplay(fmt.Sprintf("🪙 **保有 GoPoints**: `%d pt`", points)),
		discord.NewLargeSeparator(),
	)

	if position != nil {
		symbolData, _ := ticker.GetSymbolData(position.Symbol)
		ask, _ := strconv.ParseFloat(symbolData.Ask, 64)
		bid, _ := strconv.ParseFloat(symbolData.Bid, 64)

		var currentPrice float64
		var pnl float64
		if position.Direction == "BUY" {
			currentPrice = bid
			pnl = float64(position.Margin) * float64(position.Leverage) * ((currentPrice / position.EntryPrice) - 1.0)
		} else {
			currentPrice = ask
			pnl = float64(position.Margin) * float64(position.Leverage) * (1.0 - (currentPrice / position.EntryPrice))
		}

		pnlInt := int64(pnl)
		pnlStr := fmt.Sprintf("%+d", pnlInt)
		if pnlInt == 0 {
			pnlStr = "0"
		}

		dirEmoji := "🟢 買い (Long)"
		if position.Direction == "SELL" {
			dirEmoji = "🔴 売り (Short)"
		}

		liqPrice := getLiquidationPrice(position)
		positionInfo := []string{
			fmt.Sprintf("- **通貨ペア**: `%s`", strings.Replace(position.Symbol, "_", "/", 1)),
			fmt.Sprintf("- **ポジション**: %s", dirEmoji),
			fmt.Sprintf("- **レバレッジ**: `%dx`", position.Leverage),
			fmt.Sprintf("- **証拠金 (Margin)**: `%d pt` (取引数量相当: `%d pt`)", position.Margin, position.Margin*int64(position.Leverage)),
			fmt.Sprintf("- **エントリー価格**: `%.3f`", position.EntryPrice),
			fmt.Sprintf("- **現在価格**: `%.3f`", currentPrice),
			fmt.Sprintf("- **強制ロスカット価格**: `%.3f`", liqPrice),
			fmt.Sprintf("- **評価損益 (PnL)**: **`%s pt`**", pnlStr),
		}

		container = container.AddComponents(
			discord.NewTextDisplay("### 💼 保有ポジション情報"),
			discord.NewTextDisplay(strings.Join(positionInfo, "\n")),
		)

		layoutComponents = append(layoutComponents, container)

		actionRow := discord.NewActionRow().AddComponents(
			discord.NewSuccessButton("ポジション決済 (利確/損切)", fmt.Sprintf("play:fx_close:%s", session.ID)),
			discord.NewPrimaryButton("レート更新 / 損益確認", fmt.Sprintf("play:fx_refresh:%s", session.ID)),
			discord.NewSecondaryButton("画面を閉じる", fmt.Sprintf("play:fx_quit:%s", session.ID)),
		)
		layoutComponents = append(layoutComponents, actionRow)

	} else {
		formInfo := []string{
			"現在保有しているポジションはありません。注文内容を選択してください。",
			fmt.Sprintf("- **現在の選択**: `%s` | 証拠金: `%d pt` | レバレッジ: `%dx`", strings.Replace(session.SelectedSymbol, "_", "/", 1), session.SelectedMargin, session.SelectedLeverage),
		}

		container = container.AddComponents(
			discord.NewTextDisplay("### 🆕 新規注文フォーム"),
			discord.NewTextDisplay(strings.Join(formInfo, "\n")),
		)

		layoutComponents = append(layoutComponents, container)

		var symOptions []discord.StringSelectMenuOption
		for _, sym := range fxSymbols {
			symOptions = append(symOptions, discord.StringSelectMenuOption{
				Label:   strings.Replace(sym, "_", "/", 1),
				Value:   sym,
				Default: sym == session.SelectedSymbol,
			})
		}
		row1 := discord.NewActionRow().AddComponents(
			discord.StringSelectMenuComponent{
				CustomID:    "play:fx_symbol:" + session.ID.String(),
				Placeholder: "1. 通貨ペアを選択",
				Options:     symOptions,
			},
		)

		var marginOptions []discord.StringSelectMenuOption
		for _, m := range fxMargins {
			marginOptions = append(marginOptions, discord.StringSelectMenuOption{
				Label:   fmt.Sprintf("%d GoPoints", m),
				Value:   strconv.FormatInt(m, 10),
				Default: m == session.SelectedMargin,
			})
		}
		row2 := discord.NewActionRow().AddComponents(
			discord.StringSelectMenuComponent{
				CustomID:    "play:fx_margin:" + session.ID.String(),
				Placeholder: "2. 証拠金を選択",
				Options:     marginOptions,
			},
		)

		var levOptions []discord.StringSelectMenuOption
		for _, l := range fxLeverages {
			levOptions = append(levOptions, discord.StringSelectMenuOption{
				Label:   fmt.Sprintf("%dx レバレッジ", l),
				Value:   strconv.Itoa(l),
				Default: l == session.SelectedLeverage,
			})
		}
		row3 := discord.NewActionRow().AddComponents(
			discord.StringSelectMenuComponent{
				CustomID:    "play:fx_leverage:" + session.ID.String(),
				Placeholder: "3. レバレッジを選択",
				Options:     levOptions,
			},
		)

		row4 := discord.NewActionRow().AddComponents(
			discord.NewSuccessButton("🟢 買い (Long) 注文", fmt.Sprintf("play:fx_buy:%s", session.ID)),
			discord.NewDangerButton("🔴 売り (Short) 注文", fmt.Sprintf("play:fx_sell:%s", session.ID)),
			discord.NewPrimaryButton("レート更新", fmt.Sprintf("play:fx_refresh:%s", session.ID)),
			discord.NewSecondaryButton("画面を閉じる", fmt.Sprintf("play:fx_quit:%s", session.ID)),
		)

		layoutComponents = append(layoutComponents, row1, row2, row3, row4)
	}

	return layoutComponents
}

func FXPrecondition(event *events.ComponentInteractionCreate) (*FXSession, errors.Error) {
	args := strings.Split(event.Data.CustomID(), ":")
	if len(args) < 3 {
		return nil, errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	id, err := uuid.Parse(args[2])
	if err != nil {
		return nil, errors.NewError(err)
	}
	session, ok := fx_sessions.Get(id)
	if !ok {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ **セッションが期限切れです**"),
					discord.NewTextDisplay("この取引画面のセッションは終了しました。もう一度 `/play fx` を実行してください。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil, nil
	}
	if session.UserID != event.User().ID {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ **他人のセッションです**"),
					discord.NewTextDisplay("この取引画面は他のユーザーのものです。自分で `/play fx` を実行してください。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil, nil
	}
	return session, nil
}

func FXPlayCommand(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	if err := event.DeferCreateMessage(false); err != nil {
		return errors.NewError(err)
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	ticker, err := fetchTickerData()
	if err != nil {
		slog.Error("failed to fetch ticker data", "error", err)
		return errors.NewError(err)
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	session := &FXSession{
		ID:               uuid.New(),
		UserID:           event.User().ID,
		GuildID:          *event.GuildID(),
		SelectedSymbol:   "USD_JPY",
		SelectedMargin:   100,
		SelectedLeverage: 10,
	}
	fx_sessions.Set(session.ID, session)

	if pos != nil {
		liquidated, currentPrice, pnl, err := checkLiquidation(c, pos, ticker)
		if err != nil {
			return errors.NewError(err)
		}
		if liquidated {
			points, _, _ = gopoint.GetPoint(c, event.User().ID, *event.GuildID())
			msgComponents := FXMessage(c, session, nil, ticker, points)
			if len(msgComponents) > 0 {
				if mc, ok := msgComponents[0].(discord.ContainerComponent); ok {
					newComponents := append([]discord.ContainerSubComponent{
						discord.NewTextDisplay("🚨 **ロスカット (強制決済) 発生** 🚨"),
						discord.NewTextDisplay(fmt.Sprintf("保有していたポジションは、損失が証拠金を上回ったため強制決済されました。\n・決済価格: `%.3f` \n・損益: `-%d pt` (証拠金全額没収)", currentPrice, int64(-pnl))),
						discord.NewLargeSeparator(),
					}, mc.Components...)
					mc.Components = newComponents
					mc.AccentColor = 0xE74C3C
					msgComponents[0] = mc
				}
			}

			if err := event.RespondMessage(discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(msgComponents...),
			); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
	}

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, pos, ticker, points)...),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXSymbolHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
		session.SelectedSymbol = data.Values[0]
	}

	fx_sessions.Set(session.ID, session)

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, pos, ticker, points)...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXMarginHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
		m, err := strconv.ParseInt(data.Values[0], 10, 64)
		if err != nil {
			return errors.NewError(err)
		}
		session.SelectedMargin = m
	}

	fx_sessions.Set(session.ID, session)

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, pos, ticker, points)...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXLeverageHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
		l, err := strconv.Atoi(data.Values[0])
		if err != nil {
			return errors.NewError(err)
		}
		session.SelectedLeverage = l
	}

	fx_sessions.Set(session.ID, session)

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, pos, ticker, points)...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXBuyHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}
	if pos != nil {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ **既にポジションを保有しています**"),
					discord.NewTextDisplay("同時に複数のポジションを持つことはできません。現在のポジションを決済してから注文してください。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if points < session.SelectedMargin {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ **ポイント不足**"),
					discord.NewTextDisplay(fmt.Sprintf("証拠金に必要な `%d pt` が不足しています（現在の残高: `%d pt`）。", session.SelectedMargin, points)),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	symbolData, ok := ticker.GetSymbolData(session.SelectedSymbol)
	if !ok {
		return errors.NewError(fmt.Errorf("symbol data not found"))
	}

	ask, err := strconv.ParseFloat(symbolData.Ask, 64)
	if err != nil {
		return errors.NewError(err)
	}

	if err := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -session.SelectedMargin); err != nil {
		return errors.NewError(err)
	}

	pos = &models.FXPosition{
		UserID:     event.User().ID,
		GuildID:    *event.GuildID(),
		Symbol:     session.SelectedSymbol,
		Direction:  "BUY",
		EntryPrice: ask,
		Margin:     session.SelectedMargin,
		Leverage:   session.SelectedLeverage,
	}

	if err := c.GormDB().Create(pos).Error; err != nil {
		_ = gopoint.AddPoint(c, event.User().ID, *event.GuildID(), session.SelectedMargin)
		return errors.NewError(err)
	}

	points, _, _ = gopoint.GetPoint(c, event.User().ID, *event.GuildID())

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, pos, ticker, points)...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXSellHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}
	if pos != nil {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ **既にポジションを保有しています**"),
					discord.NewTextDisplay("同時に複数のポジションを持つことはできません。現在のポジションを決済してから注文してください。"),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if points < session.SelectedMargin {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("⚠️ **ポイント不足**"),
					discord.NewTextDisplay(fmt.Sprintf("証拠金に必要な `%d pt` が不足しています（現在の残高: `%d pt`）。", session.SelectedMargin, points)),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	symbolData, ok := ticker.GetSymbolData(session.SelectedSymbol)
	if !ok {
		return errors.NewError(fmt.Errorf("symbol data not found"))
	}

	bid, err := strconv.ParseFloat(symbolData.Bid, 64)
	if err != nil {
		return errors.NewError(err)
	}

	if err := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -session.SelectedMargin); err != nil {
		return errors.NewError(err)
	}

	pos = &models.FXPosition{
		UserID:     event.User().ID,
		GuildID:    *event.GuildID(),
		Symbol:     session.SelectedSymbol,
		Direction:  "SELL",
		EntryPrice: bid,
		Margin:     session.SelectedMargin,
		Leverage:   session.SelectedLeverage,
	}

	if err := c.GormDB().Create(pos).Error; err != nil {
		_ = gopoint.AddPoint(c, event.User().ID, *event.GuildID(), session.SelectedMargin)
		return errors.NewError(err)
	}

	points, _, _ = gopoint.GetPoint(c, event.User().ID, *event.GuildID())

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, pos, ticker, points)...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXRefreshHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	if pos != nil {
		liquidated, currentPrice, pnl, err := checkLiquidation(c, pos, ticker)
		if err != nil {
			return errors.NewError(err)
		}
		if liquidated {
			points, _, _ := gopoint.GetPoint(c, event.User().ID, *event.GuildID())

			msgComponents := FXMessage(c, session, nil, ticker, points)
			if len(msgComponents) > 0 {
				if mc, ok := msgComponents[0].(discord.ContainerComponent); ok {
					newComponents := append([]discord.ContainerSubComponent{
						discord.NewTextDisplay("🚨 **ロスカット (強制決済) 発生** 🚨"),
						discord.NewTextDisplay(fmt.Sprintf("保有していたポジションは、損失が証拠金を上回ったため強制決済されました。\n・決済価格: `%.3f` \n・損益: `-%d pt`", currentPrice, int64(-pnl))),
						discord.NewLargeSeparator(),
					}, mc.Components...)
					mc.Components = newComponents
					mc.AccentColor = 0xE74C3C
					msgComponents[0] = mc
				}
			}

			if err := event.UpdateMessage(discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(msgComponents...).
				BuildUpdate(),
			); err != nil {
				return errors.NewError(err)
			}
			return nil
		}
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, pos, ticker, points)...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXCloseHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay("⚠️ **ポジションが見つかりません**"),
						discord.NewTextDisplay("決済対象のポジションが見つかりません。すでにロスカットまたは他の画面で決済された可能性があります。"),
					).WithAccentColor(0xE74C3C),
				).
				AddFlags(discord.MessageFlagEphemeral)
			_ = event.RespondMessage(builder)
			return nil
		}
		return errors.NewError(err)
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	symbolData, ok := ticker.GetSymbolData(pos.Symbol)
	if !ok {
		return errors.NewError(fmt.Errorf("symbol data not found"))
	}

	ask, _ := strconv.ParseFloat(symbolData.Ask, 64)
	bid, _ := strconv.ParseFloat(symbolData.Bid, 64)

	var exitPrice float64
	var pnl float64
	if pos.Direction == "BUY" {
		exitPrice = bid
		pnl = float64(pos.Margin) * float64(pos.Leverage) * ((exitPrice / pos.EntryPrice) - 1.0)
	} else {
		exitPrice = ask
		pnl = float64(pos.Margin) * float64(pos.Leverage) * (1.0 - (exitPrice / pos.EntryPrice))
	}

	pnlInt := int64(pnl)
	refund := pos.Margin + pnlInt
	if refund < 0 {
		refund = 0
	}

	if err := c.GormDB().Delete(pos).Error; err != nil {
		return errors.NewError(err)
	}

	if refund > 0 {
		if err := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), refund); err != nil {
			slog.Error("CRITICAL: failed to refund points to user after closing position", "user_id", event.User().ID, "refund", refund, "error", err)
			return errors.NewError(err)
		}
	}

	points, _, _ := gopoint.GetPoint(c, event.User().ID, *event.GuildID())

	netWinSign := ""
	if pnlInt > 0 {
		netWinSign = "+"
	}
	pnlStr := fmt.Sprintf("%s%d", netWinSign, pnlInt)
	if pnlInt == 0 {
		pnlStr = "0"
	}

	dirText := "🟢 買い (Long)"
	if pos.Direction == "SELL" {
		dirText = "🔴 売り (Short)"
	}

	container := discord.NewContainer().WithAccentColor(0x2ECC71)
	container = container.AddComponents(
		discord.NewTextDisplay("🏁 **FXポジション決済完了** 🏁"),
		discord.NewLargeSeparator(),
		discord.NewTextDisplay("ポジションの決済が正常に完了しました。"),
		discord.NewTextDisplayf("- **通貨ペア**: `%s` (%s)", strings.Replace(pos.Symbol, "_", "/", 1), dirText),
		discord.NewTextDisplayf("- **レバレッジ**: `%dx` | **証拠金**: `%d pt`", pos.Leverage, pos.Margin),
		discord.NewTextDisplayf("- **エントリー価格**: `%.3f`", pos.EntryPrice),
		discord.NewTextDisplayf("- **決済価格**: `%.3f`", exitPrice),
		discord.NewTextDisplayf("- **獲得/損失 (PnL)**: **`%s pt`**", pnlStr),
		discord.NewTextDisplayf("- **返却証拠金**: `%d pt`", refund),
		discord.NewLargeSeparator(),
		discord.NewTextDisplayf("🪙 **現在の GoPoints**: `%d pt`", points),
	)

	actionRow := discord.NewActionRow().AddComponents(
		discord.NewPrimaryButton("もう一度取引する", fmt.Sprintf("play:fx_refresh:%s", session.ID)),
		discord.NewSecondaryButton("終了", fmt.Sprintf("play:fx_quit:%s", session.ID)),
	)

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(container, actionRow).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}

	return nil
}

func FXQuitHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	fx_sessions.Delete(session.ID)

	pos, err := getFXPosition(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	container := discord.NewContainer().WithAccentColor(0x95A5A6)
	container = container.AddComponents(
		discord.NewTextDisplay("📈 **FX TRADING CLOSED** 📉"),
		discord.NewLargeSeparator(),
		discord.NewTextDisplay("FX取引画面を閉じました。"),
	)

	if pos != nil {
		dirText := "🟢 買い"
		if pos.Direction == "SELL" {
			dirText = "🔴 売り"
		}
		container = container.AddComponents(
			discord.NewTextDisplayf("⚠️ **注意**: `%s` の %s ポジション (証拠金: `%d pt`) は決済されず、**現在も保有中**です。再度 `/play fx` を実行することで、損益確認や決済を行うことができます。", strings.Replace(pos.Symbol, "_", "/", 1), dirText, pos.Margin),
		)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(container).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}

	return nil
}
