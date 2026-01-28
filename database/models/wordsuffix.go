package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type WordSuffix struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Suffix  string    `gorm:"not null"`
	Expired *time.Time

	GuildID *snowflake.ID `gorm:"type:bigint(20);index"`
	Guild   *Guild        `gorm:"foreignKey:GuildID"`

	OwnerID snowflake.ID `gorm:"type:bigint(20);not null;index"`
	Owner   User         `gorm:"foreignKey:OwnerID"`

	Rule string `gorm:"default:'webhook'"` // webhook, warn, delete
}
