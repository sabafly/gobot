package play

import (
	"testing"
)

func TestHALPlay_Same(t *testing.T) {
	// Mock the random card function to control the outcomes deterministically.
	originalRandomCard := randomCardFunc
	originalRandomCardWithoutJoker := randomCardWithoutJokerFunc
	t.Cleanup(func() {
		randomCardFunc = originalRandomCard
		randomCardWithoutJokerFunc = originalRandomCardWithoutJoker
	})

	// Test Case 1: Same suit and same number (SAME guess succeeds)
	t.Run("SameSuit_SameNumber", func(t *testing.T) {
		randomCardFunc = func() Card {
			return NewCard(CardSuitSpades, 5)
		}

		data := &HALData{
			currentPoint: 10.0,
			multiplier:   2.0,
			currentCard:  NewCard(CardSuitSpades, 5), // start with Spade 5
			startOption:  HALStartOption{Multiplier: 1.0, MaxTurns: 99},
		}

		finishState := HALPlay(data, HALResultSame)
		if finishState != HALFinishStateNone {
			t.Errorf("expected finishState to be HALFinishStateNone, got %v", finishState)
		}
		if data.lastResult != HALResultSame {
			t.Errorf("expected lastResult to be HALResultSame, got %v", data.lastResult)
		}

		expectedMultiplier := 2.0 + float64(data.currentCard.Number())*1.0
		if data.multiplier != expectedMultiplier {
			t.Errorf("expected multiplier %f, got %f", expectedMultiplier, data.multiplier)
		}

		expectedPoints := 10.0 * float64(data.currentCard.Number()) * 1.5
		if data.currentPoint != expectedPoints {
			t.Errorf("expected points %f (same suit), got %f", expectedPoints, data.currentPoint)
		}
	})

	// Test Case 2: Different suit and same number (SAME guess succeeds)
	t.Run("DiffSuit_SameNumber", func(t *testing.T) {
		randomCardFunc = func() Card {
			return NewCard(CardSuitHearts, 5)
		}

		data := &HALData{
			currentPoint: 10.0,
			multiplier:   2.0,
			currentCard:  NewCard(CardSuitSpades, 5), // start with Spade 5
			startOption:  HALStartOption{Multiplier: 1.0, MaxTurns: 99},
		}

		finishState := HALPlay(data, HALResultSame)
		if finishState != HALFinishStateNone {
			t.Errorf("expected finishState to be HALFinishStateNone, got %v", finishState)
		}
		if data.lastResult != HALResultSame {
			t.Errorf("expected lastResult to be HALResultSame, got %v", data.lastResult)
		}

		expectedMultiplier := 2.0 + float64(data.currentCard.Number())*1.0
		if data.multiplier != expectedMultiplier {
			t.Errorf("expected multiplier %f, got %f", expectedMultiplier, data.multiplier)
		}

		expectedPoints := 10.0
		if data.currentPoint != expectedPoints {
			t.Errorf("expected points %f (diff suit), got %f", expectedPoints, data.currentPoint)
		}
	})
}
