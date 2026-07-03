package play

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"math"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
			levIdx:     0, // 25x, ratio 0.05
			points:     1000,
			margin:     50,
			wantValid:  true,
			wantMinVal: 50,
		},
		{
			name:       "25x Leverage, Ratio 0.05, Margin 49 on 1000 points (Invalid)",
			levIdx:     0, // 25x, ratio 0.05
			points:     1000,
			margin:     49,
			wantValid:  false,
			wantMinVal: 50,
		},
		{
			name:       "50x Leverage, Ratio 0.10, Margin 100 on 1000 points (Valid)",
			levIdx:     1, // 50x, ratio 0.10
			points:     1000,
			margin:     100,
			wantValid:  true,
			wantMinVal: 100,
		},
		{
			name:       "50x Leverage, Ratio 0.10, Margin 99 on 1000 points (Invalid)",
			levIdx:     1, // 50x, ratio 0.10
			points:     1000,
			margin:     99,
			wantValid:  false,
			wantMinVal: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := fxLeverages[tt.levIdx]
			minMargin := int64(float64(tt.points) * opt.MinRatio)
			if minMargin < 1 {
				minMargin = 1
			}

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
	if math.Abs(liqPrice1 - 138.0) > 1e-9 {
		t.Errorf("expected liqPrice1 to be 138.0, got %f", liqPrice1)
	}

	// 2. Add margin (so Margin becomes 200)
	pos.Margin = 200
	liqPrice2 := getLiquidationPrice(pos)
	if math.Abs(liqPrice2 - 123.0) > 1e-9 {
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

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.GoPoint{}, &models.FXPosition{}} {
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
	err = gdb.Create(&models.GoPoint{
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
	var gp models.GoPoint
	err = gdb.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&gp).Error
	if err != nil {
		t.Fatalf("failed to query gopoints: %v", err)
	}

	if gp.Points != 1250 {
		t.Errorf("expected points balance to be 1250, got %d", gp.Points)
	}
}
