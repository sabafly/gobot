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

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
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
	SelectedLeverage int // index of fxLeverages
	ActivePositionID *uuid.UUID
}

func (s *FXSession) OnDelete() error {
	return nil
}

var (
	fxSymbols = []string{
		"USD_JPY", "EUR_JPY", "GBP_JPY", "AUD_JPY", "NZD_JPY", "CAD_JPY", "CHF_JPY",
		"BTC_JPY", "ETH_JPY", "BCH_JPY", "LTC_JPY", "XRP_JPY",
	}
	fxLeverages = []LeverageOption{
		{Leverage: 1, MinRatio: 0, MarginCallRatio: 50.0, LiquidationRatio: 20.0},
		{Leverage: 25, MinRatio: 0.05, MarginCallRatio: 50.0, LiquidationRatio: 20.0},
		{Leverage: 50, MinRatio: 0.1, MarginCallRatio: 50.0, LiquidationRatio: 20.0},
		{Leverage: 150, MinRatio: 0.15, MarginCallRatio: 50.0, LiquidationRatio: 30.0},
		{Leverage: 1000, MinRatio: 0.85, MarginCallRatio: 80.0, LiquidationRatio: 50.0},
	}
)

type LeverageOption struct {
	Leverage         int     // レバレッジ倍率
	MinRatio         float64 // 総所持ポイントに対する証拠金の比率の最小値
	MarginCallRatio  float64 // 追証ライン (%)
	LiquidationRatio float64 // ロスカットライン (%)
}

func getLeverageOption(leverage int) LeverageOption {
	for _, opt := range fxLeverages {
		if opt.Leverage == leverage {
			return opt
		}
	}
	return LeverageOption{Leverage: leverage, MarginCallRatio: 50.0, LiquidationRatio: 20.0}
}

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
	// Rate limit: Reuse cached response if last fetch was less than 300 milliseconds ago.
	if tickerCache != nil && time.Since(lastFetchTime) < 300*time.Millisecond {
		res := tickerCache
		tickerCacheMu.Unlock()
		return res, nil
	}
	tickerCacheMu.Unlock()

	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Fetch Forex ticker
	resp1, err := client.Get("https://forex-api.coin.z.com/public/v1/ticker")
	if err != nil {
		return nil, err
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for forex: %d", resp1.StatusCode)
	}
	var tickerResp1 TickerResponse
	if err := json.NewDecoder(resp1.Body).Decode(&tickerResp1); err != nil {
		return nil, err
	}

	// 2. Fetch Crypto ticker
	resp2, err := client.Get("https://api.coin.z.com/public/v1/ticker")
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for crypto: %d", resp2.StatusCode)
	}
	var tickerResp2 TickerResponse
	if err := json.NewDecoder(resp2.Body).Decode(&tickerResp2); err != nil {
		return nil, err
	}

	// Combine data
	combinedData := append(tickerResp1.Data, tickerResp2.Data...)
	combinedResp := &TickerResponse{
		Status:       tickerResp1.Status,
		ResponseTime: tickerResp1.ResponseTime,
		Data:         combinedData,
	}

	tickerCacheMu.Lock()
	tickerCache = combinedResp
	lastFetchTime = time.Now()
	tickerCacheMu.Unlock()

	return combinedResp, nil
}

func getPnL(pos *models.FXPosition, currentPrice float64) float64 {
	initMargin := pos.GetInitialMargin()
	if pos.Direction == models.FXPositionDirectionBuy {
		return float64(initMargin) * float64(pos.Leverage) * ((currentPrice / pos.EntryPrice) - 1.0)
	} else {
		return float64(initMargin) * float64(pos.Leverage) * (1.0 - (currentPrice / pos.EntryPrice))
	}
}

