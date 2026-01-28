package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type MessageRemind struct {
	ID        uuid.UUID    `gorm:"type:uuid;primary_key;"`
	GuildID   snowflake.ID `gorm:"type:bigint(20);not null;index"`
	Guild     Guild        `gorm:"foreignKey:GuildID;constraint:OnDelete:CASCADE;"`
	ChannelID snowflake.ID `gorm:"type:bigint(20)"`
	AuthorID  snowflake.ID `gorm:"type:bigint(20)"`
	Time      time.Time
	Content   string `gorm:"type:text;not null"`
	Name      string `gorm:"not null"`
}
