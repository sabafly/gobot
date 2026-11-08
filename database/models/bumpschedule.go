package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BumpScheduleType string

const (
	BumpTypeBump BumpScheduleType = "bump"
	BumpTypeUp   BumpScheduleType = "up"
)

type BumpSchedule struct {
	ID        uuid.UUID        `gorm:"type:uuid;primary_key;"`
	GuildID   snowflake.ID     `gorm:"type:bigint(20) unsigned;not null;uniqueIndex:idx_guild_type"`
	Guild     Guild            `gorm:"foreignKey:GuildID;constraint:OnDelete:CASCADE;"`
	Type      BumpScheduleType `gorm:"type:varchar(16);not null;uniqueIndex:idx_guild_type"`
	ChannelID snowflake.ID     `gorm:"type:bigint(20) unsigned;not null"`
	MessageID snowflake.ID     `gorm:"type:bigint(20) unsigned;not null"`
	RemindAt  time.Time        `gorm:"not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (b *BumpSchedule) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
