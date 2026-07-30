package play

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"math"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
)

func TestFX_GetLiquidationPrice(t *testing.T) {
	tests := []struct {
		name      string
		direction models.FXPositionDirection
		entry     float64
		leverage  int
		want      float64
	}{
		{
			name:      "Buy 10x",
			direction: models.FXPositionDirectionBuy,
			entry:     150.0,
			leverage:  10,
			want:      138.0, // 150 * (1 - 0.8 / 10)
		},
		{
			name:      "Buy 25x",
			direction: models.FXPositionDirectionBuy,
			entry:     100.0,
			leverage:  25,
			want:      96.8, // 100 * (1 - 0.8 / 25)
		},
		{
			name:      "Sell 10x",
			direction: models.FXPositionDirectionSell,
			entry:     150.0,
			leverage:  10,
			want:      162.0, // 150 * (1 + 0.8 / 10)
		},
		{
			name:      "Sell 25x",
			direction: models.FXPositionDirectionSell,
			entry:     100.0,
			leverage:  25,
			want:      103.2, // 100 * (1 + 0.8 / 25)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := &models.FXPosition{
				Direction:     tt.direction,
				EntryPrice:    tt.entry,
				Margin:        100,
				InitialMargin: 100,
				Leverage:      tt.leverage,
			}
			got := getLiquidationPrice(pos)
			if got != tt.want {
				t.Errorf("getLiquidationPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFX_PnLCalculation(t *testing.T) {
	// Test BUY (Long)
	t.Run("Buy PnL Positive", func(t *testing.T) {
		pos := &models.FXPosition{
			Direction:     models.FXPositionDirectionBuy,
			EntryPrice:    160.0,
			Margin:        100,
			InitialMargin: 100,
			Leverage:      10,
		}
		currentPrice := 161.6
		pnl := getPnL(pos, currentPrice)
		if int64(pnl) != 10 {
			t.Errorf("expected PnL to be 10, got %d (raw float: %v)", int64(pnl), pnl)
		}
	})

	t.Run("Buy PnL Negative", func(t *testing.T) {
		pos := &models.FXPosition{
			Direction:     models.FXPositionDirectionBuy,
			EntryPrice:    160.0,
			Margin:        100,
			InitialMargin: 100,
			Leverage:      10,
		}
		currentPrice := 158.4
		pnl := getPnL(pos, currentPrice)
		if int64(pnl) != -10 {
			t.Errorf("expected PnL to be -10, got %d (raw float: %v)", int64(pnl), pnl)
		}
	})

	// Test SELL (Short)
	t.Run("Sell PnL Positive", func(t *testing.T) {
		pos := &models.FXPosition{
			Direction:     models.FXPositionDirectionSell,
			EntryPrice:    160.0,
			Margin:        100,
			InitialMargin: 100,
			Leverage:      10,
		}
		currentPrice := 158.4
		pnl := getPnL(pos, currentPrice)
		if int64(pnl) != 10 {
			t.Errorf("expected PnL to be 10, got %d (raw float: %v)", int64(pnl), pnl)
		}
	})

	t.Run("Sell PnL Negative", func(t *testing.T) {
		pos := &models.FXPosition{
			Direction:     models.FXPositionDirectionSell,
			EntryPrice:    160.0,
			Margin:        100,
			InitialMargin: 100,
			Leverage:      10,
		}
		currentPrice := 161.6
		pnl := getPnL(pos, currentPrice)
		if int64(pnl) != -10 {
			t.Errorf("expected PnL to be -10, got %d (raw float: %v)", int64(pnl), pnl)
		}
	})
}

func TestFX_MinRatioMarginRequirement(t *testing.T) {
	tests := []struct {
		name       string
		levIdx     int // index in fxLeverages
		points     int64
		margin     int64
		wantValid  bool
		wantMinVal int64
	}{
		{
			name:       "25x Leverage, Ratio 0.05, Margin 50 on 1000 points (Valid)",
			levIdx:     1, // 25x, ratio 0.05
			points:     1000,
			margin:     50,
			wantValid:  true,
			wantMinVal: 50,
		},
		{
			name:       "25x Leverage, Ratio 0.05, Margin 49 on 1000 points (Invalid)",
			levIdx:     1, // 25x, ratio 0.05
			points:     1000,
			margin:     49,
			wantValid:  false,
			wantMinVal: 50,
		},
		{
			name:       "50x Leverage, Ratio 0.10, Margin 100 on 1000 points (Valid)",
			levIdx:     2, // 50x, ratio 0.10
			points:     1000,
			margin:     100,
			wantValid:  true,
			wantMinVal: 100,
		},
		{
			name:       "50x Leverage, Ratio 0.10, Margin 99 on 1000 points (Invalid)",
			levIdx:     2, // 50x, ratio 0.10
			points:     1000,
			margin:     99,
			wantValid:  false,
			wantMinVal: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := fxLeverages[tt.levIdx]
			minMargin := max(int64(float64(tt.points)*opt.MinRatio), 1)

			if minMargin != tt.wantMinVal {
				t.Errorf("expected minMargin to be %d, got %d", tt.wantMinVal, minMargin)
			}

			isValid := tt.margin >= minMargin
			if isValid != tt.wantValid {
				t.Errorf("expected validity to be %v, got %v (margin: %d, minMargin: %d)", tt.wantValid, isValid, tt.margin, minMargin)
			}
		})
	}
}

func TestFX_MarginCallAndAddedMargin(t *testing.T) {
	pos := &models.FXPosition{
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    150.0,
		Margin:        100,
		InitialMargin: 100,
		Leverage:      10,
	}

	// 1. Initial liquidation price (no added margin)
	liqPrice1 := getLiquidationPrice(pos)
	if math.Abs(liqPrice1-138.0) > 1e-9 {
		t.Errorf("expected liqPrice1 to be 138.0, got %f", liqPrice1)
	}

	// 2. Add margin (so Margin becomes 200)
	pos.Margin = 200
	liqPrice2 := getLiquidationPrice(pos)
	if math.Abs(liqPrice2-123.0) > 1e-9 {
		t.Errorf("expected liqPrice2 to be 123.0, got %f", liqPrice2)
	}

	// 3. Maintenance ratio calculation:
	// Entry = 150. Current = 142.5. PnL = 100 * 10 * (142.5/150 - 1) = 1000 * (-0.05) = -50
	// Valuation = Margin (200) + PnL (-50) = 150
	// Maintenance ratio = Valuation / InitialMargin (100) * 100 = 150%
	currentPrice := 142.5
	pnl := float64(pos.GetInitialMargin()) * float64(pos.Leverage) * ((currentPrice / pos.EntryPrice) - 1.0)
	if int64(pnl) != -50 {
		t.Errorf("expected PnL to be -50, got %d (raw float: %f)", int64(pnl), pnl)
	}

	ratio := float64(pos.Margin+int64(pnl)) / float64(pos.GetInitialMargin()) * 100.0
	if int64(ratio) != 150 {
		t.Errorf("expected maintenance ratio to be 150%%, got %d%% (raw float: %f)", int64(ratio), ratio)
	}

	// 4. Test margin call trigger threshold (< 50%)
	// With Margin = 100, InitialMargin = 100:
	// Valuation < 50 => Margin + PnL < 50 => PnL < -50
	pos.Margin = 100
	pnlMC := -51.0
	ratioMC := (float64(pos.Margin) + pnlMC) / float64(pos.GetInitialMargin()) * 100.0
	isMarginCall := ratioMC < 50.0
	if !isMarginCall {
		t.Errorf("expected margin call to trigger below 50%% ratio, got ratio: %f%%", ratioMC)
	}
}

func createSQLiteTable(db *gorm.DB, model any) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return err
	}
	tableName := stmt.Schema.Table
	var columns []string
	var pks []string
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		colType := "TEXT"
		if field.DataType == "time" {
			colType = "DATETIME"
		}
		if field.PrimaryKey {
			pks = append(pks, fmt.Sprintf("`%s`", field.DBName))
		}
		columns = append(columns, fmt.Sprintf("`%s` %s", field.DBName, colType))
	}
	var pkConstraint string
	if len(pks) > 0 {
		pkConstraint = fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ", "))
	}
	query := fmt.Sprintf("CREATE TABLE `%s` (%s%s)", tableName, strings.Join(columns, ", "), pkConstraint)
	return db.Exec(query).Error
}

