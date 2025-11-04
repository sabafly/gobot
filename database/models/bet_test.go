package models

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

func TestBetHost_IsOwner(t *testing.T) {
	ownerID := snowflake.ID(123456789)
	otherUserID := snowflake.ID(987654321)

	betHost := &BetHost{
		ID:      uuid.New(),
		OwnerID: ownerID,
		Title:   "Test Bet",
		Mode:    string(BetVoteTypeGuess),
		Status:  string(BetStatusEntry),
	}

	tests := []struct {
		name     string
		userID   snowflake.ID
		expected bool
	}{
		{
			name:     "Owner should return true",
			userID:   ownerID,
			expected: true,
		},
		{
			name:     "Non-owner should return false",
			userID:   otherUserID,
			expected: false,
		},
		{
			name:     "Zero ID should return false",
			userID:   snowflake.ID(0),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := betHost.IsOwner(tt.userID)
			if result != tt.expected {
				t.Errorf("IsOwner() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBetVoteTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		voteType BetVoteType
		expected string
	}{
		{
			name:     "BetVoteTypeGuess should be 'poll'",
			voteType: BetVoteTypeGuess,
			expected: "poll",
		},
		{
			name:     "BetVoteTypeRace should be 'race'",
			voteType: BetVoteTypeRace,
			expected: "race",
		},
		{
			name:     "BetVoteTypeBattleRoyale should be 'battle_royale'",
			voteType: BetVoteTypeBattleRoyale,
			expected: "battle_royale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.voteType) != tt.expected {
				t.Errorf("VoteType = %v, want %v", tt.voteType, tt.expected)
			}
		})
	}
}

func TestBetStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   BetStatus
		expected string
	}{
		{
			name:     "BetStatusEntry should be 'entry'",
			status:   BetStatusEntry,
			expected: "entry",
		},
		{
			name:     "BetStatusVoting should be 'voting'",
			status:   BetStatusVoting,
			expected: "voting",
		},
		{
			name:     "BetStatusClosed should be 'closed'",
			status:   BetStatusClosed,
			expected: "closed",
		},
		{
			name:     "BetStatusFinished should be 'finished'",
			status:   BetStatusFinished,
			expected: "finished",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status = %v, want %v", tt.status, tt.expected)
			}
		})
	}
}
