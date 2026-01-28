package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/permissions"
	"github.com/sabafly/gobot/internal/xppoint"
)

type Member struct {
	ID                int                    `gorm:"primary_key;auto_increment"`
	GuildID           snowflake.ID           `gorm:"type:bigint(20);not null;index:idx_member_guild_user,unique"`
	Guild             Guild                  `gorm:"foreignKey:GuildID;constraint:OnDelete:CASCADE;"`
	UserID            snowflake.ID           `gorm:"type:bigint(20);not null;index:idx_member_guild_user,unique"`
	User              User                   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	Permission        permissions.Permission `gorm:"serializer:json"`
	XP                xppoint.XP             `gorm:"default:0"`
	LastXP            time.Time
	MessageCount      uint64 `gorm:"default:0"`
	LastNotifiedLevel *uint64
	LastMessageHashes []string `gorm:"serializer:json"`
}
