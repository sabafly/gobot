package ping

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/bot/components/generic"
)

func Command() *generic.GenericCommand {
	return &generic.GenericCommand{
		CommandCreate: []discord.ApplicationCommandCreate{},
	}
}
