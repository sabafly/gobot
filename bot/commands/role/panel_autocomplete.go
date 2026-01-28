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

package role

import (
	"fmt"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
)

func panelAutocomplete(c *components.Components, event *events.AutocompleteInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}

	var panels []models.RolePanel
	if err := c.GormDB().Where("guild_id = ? AND name LIKE ?", g.ID, "%"+event.Data.String("panel")+"%").Find(&panels).Error; err != nil {
		return errors.NewError(err)
	}

	choices := make([]discord.AutocompleteChoice, len(panels))
	for i, p := range panels {
		choices[i] = discord.AutocompleteChoiceString{
			Name:  builtin.Or(slices.ContainsFunc(panels, func(rp models.RolePanel) bool { return rp.ID != p.ID && rp.Name == p.Name }), fmt.Sprintf("%s (%s)", p.Name, p.ID), p.Name),
			Value: p.ID.String(),
		}
	}
	if err := event.AutocompleteResult(choices); err != nil {
		return errors.NewError(err)
	}
	return nil
}
