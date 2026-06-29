package play

import (
	"testing"
)

func TestCardJoker(t *testing.T) {
	// Verify that CardJokerBlack and CardJokerRed have correct Suit and Number
	if CardJokerBlack.Suit() != CardSuitJoker {
		t.Errorf("expected CardJokerBlack suit to be CardSuitJoker, got %v", CardJokerBlack.Suit())
	}
	if CardJokerRed.Suit() != CardSuitJoker {
		t.Errorf("expected CardJokerRed suit to be CardSuitJoker, got %v", CardJokerRed.Suit())
	}
	if CardJokerBlack.Number() != 1 {
		t.Errorf("expected CardJokerBlack number to be 1, got %d", CardJokerBlack.Number())
	}
	if CardJokerRed.Number() != 2 {
		t.Errorf("expected CardJokerRed number to be 2, got %d", CardJokerRed.Number())
	}

	// Verify IsValid()
	if !CardJokerBlack.IsValid() {
		t.Errorf("expected CardJokerBlack to be valid")
	}
	if !CardJokerRed.IsValid() {
		t.Errorf("expected CardJokerRed to be valid")
	}

	// Verify Index()
	if CardJokerBlack.Index() != 52 {
		t.Errorf("expected CardJokerBlack index to be 52, got %d", CardJokerBlack.Index())
	}
	if CardJokerRed.Index() != 53 {
		t.Errorf("expected CardJokerRed index to be 53, got %d", CardJokerRed.Index())
	}

	// Verify CardFromIndex
	if CardFromIndex(52) != CardJokerBlack {
		t.Errorf("expected CardFromIndex(52) to be CardJokerBlack, got %v", CardFromIndex(52))
	}
	if CardFromIndex(53) != CardJokerRed {
		t.Errorf("expected CardFromIndex(53) to be CardJokerRed, got %v", CardFromIndex(53))
	}

	// Verify CardNone
	if CardNone.IsValid() {
		t.Errorf("expected CardNone to be invalid")
	}
	if CardNone.String() != "N/A" {
		t.Errorf("expected CardNone string to be 'N/A', got %s", CardNone.String())
	}
}
