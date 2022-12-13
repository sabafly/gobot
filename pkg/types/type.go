package types

import (
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
	"gorm.io/gorm"
)

type GlobalBan struct {
	Code    int64  `json:"code"`
	Status  string `json:"status"`
	Content []Ban  `json:"content"`
}

type Ban struct {
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

type Res struct {
	Code    int64       `json:"code"`
	Status  string      `json:"status"`
	Content interface{} `json:"content"`
}

type MCServer struct {
	gorm.Model
	Hash    string `gorm:"uniqueIndex"`
	Address string
	Port    uint16
	Online  bool
}

type MCServers []MCServer

type PanelEmojiConfig struct {
	Message     string
	Emojis      []*discordgo.ComponentEmoji
	MessageData *discordgo.Message
	SelectMenu  discordgo.SelectMenu
}

type MessageSelect struct {
	MemberID string
	GuildID  string
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

var Twemoji = regexp2.MustCompile("((?:\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffc}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}\\x{dffd}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}\\x{dffc}\\x{dffe}\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dffd}\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dffe}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffb}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffc}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffc}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}\\x{dffd}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffd}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}\\x{dffc}\\x{dffe}\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dffe}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dffd}\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{d83c}\\x{dfff}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dffe}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffb}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffc}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffb}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffc}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffc}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}\\x{dffd}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffc}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}\\x{dffd}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffd}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}\\x{dffc}\\x{dffe}\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffd}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}\\x{dffc}\\x{dffe}\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffe}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dffd}\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dffe}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dffd}\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dfff}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc68}\\x{d83c}[\\x{dffb}-\\x{dffe}]|\\x{d83d}\\x{dc69}\\x{d83c}\\x{dfff}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83d}\\x{dc69}\\x{d83c}[\\x{dffb}-\\x{dffe}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffb}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffc}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffb}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffc}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}\\x{dffd}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffc}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffd}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}\\x{dffc}\\x{dffe}\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffd}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffe}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dffd}\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dffe}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dfff}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dffe}]|\\x{d83e}\\x{ddd1}\\x{d83c}\\x{dfff}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83e}\\x{ddd1}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc68}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}\\x{dc68}|\\x{d83d}\\x{dc69}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc8b}\\x{200d}\\x{d83d}[\\x{dc68}\\x{dc69}]|\\x{d83e}\\x{def1}\\x{d83c}\\x{dffb}\\x{200d}\\x{d83e}\\x{def2}\\x{d83c}[\\x{dffc}-\\x{dfff}]|\\x{d83e}\\x{def1}\\x{d83c}\\x{dffc}\\x{200d}\\x{d83e}\\x{def2}\\x{d83c}[\\x{dffb}\\x{dffd}-\\x{dfff}]|\\x{d83e}\\x{def1}\\x{d83c}\\x{dffd}\\x{200d}\\x{d83e}\\x{def2}\\x{d83c}[\\x{dffb}\\x{dffc}\\x{dffe}\\x{dfff}]|\\x{d83e}\\x{def1}\\x{d83c}\\x{dffe}\\x{200d}\\x{d83e}\\x{def2}\\x{d83c}[\\x{dffb}-\\x{dffd}\\x{dfff}]|\\x{d83e}\\x{def1}\\x{d83c}\\x{dfff}\\x{200d}\\x{d83e}\\x{def2}\\x{d83c}[\\x{dffb}-\\x{dffe}]|\\x{d83d}\\x{dc68}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dc68}|\\x{d83d}\\x{dc69}\\x{200d}\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}[\\x{dc68}\\x{dc69}]|\\x{d83e}\\x{ddd1}\\x{200d}\\x{d83e}\\x{dd1d}\\x{200d}\\x{d83e}\\x{ddd1}|\\x{d83d}\\x{dc6b}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc6c}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc6d}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc8f}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}\\x{dc91}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83e}\\x{dd1d}\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{d83d}[\\x{dc6b}-\\x{dc6d}\\x{dc8f}\\x{dc91}]|\\x{d83e}\\x{dd1d})|(?:\\x{d83d}[\\x{dc68}\\x{dc69}]|\\x{d83e}\\x{ddd1})(?:\\x{d83c}[\\x{dffb}-\\x{dfff}])?\\x{200d}(?:\\x{2695}\\x{fe0f}|\\x{2696}\\x{fe0f}|\\x{2708}\\x{fe0f}|\\x{d83c}[\\x{df3e}\\x{df73}\\x{df7c}\\x{df84}\\x{df93}\\x{dfa4}\\x{dfa8}\\x{dfeb}\\x{dfed}]|\\x{d83d}[\\x{dcbb}\\x{dcbc}\\x{dd27}\\x{dd2c}\\x{de80}\\x{de92}]|\\x{d83e}[\\x{ddaf}-\\x{ddb3}\\x{ddbc}\\x{ddbd}])|(?:\\x{d83c}[\\x{dfcb}\\x{dfcc}]|\\x{d83d}[\\x{dd74}\\x{dd75}]|\\x{26f9})((?:\\x{d83c}[\\x{dffb}-\\x{dfff}]|\\x{fe0f})\\x{200d}[\\x{2640}\\x{2642}]\\x{fe0f})|(?:\\x{d83c}[\\x{dfc3}\\x{dfc4}\\x{dfca}]|\\x{d83d}[\\x{dc6e}\\x{dc70}\\x{dc71}\\x{dc73}\\x{dc77}\\x{dc81}\\x{dc82}\\x{dc86}\\x{dc87}\\x{de45}-\\x{de47}\\x{de4b}\\x{de4d}\\x{de4e}\\x{dea3}\\x{deb4}-\\x{deb6}]|\\x{d83e}[\\x{dd26}\\x{dd35}\\x{dd37}-\\x{dd39}\\x{dd3d}\\x{dd3e}\\x{ddb8}\\x{ddb9}\\x{ddcd}-\\x{ddcf}\\x{ddd4}\\x{ddd6}-\\x{dddd}])(?:\\x{d83c}[\\x{dffb}-\\x{dfff}])?\\x{200d}[\\x{2640}\\x{2642}]\\x{fe0f}|(?:\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc66}\\x{200d}\\x{d83d}\\x{dc66}|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc67}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc66}\\x{200d}\\x{d83d}\\x{dc66}|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc67}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc66}\\x{200d}\\x{d83d}\\x{dc66}|\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc67}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc66}\\x{200d}\\x{d83d}\\x{dc66}|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc67}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc66}\\x{200d}\\x{d83d}\\x{dc66}|\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc67}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83c}\\x{dff3}\\x{fe0f}\\x{200d}\\x{26a7}\\x{fe0f}|\\x{d83c}\\x{dff3}\\x{fe0f}\\x{200d}\\x{d83c}\\x{df08}|\\x{d83d}\\x{de36}\\x{200d}\\x{d83c}\\x{df2b}\\x{fe0f}|\\x{2764}\\x{fe0f}\\x{200d}\\x{d83d}\\x{dd25}|\\x{2764}\\x{fe0f}\\x{200d}\\x{d83e}\\x{de79}|\\x{d83c}\\x{dff4}\\x{200d}\\x{2620}\\x{fe0f}|\\x{d83d}\\x{dc15}\\x{200d}\\x{d83e}\\x{ddba}|\\x{d83d}\\x{dc3b}\\x{200d}\\x{2744}\\x{fe0f}|\\x{d83d}\\x{dc41}\\x{200d}\\x{d83d}\\x{dde8}|\\x{d83d}\\x{dc68}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc69}\\x{200d}\\x{d83d}[\\x{dc66}\\x{dc67}]|\\x{d83d}\\x{dc6f}\\x{200d}\\x{2640}\\x{fe0f}|\\x{d83d}\\x{dc6f}\\x{200d}\\x{2642}\\x{fe0f}|\\x{d83d}\\x{de2e}\\x{200d}\\x{d83d}\\x{dca8}|\\x{d83d}\\x{de35}\\x{200d}\\x{d83d}\\x{dcab}|\\x{d83e}\\x{dd3c}\\x{200d}\\x{2640}\\x{fe0f}|\\x{d83e}\\x{dd3c}\\x{200d}\\x{2642}\\x{fe0f}|\\x{d83e}\\x{ddde}\\x{200d}\\x{2640}\\x{fe0f}|\\x{d83e}\\x{ddde}\\x{200d}\\x{2642}\\x{fe0f}|\\x{d83e}\\x{dddf}\\x{200d}\\x{2640}\\x{fe0f}|\\x{d83e}\\x{dddf}\\x{200d}\\x{2642}\\x{fe0f}|\\x{d83d}\\x{dc08}\\x{200d}\\x{2b1b})|[#*0-9]\\x{fe0f}?\\x{20e3}|(?:[©®\\x{2122}\\x{265f}]\\x{fe0f})|(?:\\x{d83c}[\\x{dc04}\\x{dd70}\\x{dd71}\\x{dd7e}\\x{dd7f}\\x{de02}\\x{de1a}\\x{de2f}\\x{de37}\\x{df21}\\x{df24}-\\x{df2c}\\x{df36}\\x{df7d}\\x{df96}\\x{df97}\\x{df99}-\\x{df9b}\\x{df9e}\\x{df9f}\\x{dfcd}\\x{dfce}\\x{dfd4}-\\x{dfdf}\\x{dff3}\\x{dff5}\\x{dff7}]|\\x{d83d}[\\x{dc3f}\\x{dc41}\\x{dcfd}\\x{dd49}\\x{dd4a}\\x{dd6f}\\x{dd70}\\x{dd73}\\x{dd76}-\\x{dd79}\\x{dd87}\\x{dd8a}-\\x{dd8d}\\x{dda5}\\x{dda8}\\x{ddb1}\\x{ddb2}\\x{ddbc}\\x{ddc2}-\\x{ddc4}\\x{ddd1}-\\x{ddd3}\\x{dddc}-\\x{ddde}\\x{dde1}\\x{dde3}\\x{dde8}\\x{ddef}\\x{ddf3}\\x{ddfa}\\x{decb}\\x{decd}-\\x{decf}\\x{dee0}-\\x{dee5}\\x{dee9}\\x{def0}\\x{def3}]|[\\x{203c}\\x{2049}\\x{2139}\\x{2194}-\\x{2199}\\x{21a9}\\x{21aa}\\x{231a}\\x{231b}\\x{2328}\\x{23cf}\\x{23ed}-\\x{23ef}\\x{23f1}\\x{23f2}\\x{23f8}-\\x{23fa}\\x{24c2}\\x{25aa}\\x{25ab}\\x{25b6}\\x{25c0}\\x{25fb}-\\x{25fe}\\x{2600}-\\x{2604}\\x{260e}\\x{2611}\\x{2614}\\x{2615}\\x{2618}\\x{2620}\\x{2622}\\x{2623}\\x{2626}\\x{262a}\\x{262e}\\x{262f}\\x{2638}-\\x{263a}\\x{2640}\\x{2642}\\x{2648}-\\x{2653}\\x{2660}\\x{2663}\\x{2665}\\x{2666}\\x{2668}\\x{267b}\\x{267f}\\x{2692}-\\x{2697}\\x{2699}\\x{269b}\\x{269c}\\x{26a0}\\x{26a1}\\x{26a7}\\x{26aa}\\x{26ab}\\x{26b0}\\x{26b1}\\x{26bd}\\x{26be}\\x{26c4}\\x{26c5}\\x{26c8}\\x{26cf}\\x{26d1}\\x{26d3}\\x{26d4}\\x{26e9}\\x{26ea}\\x{26f0}-\\x{26f5}\\x{26f8}\\x{26fa}\\x{26fd}\\x{2702}\\x{2708}\\x{2709}\\x{270f}\\x{2712}\\x{2714}\\x{2716}\\x{271d}\\x{2721}\\x{2733}\\x{2734}\\x{2744}\\x{2747}\\x{2757}\\x{2763}\\x{2764}\\x{27a1}\\x{2934}\\x{2935}\\x{2b05}-\\x{2b07}\\x{2b1b}\\x{2b1c}\\x{2b50}\\x{2b55}\\x{3030}\\x{303d}\\x{3297}\\x{3299}])(?:\\x{fe0f}|(?!\\x{fe0e}))|(?:(?:\\x{d83c}[\\x{dfcb}\\x{dfcc}]|\\x{d83d}[\\x{dd74}\\x{dd75}\\x{dd90}]|[\\x{261d}\\x{26f7}\\x{26f9}\\x{270c}\\x{270d}])(?:\\x{fe0f}|(?!\\x{fe0e}))|(?:\\x{d83c}[\\x{df85}\\x{dfc2}-\\x{dfc4}\\x{dfc7}\\x{dfca}]|\\x{d83d}[\\x{dc42}\\x{dc43}\\x{dc46}-\\x{dc50}\\x{dc66}-\\x{dc69}\\x{dc6e}\\x{dc70}-\\x{dc78}\\x{dc7c}\\x{dc81}-\\x{dc83}\\x{dc85}-\\x{dc87}\\x{dcaa}\\x{dd7a}\\x{dd95}\\x{dd96}\\x{de45}-\\x{de47}\\x{de4b}-\\x{de4f}\\x{dea3}\\x{deb4}-\\x{deb6}\\x{dec0}\\x{decc}]|\\x{d83e}[\\x{dd0c}\\x{dd0f}\\x{dd18}-\\x{dd1c}\\x{dd1e}\\x{dd1f}\\x{dd26}\\x{dd30}-\\x{dd39}\\x{dd3d}\\x{dd3e}\\x{dd77}\\x{ddb5}\\x{ddb6}\\x{ddb8}\\x{ddb9}\\x{ddbb}\\x{ddcd}-\\x{ddcf}\\x{ddd1}-\\x{dddd}\\x{dec3}-\\x{dec5}\\x{def0}-\\x{def6}]|[\\x{270a}\\x{270b}]))(?:\\x{d83c}[\\x{dffb}-\\x{dfff}])?|(?:\\x{d83c}\\x{dff4}\\x{db40}\\x{dc67}\\x{db40}\\x{dc62}\\x{db40}\\x{dc65}\\x{db40}\\x{dc6e}\\x{db40}\\x{dc67}\\x{db40}\\x{dc7f}|\\x{d83c}\\x{dff4}\\x{db40}\\x{dc67}\\x{db40}\\x{dc62}\\x{db40}\\x{dc73}\\x{db40}\\x{dc63}\\x{db40}\\x{dc74}\\x{db40}\\x{dc7f}|\\x{d83c}\\x{dff4}\\x{db40}\\x{dc67}\\x{db40}\\x{dc62}\\x{db40}\\x{dc77}\\x{db40}\\x{dc6c}\\x{db40}\\x{dc73}\\x{db40}\\x{dc7f}|\\x{d83c}\\x{dde6}\\x{d83c}[\\x{dde8}-\\x{ddec}\\x{ddee}\\x{ddf1}\\x{ddf2}\\x{ddf4}\\x{ddf6}-\\x{ddfa}\\x{ddfc}\\x{ddfd}\\x{ddff}]|\\x{d83c}\\x{dde7}\\x{d83c}[\\x{dde6}\\x{dde7}\\x{dde9}-\\x{ddef}\\x{ddf1}-\\x{ddf4}\\x{ddf6}-\\x{ddf9}\\x{ddfb}\\x{ddfc}\\x{ddfe}\\x{ddff}]|\\x{d83c}\\x{dde8}\\x{d83c}[\\x{dde6}\\x{dde8}\\x{dde9}\\x{ddeb}-\\x{ddee}\\x{ddf0}-\\x{ddf5}\\x{ddf7}\\x{ddfa}-\\x{ddff}]|\\x{d83c}\\x{dde9}\\x{d83c}[\\x{ddea}\\x{ddec}\\x{ddef}\\x{ddf0}\\x{ddf2}\\x{ddf4}\\x{ddff}]|\\x{d83c}\\x{ddea}\\x{d83c}[\\x{dde6}\\x{dde8}\\x{ddea}\\x{ddec}\\x{dded}\\x{ddf7}-\\x{ddfa}]|\\x{d83c}\\x{ddeb}\\x{d83c}[\\x{ddee}-\\x{ddf0}\\x{ddf2}\\x{ddf4}\\x{ddf7}]|\\x{d83c}\\x{ddec}\\x{d83c}[\\x{dde6}\\x{dde7}\\x{dde9}-\\x{ddee}\\x{ddf1}-\\x{ddf3}\\x{ddf5}-\\x{ddfa}\\x{ddfc}\\x{ddfe}]|\\x{d83c}\\x{dded}\\x{d83c}[\\x{ddf0}\\x{ddf2}\\x{ddf3}\\x{ddf7}\\x{ddf9}\\x{ddfa}]|\\x{d83c}\\x{ddee}\\x{d83c}[\\x{dde8}-\\x{ddea}\\x{ddf1}-\\x{ddf4}\\x{ddf6}-\\x{ddf9}]|\\x{d83c}\\x{ddef}\\x{d83c}[\\x{ddea}\\x{ddf2}\\x{ddf4}\\x{ddf5}]|\\x{d83c}\\x{ddf0}\\x{d83c}[\\x{ddea}\\x{ddec}-\\x{ddee}\\x{ddf2}\\x{ddf3}\\x{ddf5}\\x{ddf7}\\x{ddfc}\\x{ddfe}\\x{ddff}]|\\x{d83c}\\x{ddf1}\\x{d83c}[\\x{dde6}-\\x{dde8}\\x{ddee}\\x{ddf0}\\x{ddf7}-\\x{ddfb}\\x{ddfe}]|\\x{d83c}\\x{ddf2}\\x{d83c}[\\x{dde6}\\x{dde8}-\\x{dded}\\x{ddf0}-\\x{ddff}]|\\x{d83c}\\x{ddf3}\\x{d83c}[\\x{dde6}\\x{dde8}\\x{ddea}-\\x{ddec}\\x{ddee}\\x{ddf1}\\x{ddf4}\\x{ddf5}\\x{ddf7}\\x{ddfa}\\x{ddff}]|\\x{d83c}\\x{ddf4}\\x{d83c}\\x{ddf2}|\\x{d83c}\\x{ddf5}\\x{d83c}[\\x{dde6}\\x{ddea}-\\x{dded}\\x{ddf0}-\\x{ddf3}\\x{ddf7}-\\x{ddf9}\\x{ddfc}\\x{ddfe}]|\\x{d83c}\\x{ddf6}\\x{d83c}\\x{dde6}|\\x{d83c}\\x{ddf7}\\x{d83c}[\\x{ddea}\\x{ddf4}\\x{ddf8}\\x{ddfa}\\x{ddfc}]|\\x{d83c}\\x{ddf8}\\x{d83c}[\\x{dde6}-\\x{ddea}\\x{ddec}-\\x{ddf4}\\x{ddf7}-\\x{ddf9}\\x{ddfb}\\x{ddfd}-\\x{ddff}]|\\x{d83c}\\x{ddf9}\\x{d83c}[\\x{dde6}\\x{dde8}\\x{dde9}\\x{ddeb}-\\x{dded}\\x{ddef}-\\x{ddf4}\\x{ddf7}\\x{ddf9}\\x{ddfb}\\x{ddfc}\\x{ddff}]|\\x{d83c}\\x{ddfa}\\x{d83c}[\\x{dde6}\\x{ddec}\\x{ddf2}\\x{ddf3}\\x{ddf8}\\x{ddfe}\\x{ddff}]|\\x{d83c}\\x{ddfb}\\x{d83c}[\\x{dde6}\\x{dde8}\\x{ddea}\\x{ddec}\\x{ddee}\\x{ddf3}\\x{ddfa}]|\\x{d83c}\\x{ddfc}\\x{d83c}[\\x{ddeb}\\x{ddf8}]|\\x{d83c}\\x{ddfd}\\x{d83c}\\x{ddf0}|\\x{d83c}\\x{ddfe}\\x{d83c}[\\x{ddea}\\x{ddf9}]|\\x{d83c}\\x{ddff}\\x{d83c}[\\x{dde6}\\x{ddf2}\\x{ddfc}]|\\x{d83c}[\\x{dccf}\\x{dd8e}\\x{dd91}-\\x{dd9a}\\x{dde6}-\\x{ddff}\\x{de01}\\x{de32}-\\x{de36}\\x{de38}-\\x{de3a}\\x{de50}\\x{de51}\\x{df00}-\\x{df20}\\x{df2d}-\\x{df35}\\x{df37}-\\x{df7c}\\x{df7e}-\\x{df84}\\x{df86}-\\x{df93}\\x{dfa0}-\\x{dfc1}\\x{dfc5}\\x{dfc6}\\x{dfc8}\\x{dfc9}\\x{dfcf}-\\x{dfd3}\\x{dfe0}-\\x{dff0}\\x{dff4}\\x{dff8}-\\x{dfff}]|\\x{d83d}[\\x{dc00}-\\x{dc3e}\\x{dc40}\\x{dc44}\\x{dc45}\\x{dc51}-\\x{dc65}\\x{dc6a}\\x{dc6f}\\x{dc79}-\\x{dc7b}\\x{dc7d}-\\x{dc80}\\x{dc84}\\x{dc88}-\\x{dc8e}\\x{dc90}\\x{dc92}-\\x{dca9}\\x{dcab}-\\x{dcfc}\\x{dcff}-\\x{dd3d}\\x{dd4b}-\\x{dd4e}\\x{dd50}-\\x{dd67}\\x{dda4}\\x{ddfb}-\\x{de44}\\x{de48}-\\x{de4a}\\x{de80}-\\x{dea2}\\x{dea4}-\\x{deb3}\\x{deb7}-\\x{debf}\\x{dec1}-\\x{dec5}\\x{ded0}-\\x{ded2}\\x{ded5}-\\x{ded7}\\x{dedd}-\\x{dedf}\\x{deeb}\\x{deec}\\x{def4}-\\x{defc}\\x{dfe0}-\\x{dfeb}\\x{dff0}]|\\x{d83e}[\\x{dd0d}\\x{dd0e}\\x{dd10}-\\x{dd17}\\x{dd20}-\\x{dd25}\\x{dd27}-\\x{dd2f}\\x{dd3a}\\x{dd3c}\\x{dd3f}-\\x{dd45}\\x{dd47}-\\x{dd76}\\x{dd78}-\\x{ddb4}\\x{ddb7}\\x{ddba}\\x{ddbc}-\\x{ddcc}\\x{ddd0}\\x{ddde}-\\x{ddff}\\x{de70}-\\x{de74}\\x{de78}-\\x{de7c}\\x{de80}-\\x{de86}\\x{de90}-\\x{deac}\\x{deb0}-\\x{deba}\\x{dec0}-\\x{dec2}\\x{ded0}-\\x{ded9}\\x{dee0}-\\x{dee7}]|[\\x{23e9}-\\x{23ec}\\x{23f0}\\x{23f3}\\x{267e}\\x{26ce}\\x{2705}\\x{2728}\\x{274c}\\x{274e}\\x{2753}-\\x{2755}\\x{2795}-\\x{2797}\\x{27b0}\\x{27bf}\\x{e50a}])|\\x{fe0f}|<(a|):[A-z0-9_~]+:[0-9]{18,20}>)", regexp2.None)

var CustomEmojiRegex = regexp.MustCompile(`<(a|):[A-z0-9_~]+:[0-9]{18,20}>`)
