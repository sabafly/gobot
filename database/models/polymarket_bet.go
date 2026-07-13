package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PolymarketBet struct {
	ID          uuid.UUID    `gorm:"type:char(36);primary_key;"`
	UserID      snowflake.ID `gorm:"column:user_id;type:bigint(20) unsigned;not null;index"`
	User        User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	GuildID     snowflake.ID `gorm:"column:guild_id;type:bigint(20) unsigned;not null;index"`
	Guild       Guild        `gorm:"foreignKey:GuildID;constraint:OnDelete:CASCADE;"`
	MarketID    string       `gorm:"column:market_id;type:varchar(255);not null;index"`
	MarketTitle string       `gorm:"column:market_title;type:text;not null"`
	TokenID     string       `gorm:"column:token_id;type:varchar(255);not null"`
	Outcome     string       `gorm:"column:outcome;type:varchar(100);not null"` // e.g. "Yes" or "No"
	BetAmount   int64        `gorm:"column:bet_amount;type:bigint(20) unsigned;not null"`
	EntryPrice  float64      `gorm:"column:entry_price;type:double;not null"` // price of token at bet time (between 0.0 and 1.0)
	Resolved    bool         `gorm:"column:resolved;not null;default:false"`
	Winner      bool         `gorm:"column:winner;not null;default:false"`
	Payout      int64        `gorm:"column:payout;type:bigint(20) unsigned;not null;default:0"`
	CreatedAt   time.Time    `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
}

func (b *PolymarketBet) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
