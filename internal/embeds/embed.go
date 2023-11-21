package embeds

import (
	"time"

	"github.com/disgoorg/disgo/discord"
)

var (
	Color   = 0x00AED9
	BotName = "gobot"
)

func SetEmbedProperties(embed discord.Embed) discord.Embed {
	now := time.Now()
	if embed.Color == 0 {
		embed.Color = Color
	}
	if embed.Footer == nil {
		embed.Footer = &discord.EmbedFooter{}
	}
	if embed.Footer.Text == "" {
		embed.Footer.Text = BotName
	}
	if embed.Timestamp == nil {
		embed.Timestamp = &now
	}
	return embed
}

func SetEmbedsProperties(embeds []discord.Embed) []discord.Embed {
	now := time.Now()
	for i := range embeds {
		if embeds[i].Color == 0 {
			embeds[i].Color = Color
		}
		if i == len(embeds)-1 {
			if embeds[i].Footer == nil {
				embeds[i].Footer = &discord.EmbedFooter{}
			}
			if embeds[i].Footer.Text == "" {
				embeds[i].Footer.Text = BotName
			}
			if embeds[i].Timestamp == nil {
				embeds[i].Timestamp = &now
			}
		}
	}
	return embeds
}
