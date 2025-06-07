package i18n

import (
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type Emoji struct {
	Name     string       `yaml:"name"`
	ID       snowflake.ID `yaml:"id,omitempty"`
	Animated bool         `yaml:"animated,omitempty"`
}

func (e Emoji) Emoji() *discord.ComponentEmoji {
	return &discord.ComponentEmoji{
		Name:     e.Name,
		ID:       e.ID,
		Animated: e.Animated,
	}
}

func (e *Emoji) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	text := string(b)
	if text[0] == '<' && text[len(text)-1] == '>' {
		// separate by ':'
		parts := strings.SplitN(text[1:len(text)-1], ":", 3)
		if len(parts) == 3 && parts[0] == "a" {
			e.Animated = true
		} else {
			return ErrInvalidEmojiFormat.Format(text)
		}
		if len(parts) >= 2 {
			e.Name = parts[len(parts)-2]
		}
		if len(parts) >= 2 && parts[0] != "" {
			id, err := snowflake.Parse(parts[0])
			if err != nil {
				return err
			}
			e.ID = id
		}
	} else {
		e.Name = text
		e.ID = 0
		e.Animated = false
	}
	return nil
}
