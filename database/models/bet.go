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
	Mode      string       `gorm:"not null;"` // poll, race, battle_royale
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
	BetStatusEntry   BetStatus = "entry"
	BetStatusVoting  BetStatus = "voting"
	BetStatusClosed  BetStatus = "closed"
	BetStatusFinished BetStatus = "finished"
)

type BetOption struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;"`
	HostID     uuid.UUID `gorm:"type:uuid;not null"`
	Host       BetHost   `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	OptionText string    `gorm:"not null;"`
}

type BetParticipant struct {
	ID       uuid.UUID    `gorm:"type:uuid;primary_key;"`
	HostID   uuid.UUID    `gorm:"type:uuid;index:idx_participant;not null"`
	Host     BetHost      `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE;"`
	UserID   snowflake.ID `gorm:"type:bigint(20);index:idx_participant;not null;"`
	User     User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	OptionID uuid.UUID    `gorm:"type:uuid;not null"`
	Option   BetOption    `gorm:"foreignKey:OptionID;constraint:OnDelete:CASCADE;"`
}

// IsOwner checks if the given userID is the owner of this bet session
func (b *BetHost) IsOwner(userID snowflake.ID) bool {
	return b.OwnerID == userID
}
