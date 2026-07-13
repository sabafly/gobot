package models

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/permissions"
)

type Guild struct {
	ID                     snowflake.ID                            `gorm:"primary_key;column:id;type:bigint(20) unsigned;not null"`
	Name                   string                                  `gorm:"not null"`
	Locale                 discord.Locale                          `gorm:"default:'ja'"`
	LevelUpMessage         string                                  `gorm:"default:'{user}がレベルアップしたよ！🥳\n**{before_level} レベル → {after_level} レベル**'"`
	LevelUpChannel         *snowflake.ID                           `gorm:"type:bigint(20)"`
	LevelUpExcludeChannel  []snowflake.ID                          `gorm:"serializer:json"`
	LevelMee6Imported      bool                                    `gorm:"default:false"`
	LevelRole              map[int]snowflake.ID                    `gorm:"serializer:json"`
	Permissions            map[snowflake.ID]permissions.Permission `gorm:"serializer:json"`
	RemindCount            int                                     `gorm:"default:0"`
	RolePanelEditTimes     []time.Time                             `gorm:"serializer:json"`
	BumpEnabled            bool                                    `gorm:"default:true"`
	BumpMessageTitle       string                                  `gorm:"default:'Bumpを検知しました'"`
	BumpMessage            string                                  `gorm:"default:'２時間後に通知します'"`
	BumpRemindMessageTitle string                                  `gorm:"default:'Bumpの時間です'"`
	BumpRemindMessage      string                                  `gorm:"default:'</bump:947088344167366698>でBumpしましょう'"`
	UpEnabled              bool                                    `gorm:"default:true"`
	UpMessageTitle         string                                  `gorm:"default:'UPを検知しました'"`
	UpMessage              string                                  `gorm:"default:'１時間後に通知します'"`
	UpRemindMessageTitle   string                                  `gorm:"default:'UPの時間です'"`
	UpRemindMessage        string                                  `gorm:"default:'</dissoku up:828002256690610256>でUPしましょう'"`
	BumpMention            *snowflake.ID                           `gorm:"type:bigint(20)"`
	UpMention              *snowflake.ID                           `gorm:"type:bigint(20)"`
	LevelingDisabled       bool                                    `gorm:"default:false"`

	OwnerID *snowflake.ID `gorm:"type:bigint(20) unsigned;column:owner_id;index:idx_guild_owner"`
	Owner   *User         `gorm:"foreignKey:OwnerID;constraint:OnDelete:SET NULL;"`
}
