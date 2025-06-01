package star

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/models"
)

type Star struct {
	ChannelID snowflake.ID `gorm:"primary_key;column:channel_id;type:bigint(20) unsigned;not null"`
	MessageID snowflake.ID `gorm:"primary_key;column:message_id;type:bigint(20) unsigned;not null"`

	GuildID snowflake.ID `gorm:"column:guild_id;type:bigint(20) unsigned;not null"`
	Guild   models.Guild `gorm:"foreignKey:GuildID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
