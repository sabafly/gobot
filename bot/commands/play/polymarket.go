package play

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoPolymarket/polymarket-go-sdk/v2"
	"github.com/GoPolymarket/polymarket-go-sdk/v2/pkg/gamma"
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
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	pm_sessions = database.NewMemoryValues[uuid.UUID, *PolymarketSession](time.Minute * 15)
	pmClient    *polymarket.Client
	pmClientMu  sync.Mutex
)

type PolymarketSession struct {
	ID             uuid.UUID
	UserID         snowflake.ID
	GuildID        snowflake.ID
	Markets        []gamma.Market
	SelectedMarket *gamma.Market
	SelectedToken  *gamma.Token
	SearchQuery    string
	StatusMsg      string
	StatusHeader   string
}

func (s *PolymarketSession) OnDelete() error {
	return nil
}

func getPMClient() *polymarket.Client {
	pmClientMu.Lock()
	defer pmClientMu.Unlock()
	if pmClient == nil {
		pmClient = polymarket.NewClient()
	}
	return pmClient
}

func GetMarketTokens(m *gamma.Market) []gamma.Token {
	if m == nil {
		return nil
	}
	tokens := m.ParsedTokens()
	if len(tokens) == 0 {
		return nil
	}
	// Parse prices from OutcomePrices if not populated
	var prices []string
	if err := json.Unmarshal([]byte(m.OutcomePrices), &prices); err == nil {
		for i, pStr := range prices {
			if i < len(tokens) {
				if pVal, err := strconv.ParseFloat(pStr, 64); err == nil {
					tokens[i].Price = pVal
				}
			}
		}
	}
	return tokens
}

func formatDecimal(d decimal.Decimal) string {
	val, _ := d.Float64()
	if val >= 1000000000 {
		return fmt.Sprintf("%.2fB", val/1000000000.0)
	} else if val >= 1000000 {
		return fmt.Sprintf("%.2fM", val/1000000.0)
	} else if val >= 1000 {
		return fmt.Sprintf("%.2fk", val/1000.0)
	}
	return fmt.Sprintf("%.2f", val)
}

func formatTimeStr(utcStr string) string {
	t, err := time.Parse(time.RFC3339, utcStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z", utcStr)
		if err != nil {
			return utcStr
		}
	}
	return t.Local().Format("2006/01/02 15:04")
}

func PolymarketSearchMessage(c *components.Components, session *PolymarketSession, points int64, locale discord.Locale) []discord.LayoutComponent {
	ctx := i18n.BuildContext()

	var marketOpts []discord.StringSelectMenuOption
	for _, m := range session.Markets {
		label := m.Question
		if len(label) > 100 {
			label = label[:97] + "..."
		}
		desc := i18n.TranslateText(locale, "components.play.polymarket.market_option_desc", map[string]any{
			"volume":   formatDecimal(m.Volume),
			"end_date": formatTimeStr(m.EndDate),
		})
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		marketOpts = append(marketOpts, discord.StringSelectMenuOption{
			Label:       label,
			Value:       m.ID,
			Description: desc,
		})
	}

	if len(marketOpts) == 0 {
		marketOpts = append(marketOpts, discord.StringSelectMenuOption{
			Label: i18n.TranslateText(locale, "components.play.polymarket.market_not_found_option"),
			Value: "none",
		})
	}

	queryInfo := i18n.TranslateText(locale, "components.play.polymarket.default_query_info")
	if session.SearchQuery != "" {
		queryInfo = i18n.TranslateText(locale, "components.play.polymarket.search_query_info", map[string]any{
			"query": session.SearchQuery,
		})
	}

	status := ""
	if session.StatusMsg != "" {
		status = "\n\n" + session.StatusMsg
	}
	ctx.WithText("status", status)
	ctx.WithText("query_info", queryInfo)
	ctx.WithText("points", strconv.FormatInt(points, 10))
	ctx.WithCustomID("uuid", session.ID.String())
	ctx.WithStringOptions("market_options", marketOpts)

	return ctx.Translate(i18n.TranslateLayout(locale, "command.play.polymarket.search_screen"))
}

