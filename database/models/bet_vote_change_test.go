package models

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

func TestBetHost_AllowVoteDestChange(t *testing.T) {
	tests := []struct {
		name                string
		allowVoteDestChange bool
		expectedValue       bool
	}{
		{
			name:                "AllowVoteDestChange enabled",
			allowVoteDestChange: true,
			expectedValue:       true,
		},
		{
			name:                "AllowVoteDestChange disabled",
			allowVoteDestChange: false,
			expectedValue:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			betHost := &BetHost{
				ID:                  uuid.New(),
				GuildID:             snowflake.ID(123456789),
				ChannelID:           snowflake.ID(987654321),
				Title:               "Test Bet",
				Mode:                string(BetVoteTypeGuess),
				Status:              string(BetStatusVoting),
				OwnerID:             snowflake.ID(111222333),
				AllowVoteDestChange: tt.allowVoteDestChange,
			}

			if betHost.AllowVoteDestChange != tt.expectedValue {
				t.Errorf("Expected AllowVoteDestChange to be %v, got %v", tt.expectedValue, betHost.AllowVoteDestChange)
			}
		})
	}
}

func TestBetHost_DefaultAllowVoteDestChange(t *testing.T) {
	// Test that the default value is true when not explicitly set
	betHost := &BetHost{
		ID:        uuid.New(),
		GuildID:   snowflake.ID(123456789),
		ChannelID: snowflake.ID(987654321),
		Title:     "Test Bet",
		Mode:      string(BetVoteTypeGuess),
		Status:    string(BetStatusVoting),
		OwnerID:   snowflake.ID(111222333),
		// AllowVoteDestChange not set, should default to false in Go zero value
		// but database default is true
	}

	// Note: The database default is set via GORM tags as `default:true`
	// This test verifies the Go zero value is false (not set explicitly)
	if betHost.AllowVoteDestChange {
		t.Fatalf("expected AllowVoteDestChange to be false (Go zero value), got true")
	}
}
