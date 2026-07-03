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
				Direction:  tt.direction,
				EntryPrice: tt.entry,
				Leverage:   tt.leverage,
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
