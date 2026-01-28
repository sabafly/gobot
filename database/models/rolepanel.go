package models

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type Role struct {
	ID    snowflake.ID            `json:"id"`
	Name  string                  `json:"name"`
	Emoji *discord.ComponentEmoji `json:"emoji"`
}

type RolePanel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"type:text"`
	Roles       []Role    `gorm:"serializer:json"`
	UpdatedAt   time.Time
	AppliedAt   time.Time

	GuildID snowflake.ID `gorm:"type:bigint(20);not null;index"`
	Guild   Guild        `gorm:"foreignKey:GuildID;constraint:OnDelete:CASCADE;"`

	Placements []RolePanelPlaced `gorm:"foreignKey:RolePanelID"`
	Edit       *RolePanelEdit    `gorm:"foreignKey:ParentID"`
}

type RolePanelEdit struct {
	ID           uuid.UUID     `gorm:"type:uuid;primary_key;"`
	ChannelID    snowflake.ID  `gorm:"type:bigint(20)"`
	EmojiAuthor  *snowflake.ID `gorm:"type:bigint(20)"`
	Token        *string
	SelectedRole *snowflake.ID `gorm:"type:bigint(20)"`
	Modified     bool          `gorm:"default:false"`
	Name         *string
	Description  *string
	Roles        []Role `gorm:"serializer:json"`

	GuildID snowflake.ID `gorm:"type:bigint(20);not null;index"`
	Guild   Guild        `gorm:"foreignKey:GuildID"`

	ParentID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Parent   RolePanel `gorm:"foreignKey:ParentID"`
}

type RolePanelPlaced struct {
	ID                uuid.UUID           `gorm:"type:uuid;primary_key;"`
	MessageID         *snowflake.ID       `gorm:"type:bigint(20)"`
	ChannelID         snowflake.ID        `gorm:"type:bigint(20)"`
	Type              string              `gorm:"type:varchar(20)"` // button, reaction, select_menu
	ButtonType        discord.ButtonStyle `gorm:"default:1"`
	ShowName          bool                `gorm:"default:false"`
	FoldingSelectMenu bool                `gorm:"default:true"`
	HideNotice        bool                `gorm:"default:false"`
	UseDisplayName    bool                `gorm:"default:false"`
	CreatedAt         time.Time           `gorm:"default:CURRENT_TIMESTAMP"`
	Uses              int                 `gorm:"default:0"`
	Name              string              `gorm:"not null"`
	Description       string              `gorm:"type:text"`
	Roles             []Role              `gorm:"serializer:json"`
	UpdatedAt         time.Time

	GuildID snowflake.ID `gorm:"type:bigint(20);not null;index"`
	Guild   Guild        `gorm:"foreignKey:GuildID;constraint:OnDelete:CASCADE;"`

	RolePanelID uuid.UUID `gorm:"type:uuid;not null;index"`
	RolePanel   RolePanel `gorm:"foreignKey:RolePanelID;constraint:OnDelete:CASCADE;"`
}
