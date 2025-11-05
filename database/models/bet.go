package models

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type BetHost struct {
	ID        uuid.UUID    `gorm:"type:uuid;primary_key;"`
	GuildID   snowflake.ID `gorm:"type:bigint(20);not null;index:idx_bet_guild"`
	ChannelID snowflake.ID `gorm:"type:bigint(20);not null;"`
	MessageID snowflake.ID `gorm:"type:bigint(20);"`
	Title     string       `gorm:"not null;"`
	Mode      string       `gorm:"not null;"`                // poll, race, battle_royale
	Status    string       `gorm:"not null;default:'entry'"` // entry, voting, closed, finished
	Winner    *uuid.UUID   `gorm:"type:uuid;"`
	EntryFee  *int64
	OwnerID   snowflake.ID `gorm:"type:bigint(20);not null;"`
	Owner     User         `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE;"`
}

type BetVoteType string

const (
	BetVoteTypeGuess        BetVoteType = "poll"
	BetVoteTypeRace         BetVoteType = "race"
	BetVoteTypeBattleRoyale BetVoteType = "battle_royale"
)

type BetStatus string

const (
	BetStatusEntry    BetStatus = "entry"
	BetStatusVoting   BetStatus = "voting"
	BetStatusClosed   BetStatus = "closed"
	BetStatusFinished BetStatus = "finished"
)

type BetOption struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;"`
	HostID     uuid.UUID `gorm:"type:uuid;not null"`
	Host       BetHost   `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	OptionText string    `gorm:"not null;"`
	Index      int
}

// Bet represents a user's bet on an option (used in poll and race modes)
type Bet struct {
	ID        uuid.UUID    `gorm:"type:uuid;primary_key;"`
	HostID    uuid.UUID    `gorm:"type:uuid;index:idx_bet;not null"`
	Host      BetHost      `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	UserID    snowflake.ID `gorm:"type:bigint(20);index:idx_bet;not null;"`
	User      User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	OptionID  uuid.UUID    `gorm:"type:uuid;not null"`
	Option    BetOption    `gorm:"foreignKey:OptionID;constraint:OnDelete:CASCADE;"`
	Amount    int64        `gorm:"not null;"`
	Timestamp int64        `gorm:"not null;"`
}

// BetEntrant represents a user entering as a competitor (used in race and battle_royale modes)
type BetEntrant struct {
	ID       uuid.UUID    `gorm:"type:uuid;primary_key;"`
	HostID   uuid.UUID    `gorm:"type:uuid;index:idx_entrant;not null"`
	Host     BetHost      `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	UserID   snowflake.ID `gorm:"type:bigint(20);index:idx_entrant;not null;unique:idx_host_user"`
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