func TestFX_LiquidationWithDeficitCoverage(t *testing.T) {
	// Setup SQLite in-memory DB
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.Currency{}, &models.FXPosition{}, &models.CurrencySeason{}, &models.CurrencySeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	c := components.New(ctx, components.Config{}, dbWrapper)

	userID := snowflake.ID(12345)
	guildID := snowflake.ID(67890)

	// Seed User, Guild, and GoPoint
	err = gdb.Create(&models.User{ID: userID}).Error
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	err = gdb.Create(&models.Guild{ID: guildID}).Error
	if err != nil {
		t.Fatalf("failed to create guild: %v", err)
	}

	// Seed initial GoPoints balance
	err = gdb.Create(&models.Currency{
		UserID:  userID,
		GuildID: guildID,
		Points:  1000,
	}).Error
	if err != nil {
		t.Fatalf("failed to create gopoints: %v", err)
	}

	// Insert positions
	// Position 1: margin 100, leverage 10 -> size 1000. Under BUY, Entry 150.
	// We will liquidate pos1.
	pos1 := &models.FXPosition{
		ID:            uuid.New(),
		UserID:        userID,
		GuildID:       guildID,
		Symbol:        "USD_JPY",
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    150.0,
		Margin:        100,
		InitialMargin: 100,
		Leverage:      10,
	}
	err = gdb.Create(pos1).Error
	if err != nil {
		t.Fatalf("failed to create pos1: %v", err)
	}

	// Position 2: margin 200, leverage 10 -> size 2000. Under BUY, Entry 150.
	pos2 := &models.FXPosition{
		ID:            uuid.New(),
		UserID:        userID,
		GuildID:       guildID,
		Symbol:        "USD_JPY",
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    150.0,
		Margin:        200,
		InitialMargin: 200,
		Leverage:      10,
	}
	err = gdb.Create(pos2).Error
	if err != nil {
		t.Fatalf("failed to create pos2: %v", err)
	}

	// Deficit calculations:
	// Let's liquidate pos1 at price 127.5 (pnl = -150). Deficit = 50.
	// Pos2 is at price 157.5 (pnl = +100, valuation = 300).
	// When pos1 liquidates, it should close pos2 as well because of deficit.
	// Valuation of pos2 (300) covers deficit (50), leaving 250.
	// Remaining 250 is added back to points.
	// Total points should be: initial (1000) + 250 = 1250.
	// Pos1 and pos2 should both be deleted.
	ticker := &TickerResponse{
		Data: []TickerData{
			{Symbol: "USD_JPY", Ask: "157.5", Bid: "157.5"},
		},
	}

	err = liquidatePosition(c, nil, pos1, ticker, 127.5, -150.0)
	if err != nil {
		t.Fatalf("liquidatePosition failed: %v", err)
	}

	// Verify pos1 and pos2 are deleted
	var count int64
	gdb.Model(&models.FXPosition{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 positions remaining, got %d", count)
	}

	// Verify points balance
	var gp models.Currency
	err = gdb.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&gp).Error
	if err != nil {
		t.Fatalf("failed to query gopoints: %v", err)
	}

	if gp.Points != 1250 {
		t.Errorf("expected points balance to be 1250, got %d", gp.Points)
	}
}

