package models

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type BetHost struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;"`
	Title    string    `gorm:"not null;"`
	VoteType string    `gorm:"not null;"`
	Price    *int64
	OwnerID  snowflake.ID `gorm:"type:bigint(20);not null;"`
	Owner    User         `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE;"`
}

type BetVoteType string

const (
	BetVoteTypeGuess BetVoteType = "GUESS"
	BetVoteTypeRace  BetVoteType = "RACE"
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
