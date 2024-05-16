package game

import (
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/translate"
)

func chinchiroEmbed(locale discord.Locale, players []*ent.ChinchiroPlayer) discord.Embed {
	str := ""
	for _, player := range players {
		str += fmt.Sprintf("- %s %dp (%dbet)\n", discord.UserMention(player.UserID), player.Point, builtin.NonNilOrDefault(player.Bet, 0))
	}
	return discord.NewEmbedBuilder().
		SetTitle(translate.Message(locale, "component.game.cinchiro.game.base.embed.title")).
		SetDescription(translate.Message(locale, "component.game.cinchiro.game.base.embed.description")).
		SetFields(
			discord.EmbedField{
				Name:  translate.Message(locale, "component.game.cinchiro.game.base.embed.field.players.name"),
				Value: str,
			},
		).
		Build()
}
