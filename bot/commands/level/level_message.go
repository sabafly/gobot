package level

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/sabafly/gobot/internal/xppoint"
)

func levelMessage(
	g *ent.Guild,
	gl *discord.Guild,
	m *ent.Member,
	index int,
	member discord.Member,
	event interface {
		Locale() discord.Locale
	},
) discord.Embed {
	return discord.NewEmbedBuilder().
		SetEmbedAuthor(
			&discord.EmbedAuthor{
				Name:    g.Name,
				IconURL: builtin.NonNil(gl.IconURL()),
			},
		).
		SetThumbnail(member.EffectiveAvatarURL()).
		SetTitle(
			translate.Message(
				event.Locale(), "components.level.rank.embed.title",
				translate.WithTemplate(
					map[string]any{
						"User": member.EffectiveName(),
					},
				),
			),
		).
		SetDescription("## "+translate.Message(event.Locale(), "components.level.rank.embed.description",
			translate.WithTemplate(map[string]any{
				"Level": m.Xp.Level(),
				"Xp":    m.Xp,
			}),
		)).
		SetFields(
			discord.EmbedField{
				Name:   translate.Message(event.Locale(), "components.level.rank.embed.fields.place"),
				Value:  fmt.Sprintf("**#%d**", index+1),
				Inline: builtin.Ptr(true),
			},
			discord.EmbedField{
				Name: translate.Message(event.Locale(), "components.level.rank.embed.fields.next_level",
					translate.WithTemplate(map[string]any{"NextLevel": m.Xp.Level() + 1}),
				),
				Value: fmt.Sprintf("`%d`xp / `%d`xp",
					xppoint.RequiredPoint(m.Xp.Level())-(xppoint.TotalPoint(m.Xp.Level()+1)-uint64(m.Xp)),
					xppoint.RequiredPoint(m.Xp.Level()),
				),
				Inline: builtin.Ptr(true),
			},
		).
		Build()
}