func getLiquidationPrice(pos *models.FXPosition) float64 {
	initMargin := pos.GetInitialMargin()
	opt := getLeverageOption(pos.Leverage)
	targetValuation := float64(initMargin) * (opt.LiquidationRatio / 100.0)
	diff := targetValuation - float64(pos.Margin)

	if pos.Direction == models.FXPositionDirectionBuy {
		return pos.EntryPrice * (1.0 + diff/(float64(initMargin)*float64(pos.Leverage)))
	} else {
		return pos.EntryPrice * (1.0 - diff/(float64(initMargin)*float64(pos.Leverage)))
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
	if pos.Direction == models.FXPositionDirectionBuy {
		currentPrice = bid
	} else {
		currentPrice = ask
	}
	pnl := getPnL(pos, currentPrice)

	opt := getLeverageOption(pos.Leverage)
	initMargin := pos.GetInitialMargin()
	ratio := (float64(pos.Margin) + pnl) / float64(initMargin) * 100.0
	if ratio <= opt.LiquidationRatio {
		return true, currentPrice, pnl, nil
	}

	return false, currentPrice, pnl, nil
}

func getFXPositions(c *components.Components, userID snowflake.ID, guildID snowflake.ID) ([]models.FXPosition, error) {
	var positions []models.FXPosition
	err := c.GormDB().Where("user_id = ? AND guild_id = ?", userID, guildID).Find(&positions).Error
	if err != nil {
		return nil, err
	}
	return positions, nil
}

func getFXPositionByID(c *components.Components, id uuid.UUID) (*models.FXPosition, error) {
	var pos models.FXPosition
	err := c.GormDB().Where("id = ?", id).First(&pos).Error
	if err != nil {
		return nil, err
	}
	return &pos, nil
}

func hasMarginCall(c *components.Components, userID snowflake.ID, guildID snowflake.ID) (bool, error) {
	positions, err := getFXPositions(c, userID, guildID)
	if err != nil {
		return false, err
	}
	if len(positions) == 0 {
		return false, nil
	}
	ticker, err := fetchTickerData()
	if err != nil {
		return false, err
	}
	for _, pos := range positions {
		symbolData, ok := ticker.GetSymbolData(pos.Symbol)
		if !ok {
			continue
		}
		ask, _ := strconv.ParseFloat(symbolData.Ask, 64)
		bid, _ := strconv.ParseFloat(symbolData.Bid, 64)
		var currentPrice float64
		if pos.Direction == models.FXPositionDirectionBuy {
			currentPrice = bid
		} else {
			currentPrice = ask
		}
		pnl := getPnL(&pos, currentPrice)
		initMargin := pos.GetInitialMargin()
		ratio := (float64(pos.Margin) + pnl) / float64(initMargin) * 100.0
		opt := getLeverageOption(pos.Leverage)
		if ratio < opt.MarginCallRatio {
			return true, nil
		}
	}
	return false, nil
}

func liquidatePosition(c *components.Components, client *bot.Client, pos *models.FXPosition, ticker *TickerResponse, currentPrice float64, pnl float64) error {
	pnlInt := int64(pnl)
	var deficit int64
	if pos.Margin+pnlInt < 0 {
		deficit = -(pos.Margin + pnlInt)
	}

	type closedInfo struct {
		symbol    string
		direction string
		exitPrice float64
		pnl       int64
		margin    int64
		valuation int64
	}
	var closedPositions []closedInfo

	err := c.GormDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(pos).Error; err != nil {
			return err
		}

		if deficit > 0 {
			var otherPositions []models.FXPosition
			if err := tx.Where("user_id = ? AND guild_id = ? AND id != ?", pos.UserID, pos.GuildID, pos.ID).Find(&otherPositions).Error; err != nil {
				return err
			}

			for _, other := range otherPositions {
				if deficit <= 0 {
					break
				}

				otherSymData, ok := ticker.GetSymbolData(other.Symbol)
				if !ok {
					continue
				}
				otherAsk, _ := strconv.ParseFloat(otherSymData.Ask, 64)
				otherBid, _ := strconv.ParseFloat(otherSymData.Bid, 64)
				var otherPrice float64
				if other.Direction == models.FXPositionDirectionBuy {
					otherPrice = otherBid
				} else {
					otherPrice = otherAsk
				}
				otherPnl := getPnL(&other, otherPrice)
				otherVal := other.Margin + int64(otherPnl)

				if err := tx.Delete(&other).Error; err != nil {
					return err
				}

				closedPositions = append(closedPositions, closedInfo{
					symbol:    other.Symbol,
					direction: string(other.Direction),
					exitPrice: otherPrice,
					pnl:       int64(otherPnl),
					margin:    other.Margin,
					valuation: otherVal,
				})

				if otherVal > 0 {
					if otherVal >= deficit {
						remaining := otherVal - deficit
						if err := gopoint.AddPointTx(tx, pos.UserID, pos.GuildID, remaining); err != nil {
							return err
						}
						deficit = 0
					} else {
						deficit -= otherVal
					}
				}
			}
		}

		if deficit > 0 {
			userPoint := models.GoPoint{
				UserID:  pos.UserID,
				GuildID: pos.GuildID,
			}
			if err := tx.Where(userPoint).First(&userPoint).Error; err == nil {
				points := userPoint.Points
				deduct := points
				if points > deficit {
					deduct = deficit
				}
				userPoint.Points -= deduct
				if err := tx.Save(&userPoint).Error; err != nil {
					return err
				}
				deficit -= deduct
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if client != nil && client.Rest != nil {
		ch, err := client.Rest.CreateDMChannel(pos.UserID)
		if err == nil {
			locale := discord.LocaleJapanese
			var detailStr strings.Builder

			if len(closedPositions) > 0 {
				detailStr.WriteString(i18n.TranslateText(locale, "components.play.fx.liquidation_detail_header"))
				for _, cp := range closedPositions {
					detailStr.WriteString(i18n.TranslateText(locale, "components.play.fx.liquidation_detail_item", map[string]any{
						"symbol":    strings.Replace(cp.symbol, "_", "/", 1),
						"direction": cp.direction,
						"margin":    cp.margin,
						"exit":      fmt.Sprintf("%.3f", cp.exitPrice),
						"pnl":       fmt.Sprintf("%+d", cp.pnl),
						"val":       cp.valuation,
					}))
				}
			}

			if deficit > 0 {
				detailStr.WriteString(i18n.TranslateText(locale, "components.play.fx.liquidation_detail_deficit", map[string]any{
					"deficit": deficit,
				}))
			}

			descKey := "components.play.fx.liquidation_desc"
			templateMap := map[string]any{
				"symbol":  strings.Replace(pos.Symbol, "_", "/", 1),
				"exit":    fmt.Sprintf("%.3f", currentPrice),
				"pnl":     strconv.FormatInt(-pnlInt, 10),
				"margin":  strconv.FormatInt(pos.Margin, 10),
				"deficit": strconv.FormatInt(pos.Margin+pnlInt, 10),
			}
			if len(closedPositions) > 0 {
				descKey = "components.play.fx.liquidation_deficit_desc"
				templateMap["deficit"] = strconv.FormatInt(pos.Margin+pnlInt, 10)
			}

			descText := i18n.TranslateText(locale, descKey, templateMap) + detailStr.String()

			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(locale, "components.play.fx.liquidation_title")),
						discord.NewTextDisplay(descText),
					).WithAccentColor(0xE74C3C),
				)
			_, _ = client.Rest.CreateMessage(ch.ID(), builder.BuildCreate())
		}
	}

	return nil
}

