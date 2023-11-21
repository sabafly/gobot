package emoji

import (
	"regexp"

	"github.com/forPelevin/gomoji"
)

func MatchString(str string) bool {
	return DiscordEmoji.MatchString(str) || gomoji.ContainsEmoji(str)
}

func FindAllString(str string) []string {
	s := []string{}
	emojis := gomoji.CollectAll(str)
	for _, e := range emojis {
		s = append(s, e.Character)
	}
	discord_emojis := DiscordEmoji.FindAllString(str, -1)
	s = append(s, discord_emojis...)
	return s
}

var DiscordEmoji = regexp.MustCompile("<a?:[A-z0-9_~]+:[0-9]{18,20}>")
