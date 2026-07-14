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
	"context"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"

	"github.com/sabafly/gobot/internal/discordutil"
)

const (
	RolePanelPlacedTypeReaction   = "reaction"
	RolePanelPlacedTypeButton     = "button"
	RolePanelPlacedTypeSelectMenu = "select_menu"
)

func rolePanelPlace(ctx context.Context, place *models.RolePanelPlaced, locale discord.Locale, client *bot.Client, react bool, c *components.Components) error {
	builder := rpPlacedMessage(place, locale)
	if place.MessageID != nil {
		if _, err := client.Rest.UpdateMessage(place.ChannelID, *place.MessageID, builder.BuildUpdate()); err != nil {
			return err
		}
		if place.Type == RolePanelPlacedTypeReaction && react {
			if err := client.Rest.RemoveAllReactions(place.ChannelID, *place.MessageID); err != nil {
				return err
			}
		}
	} else {
		m, err := client.Rest.CreateMessage(place.ChannelID, builder.BuildCreate())
		if err != nil {
			return err
		}
		place.MessageID = &m.ID
		if err := c.GormDB().Save(place).Error; err != nil {
			return err
		}
	}

	if place.Type == RolePanelPlacedTypeReaction && react {
		for i, r := range place.Roles {
			if r.Emoji == nil {
				r.Emoji = &discord.ComponentEmoji{
					Name: discordutil.Index2Emoji(i),
				}
			}
			if err := client.Rest.AddReaction(place.ChannelID, *place.MessageID, discordutil.FormatComponentEmoji(*r.Emoji)); err != nil {
				return err
			}
		}
	}
	return nil
}

func createPanelPlace(ctx context.Context, c *components.Components, panelID uuid.UUID, channelID snowflake.ID, g *models.Guild) (*models.RolePanelPlaced, error) {

	// Clean up incomplete placements
	c.GormDB().Where("(message_id IS NULL OR type = '') AND guild_id = ?", g.ID).Delete(&models.RolePanelPlaced{})

	var panel models.RolePanel
	if err := c.GormDB().Where("id = ? AND guild_id = ?", panelID, g.ID).First(&panel).Error; err != nil {
		return nil, errors.New("rolepanel not found")
	}

	placed := models.RolePanelPlaced{
		GuildID:     g.ID,
		ChannelID:   channelID,
		RolePanelID: panel.ID,
		Name:        panel.Name,
		Description: panel.Description,
		Roles:       panel.Roles,
		UpdatedAt:   time.Now(),
	}

	if err := c.GormDB().Create(&placed).Error; err != nil {
		return nil, err
	}

	return &placed, nil
}
