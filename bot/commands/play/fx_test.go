package play

import (
	"testing"

	"github.com/sabafly/gobot/database/models"
)

func TestFX_GetLiquidationPrice(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		entry     float64
		leverage  int
		want      float64
	}{
		{
			name:      "Buy 10x",
			direction: "BUY",
			entry:     150.0,
			leverage:  10,
			want:      135.0, // 150 * (1 - 0.1)
		},
		{
			name:      "Buy 25x",
			direction: "BUY",
			entry:     100.0,
			leverage:  25,
			want:      96.0, // 100 * (1 - 0.04)
		},
		{
			name:      "Sell 10x",
			direction: "SELL",
			entry:     150.0,
			leverage:  10,
			want:      165.0, // 150 * (1 + 0.1)
		},
		{
			name:      "Sell 25x",
			direction: "SELL",
			entry:     100.0,
			leverage:  25,
			want:      104.0, // 100 * (1 + 0.04)
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
			Direction:  "BUY",
			EntryPrice: 160.0,
			Margin:     100,
			Leverage:   10,
		}
		currentPrice := 161.6
		pnl := float64(pos.Margin) * float64(pos.Leverage) * ((currentPrice / pos.EntryPrice) - 1.0)
		if int64(pnl) != 10 {
			t.Errorf("expected PnL to be 10, got %d (raw float: %v)", int64(pnl), pnl)
		}
	})

	t.Run("Buy PnL Negative", func(t *testing.T) {
		pos := &models.FXPosition{
			Direction:  "BUY",
			EntryPrice: 160.0,
			Margin:     100,
			Leverage:   10,
		}
		currentPrice := 158.4
		pnl := float64(pos.Margin) * float64(pos.Leverage) * ((currentPrice / pos.EntryPrice) - 1.0)
		if int64(pnl) != -10 {
			t.Errorf("expected PnL to be -10, got %d (raw float: %v)", int64(pnl), pnl)
		}
	})

	// Test SELL (Short)
	t.Run("Sell PnL Positive", func(t *testing.T) {
		pos := &models.FXPosition{
			Direction:  "SELL",
			EntryPrice: 160.0,
			Margin:     100,
			Leverage:   10,
		}
		currentPrice := 158.4
		pnl := float64(pos.Margin) * float64(pos.Leverage) * (1.0 - (currentPrice / pos.EntryPrice))
		if int64(pnl) != 10 {
			t.Errorf("expected PnL to be 10, got %d (raw float: %v)", int64(pnl), pnl)
		}
	})

	t.Run("Sell PnL Negative", func(t *testing.T) {
		pos := &models.FXPosition{
			Direction:  "SELL",
			EntryPrice: 160.0,
			Margin:     100,
			Leverage:   10,
		}
		currentPrice := 161.6
		pnl := float64(pos.Margin) * float64(pos.Leverage) * (1.0 - (currentPrice / pos.EntryPrice))
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
		Direction:     "BUY",
		EntryPrice:    150.0,
		Margin:        100,
		InitialMargin: 100,
		Leverage:      10,
	}

	// 1. Initial liquidation price (no added margin)
	liqPrice1 := getLiquidationPrice(pos)
	if liqPrice1 != 135.0 {
		t.Errorf("expected liqPrice1 to be 135.0, got %f", liqPrice1)
	}

	// 2. Add margin (so Margin becomes 200)
	pos.Margin = 200
	liqPrice2 := getLiquidationPrice(pos)
	if liqPrice2 != 120.0 {
		t.Errorf("expected liqPrice2 to be 120.0, got %f", liqPrice2)
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