func PolymarketDetailMessage(c *components.Components, session *PolymarketSession, points int64, locale discord.Locale) []discord.LayoutComponent {
	ctx := i18n.BuildContext()

	m := session.SelectedMarket
	tokens := GetMarketTokens(m)

	var oddsText []string
	var outcomeOpts []discord.StringSelectMenuOption

	for _, tok := range tokens {
		percentage := tok.Price * 100.0
		oddsText = append(oddsText, i18n.TranslateText(locale, "components.play.polymarket.odds_item", map[string]any{
			"outcome":    tok.Outcome,
			"percentage": fmt.Sprintf("%.1f", percentage),
			"price":      fmt.Sprintf("%.2f", tok.Price),
		}))

		var label string
		if len(tok.Outcome) > 100 {
			label = tok.Outcome[:97] + "..."
		} else {
			label = tok.Outcome
		}

		desc := i18n.TranslateText(locale, "components.play.polymarket.outcome_option_desc", map[string]any{
			"price":      fmt.Sprintf("%.2f", tok.Price),
			"multiplier": fmt.Sprintf("%.2f", 1.0/tok.Price),
		})
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}

		outcomeOpts = append(outcomeOpts, discord.StringSelectMenuOption{
			Label:       label,
			Value:       tok.TokenID,
			Description: desc,
		})
	}

	if len(outcomeOpts) == 0 {
		outcomeOpts = append(outcomeOpts, discord.StringSelectMenuOption{
			Label: i18n.TranslateText(locale, "components.play.polymarket.no_outcomes_option"),
			Value: "none",
		})
	}

	questionLink := fmt.Sprintf("[%s](https://polymarket.com/event/%s)", m.Question, m.Slug)
	ctx.WithText("market_question", questionLink)
	ctx.WithText("end_date", formatTimeStr(m.EndDate))
	ctx.WithText("volume", formatDecimal(m.Volume))
	ctx.WithText("liquidity", formatDecimal(m.Liquidity))
	ctx.WithText("odds_text", i18n.TranslateText(locale, "components.play.polymarket.odds_label")+"\n"+strings.Join(oddsText, "\n"))
	ctx.WithText("points", strconv.FormatInt(points, 10))
	ctx.WithCustomID("uuid", session.ID.String())
	ctx.WithStringOptions("outcome_options", outcomeOpts)

	return ctx.Translate(i18n.TranslateLayout(locale, "command.play.polymarket.market_detail_screen"))
}

func PolymarketBetConfirmMessage(c *components.Components, session *PolymarketSession, points int64, locale discord.Locale) []discord.LayoutComponent {
	ctx := i18n.BuildContext()

	m := session.SelectedMarket
	tok := session.SelectedToken

	questionLink := fmt.Sprintf("[%s](https://polymarket.com/event/%s)", m.Question, m.Slug)
	ctx.WithText("market_question", questionLink)
	ctx.WithText("selected_outcome", tok.Outcome)
	ctx.WithText("outcome_price", fmt.Sprintf("%.2f", tok.Price))
	ctx.WithText("multiplier", fmt.Sprintf("%.2f", 1.0/tok.Price))
	ctx.WithText("points", strconv.FormatInt(points, 10))
	ctx.WithCustomID("uuid", session.ID.String())

	return ctx.Translate(i18n.TranslateLayout(locale, "command.play.polymarket.bet_placement_screen"))
}

func PolymarketMyBetsMessage(c *components.Components, session *PolymarketSession, points int64, locale discord.Locale) []discord.LayoutComponent {
	ctx := i18n.BuildContext()

	var bets []models.PolymarketBet
	err := c.GormDB().Where("user_id = ? AND guild_id = ?", session.UserID, session.GuildID).Order("created_at desc").Limit(10).Find(&bets).Error

	var betsList string
	if err != nil || len(bets) == 0 {
		betsList = i18n.TranslateText(locale, "components.play.polymarket.no_bet_history")
	} else {
		var lines []string
		for _, b := range bets {
			status := i18n.TranslateText(locale, "components.play.polymarket.status_pending")
			if b.Resolved {
				if b.Winner {
					status = i18n.TranslateText(locale, "components.play.polymarket.status_won", map[string]any{"payout": b.Payout})
				} else {
					status = i18n.TranslateText(locale, "components.play.polymarket.status_lost")
				}
			}
			timeStr := b.CreatedAt.Format("2006/01/02 15:04")
			lines = append(lines, i18n.TranslateText(locale, "components.play.polymarket.bet_history_item", map[string]any{
				"market":  b.MarketTitle,
				"outcome": b.Outcome,
				"price":   fmt.Sprintf("%.2f", b.EntryPrice),
				"amount":  b.BetAmount,
				"status":  status,
				"time":    timeStr,
			}))
		}
		betsList = strings.Join(lines, "\n")
	}

	statusHeader := ""
	if session.StatusHeader != "" {
		statusHeader = session.StatusHeader + "\n"
	}
	ctx.WithText("status_header", statusHeader)
	ctx.WithText("bets_list", betsList)
	ctx.WithCustomID("uuid", session.ID.String())

	return ctx.Translate(i18n.TranslateLayout(locale, "command.play.polymarket.my_bets_screen"))
}

