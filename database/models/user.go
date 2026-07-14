package models

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/internal/xppoint"
)

type User struct {
	ID        snowflake.ID   `gorm:"primary_key;column:id;type:bigint(20) unsigned;not null"`
	Name      string         `gorm:"not null"`
	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
	Locale    discord.Locale `gorm:"default:'ja'"`
	XP        xppoint.XP     `gorm:"default:0"`
}