func FXMessage(c *components.Components, session *FXSession, positions []models.FXPosition, ticker *TickerResponse, points int64, locale discord.Locale) []discord.LayoutComponent {
	ctx := i18n.BuildContext()

	var ratesText []string
	for _, sym := range fxSymbols {
		if data, ok := ticker.GetSymbolData(sym); ok {
			ask, _ := strconv.ParseFloat(data.Ask, 64)
			bid, _ := strconv.ParseFloat(data.Bid, 64)
			spread := (ask - bid) * 100.0
			ratesText = append(ratesText, i18n.TranslateText(locale, "components.play.fx.rates_entry", map[string]any{
				"symbol": strings.Replace(sym, "_", "/", 1),
				"bid":    fmt.Sprintf("%.3f", bid),
				"ask":    fmt.Sprintf("%.3f", ask),
				"spread": fmt.Sprintf("%.1f", spread),
			}))
		}
	}
	ctx.WithText("rates_text", strings.Join(ratesText, "\n"))
	ctx.WithText("points", strconv.FormatInt(points, 10))
	ctx.WithCustomID("uuid", session.ID.String())

	var positionOpts []discord.StringSelectMenuOption
	positionOpts = append(positionOpts, discord.StringSelectMenuOption{
		Label:   i18n.TranslateText(locale, "components.play.fx.option_new_order"),
		Value:   "new_order",
		Default: session.ActivePositionID == nil,
	})

	hasMc := false
	var activePos *models.FXPosition

	for _, p := range positions {
		pSymData, ok := ticker.GetSymbolData(p.Symbol)
		var pPrice float64
		if ok {
			pAsk, _ := strconv.ParseFloat(pSymData.Ask, 64)
			pBid, _ := strconv.ParseFloat(pSymData.Bid, 64)
			if p.Direction == models.FXPositionDirectionBuy {
				pPrice = pBid
			} else {
				pPrice = pAsk
			}
		}
		pPnl := getPnL(&p, pPrice)
		pRatio := (float64(p.Margin) + pPnl) / float64(p.GetInitialMargin()) * 100.0
		pOpt := getLeverageOption(p.Leverage)
		if pRatio < pOpt.MarginCallRatio {
			hasMc = true
		}

		pnlSign := ""
		if int64(pPnl) > 0 {
			pnlSign = "+"
		}
		dirStr := "L"
		if p.Direction == models.FXPositionDirectionSell {
			dirStr = "S"
		}
		label := i18n.TranslateText(locale, "components.play.fx.option_pos_item", map[string]any{
			"symbol":    strings.Replace(p.Symbol, "_", "/", 1),
			"direction": dirStr,
			"leverage":  p.Leverage,
			"margin":    strconv.FormatInt(p.Margin, 10),
			"pnl":       fmt.Sprintf("%s%d", pnlSign, int64(pPnl)),
		})
		positionOpts = append(positionOpts, discord.StringSelectMenuOption{
			Label:   label,
			Value:   p.ID.String(),
			Default: session.ActivePositionID != nil && *session.ActivePositionID == p.ID,
		})

		if session.ActivePositionID != nil && *session.ActivePositionID == p.ID {
			pCopy := p
			activePos = &pCopy
		}
	}

	ctx.WithStringOptions("position_options", positionOpts)

	var symbolOpts []discord.StringSelectMenuOption
	for _, sym := range fxSymbols {
		symbolOpts = append(symbolOpts, discord.StringSelectMenuOption{
			Label:   strings.Replace(sym, "_", "/", 1),
			Value:   sym,
			Default: sym == session.SelectedSymbol,
		})
	}
	ctx.WithStringOptions("symbol_options", symbolOpts)

	var leverageOpts []discord.StringSelectMenuOption
	for i, l := range fxLeverages {
		leverageOpts = append(leverageOpts, discord.StringSelectMenuOption{
			Label:   fmt.Sprintf("%dx レバレッジ", l.Leverage),
			Value:   strconv.Itoa(i),
			Default: i == session.SelectedLeverage,
		})
	}
	ctx.WithStringOptions("leverage_options", leverageOpts)

	restrictionWarning := ""
	if hasMc {
		restrictionWarning = "\n" + i18n.TranslateText(locale, "components.play.fx.trade_restricted_warning")
		ctx.WithDisabled("play:fx_buy:"+session.ID.String(), true)
		ctx.WithDisabled("play:fx_sell:"+session.ID.String(), true)
		ctx.WithDisabled("play:fx_close:"+session.ID.String(), true)
	}
	ctx.WithText("restriction_warning", restrictionWarning)

	if activePos == nil {
		opt := fxLeverages[session.SelectedLeverage]
		formInfo := i18n.TranslateText(locale, "components.play.fx.form_selected", map[string]any{
			"symbol":   strings.Replace(session.SelectedSymbol, "_", "/", 1),
			"margin":   strconv.FormatInt(session.SelectedMargin, 10),
			"leverage": opt.Leverage,
		})
		ctx.WithText("form_info", formInfo)
		ctx.WithText("btn_input_margin", i18n.TranslateText(locale, "components.play.fx.btn_input_margin", map[string]any{
			"margin": strconv.FormatInt(session.SelectedMargin, 10),
		}))

		return ctx.Translate(i18n.TranslateLayout(locale, "command.play.fx.order_screen"))
	} else {
		pSymData, _ := ticker.GetSymbolData(activePos.Symbol)
		pAsk, _ := strconv.ParseFloat(pSymData.Ask, 64)
		pBid, _ := strconv.ParseFloat(pSymData.Bid, 64)
		var currentPrice float64
		if activePos.Direction == models.FXPositionDirectionBuy {
			currentPrice = pBid
		} else {
			currentPrice = pAsk
		}
		pPnl := getPnL(activePos, currentPrice)
		pnlInt := int64(pPnl)
		pnlStr := fmt.Sprintf("%+d", pnlInt)
		if pnlInt == 0 {
			pnlStr = "0"
		}

		dirEmoji := i18n.TranslateText(locale, "components.play.fx.direction.buy")
		if activePos.Direction == models.FXPositionDirectionSell {
			dirEmoji = i18n.TranslateText(locale, "components.play.fx.direction.sell")
		}

		initMargin := activePos.GetInitialMargin()
		ratio := (float64(activePos.Margin) + pPnl) / float64(initMargin) * 100.0
		ratioText := fmt.Sprintf("%.1f%%", ratio)
		liqPrice := getLiquidationPrice(activePos)
		activeOpt := getLeverageOption(activePos.Leverage)

		posDesc := i18n.TranslateText(locale, "components.play.fx.active_position_desc", map[string]any{
			"symbol":    strings.Replace(activePos.Symbol, "_", "/", 1),
			"direction": dirEmoji,
			"leverage":  activePos.Leverage,
			"margin":    strconv.FormatInt(activePos.Margin, 10),
			"init":      strconv.FormatInt(initMargin, 10),
			"size":      strconv.FormatInt(initMargin*int64(activePos.Leverage), 10),
			"entry":     fmt.Sprintf("%.3f", activePos.EntryPrice),
			"current":   fmt.Sprintf("%.3f", currentPrice),
			"liq":       fmt.Sprintf("%.3f", liqPrice),
			"pnl":       pnlStr,
			"ratio":     ratioText,
			"mc_line":   fmt.Sprintf("%.1f%%", activeOpt.MarginCallRatio),
			"liq_line":  fmt.Sprintf("%.1f%%", activeOpt.LiquidationRatio),
		})

		var posInfoBuilder strings.Builder
		totalPos := len(positions)
		currentIndex := 1
		for idx, p := range positions {
			if p.ID == activePos.ID {
				currentIndex = idx + 1
				break
			}
		}

		posInfoBuilder.WriteString(i18n.TranslateText(locale, "components.play.fx.active_position_title", map[string]any{
			"current": currentIndex,
			"total":   totalPos,
		}))
		posInfoBuilder.WriteString("\n" + posDesc)

		if ratio < activeOpt.MarginCallRatio {
			posInfoBuilder.WriteString("\n\n" + i18n.TranslateText(locale, "components.play.fx.margin_call_warning", map[string]any{
				"mc_line": fmt.Sprintf("%.1f%%", activeOpt.MarginCallRatio),
			}))
		}
		ctx.WithText("position_info", posInfoBuilder.String())

		return ctx.Translate(i18n.TranslateLayout(locale, "command.play.fx.position_screen"))
	}
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
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_session_expired")),
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
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_not_your_session")),
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

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	var remainingPositions []models.FXPosition
	for _, p := range positions {
		posCopy := p
		liquidated, currentPrice, pnl, err := checkLiquidation(c, &posCopy, ticker)
		if err != nil {
			return errors.NewError(err)
		}
		if liquidated {
			if err := liquidatePosition(c, event.Client(), &p, ticker, currentPrice, pnl); err != nil {
				return errors.NewError(err)
			}
		} else {
			remainingPositions = append(remainingPositions, p)
		}
	}
	positions = remainingPositions

	session := &FXSession{
		ID:               uuid.New(),
		UserID:           event.User().ID,
		GuildID:          *event.GuildID(),
		SelectedSymbol:   "USD_JPY",
		SelectedMargin:   100,
		SelectedLeverage: 0,
	}
	if len(positions) > 0 {
		session.ActivePositionID = &positions[0].ID
	}
	fx_sessions.Set(session.ID, session)

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...),
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
		val := data.Values[0]
		isValid := false
		for _, s := range fxSymbols {
			if s == val {
				isValid = true
				break
			}
		}
		if !isValid {
			return errors.NewError(fmt.Errorf("invalid symbol selected"))
		}
		session.SelectedSymbol = val
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

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXMarginButtonHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	modal := discord.NewModalCreateBuilder().
		SetTitle(i18n.TranslateText(event.Locale(), "components.play.fx.modal_input_margin_title")).
		SetCustomID("play:fx_margin_modal:" + session.ID.String()).
		SetComponents(
			discord.NewLabel(i18n.TranslateText(event.Locale(), "components.play.fx.modal_input_margin_label"),
				discord.TextInputComponent{
					CustomID:    "margin",
					Style:       discord.TextInputStyleShort,
					MinLength:   ptr(1),
					MaxLength:   10,
					Required:    true,
					Value:       strconv.FormatInt(session.SelectedMargin, 10),
					Placeholder: "例: 100",
				},
			),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXMarginModalHandler(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	args := strings.Split(event.Data.CustomID, ":")
	if len(args) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	id, err := uuid.Parse(args[2])
	if err != nil {
		return errors.NewError(err)
	}
	session, ok := fx_sessions.Get(id)
	if !ok {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_session_expired")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}
	if session.UserID != event.User().ID {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_not_your_session")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	marginStr := event.Data.Text("margin")
	margin, err := strconv.ParseInt(strings.TrimSpace(marginStr), 10, 64)
	if err != nil || margin <= 0 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_invalid_input")),
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

	opt := fxLeverages[session.SelectedLeverage]
	minMargin := int64(float64(points) * opt.MinRatio)
	if minMargin < 1 {
		minMargin = 1
	}

	if margin < minMargin {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_min_ratio_margin", map[string]any{
						"leverage": opt.Leverage,
						"points":   points,
						"ratio":    opt.MinRatio * 100.0,
						"min":      minMargin,
					})),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	session.SelectedMargin = margin
	fx_sessions.Set(session.ID, session)

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXAddMarginButtonHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	modal := discord.NewModalCreateBuilder().
		SetTitle(i18n.TranslateText(event.Locale(), "components.play.fx.modal_add_margin_title")).
		SetCustomID("play:fx_add_margin_modal:" + session.ID.String()).
		SetComponents(
			discord.NewLabel(i18n.TranslateText(event.Locale(), "components.play.fx.modal_add_margin_label"),
				discord.TextInputComponent{
					CustomID:    "amount",
					Style:       discord.TextInputStyleShort,
					MinLength:   ptr(1),
					MaxLength:   10,
					Required:    true,
					Placeholder: "例: 500",
				},
			),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func FXAddMarginModalHandler(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	args := strings.Split(event.Data.CustomID, ":")
	if len(args) < 3 {
		return errors.NewError(fmt.Errorf("invalid custom ID"))
	}
	id, err := uuid.Parse(args[2])
	if err != nil {
		return errors.NewError(err)
	}
	session, ok := fx_sessions.Get(id)
	if !ok {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_session_expired")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}
	if session.UserID != event.User().ID {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_not_your_session")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	amountStr := event.ModalSubmitInteraction.Data.Text("amount")
	amount, err := strconv.ParseInt(strings.TrimSpace(amountStr), 10, 64)
	if err != nil || amount <= 0 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_invalid_input")),
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

	if points < amount {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_insufficient_points", map[string]any{
						"margin": amount,
						"points": points,
					})),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	if session.ActivePositionID == nil {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_pos_not_found")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	pos, err := getFXPositionByID(c, *session.ActivePositionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_pos_not_found")),
					).WithAccentColor(0xE74C3C),
				).
				AddFlags(discord.MessageFlagEphemeral)
			_ = event.RespondMessage(builder)
			return nil
		}
		return errors.NewError(err)
	}

	if err := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -amount); err != nil {
		return errors.NewError(err)
	}

	pos.Margin += amount
	pos.MarginCallNotified = false
	if err := c.GormDB().Save(pos).Error; err != nil {
		if refundErr := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), amount); refundErr != nil {
			slog.Error("CRITICAL: failed to refund points to user after margin addition failed", "user_id", event.User().ID, "guild_id", *event.GuildID(), "session_id", session.ID, "refund", amount, "error", refundErr)
		}
		return errors.NewError(err)
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return errors.NewError(err)
	}

	points, _, _ = gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
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
		if l < 0 || l >= len(fxLeverages) {
			return errors.NewError(fmt.Errorf("invalid leverage option selected"))
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

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
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

	mcRestricted, err := hasMarginCall(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	if mcRestricted {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_restricted")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if len(positions) >= 10 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_max_positions")),
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

	opt := fxLeverages[session.SelectedLeverage]
	minMargin := int64(float64(points) * opt.MinRatio)
	if minMargin < 1 {
		minMargin = 1
	}

	if session.SelectedMargin < minMargin {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_min_ratio_margin", map[string]any{
						"leverage": opt.Leverage,
						"points":   points,
						"ratio":    opt.MinRatio * 100.0,
						"min":      minMargin,
					})),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	if points < session.SelectedMargin {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_insufficient_points", map[string]any{
						"margin": session.SelectedMargin,
						"points": points,
					})),
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

	pos := &models.FXPosition{
		UserID:        event.User().ID,
		GuildID:       *event.GuildID(),
		Symbol:        session.SelectedSymbol,
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    ask,
		Margin:        session.SelectedMargin,
		InitialMargin: session.SelectedMargin,
		Leverage:      opt.Leverage,
	}

	if err := c.GormDB().Create(pos).Error; err != nil {
		if refundErr := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), session.SelectedMargin); refundErr != nil {
			slog.Error("CRITICAL: failed to refund points to user after position creation failed", "user_id", event.User().ID, "guild_id", *event.GuildID(), "session_id", session.ID, "refund", session.SelectedMargin, "error", refundErr)
		}
		return errors.NewError(err)
	}

	session.ActivePositionID = &pos.ID
	fx_sessions.Set(session.ID, session)

	points, _, _ = gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	positions, _ = getFXPositions(c, event.User().ID, *event.GuildID())

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
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

	mcRestricted, err := hasMarginCall(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	if mcRestricted {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_restricted")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if len(positions) >= 10 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_max_positions")),
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

	opt := fxLeverages[session.SelectedLeverage]
	minMargin := int64(float64(points) * opt.MinRatio)
	if minMargin < 1 {
		minMargin = 1
	}

	if session.SelectedMargin < minMargin {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_min_ratio_margin", map[string]any{
						"leverage": opt.Leverage,
						"points":   points,
						"ratio":    opt.MinRatio * 100.0,
						"min":      minMargin,
					})),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	if points < session.SelectedMargin {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_insufficient_points", map[string]any{
						"margin": session.SelectedMargin,
						"points": points,
					})),
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

	pos := &models.FXPosition{
		UserID:        event.User().ID,
		GuildID:       *event.GuildID(),
		Symbol:        session.SelectedSymbol,
		Direction:     models.FXPositionDirectionSell,
		EntryPrice:    bid,
		Margin:        session.SelectedMargin,
		InitialMargin: session.SelectedMargin,
		Leverage:      opt.Leverage,
	}

	if err := c.GormDB().Create(pos).Error; err != nil {
		if refundErr := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), session.SelectedMargin); refundErr != nil {
			slog.Error("CRITICAL: failed to refund points to user after position creation failed", "user_id", event.User().ID, "guild_id", *event.GuildID(), "session_id", session.ID, "refund", session.SelectedMargin, "error", refundErr)
		}
		return errors.NewError(err)
	}

	session.ActivePositionID = &pos.ID
	fx_sessions.Set(session.ID, session)

	points, _, _ = gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	positions, _ = getFXPositions(c, event.User().ID, *event.GuildID())

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
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

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	var remainingPositions []models.FXPosition
	activeLiquidated := false
	for _, p := range positions {
		posCopy := p
		liquidated, currentPrice, pnl, err := checkLiquidation(c, &posCopy, ticker)
		if err != nil {
			return errors.NewError(err)
		}
		if liquidated {
			if err := liquidatePosition(c, event.Client(), &p, ticker, currentPrice, pnl); err != nil {
				return errors.NewError(err)
			}
			if session.ActivePositionID != nil && *session.ActivePositionID == p.ID {
				activeLiquidated = true
			}
		} else {
			remainingPositions = append(remainingPositions, p)
		}
	}
	positions = remainingPositions

	if activeLiquidated {
		if len(positions) > 0 {
			session.ActivePositionID = &positions[0].ID
		} else {
			session.ActivePositionID = nil
		}
		fx_sessions.Set(session.ID, session)
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
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

	mcRestricted, err := hasMarginCall(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	if mcRestricted {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_restricted")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	if session.ActivePositionID == nil {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_pos_not_found")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	pos, err := getFXPositionByID(c, *session.ActivePositionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.err_pos_not_found")),
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

	ask, err := strconv.ParseFloat(symbolData.Ask, 64)
	if err != nil {
		return errors.NewError(err)
	}
	bid, err := strconv.ParseFloat(symbolData.Bid, 64)
	if err != nil {
		return errors.NewError(err)
	}

	var exitPrice float64
	if pos.Direction == models.FXPositionDirectionBuy {
		exitPrice = bid
	} else {
		exitPrice = ask
	}
	pnl := getPnL(pos, exitPrice)

	pnlInt := int64(pnl)
	refund := pos.Margin + pnlInt
	var deficit int64
	if refund < 0 {
		deficit = -refund
		refund = 0
	}

	txErr := c.GormDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(pos).Error; err != nil {
			return err
		}
		if refund > 0 {
			if err := gopoint.AddPointTx(tx, event.User().ID, *event.GuildID(), refund); err != nil {
				return err
			}
		}

		if deficit > 0 {
			var otherPositions []models.FXPosition
			if err := tx.Where("user_id = ? AND guild_id = ? AND id != ?", pos.UserID, pos.GuildID, pos.ID).Find(&otherPositions).Error; err != nil {
				return err
			}

			for _, other := range otherPositions {
				if deficit <= 0 {
					break
				}

				otherSymData, ok := ticker.GetSymbolData(other.Symbol)
				if !ok {
					continue
				}
				otherAsk, _ := strconv.ParseFloat(otherSymData.Ask, 64)
				otherBid, _ := strconv.ParseFloat(otherSymData.Bid, 64)
				var otherPrice float64
				if other.Direction == models.FXPositionDirectionBuy {
					otherPrice = otherBid
				} else {
					otherPrice = otherAsk
				}
				otherPnl := getPnL(&other, otherPrice)
				otherVal := other.Margin + int64(otherPnl)

				if err := tx.Delete(&other).Error; err != nil {
					return err
				}

				if otherVal > 0 {
					if otherVal >= deficit {
						remaining := otherVal - deficit
						if err := gopoint.AddPointTx(tx, pos.UserID, pos.GuildID, remaining); err != nil {
							return err
						}
						deficit = 0
					} else {
						deficit -= otherVal
					}
				}
			}
		}

		if deficit > 0 {
			userPoint := models.GoPoint{
				UserID:  pos.UserID,
				GuildID: pos.GuildID,
			}
			if err := tx.Where(userPoint).First(&userPoint).Error; err == nil {
				points := userPoint.Points
				deduct := points
				if points > deficit {
					deduct = deficit
				}
				userPoint.Points -= deduct
				if err := tx.Save(&userPoint).Error; err != nil {
					return err
				}
				deficit -= deduct
			}
		}
		return nil
	})
	if txErr != nil {
		return errors.NewError(txErr)
	}

	points, _, _ := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	positions, _ := getFXPositions(c, event.User().ID, *event.GuildID())
	if len(positions) > 0 {
		session.ActivePositionID = &positions[0].ID
	} else {
		session.ActivePositionID = nil
	}
	fx_sessions.Set(session.ID, session)

	netWinSign := ""
	if pnlInt > 0 {
		netWinSign = "+"
	}
	pnlStr := fmt.Sprintf("%s%d", netWinSign, pnlInt)
	if pnlInt == 0 {
		pnlStr = "0"
	}

	dirText := i18n.TranslateText(event.Locale(), "components.play.fx.direction.buy")
	if pos.Direction == models.FXPositionDirectionSell {
		dirText = i18n.TranslateText(event.Locale(), "components.play.fx.direction.sell")
	}

	descText := i18n.TranslateText(event.Locale(), "components.play.fx.close_success_desc", map[string]any{
		"symbol":    strings.Replace(pos.Symbol, "_", "/", 1),
		"direction": dirText,
		"leverage":  pos.Leverage,
		"margin":    pos.Margin,
		"entry":     fmt.Sprintf("%.3f", pos.EntryPrice),
		"exit":      fmt.Sprintf("%.3f", exitPrice),
		"pnl":       pnlStr,
		"refund":    refund,
		"points":    points,
	})

	container := discord.NewContainer().WithAccentColor(0x2ECC71)
	container = container.AddComponents(
		discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.close_success_title")),
		discord.NewLargeSeparator(),
		discord.NewTextDisplay(descText),
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

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	container := discord.NewContainer().WithAccentColor(0x95A5A6)
	container = container.AddComponents(
		discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.close_quit_title")),
		discord.NewLargeSeparator(),
		discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.fx.close_quit_desc")),
	)

	if len(positions) > 0 {
		for _, p := range positions {
			dirText := i18n.TranslateText(event.Locale(), "components.play.fx.direction.buy")
			if p.Direction == models.FXPositionDirectionSell {
				dirText = i18n.TranslateText(event.Locale(), "components.play.fx.direction.sell")
			}
			noticeText := i18n.TranslateText(event.Locale(), "components.play.fx.close_quit_active_notice", map[string]any{
				"symbol":    strings.Replace(p.Symbol, "_", "/", 1),
				"direction": dirText,
				"margin":    p.Margin,
			})
			container = container.AddComponents(discord.NewTextDisplay(noticeText))
		}
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

func FXSwitchPositionHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := FXPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
		val := data.Values[0]
		if val == "new_order" {
			session.ActivePositionID = nil
		} else {
			id, err := uuid.Parse(val)
			if err != nil {
				return errors.NewError(err)
			}
			session.ActivePositionID = &id
		}
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

	positions, err := getFXPositions(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(FXMessage(c, session, positions, ticker, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func CheckAllPositionsLiquidation(c *components.Components, client *bot.Client) error {
	var positions []models.FXPosition
	if err := c.GormDB().Find(&positions).Error; err != nil {
		return err
	}
	if len(positions) == 0 {
		return nil
	}

	ticker, err := fetchTickerData()
	if err != nil {
		return err
	}

	for _, pos := range positions {
		posCopy := pos
		liquidated, currentPrice, pnl, err := checkLiquidation(c, &posCopy, ticker)
		if err != nil {
			slog.Error("failed to check background liquidation", "user_id", pos.UserID, "error", err)
			continue
		}
		if liquidated {
			slog.Info("position background liquidated", "user_id", pos.UserID, "symbol", pos.Symbol)
			if err := liquidatePosition(c, client, &posCopy, ticker, currentPrice, pnl); err != nil {
				slog.Error("failed to liquidate position background", "user_id", pos.UserID, "error", err)
			}
		} else {
			opt := getLeverageOption(posCopy.Leverage)
			initMargin := posCopy.GetInitialMargin()
			ratio := (float64(posCopy.Margin) + pnl) / float64(initMargin) * 100.0
			if ratio < opt.MarginCallRatio && !posCopy.MarginCallNotified {
				if client != nil && client.Rest != nil {
					ch, err := client.Rest.CreateDMChannel(posCopy.UserID)
					if err == nil {
						locale := discord.LocaleJapanese
						descText := i18n.TranslateText(locale, "components.play.fx.dm_margin_call_desc", map[string]any{
							"symbol":  strings.Replace(posCopy.Symbol, "_", "/", 1),
							"ratio":   fmt.Sprintf("%.1f%%", ratio),
							"mc_line": fmt.Sprintf("%.1f%%", opt.MarginCallRatio),
						})
						builder := discord.NewMessageBuilder().
							SetIsComponentsV2(true).
							SetComponents(
								discord.NewContainer(
									discord.NewTextDisplay(i18n.TranslateText(locale, "components.play.fx.dm_margin_call_title")),
									discord.NewTextDisplay(descText),
								).WithAccentColor(0xF1C40F),
							)
						_, _ = client.Rest.CreateMessage(ch.ID(), builder.BuildCreate())
					}
				}
				posCopy.MarginCallNotified = true
				if err := c.GormDB().Save(&posCopy).Error; err != nil {
					slog.Error("failed to save margin call notified state", "pos_id", posCopy.ID, "error", err)
				}
			}
		}
	}
	return nil
}
