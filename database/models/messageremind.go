package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

func (m *MessageRemind) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
