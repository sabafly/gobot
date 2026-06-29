package play

import (
	"testing"
)

func TestHALPlay_Same(t *testing.T) {
	// Test the SAME payout logic by running HALPlay repeatedly until we get SAME results.
	gotSameSuit := false
	gotDiffSuit := false

	for i := 0; i < 20000 && !(gotSameSuit && gotDiffSuit); i++ {
		data := &HALData{
			currentPoint: 10.0,
			multiplier:   2.0,
			currentCard:  NewCard(CardSuitSpades, 5), // start with Spade 5
		}

		// HALPlay will roll a new card.
		// We guess HALChoiceSame.
		success, _ := HALPlay(data, HALChoiceSame)
		if !success {
			continue
		}

		// If success is true, it must be that result == HALChoiceSame.
		// Let's assert:
		expectedMultiplier := 2.0 + float64(data.currentCard.Number())
		if data.multiplier != expectedMultiplier {
			t.Errorf("expected multiplier %f, got %f", expectedMultiplier, data.multiplier)
		}

		if data.previousCard.Suit() == data.currentCard.Suit() {
			expectedPoints := 10.0 * float64(data.currentCard.Number()) * 1.5
			if data.currentPoint != expectedPoints {
				t.Errorf("expected points %f (same suit), got %f", expectedPoints, data.currentPoint)
			}
			gotSameSuit = true
		} else {
			expectedPoints := 10.0
			if data.currentPoint != expectedPoints {
				t.Errorf("expected points %f (diff suit), got %f", expectedPoints, data.currentPoint)
			}
			gotDiffSuit = true
		}
	}

	if !gotSameSuit {
		t.Error("failed to test same suit SAME choice case")
	}
	if !gotDiffSuit {
		t.Error("failed to test different suit SAME choice case")
	}
}
