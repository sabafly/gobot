package errors

import (
	"errors"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/translate"
)

var (
	As     = errors.As
	Is     = errors.Is
	Join   = errors.Join
	New    = errors.New
	Unwrap = errors.Unwrap
)

type (
	config struct {
		desc *string
	}

	Option func(*config)
)

func (c *config) options(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

func WithDescription(s string) Option {
	return func(c *config) {
		c.desc = &s
	}
}

func ErrorMessage(
	key string,
	event interface {
		RespondMessage(messageBuilder discord.MessageBuilder, opts ...rest.RequestOpt) error
		Locale() discord.Locale
	},
	opts ...Option,
) error {
	cfg := config{}
	cfg.options(opts...)

	var desc string
	if cfg.desc != nil {
		desc = *cfg.desc
	} else {
		d, err := translate.Localize(event.Locale(), key+".description", nil, 0)
		if err == nil {
			desc = d
		}
	}

	return event.RespondMessage(
		discord.NewMessageBuilder().
			SetEmbeds(
				embeds.SetEmbedProperties(
					discord.NewEmbedBuilder().
						SetTitlef("❗ %s", translate.Message(event.Locale(), key)).
						SetDescription(desc).
						SetColor(0xff2121).
						Build(),
				),
			).
			SetFlags(discord.MessageFlagEphemeral),
	)
}
