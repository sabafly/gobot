package errors

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/sabafly/gobot/internal/translate"
)

func ErrorMessage(
	key string,
	event interface {
		CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
		Locale() discord.Locale
	},
) error {
	desc, err := translate.Localize(event.Locale(), key+".description", nil, 0)
	if err != nil {
		desc = ""
	}

	return event.CreateMessage(
		discord.NewMessageBuilder().
			SetEmbeds(
				discord.NewEmbedBuilder().
					SetTitlef("❗ %s", translate.Message(event.Locale(), key)).
					SetDescription(desc).
					SetColor(0xff2121).
					Build(),
			).
			SetFlags(discord.MessageFlagEphemeral).
			Create(),
	)
}
