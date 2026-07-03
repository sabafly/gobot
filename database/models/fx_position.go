package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
)

// primary key: user_id, guild_id
// references: user_id -> user.id, guild_id -> guild.id

type FXPositionDirection string

const (
	FXPositionDirectionBuy  FXPositionDirection = "BUY"
	FXPositionDirectionSell FXPositionDirection = "SELL"
)

type FXPosition struct {
	UserID     snowflake.ID `gorm:"primary_key;column:user_id;type:bigint(20) unsigned;not null"`
	User       User         `gorm:"foreignKey:UserID;onDelete:CASCADE"`
	GuildID    snowflake.ID `gorm:"primary_key;column:guild_id;type:bigint(20) unsigned;not null"`
	Guild      Guild        `gorm:"foreignKey:GuildID;onDelete:CASCADE"`
	Symbol     string       `gorm:"column:symbol;type:varchar(20);not null"`
	Direction  FXPositionDirection `gorm:"column:direction;type:varchar(10);not null"` // "BUY" or "SELL"
	EntryPrice float64      `gorm:"column:entry_price;type:double;not null"`
	Margin        int64        `gorm:"column:margin;type:bigint(20) unsigned;not null"`
	InitialMargin int64        `gorm:"column:initial_margin;type:bigint(20) unsigned;not null;default:0"`
	Leverage      int          `gorm:"column:leverage;type:int;not null"`
	CreatedAt     time.Time    `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
}

func (pos *FXPosition) GetInitialMargin() int64 {
	if pos.InitialMargin == 0 {
		return pos.Margin
	}
	return pos.InitialMargin
}