func TestFX_LiquidationWithRefund(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.Currency{}, &models.FXPosition{}, &models.CurrencySeason{}, &models.CurrencySeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	c := components.New(ctx, components.Config{}, dbWrapper)

	userID := snowflake.ID(12345)
	guildID := snowflake.ID(67890)

	_ = gdb.Create(&models.User{ID: userID})
	_ = gdb.Create(&models.Guild{ID: guildID})
	_ = gdb.Create(&models.Currency{UserID: userID, GuildID: guildID, Points: 1000})

	// Seed position with 100 margin, 10 leverage, entry 150.0
	pos := &models.FXPosition{
		ID:            uuid.New(),
		UserID:        userID,
		GuildID:       guildID,
		Symbol:        "USD_JPY",
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    150.0,
		Margin:        100,
		InitialMargin: 100,
		Leverage:      10,
	}
	_ = gdb.Create(pos)

	// Liquidate at 142.5 -> PnL = -50 -> Valuation = 50.
	// Remaining 50 should be refunded to user points.
	// Total points should be: 1000 + 50 = 1050.
	ticker := &TickerResponse{
		Data: []TickerData{
			{Symbol: "USD_JPY", Ask: "142.5", Bid: "142.5"},
		},
	}

	err = liquidatePosition(c, nil, pos, ticker, 142.5, -50.0)
	if err != nil {
		t.Fatalf("liquidatePosition failed: %v", err)
	}

	// Verify position is deleted
	var count int64
	gdb.Model(&models.FXPosition{}).Count(&count)
	if count != 0 {
		t.Errorf("expected position to be deleted, got count %d", count)
	}

	// Verify points balance is refunded
	var gp models.Currency
	err = gdb.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&gp).Error
	if err != nil {
		t.Fatalf("failed to query gopoints: %v", err)
	}

	if gp.Points != 1050 {
		t.Errorf("expected points balance to be 1050, got %d", gp.Points)
	}
}

