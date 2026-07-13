package discordutil

import (
	"time"

	"github.com/disgoorg/disgo/discord"
)

func LocaleToLocation(locale discord.Locale) *time.Location {
	switch locale {
	case discord.LocaleEnglishUS:
		return time.UTC
	case discord.LocaleJapanese:
		return time.FixedZone("JST", 9*60*60)
	default:
		return time.UTC
	}
}
