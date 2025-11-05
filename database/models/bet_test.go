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
		{
			name:     "BetStatusCancelled should be 'cancelled'",
			status:   BetStatusCancelled,
			expected: "cancelled",
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

func TestBetHost_GetWinners(t *testing.T) {
	tests := []struct {
		name     string
		winners  string
		expected []uuid.UUID
	}{
		{
			name:     "Empty winners should return empty slice",
			winners:  "",
			expected: []uuid.UUID{},
		},
		{
			name:     "Single winner should parse correctly",
			winners:  "550e8400-e29b-41d4-a716-446655440000",
			expected: []uuid.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
		},
		{
			name:    "Multiple winners should parse correctly",
			winners: "550e8400-e29b-41d4-a716-446655440000,6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			expected: []uuid.UUID{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			betHost := &BetHost{Winners: tt.winners}
			result := betHost.GetWinners()
			if len(result) != len(tt.expected) {
				t.Errorf("GetWinners() returned %d winners, want %d", len(result), len(tt.expected))
				return
			}
			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("GetWinners()[%d] = %v, want %v", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestBetHost_SetWinners(t *testing.T) {
	tests := []struct {
		name     string
		winners  []uuid.UUID
		expected string
	}{
		{
			name:     "Empty winners should set empty string",
			winners:  []uuid.UUID{},
			expected: "",
		},
		{
			name:     "Single winner should set correctly",
			winners:  []uuid.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "Multiple winners should set correctly",
			winners: []uuid.UUID{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			},
			expected: "550e8400-e29b-41d4-a716-446655440000,6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			betHost := &BetHost{}
			betHost.SetWinners(tt.winners)
			if betHost.Winners != tt.expected {
				t.Errorf("SetWinners() set Winners = %v, want %v", betHost.Winners, tt.expected)
			}
		})
	}
}

func TestBetHost_IsCancelled(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "Cancelled status should return true",
			status:   string(BetStatusCancelled),
			expected: true,
		},
		{
			name:     "Finished status should return false",
			status:   string(BetStatusFinished),
			expected: false,
		},
		{
			name:     "Voting status should return false",
			status:   string(BetStatusVoting),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			betHost := &BetHost{Status: tt.status}
			result := betHost.IsCancelled()
			if result != tt.expected {
				t.Errorf("IsCancelled() = %v, want %v", result, tt.expected)
			}
		})
	}
}