func TestFX_PendingOrders(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.Currency{}, &models.FXPosition{}, &models.FXOrder{}, &models.CurrencySeason{}, &models.CurrencySeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	c := components.New(ctx, components.Config{}, dbWrapper)

	userID := snowflake.ID(99999)
	guildID := snowflake.ID(88888)

	// Seed User, Guild, and GoPoint
	_ = gdb.Create(&models.User{ID: userID})
	_ = gdb.Create(&models.Guild{ID: guildID})
	_ = gdb.Create(&models.Currency{UserID: userID, GuildID: guildID, Points: 1000})

	// 1. Create a pending LIMIT BUY order (Target: 145.000, current price is higher, e.g. 150.000)
	orderID := uuid.New()
	order := &models.FXOrder{
		ID:          orderID,
		UserID:      userID,
		GuildID:     guildID,
		Symbol:      "USD_JPY",
		Direction:   models.FXPositionDirectionBuy,
		OrderType:   "LIMIT",
		TargetPrice: 145.0,
		Margin:      100,
		Leverage:    25,
	}
	err = gdb.Create(order).Error
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// 2. Run CheckAllPositionsLiquidation with price = 146.000. It should NOT trigger.
	ticker := &TickerResponse{
		Data: []TickerData{
			{Symbol: "USD_JPY", Ask: "146.0", Bid: "146.0"},
		},
	}
	tickerCacheMu.Lock()
	tickerCache = ticker
	lastFetchTime = time.Now().Add(time.Hour)
	tickerCacheMu.Unlock()

	err = CheckAllPositionsLiquidation(c, nil)
	if err != nil {
		t.Fatalf("CheckAllPositionsLiquidation failed: %v", err)
	}

	// Verify order still exists
	var count int64
	gdb.Model(&models.FXOrder{}).Count(&count)
	if count != 1 {
		t.Errorf("expected order to remain, count was %d", count)
	}

	// 3. Run CheckAllPositionsLiquidation with price = 144.500. It SHOULD trigger LIMIT BUY.
	ticker.Data[0].Ask = "144.5"
	ticker.Data[0].Bid = "144.5"
	tickerCacheMu.Lock()
	tickerCache = ticker
	lastFetchTime = time.Now().Add(time.Hour)
	tickerCacheMu.Unlock()

	err = CheckAllPositionsLiquidation(c, nil)
	if err != nil {
		t.Fatalf("CheckAllPositionsLiquidation failed: %v", err)
	}

	// Verify order is deleted
	gdb.Model(&models.FXOrder{}).Count(&count)
	if count != 0 {
		t.Errorf("expected order to be deleted, count was %d", count)
	}

	// Verify position is created with correct fields
	var pos models.FXPosition
	err = gdb.First(&pos).Error
	if err != nil {
		t.Fatalf("expected position to be created: %v", err)
	}

	if pos.ID != orderID {
		t.Errorf("expected position ID to match order ID, got %s vs %s", pos.ID, orderID)
	}
	if pos.EntryPrice != 145.0 {
		t.Errorf("expected entry price to be 145.0, got %f", pos.EntryPrice)
	}
}

