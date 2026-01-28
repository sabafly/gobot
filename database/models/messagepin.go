package models

import (
	"encoding/json"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RateLimit struct {
	Limit []time.Time
}

func (r RateLimit) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Limit)
}

func (r *RateLimit) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &r.Limit)
}

func (r *RateLimit) CheckLimit() bool {
	if (len(r.Limit) >= 3 && time.Since(r.Limit[2]) < time.Second*5) || (len(r.Limit) >= 10 && time.Since(r.Limit[9]) < time.Second*30) {
		return false
	}
	r.Limit = append([]time.Time{time.Now()}, r.Limit[0:min(10, len(r.Limit))]...)
	ok := (len(r.Limit) < 3 || time.Since(r.Limit[2]) >= time.Second*5) && (len(r.Limit) < 10 || time.Since(r.Limit[9]) >= time.Second*30)
	return ok
}

type MessagePin struct {
	ID        uuid.UUID       `gorm:"type:uuid;primary_key;"`
	GuildID   snowflake.ID    `gorm:"type:bigint(20);not null;index"`
	Guild     Guild           `gorm:"foreignKey:GuildID;constraint:OnDelete:CASCADE;"`
	ChannelID snowflake.ID    `gorm:"type:bigint(20);uniqueIndex"`
	Content   string          `gorm:"type:text"`
	Embeds    []discord.Embed `gorm:"serializer:json"`
	BeforeID  *snowflake.ID   `gorm:"type:bigint(20)"`
	RateLimit RateLimit       `gorm:"serializer:json"`
}

func (m *MessagePin) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