func PolymarketPrecondition(event *events.ComponentInteractionCreate) (*PolymarketSession, errors.Error) {
	customID := strings.Split(event.Data.CustomID(), ":")
	if len(customID) < 3 {
		return nil, errors.NewError(fmt.Errorf("invalid component custom ID format"))
	}
	sessionUUIDStr := customID[len(customID)-1]
	sessionUUID, err := uuid.Parse(sessionUUIDStr)
	if err != nil {
		return nil, errors.NewError(err)
	}
	session, ok := pm_sessions.Get(sessionUUID)
	if !ok {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.err_session_expired")),
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
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.err_not_your_session")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil, nil
	}
	return session, nil
}

type marketWithScore struct {
	market gamma.Market
	score  float64
}

func PolymarketPlayCommand(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	if err := event.DeferCreateMessage(false); err != nil {
		return errors.NewError(err)
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	query := ""
	if opt, ok := event.SlashCommandInteractionData().OptString("query"); ok {
		query = opt
	}

	includeClosed := false
	if opt, ok := event.SlashCommandInteractionData().OptBool("closed"); ok {
		includeClosed = opt
	}

	pm := getPMClient()
	var markets []gamma.Market
	var apiErr error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if query != "" {
		limit := 15
		var activeStatus string
		if !includeClosed {
			activeStatus = "active"
		}
		results, err := pm.Gamma.PublicSearch(ctx, &gamma.PublicSearchRequest{
			Query:        query,
			EventsStatus: activeStatus,
			LimitPerType: &limit,
		})
		if err == nil {
			for _, ev := range results.Events {
				if includeClosed || (ev.Active && !ev.Closed) {
					for _, mk := range ev.Markets {
						if includeClosed || (mk.Active && !mk.Closed) {
							if len(mk.ClobTokenIds) > 0 {
								markets = append(markets, mk)
							}
						}
					}
				}
			}
		} else {
			apiErr = err
			slog.Error("Polymarket PublicSearch failed", "error", err, "query", query)
		}
	} else {
		limit := 20
		var active *bool
		var closed *bool
		if !includeClosed {
			activeVal := true
			closedVal := false
			active = &activeVal
			closed = &closedVal
		}
		var err error
		markets, err = pm.Gamma.Markets(ctx, &gamma.MarketsRequest{
			Limit:  &limit,
			Active: active,
			Closed: closed,
			Order:  "volume",
		})
		if err != nil {
			apiErr = err
			slog.Error("Polymarket Markets fetch failed", "error", err)
		}
	}

	if apiErr != nil {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(fmt.Sprintf("❌ Polymarket APIとの通信に失敗しました。\nエラー: `%s`", apiErr.Error())),
				).WithAccentColor(0xE74C3C),
			)
		_ = event.RespondMessage(builder)
		return nil
	}

	var validMarkets []gamma.Market
	for _, m := range markets {
		if !includeClosed {
			if !m.Active || m.Closed {
				continue
			}
		}
		toks := GetMarketTokens(&m)
		if len(toks) > 0 {
			validMarkets = append(validMarkets, m)
		}
	}

	// Score and sort validMarkets: future, recent, high volume/trend
	now := time.Now()
	scores := make([]marketWithScore, len(validMarkets))
	for i, m := range validMarkets {
		endDate, parseErr := time.Parse(time.RFC3339, m.EndDate)
		if parseErr != nil {
			endDate, parseErr = time.Parse("2006-01-02T15:04:05Z", m.EndDate)
		}

		volVal, _ := m.Volume.Float64()
		liqVal, _ := m.Liquidity.Float64()
		trendiness := volVal
		if liqVal > trendiness {
			trendiness = liqVal
		}
		if trendiness < 1.0 {
			trendiness = 1.0
		}

		var score float64
		if parseErr == nil {
			duration := endDate.Sub(now)
			if duration > 0 {
				days := duration.Hours() / 24.0
				score = trendiness / (days + 3.0)
			} else {
				daysPast := -duration.Hours() / 24.0
				score = (trendiness / (daysPast + 100.0)) * 0.01
			}
		} else {
			score = trendiness / 100.0
		}

		scores[i] = marketWithScore{
			market: m,
			score:  score,
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	sortedMarkets := make([]gamma.Market, len(validMarkets))
	for i, s := range scores {
		sortedMarkets[i] = s.market
	}

	if len(sortedMarkets) > 20 {
		sortedMarkets = sortedMarkets[:20]
	}

	if len(sortedMarkets) == 0 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay("❌ 条件に合う予測市場が見つかりませんでした。別のキーワードで検索するか、終了した市場を含めてください。"),
				).WithAccentColor(0xE74C3C),
			)
		_ = event.RespondMessage(builder)
		return nil
	}

	session := &PolymarketSession{
		ID:          uuid.New(),
		UserID:      event.User().ID,
		GuildID:     *event.GuildID(),
		Markets:     sortedMarkets,
		SearchQuery: query,
	}
	pm_sessions.Set(session.ID, session)

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketSearchMessage(c, session, points, event.Locale())...),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketSelectMarketHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
		marketID := data.Values[0]
		if marketID == "none" {
			return nil
		}
		var selected *gamma.Market
		for _, m := range session.Markets {
			if m.ID == marketID {
				mCopy := m
				selected = &mCopy
				break
			}
		}

		if selected == nil {
			pm := getPMClient()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			m, err := pm.Gamma.GetMarket(ctx, marketID)
			if err != nil {
				return errors.NewError(err)
			}
			selected = m
		}

		session.SelectedMarket = selected
		session.SelectedToken = nil
		session.StatusMsg = ""
		pm_sessions.Set(session.ID, session)
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketDetailMessage(c, session, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketSelectOutcomeHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
		tokenID := data.Values[0]
		if tokenID == "none" {
			return nil
		}
		tokens := GetMarketTokens(session.SelectedMarket)
		var selectedToken *gamma.Token
		for _, t := range tokens {
			if t.TokenID == tokenID {
				tCopy := t
				selectedToken = &tCopy
				break
			}
		}
		if selectedToken == nil {
			return errors.NewError(fmt.Errorf("選択された予測トークンが見つかりませんでした。"))
		}

		session.SelectedToken = selectedToken
		session.StatusMsg = ""
		pm_sessions.Set(session.ID, session)
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketBetConfirmMessage(c, session, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketViewMyBetsHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	session.StatusHeader = ""
	pm_sessions.Set(session.ID, session)

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketMyBetsMessage(c, session, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketBackToListHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	session.SelectedMarket = nil
	session.SelectedToken = nil
	session.StatusMsg = ""
	pm_sessions.Set(session.ID, session)

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketSearchMessage(c, session, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketBackToDetailHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	session.SelectedToken = nil
	session.StatusMsg = ""
	pm_sessions.Set(session.ID, session)

	points, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketDetailMessage(c, session, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketQuitHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	pm_sessions.Delete(session.ID)

	container := discord.NewContainer().WithAccentColor(0x95A5A6)
	container = container.AddComponents(
		discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.quit_title")),
		discord.NewLargeSeparator(),
		discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.quit_desc")),
	)

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(container).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketBetAmountHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	customID := strings.Split(event.Data.CustomID(), ":")
	if len(customID) < 4 {
		return errors.NewError(fmt.Errorf("invalid component custom ID format for bet amount"))
	}
	amountStr := customID[2]

	if amountStr == "custom_btn" {
		if err := event.Modal(discord.NewModalCreateBuilder().
			SetTitle(i18n.TranslateText(event.Locale(), "components.play.polymarket.modal_bet_title")).
			SetCustomID(fmt.Sprintf("play:pm_custom_bet_modal:%s", session.ID.String())).
			SetComponents(
				discord.NewLabel(i18n.TranslateText(event.Locale(), "components.play.polymarket.modal_bet_label"),
					discord.TextInputComponent{
						CustomID:    "amount",
						Style:       discord.TextInputStyleShort,
						Placeholder: i18n.TranslateText(event.Locale(), "components.play.polymarket.modal_bet_placeholder"),
						Required:    true,
					}),
			).
			Build(),
		); err != nil {
			return errors.NewError(err)
		}
		return nil
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return errors.NewError(err)
	}

	return processBetPlacement(c, event, session, amount)
}

func dbPlaceBet(c *components.Components, userID snowflake.ID, guildID snowflake.ID, session *PolymarketSession, amount int64, locale discord.Locale) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("ベット額は正の値である必要があります。")
	}

	points, _, err := gopoint.GetPoint(c, userID, guildID)
	if err != nil {
		return "", err
	}

	if points < amount {
		return "", fmt.Errorf("insufficient_points")
	}

	if err := gopoint.AddPoint(c, userID, guildID, -amount); err != nil {
		return "", err
	}

	bet := models.PolymarketBet{
		ID:          uuid.New(),
		UserID:      userID,
		GuildID:     guildID,
		MarketID:    session.SelectedMarket.ID,
		MarketTitle: session.SelectedMarket.Question,
		TokenID:     session.SelectedToken.TokenID,
		Outcome:     session.SelectedToken.Outcome,
		BetAmount:   amount,
		EntryPrice:  session.SelectedToken.Price,
		Resolved:    false,
		Winner:      false,
		Payout:      0,
	}

	if err := c.GormDB().Create(&bet).Error; err != nil {
		_ = gopoint.AddPoint(c, userID, guildID, amount)
		return "", err
	}

	multiplier := 1.0 / bet.EntryPrice
	estPayout := int64(float64(amount) * multiplier)

	msg := i18n.TranslateText(locale, "components.play.polymarket.bet_success", map[string]any{
		"market":     bet.MarketTitle,
		"outcome":    bet.Outcome,
		"price":      fmt.Sprintf("%.2f", bet.EntryPrice),
		"amount":     amount,
		"est_payout": estPayout,
		"multiplier": fmt.Sprintf("%.2f", multiplier),
	})

	return msg, nil
}

func processBetPlacement(c *components.Components, event *events.ComponentInteractionCreate, session *PolymarketSession, amount int64) errors.Error {
	pointsBefore, _, _ := gopoint.GetPoint(c, session.UserID, session.GuildID)

	msg, err := dbPlaceBet(c, session.UserID, session.GuildID, session, amount, event.Locale())
	if err != nil {
		if err.Error() == "insufficient_points" {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.err_insufficient_points", map[string]any{
							"amount": amount,
							"points": pointsBefore,
						})),
					).WithAccentColor(0xE74C3C),
				).
				AddFlags(discord.MessageFlagEphemeral)
			_ = event.RespondMessage(builder)
			return nil
		}
		return errors.NewError(err)
	}

	session.SelectedToken = nil
	session.SelectedMarket = nil
	session.StatusMsg = msg
	pm_sessions.Set(session.ID, session)

	pointsAfter, _, _ := gopoint.GetPoint(c, session.UserID, session.GuildID)

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketSearchMessage(c, session, pointsAfter, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketCustomBetModalHandler(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	customID := strings.Split(event.Data.CustomID, ":")
	if len(customID) < 3 {
		return errors.NewError(fmt.Errorf("invalid modal custom ID format"))
	}
	sessionUUIDStr := customID[2]
	sessionUUID, err := uuid.Parse(sessionUUIDStr)
	if err != nil {
		return errors.NewError(err)
	}
	session, ok := pm_sessions.Get(sessionUUID)
	if !ok {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.err_session_expired_modal")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	amountStr := event.Data.Text("amount")

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		builder := discord.NewMessageBuilder().
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.err_invalid_amount")),
				).WithAccentColor(0xE74C3C),
			).
			AddFlags(discord.MessageFlagEphemeral)
		_ = event.RespondMessage(builder)
		return nil
	}

	pointsBefore, _, _ := gopoint.GetPoint(c, session.UserID, session.GuildID)

	msg, err := dbPlaceBet(c, session.UserID, session.GuildID, session, amount, event.Locale())
	if err != nil {
		if err.Error() == "insufficient_points" {
			builder := discord.NewMessageBuilder().
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.play.polymarket.err_insufficient_points", map[string]any{
							"amount": amount,
							"points": pointsBefore,
						})),
					).WithAccentColor(0xE74C3C),
				).
				AddFlags(discord.MessageFlagEphemeral)
			_ = event.RespondMessage(builder)
			return nil
		}
		return errors.NewError(err)
	}

	session.SelectedToken = nil
	session.SelectedMarket = nil
	session.StatusMsg = msg
	pm_sessions.Set(session.ID, session)

	pointsAfter, _, _ := gopoint.GetPoint(c, session.UserID, session.GuildID)

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketSearchMessage(c, session, pointsAfter, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func PolymarketRefreshBetsHandler(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
	session, err1 := PolymarketPrecondition(event)
	if err1 != nil {
		return err1
	}
	if session == nil {
		return nil
	}

	var bets []models.PolymarketBet
	if err := c.GormDB().Where("user_id = ? AND guild_id = ? AND resolved = ?", session.UserID, session.GuildID, false).Find(&bets).Error; err != nil {
		return errors.NewError(err)
	}

	pm := getPMClient()
	resolvedCount := 0
	var resolvedMessages []string

	for _, b := range bets {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		m, err := pm.Gamma.GetMarket(ctx, b.MarketID)
		cancel()
		if err != nil {
			slog.Error("failed to fetch market details for resolution", "market_id", b.MarketID, "error", err)
			continue
		}

		if m.Closed {
			tokens := GetMarketTokens(m)
			var winningToken *gamma.Token
			for _, t := range tokens {
				if t.Winner {
					tCopy := t
					winningToken = &tCopy
					break
				}
			}

			if winningToken == nil {
				continue
			}

			won := (b.TokenID == winningToken.TokenID)
			payout := int64(0)

			errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
				var dbBet models.PolymarketBet
				if err := tx.Where("id = ?", b.ID).First(&dbBet).Error; err != nil {
					return err
				}
				if dbBet.Resolved {
					return nil
				}

				dbBet.Resolved = true
				if won {
					payout = int64(float64(b.BetAmount) / b.EntryPrice)
					dbBet.Winner = true
					dbBet.Payout = payout

					if err := gopoint.AddPointTx(tx, b.UserID, b.GuildID, payout); err != nil {
						return err
					}
				} else {
					dbBet.Winner = false
					dbBet.Payout = 0
				}

				return tx.Save(&dbBet).Error
			})

			if errTx == nil {
				resolvedCount++
				if won {
					resolvedMessages = append(resolvedMessages, i18n.TranslateText(event.Locale(), "components.play.polymarket.resolution_won", map[string]any{
						"amount":  b.BetAmount,
						"market":  b.MarketTitle,
						"outcome": b.Outcome,
						"payout":  payout,
					}))
				} else {
					resolvedMessages = append(resolvedMessages, i18n.TranslateText(event.Locale(), "components.play.polymarket.resolution_lost", map[string]any{
						"amount":  b.BetAmount,
						"market":  b.MarketTitle,
						"outcome": b.Outcome,
					}))
				}
			}
		}
	}

	points, _, err := gopoint.GetPoint(c, session.UserID, session.GuildID)
	if err != nil {
		return errors.NewError(err)
	}

	var statusHeader string
	if resolvedCount > 0 {
		statusHeader = i18n.TranslateText(event.Locale(), "components.play.polymarket.resolution_header", map[string]any{
			"details": strings.Join(resolvedMessages, "\n"),
		})
	} else {
		statusHeader = i18n.TranslateText(event.Locale(), "components.play.polymarket.no_new_resolutions")
	}

	session.StatusHeader = statusHeader
	pm_sessions.Set(session.ID, session)

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(PolymarketMyBetsMessage(c, session, points, event.Locale())...).
		BuildUpdate(),
	); err != nil {
		return errors.NewError(err)
	}

	return nil
}
