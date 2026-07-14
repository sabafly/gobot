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

package generic

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/translate"
)

func noPermissionMessage(event interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
	Locale() discord.Locale
}, perms []Permission) error {
	var permStr strings.Builder
	for _, p := range perms {
		fmt.Fprintf(&permStr, "`%s` ", p.PermString())
	}
	return event.CreateMessage(
		discord.NewMessageBuilder().
			SetEmbeds(
				discord.NewEmbedBuilder().
					SetTitlef("⚠️ %s", translate.Message(event.Locale(), "errors.invalid.permission")).
					SetDescription(translate.Message(event.Locale(), "errors.invalid.permission.description",
						translate.WithTemplate(map[string]any{"Permission": permStr.String()}),
					)).
					SetColor(0xEEE731).
					Build(),
			).
			SetFlags(discord.MessageFlagEphemeral).
			BuildCreate(),
	)
}

func PermissionCheck(ctx context.Context, c *components.Components, g *models.Guild, client *bot.Client, m discord.ResolvedMember, guildID snowflake.ID, perms []Permission) bool {

	if len(perms) == 0 {
		return true
	}

	var member models.Member
	if err := c.GormDB().Where("guild_id = ? AND user_id = ?", guildID, m.User.ID).First(&member).Error; err == nil {
		for _, p := range perms {
			var r bool
			if p.Default() {
				if member.Permission.Disabled(p.PermString()) {
					return false
				} else {
					r = true
				}
			} else {
				if member.Permission.Enabled(p.PermString()) {
					r = true
				} else if member.Permission.Disabled(p.PermString()) {
					return false
				}
			}
			if r {
				return r
			}
		}
	}

	m.RoleIDs = append(m.RoleIDs, guildID)

	return RolePermissionCheck(g, guildID, client, m.RoleIDs, perms)
}

func RolePermissionCheck(g *models.Guild, guildID snowflake.ID, client *bot.Client, roleIds []snowflake.ID, perms []Permission) bool {
	if len(perms) == 0 {
		return true
	}

	var roles []discord.Role
	for role := range client.Caches.Roles(guildID) {
		roles = append(roles, role)
	}
	slices.SortStableFunc(roles, func(a, b discord.Role) int {
		return a.Compare(b)
	})
	var memberRoles []discord.Role
	for _, role := range roles {
		if !slices.Contains(roleIds, role.ID) {
			continue
		}
		memberRoles = append(memberRoles, role)
	}

	for _, p := range perms {
		var r bool
		hasExplicit := false
		for i := len(memberRoles) - 1; i >= 0; i-- {
			role := memberRoles[i]
			l := g.Permissions[role.ID]
			if l.Enabled(p.PermString()) {
				r = true
				hasExplicit = true
				break
			} else if l.Disabled(p.PermString()) {
				r = false
				hasExplicit = true
				break
			}
		}

		if !hasExplicit {
			if p.Default() {
				r = true
			}
		}

		if r {
			return true
		}
	}

	return false
}

func permissionCheck(event interface {
	context.Context
	Member() *discord.ResolvedMember
	GuildID() *snowflake.ID
	User() discord.User
	Client() *bot.Client
}, c *components.Components, perms []Permission, dPerm discord.Permissions) bool {

	if len(perms) == 0 {
		return true
	}

	if slices.Contains(c.Config().Debug.DebugUsers, event.User().ID) {
		return true
	}

	if dPerm != 0 && event.Member().Permissions.Has(dPerm) {
		var member models.Member
		if err := c.GormDB().Where("guild_id = ? AND user_id = ?", *event.GuildID(), event.User().ID).First(&member).Error; err == nil {
			for _, p := range perms {
				if member.Permission.Disabled(p.PermString()) {
					return false
				}
			}
		}
		return true
	}

	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		slog.Warn("failed to GuildCreateID", "id", event.GuildID())
		return false
	}

	return PermissionCheck(event, c, g, event.Client(), *event.Member(), *event.GuildID(), perms)
}