func TestFX_ActivePositionTPSL(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.Currency{}, &models.FXPosition{}, &models.CurrencySeason{}, &models.CurrencySeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	c := components.New(ctx, components.Config{}, dbWrapper)

	userID := snowflake.ID(77777)
	guildID := snowflake.ID(66666)

	// Seed User, Guild, and GoPoint
	_ = gdb.Create(&models.User{ID: userID})
	_ = gdb.Create(&models.Guild{ID: guildID})
	_ = gdb.Create(&models.Currency{UserID: userID, GuildID: guildID, Points: 1000})

	// 1. Create a position with BUY entry at 150.0, Margin 100, Leverage 25.
	// We set TakeProfitPrice to 155.0.
	tpPrice := 155.0
	pos := &models.FXPosition{
		ID:              uuid.New(),
		UserID:          userID,
		GuildID:         guildID,
		Symbol:          "USD_JPY",
		Direction:       models.FXPositionDirectionBuy,
		EntryPrice:      150.0,
		Margin:          100,
		InitialMargin:   100,
		Leverage:        25,
		TakeProfitPrice: &tpPrice,
	}
	err = gdb.Create(pos).Error
	if err != nil {
		t.Fatalf("failed to create position: %v", err)
	}

	// 2. Run CheckAllPositionsLiquidation with price = 152.0. It should NOT trigger.
	ticker := &TickerResponse{
		Data: []TickerData{
			{Symbol: "USD_JPY", Ask: "152.0", Bid: "152.0"},
		},
	}
	tickerCacheMu.Lock()
	tickerCache = ticker
	lastFetchTime = time.Now().Add(time.Hour)
	tickerCacheMu.Unlock()

	err = CheckAllPositionsLiquidation(c, nil)
	if err != nil {
		t.Fatalf("CheckAllPositionsLiquidation failed: %v", err)
	}

	var count int64
	gdb.Model(&models.FXPosition{}).Count(&count)
	if count != 1 {
		t.Errorf("expected position to remain, count was %d", count)
	}

	// 3. Run CheckAllPositionsLiquidation with price = 156.0. It SHOULD trigger TP.
	ticker.Data[0].Ask = "156.0"
	ticker.Data[0].Bid = "156.0"
	tickerCacheMu.Lock()
	tickerCache = ticker
	lastFetchTime = time.Now().Add(time.Hour)
	tickerCacheMu.Unlock()

	err = CheckAllPositionsLiquidation(c, nil)
	if err != nil {
		t.Fatalf("CheckAllPositionsLiquidation failed: %v", err)
	}

	// Verify position is deleted
	gdb.Model(&models.FXPosition{}).Count(&count)
	if count != 0 {
		t.Errorf("expected position to be deleted by TP, count was %d", count)
	}

	// Verify points balance
	// PnL at TP (155.0): (155.0 - 150.0) / 150.0 * 100 * 25 = 83.333 pt.
	// Valuation: 100 + 83 = 183 pt.
	// Total points: 1000 + 183 = 1183 pt.
	var gp models.Currency
	err = gdb.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&gp).Error
	if err != nil {
		t.Fatalf("failed to query points: %v", err)
	}
	if gp.Points != 1183 {
		t.Errorf("expected points balance to be 1183, got %d", gp.Points)
	}
}

func TestFX_PortfolioLogic(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.Currency{}, &models.FXPosition{}, &models.FXOrder{}, &models.CurrencySeason{}, &models.CurrencySeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	c := components.New(ctx, components.Config{}, dbWrapper)

	userID := snowflake.ID(77777)
	guildID := snowflake.ID(66666)

	// Seed User, Guild, GoPoint
	_ = gdb.Create(&models.User{ID: userID})
	_ = gdb.Create(&models.Guild{ID: guildID})
	_ = gdb.Create(&models.Currency{UserID: userID, GuildID: guildID, Points: 1000})

	// 1. Check getFXPositions and getFXOrders when empty
	positions, err := getFXPositions(c, userID, guildID)
	if err != nil {
		t.Fatalf("failed to get positions: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("expected 0 positions, got %d", len(positions))
	}

	orders, err := getFXOrders(c, userID, guildID)
	if err != nil {
		t.Fatalf("failed to get orders: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected 0 orders, got %d", len(orders))
	}

	// 2. Add a position and an order
	pos := &models.FXPosition{
		ID:            uuid.New(),
		UserID:        userID,
		GuildID:       guildID,
		Symbol:        "USD_JPY",
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    150.0,
		Margin:        100,
		InitialMargin: 100,
		Leverage:      25,
	}
	if err := gdb.Create(pos).Error; err != nil {
		t.Fatalf("failed to create position: %v", err)
	}

	order := &models.FXOrder{
		ID:          uuid.New(),
		UserID:      userID,
		GuildID:     guildID,
		Symbol:      "EUR_USD",
		Direction:   models.FXPositionDirectionSell,
		OrderType:   "LIMIT",
		TargetPrice: 1.10,
		Margin:      50,
		Leverage:    10,
	}
	if err := gdb.Create(order).Error; err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// 3. Retrieve and verify
	positions, err = getFXPositions(c, userID, guildID)
	if err != nil {
		t.Fatalf("failed to get positions: %v", err)
	}
	if len(positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(positions))
	}
	if positions[0].Symbol != "USD_JPY" {
		t.Errorf("expected symbol USD_JPY, got %s", positions[0].Symbol)
	}

	orders, err = getFXOrders(c, userID, guildID)
	if err != nil {
		t.Fatalf("failed to get orders: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 order, got %d", len(orders))
	}
	if orders[0].Symbol != "EUR_USD" {
		t.Errorf("expected symbol EUR_USD, got %s", orders[0].Symbol)
	}
}

