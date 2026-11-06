/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package level

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"

	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/i18n"
	"github.com/sabafly/gobot/internal/xppoint"
)

func levelMessage(
	g *models.Guild,
	gl *discord.Guild,
	m *models.Member,
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
			i18n.TranslateText(
				event.Locale(), "components.level.rank.embed.title",
				map[string]any{
					"user": member.EffectiveName(),
				},
			),
		).
		SetDescription("## "+i18n.TranslateText(event.Locale(), "components.level.rank.embed.description",
			map[string]any{
				"level": m.XP.Level(),
				"xp":    m.XP,
			},
		)).
		SetFields(
			discord.EmbedField{
				Name:   i18n.TranslateText(event.Locale(), "components.level.rank.embed.fields.place", nil),
				Value:  fmt.Sprintf("**#%d**", index+1),
				Inline: builtin.Ptr(true),
			},
			discord.EmbedField{
				Name: i18n.TranslateText(event.Locale(), "components.level.rank.embed.fields.next_level",
					map[string]any{"next_level": m.XP.Level() + 1},
				),
				Value: fmt.Sprintf("`%d`xp / `%d`xp",
					xppoint.RequiredPoint(m.XP.Level())-(xppoint.TotalPoint(m.XP.Level()+1)-uint64(m.XP)),
					xppoint.RequiredPoint(m.XP.Level()),
				),
				Inline: builtin.Ptr(true),
			},
		).
		Build()
}
