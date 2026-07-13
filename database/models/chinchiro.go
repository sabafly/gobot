package models

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChinchiroState string

const (
	ChinchiroStateLobby       ChinchiroState = "LOBBY"
	ChinchiroStateHostRolling ChinchiroState = "HOST_ROLLING"
	ChinchiroStateKidsRolling ChinchiroState = "KIDS_ROLLING"
	ChinchiroStateFinished    ChinchiroState = "FINISHED"
)

type ChinchiroSession struct {
	ID                 uuid.UUID      `gorm:"type:char(36);primary_key;"`
	GuildID            snowflake.ID   `gorm:"column:guild_id;type:bigint(20) unsigned;not null;index"`
	ChannelID          snowflake.ID   `gorm:"column:channel_id;type:bigint(20) unsigned;not null"`
	MessageID          snowflake.ID   `gorm:"column:message_id;type:bigint(20) unsigned;not null"`
	HostUserID         snowflake.ID   `gorm:"column:host_user_id;type:bigint(20) unsigned;not null"`
	Bet                int64          `gorm:"column:bet;type:bigint(20) unsigned;not null"`
	State              ChinchiroState `gorm:"column:state;type:varchar(20);not null"`
	HostDices          string         `gorm:"column:host_dices;type:varchar(20)"`
	HostRollCount      int            `gorm:"column:host_roll_count;type:int;not null;default:0"`
	HostPoint          int            `gorm:"column:host_point;type:int;not null;default:0"`
	CurrentPlayerIndex int            `gorm:"column:current_player_index;type:int;not null;default:0"`
	CurrentHostIndex   int            `gorm:"column:current_host_index;type:int;not null;default:0"`
	CreatedAt          time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`

	Players []ChinchiroPlayer `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

func (s *ChinchiroSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type ChinchiroPlayer struct {
	ID        uuid.UUID    `gorm:"type:char(36);primary_key;"`
	SessionID uuid.UUID    `gorm:"column:session_id;type:char(36);not null;index"`
	UserID    snowflake.ID `gorm:"column:user_id;type:bigint(20) unsigned;not null;index"`
	IsHost    bool         `gorm:"column:is_host;not null;default:false"`
	Dices     string       `gorm:"column:dices;type:varchar(20)"`
	RollCount int          `gorm:"column:roll_count;type:int;not null;default:0"`
	Point     int          `gorm:"column:point;type:int;not null;default:0"`
	CreatedAt time.Time    `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
}

func (p *ChinchiroPlayer) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
