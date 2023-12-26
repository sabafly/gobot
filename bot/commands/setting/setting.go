package setting

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/builtin"
)

// TODO: 君たちはどう設定を作るか

func Command(c *components.Components) components.Command {
	return (&generic.GenericCommand{
		Namespace: "setting",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:         "setting",
				Description:  "setting",
				DMPermission: builtin.Ptr(false),
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommandGroup{},
				},
			},
		},
	}).SetComponent(c)
}
