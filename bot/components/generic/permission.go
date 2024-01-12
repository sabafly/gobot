package generic

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/user"
	"github.com/sabafly/gobot/internal/translate"
)

func noPermissionMessage(event interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
	Locale() discord.Locale
}, perms []Permission) error {
	var permStr string
	for _, p := range perms {
		permStr += fmt.Sprintf("`%s` ", p.PermString())
	}
	return event.CreateMessage(
		discord.NewMessageBuilder().
			SetEmbeds(
				discord.NewEmbedBuilder().
					SetTitlef("⚠️ %s", translate.Message(event.Locale(), "errors.invalid.permission")).
					SetDescription(translate.Message(event.Locale(), "errors.invalid.permission.description",
						translate.WithTemplate(map[string]any{"Permission": permStr}),
					)).
					SetColor(0xEEE731).
					Build(),
			).
			SetFlags(discord.MessageFlagEphemeral).
			Create(),
	)
}

func PermissionCheck(ctx context.Context, c *components.Components, g *ent.Guild, client bot.Client, m discord.ResolvedMember, guildID snowflake.ID, perms []Permission) bool {

	if len(perms) == 0 {
		return true
	}

	if m := c.DB().Guild.Query().
		Where(guild.ID(guildID)).
		FirstX(ctx).
		QueryMembers().
		Where(member.HasUserWith(user.ID(m.User.ID))).
		FirstX(ctx); m != nil {
		for _, p := range perms {
			var r bool
			if p.Default() {
				if m.Permission.Disabled(p.PermString()) {
					return false
				} else {
					r = true
				}
			} else {
				if m.Permission.Enabled(p.PermString()) {
					r = true
				} else if m.Permission.Disabled(p.PermString()) {
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

func RolePermissionCheck(g *ent.Guild, guildID snowflake.ID, client bot.Client, roleIds []snowflake.ID, perms []Permission) bool {

	if len(perms) == 0 {
		return true
	}

	var roles []discord.Role
	client.Caches().RolesForEach(guildID, func(role discord.Role) {
		roles = append(roles, role)
	})
	slices.SortStableFunc(roles, func(a, b discord.Role) int {
		return cmp.Compare(a.Position, b.Position)
	})
	var memberRoles []discord.Role
	for _, role := range roles {
		if !slices.Contains(roleIds, role.ID) {
			continue
		}
		memberRoles = append(memberRoles, role)
	}

	ok := false
	for _, p := range perms {
		for _, r := range memberRoles {
			l := g.Permissions[r.ID]
			if l.Enabled(p.PermString()) {
				ok = true
			} else if l.Disabled(p.PermString()) {
				ok = false
			}
		}
	}

	return ok
}

func permissionCheck(event interface {
	context.Context
	Member() *discord.ResolvedMember
	GuildID() *snowflake.ID
	User() discord.User
	Client() bot.Client
}, c *components.Components, perms []Permission, dPerm discord.Permissions) bool {

	if len(perms) == 0 {
		return true
	}

	if slices.Contains(c.Config().Debug.DebugUsers, event.User().ID) {
		return true
	}

	if dPerm != 0 && event.Member().Permissions.Has(dPerm) {
		if m := c.DB().Guild.Query().
			Where(guild.ID(*event.GuildID())).
			FirstX(event).
			QueryMembers().
			Where(member.HasUserWith(user.ID(event.User().ID))).
			FirstX(event); m != nil {
			for _, p := range perms {
				if m.Permission.Disabled(p.PermString()) {
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
