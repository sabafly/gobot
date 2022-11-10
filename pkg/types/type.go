package types

import (
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type GlobalBan struct {
	Code    int64     `json:"code"`
	Status  string    `json:"status"`
	Content []Content `json:"content"`
}

type Content struct {
	ID        int64       `json:"ID"`
	CreatedAt string      `json:"CreatedAt"`
	UpdatedAt string      `json:"UpdatedAt"`
	DeletedAt interface{} `json:"DeletedAt"`
	Reason    string      `json:"Reason"`
}

type TransMCServer struct {
	FeedMCServer
	Address string
	Port    uint16
}

type FeedMCServer struct {
	gorm.Model
	Hash      string `gorm:"uniqueIndex"`
	GuildID   string
	ChannelID string
	RoleID    string
	Name      string
	Locale    discordgo.Locale
}

type FeedMCServers []FeedMCServer

type ImagePngHash struct {
	gorm.Model
	Hash string `gorm:"primarykey"`
	Data string `gorm:"primarykey"`
}
