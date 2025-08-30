package models

import (
	"github.com/disgoorg/snowflake/v2"
)

// primary key: user_id, guild_id
// references: user_id -> user.id, guild_id -> guild.id

type GoPoint struct {
	UserID  snowflake.ID `gorm:"primary_key;column:user_id;type:bigint(20) unsigned;not null"`
	User    User         `gorm:"foreignKey:UserID;onDelete:CASCADE"`
	GuildID snowflake.ID `gorm:"primary_key;column:guild_id;type:bigint(20) unsigned;not null"`
	Guild   Guild        `gorm:"foreignKey:GuildID;onDelete:CASCADE"`
	Points  int64        `gorm:"column:points;type:bigint(20) unsigned;not null;default:0"`
}
