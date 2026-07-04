package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FXOrder struct {
	ID          uuid.UUID           `gorm:"type:char(36);primary_key;"`
	UserID      snowflake.ID        `gorm:"column:user_id;type:bigint(20) unsigned;not null;index"`
	User        User                `gorm:"foreignKey:UserID;onDelete:CASCADE"`
	GuildID     snowflake.ID        `gorm:"column:guild_id;type:bigint(20) unsigned;not null;index"`
	Guild       Guild               `gorm:"foreignKey:GuildID;onDelete:CASCADE"`
	Symbol      string              `gorm:"column:symbol;type:varchar(20);not null"`
	Direction   FXPositionDirection `gorm:"column:direction;type:varchar(10);not null"` // "BUY" or "SELL"
	OrderType   string              `gorm:"column:order_type;type:varchar(10);not null"` // "LIMIT" or "STOP"
	TargetPrice float64             `gorm:"column:target_price;type:double;not null"`
	Margin      int64               `gorm:"column:margin;type:bigint(20) unsigned;not null"`
	Leverage    int                 `gorm:"column:leverage;type:int;not null"`
	CreatedAt   time.Time           `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
}

func (order *FXOrder) BeforeCreate(tx *gorm.DB) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	return nil
}
