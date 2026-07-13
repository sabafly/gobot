package models

import (
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BetHost struct {
	ID                  uuid.UUID    `gorm:"type:uuid;primary_key;"`
	GuildID             snowflake.ID `gorm:"type:bigint(20);not null;index:idx_bet_guild"`
	ChannelID           snowflake.ID `gorm:"type:bigint(20);not null;"`
	MessageID           snowflake.ID `gorm:"type:bigint(20);"`
	Title               string       `gorm:"not null;"`
	Mode                string       `gorm:"not null;"`                // poll, race, battle_royale
	Status              string       `gorm:"not null;default:'entry'"` // entry, voting, closed, finished, cancelled
	Winners             string       `gorm:"type:text;"`               // Comma-separated winner UUIDs, empty for cancellation
	EntryFee            *int64
	PrizePool           *int64       // Organizer-contributed prize pool
	OwnerID             snowflake.ID `gorm:"type:bigint(20);not null;"`
	Owner               User         `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE;"`
	AllowVoteDestChange bool         `gorm:"not null;"` // Allow users to change their vote destination after voting
	CreatedAt           time.Time
	EntryDeadline       *time.Time `gorm:"index:idx_entry_deadline;"` // Deadline for race/battle_royale entry
	VoteDeadline        *time.Time `gorm:"index:idx_vote_deadline;"`
	Locale              string     `gorm:"type:varchar(10);not null;default:'en';"`
}

func (b *BetHost) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type BetVoteType string

const (
	BetVoteTypeGuess        = "poll"
	BetVoteTypeRace         = "race"
	BetVoteTypeBattleRoyale = "battle_royale"
)

type BetStatus string

const (
	BetStatusEntry     = "entry"     // Accepting entries
	BetStatusVoting    = "voting"    // Accepting votes
	BetStatusClosed    = "closed"    // No more bets can be placed
	BetStatusFinished  = "finished"  // Bet session finished
	BetStatusCancelled = "cancelled" // Bet session cancelled
)

type BetOption struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;"`
	HostID     uuid.UUID `gorm:"type:uuid;not null"`
	Host       BetHost   `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	OptionText string    `gorm:"not null;"`
	// auto-incremented index for ordering options
	Index int `gorm:"not null;"`
}

// Bet represents a user's bet on an option (used in poll and race modes)
type Bet struct {
	ID        uuid.UUID    `gorm:"type:uuid;primary_key;"`
	HostID    uuid.UUID    `gorm:"type:uuid;index:idx_bet;not null;uniqueIndex:idx_bet_host_user"`
	Host      BetHost      `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	UserID    snowflake.ID `gorm:"type:bigint(20);index:idx_bet;not null;uniqueIndex:idx_bet_host_user"`
	User      User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	OptionID  uuid.UUID    `gorm:"type:uuid;not null"`
	Option    BetOption    `gorm:"foreignKey:OptionID;constraint:OnDelete:CASCADE;"`
	Amount    int64        `gorm:"not null;"`
	Timestamp int64        `gorm:"not null;"` // Unix timestamp of when the bet was placed
}

// BetEntrant represents a user entering as a competitor (used in race and battle_royale modes)
type BetEntrant struct {
	ID       uuid.UUID    `gorm:"type:uuid;primary_key;"`
	HostID   uuid.UUID    `gorm:"type:uuid;index:idx_entrant;not null;uniqueIndex:idx_host_user"`
	Host     BetHost      `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	UserID   snowflake.ID `gorm:"type:bigint(20);index:idx_entrant;not null;uniqueIndex:idx_host_user"`
	User     User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	OptionID uuid.UUID    `gorm:"type:uuid;not null"`
	Option   BetOption    `gorm:"foreignKey:OptionID;constraint:OnDelete:CASCADE;"`
}

// BetParticipant is an alias for backward compatibility
type BetParticipant = Bet

// IsOwner checks if the given userID is the owner of this bet session
func (b *BetHost) IsOwner(userID snowflake.ID) bool {
	return b.OwnerID == userID
}

// GetWinners returns the list of winner UUIDs
func (b *BetHost) GetWinners() []uuid.UUID {
	if b.Winners == "" {
		return []uuid.UUID{}
	}
	parts := strings.Split(b.Winners, ",")
	winners := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		if id, err := uuid.Parse(strings.TrimSpace(part)); err == nil {
			winners = append(winners, id)
		}
	}
	return winners
}

// SetWinners sets the winners from a list of UUIDs
func (b *BetHost) SetWinners(winners []uuid.UUID) {
	if len(winners) == 0 {
		b.Winners = ""
		return
	}
	parts := make([]string, len(winners))
	for i, w := range winners {
		parts[i] = w.String()
	}
	b.Winners = strings.Join(parts, ",")
}

// IsCancelled checks if the bet session was cancelled
func (b *BetHost) IsCancelled() bool {
	return b.Status == string(BetStatusCancelled)
}
