package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type GoPointTaxConfig struct {
	GuildID      snowflake.ID `gorm:"primary_key;column:guild_id;type:bigint(20) unsigned;not null"`
	Guild        Guild        `gorm:"foreignKey:GuildID;onDelete:CASCADE"`
	Rate         int          `gorm:"column:rate;type:int;not null;default:0"`          // percentage, e.g. 5 for 5%
	IntervalDays int          `gorm:"column:interval_days;type:int;not null;default:0"` // interval in days, e.g. 7
	NextTaxTime  time.Time    `gorm:"column:next_tax_time;type:datetime"`               // next scheduled calculation time
	Enabled      bool         `gorm:"column:enabled;type:tinyint(1);not null;default:0"`
	MinPoints    int64        `gorm:"column:min_points;type:bigint(20);not null;default:0"`
	Brackets     string       `gorm:"column:brackets;type:text"`
}

type GoPointPendingTax struct {
	ID            uuid.UUID    `gorm:"primary_key;column:id;type:varchar(36);not null"`
	GuildID       snowflake.ID `gorm:"column:guild_id;type:bigint(20) unsigned;not null;index"`
	Guild         Guild        `gorm:"foreignKey:GuildID;onDelete:CASCADE"`
	UserID        snowflake.ID `gorm:"column:user_id;type:bigint(20) unsigned;not null;index"`
	User          User         `gorm:"foreignKey:UserID;onDelete:CASCADE"`
	BasePoints    int64        `gorm:"column:base_points;type:bigint(20) unsigned;not null"` // point amount when calculated
	TaxAmount     int64        `gorm:"column:tax_amount;type:bigint(20) unsigned;not null"`  // calculated tax amount
	CalculateTime time.Time    `gorm:"column:calculate_time;type:datetime;not null"`         // when calculation occurred
	CollectTime   time.Time    `gorm:"column:collect_time;type:datetime;not null;index"`     // calculate_time + 1 week
	Collected     bool         `gorm:"column:collected;type:tinyint(1);not null;default:0"`
	Exempted      bool         `gorm:"column:exempted;type:tinyint(1);not null;default:0"`
}

type GoPointSeason struct {
	ID         uuid.UUID    `gorm:"primary_key;column:id;type:varchar(36);not null"`
	GuildID    snowflake.ID `gorm:"column:guild_id;type:bigint(20) unsigned;not null;index"`
	Guild      Guild        `gorm:"foreignKey:GuildID;onDelete:CASCADE"`
	Name       string       `gorm:"column:name;type:varchar(255);not null"`
	StartTime  time.Time    `gorm:"column:start_time;type:datetime;not null"`
	EndTime    time.Time    `gorm:"column:end_time;type:datetime;not null"`
	IsActive   bool         `gorm:"column:is_active;type:tinyint(1);not null;default:0"`
	HasAwarded bool         `gorm:"column:has_awarded;type:tinyint(1);not null;default:0"`
	ChannelID  snowflake.ID `gorm:"column:channel_id;type:bigint(20) unsigned;not null;default:0"`
	Criteria   string       `gorm:"column:criteria;type:varchar(50);not null;default:'earned'"`
}

type GoPointSeasonUser struct {
	SeasonID     uuid.UUID     `gorm:"primary_key;column:season_id;type:varchar(36);not null"`
	Season       GoPointSeason `gorm:"foreignKey:SeasonID;onDelete:CASCADE"`
	UserID       snowflake.ID  `gorm:"primary_key;column:user_id;type:bigint(20) unsigned;not null"`
	User         User          `gorm:"foreignKey:UserID;onDelete:CASCADE"`
	GuildID      snowflake.ID  `gorm:"column:guild_id;type:bigint(20) unsigned;not null;index"`
	Guild        Guild         `gorm:"foreignKey:GuildID;onDelete:CASCADE"`
	PointsEarned int64         `gorm:"column:points_earned;type:bigint(20);not null;default:0"`
}