func TestFX_MarginCallRedirection(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.Currency{}, &models.FXPosition{}, &models.FXOrder{}, &models.CurrencySeason{}, &models.CurrencySeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	userID := snowflake.ID(77777)
	guildID := snowflake.ID(66666)

	_ = gdb.Create(&models.User{ID: userID})
	_ = gdb.Create(&models.Guild{ID: guildID})
	_ = gdb.Create(&models.Currency{UserID: userID, GuildID: guildID, Points: 1000})

	// 1. Create a warned position (under margin call)
	// Leverage = 25 -> MarginCallRatio = 50.0%
	// Entry = 150.0. Bid = 145.5.
	// PnL = 100 * 25 * (145.5 / 150.0 - 1.0) = 2500 * (-0.03) = -75
	// Valuation = Margin (100) + PnL (-75) = 25
	// Ratio = 25 / 100 * 100 = 25% (under 50%)
	// Needed to reach 50% ratio:
	// initMargin = 100, opt.MarginCallRatio = 50% -> target valuation = 50.
	// Current valuation = 25.
	// needed = 50 - 25 = 25.
	warnedPos := &models.FXPosition{
		ID:            uuid.New(),
		UserID:        userID,
		GuildID:       guildID,
		Symbol:        "USD_JPY",
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    150.0,
		Margin:        100,
		InitialMargin: 100,
		Leverage:      25,
	}
	if err := gdb.Create(warnedPos).Error; err != nil {
		t.Fatalf("failed to create warned position: %v", err)
	}

	// 2. Create a healthy position
	// Entry = 100.0, Bid = 110.0, Leverage = 1.
	// PnL = 100 * 1 * (110 / 100 - 1) = 10.
	// refund = 100 + 10 = 110.
	healthyPos := &models.FXPosition{
		ID:            uuid.New(),
		UserID:        userID,
		GuildID:       guildID,
		Symbol:        "EUR_USD",
		Direction:     models.FXPositionDirectionBuy,
		EntryPrice:    100.0,
		Margin:        100,
		InitialMargin: 100,
		Leverage:      1,
	}
	if err := gdb.Create(healthyPos).Error; err != nil {
		t.Fatalf("failed to create healthy position: %v", err)
	}

	// Construct mock TickerResponse
	ticker := &TickerResponse{
		Data: []TickerData{
			{
				Symbol: "USD_JPY",
				Ask:    "145.500",
				Bid:    "145.500",
			},
			{
				Symbol: "EUR_USD",
				Ask:    "110.000",
				Bid:    "110.000",
			},
		},
	}

	// 3. Test redirectRefundToMarginCalls
	// We close healthyPos. Its refund is 100 + 10 = 110.
	// Since warnedPos is warned and needs 25 pt to clear the margin call:
	// redirectRefundToMarginCalls should:
	// - add 25 to warnedPos.Margin (so it becomes 125)
	// - return the remaining 85 (110 - 25)
	err = gdb.Transaction(func(tx *gorm.DB) error {
		remainingRefund, errRedirect := redirectRefundToMarginCalls(tx, userID, guildID, healthyPos.ID, 110, ticker)
		if errRedirect != nil {
			return errRedirect
		}
		if remainingRefund != 85 {
			t.Errorf("expected remaining refund to be 85, got %d", remainingRefund)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	// Verify warned position margin is updated to 125
	var updatedWarned models.FXPosition
	if err := gdb.Where("id = ?", warnedPos.ID).First(&updatedWarned).Error; err != nil {
		t.Fatalf("failed to get updated warned position: %v", err)
	}
	if updatedWarned.Margin != 125 {
		t.Errorf("expected warned position margin to be 125, got %d", updatedWarned.Margin)
	}
}

func TestFX_MarketOrders(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.Currency{}, &models.FXPosition{}, &models.FXOrder{}, &models.CurrencySeason{}, &models.CurrencySeasonUser{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	c := components.New(ctx, components.Config{}, dbWrapper)

	userID := snowflake.ID(99999)
	guildID := snowflake.ID(88888)

	// Seed User, Guild, and GoPoint
	_ = gdb.Create(&models.User{ID: userID})
	_ = gdb.Create(&models.Guild{ID: guildID})
	_ = gdb.Create(&models.Currency{UserID: userID, GuildID: guildID, Points: 1000})

	// 1. Create a pending MARKET BUY order that should execute successfully
	orderID := uuid.New()
	order := &models.FXOrder{
		ID:                orderID,
		UserID:            userID,
		GuildID:           guildID,
		Symbol:            "USD_JPY",
		Direction:         models.FXPositionDirectionBuy,
		OrderType:         "MARKET",
		TargetPrice:       150.0,
		ExpectedPrice:     150.0,
		SlippageTolerance: 0.002, // 0.2%
		Margin:            100,
		Leverage:          25,
	}
	err = gdb.Create(order).Error
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// Ticker price matches ExpectedPrice (150.0)
	ticker := &TickerResponse{
		Data: []TickerData{
			{Symbol: "USD_JPY", Ask: "150.0", Bid: "150.0"},
		},
	}
	tickerCacheMu.Lock()
	tickerCache = ticker
	lastFetchTime = time.Now().Add(time.Hour)
	tickerCacheMu.Unlock()

	err = CheckAllPositionsLiquidation(c, nil)
	if err != nil {
		t.Fatalf("CheckAllPositionsLiquidation failed: %v", err)
	}

	// Verify order is deleted (since it was processed)
	var count int64
	gdb.Model(&models.FXOrder{}).Count(&count)
	if count != 0 {
		t.Errorf("expected market order to be processed and deleted, count was %d", count)
	}

	// Verify position is created
	var pos models.FXPosition
	err = gdb.First(&pos).Error
	if err != nil {
		t.Fatalf("expected position to be created, error: %v", err)
	}
	if pos.EntryPrice < 150.0*0.999 || pos.EntryPrice > 150.0*1.002 {
		t.Errorf("unexpected entry price: %f", pos.EntryPrice)
	}

	// Clean up position
	gdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.FXPosition{})

	// 2. Create another MARKET BUY order that should be canceled due to high slippage
	orderID2 := uuid.New()
	order2 := &models.FXOrder{
		ID:                orderID2,
		UserID:            userID,
		GuildID:           guildID,
		Symbol:            "USD_JPY",
		Direction:         models.FXPositionDirectionBuy,
		OrderType:         "MARKET",
		TargetPrice:       150.0,
		ExpectedPrice:     150.0,
		SlippageTolerance: 0.002, // 0.2%
		Margin:            100,
		Leverage:          25,
	}
	err = gdb.Create(order2).Error
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// Ticker price jumps to 151.0 (above 150.0 * 1.002 tolerance price)
	ticker.Data[0].Ask = "151.0"
	ticker.Data[0].Bid = "151.0"
	tickerCacheMu.Lock()
	tickerCache = ticker
	lastFetchTime = time.Now().Add(time.Hour)
	tickerCacheMu.Unlock()

	err = CheckAllPositionsLiquidation(c, nil)
	if err != nil {
		t.Fatalf("CheckAllPositionsLiquidation failed: %v", err)
	}

	// Verify order is deleted
	gdb.Model(&models.FXOrder{}).Count(&count)
	if count != 0 {
		t.Errorf("expected market order to be deleted, count was %d", count)
	}

	// Verify position was NOT created
	gdb.Model(&models.FXPosition{}).Count(&count)
	if count != 0 {
		t.Errorf("expected no position to be created, count was %d", count)
	}
}
