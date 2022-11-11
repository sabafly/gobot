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

var StL = map[string]discordgo.Locale{
	"English (Great Britain)": discordgo.EnglishGB,
	"Bulgarian":               discordgo.Bulgarian,
	"Chinese (China)":         discordgo.ChineseCN,
	"Chinese (Taiwan)":        discordgo.ChineseTW,
	"Croatian":                discordgo.Croatian,
	"Czech":                   discordgo.Czech,
	"Danish":                  discordgo.Danish,
	"Dutch":                   discordgo.Dutch,
	"Finnish":                 discordgo.Finnish,
	"French":                  discordgo.French,
	"German":                  discordgo.German,
	"Greek":                   discordgo.Greek,
	"Hindi":                   discordgo.Hindi,
	"Hungarian":               discordgo.Hungarian,
	"Italian":                 discordgo.Italian,
	"Japanese":                discordgo.Japanese,
	"Korean":                  discordgo.Korean,
	"Lithuanian":              discordgo.Lithuanian,
	"Norwegian":               discordgo.Norwegian,
	"Polish":                  discordgo.Polish,
	"Portuguese (Brazil)":     discordgo.PortugueseBR,
	"Romanian":                discordgo.Romanian,
	"Russian":                 discordgo.Russian,
	"Spanish (Spain)":         discordgo.SpanishES,
	"Swedish":                 discordgo.Swedish,
	"Thai":                    discordgo.Thai,
	"Turkish":                 discordgo.Turkish,
	"Ukrainian":               discordgo.Ukrainian,
	"Vietnamese":              discordgo.Vietnamese,
	"unknown":                 discordgo.Unknown,
}
