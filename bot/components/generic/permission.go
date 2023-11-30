package generic

import (
	"context"
	"slices"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/user"
	"github.com/sabafly/gobot/internal/translate"
)

func PermissionCommandCheck(perm string, perms ...discord.Permissions) PEventHandler[*events.ApplicationCommandInteractionCreate] {
	return func(c *components.Components, event *events.ApplicationCommandInteractionCreate) bool {
		ok := permissionCheck(event, perms, c, perm)
		if ok {
			return true
		}

		noPermissionMessage(event, perm)

		return false
	}
}

func PermissionAutocompleteCheck(perm string, perms ...discord.Permissions) PEventHandler[*events.AutocompleteInteractionCreate] {
	return func(c *components.Components, event *events.AutocompleteInteractionCreate) bool {
		return permissionCheck(event, perms, c, perm)
	}
}

func PermissionComponentCheck(perm string, perms ...discord.Permissions) PEventHandler[*events.ComponentInteractionCreate] {
	return func(c *components.Components, event *events.ComponentInteractionCreate) bool {
		ok := permissionCheck(event, perms, c, perm)
		if ok {
			return true
		}

		noPermissionMessage(event, perm)

		return false

	}
}

func noPermissionMessage(event interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
	Locale() discord.Locale
}, perm string) {
	_ = event.CreateMessage(
		discord.NewMessageBuilder().
			SetEmbeds(
				discord.NewEmbedBuilder().
					SetTitlef("⚠️ %s", translate.Message(event.Locale(), "errors.invalid.permission")).
					SetDescription(translate.Message(event.Locale(), "errors.invalid.permission.description", translate.WithTemplate(map[string]any{"Permission": perm}))).
					SetColor(0xEEE731).
					Build(),
			).
			SetFlags(discord.MessageFlagEphemeral).
			Create(),
	)
}

func permissionCheck(event interface {
	context.Context
	Member() *discord.ResolvedMember
	GuildID() *snowflake.ID
	User() discord.User
	Client() bot.Client
}, perms []discord.Permissions, c *components.Components, perm string) bool {
	if slices.Contains(c.Config().Debug.DebugUsers, event.User().ID) {
		return true
	}

	if event.Member().Permissions.Has(perms...) {
		return true
	}

	if m := c.DB().Guild.Query().
		Where(guild.ID(*event.GuildID())).
		FirstX(event).
		QueryMembers().
		Where(member.HasUserWith(user.ID(event.User().ID))).
		FirstX(event); m != nil && m.Permission.Enabled(perm) {
		return true
	}

	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return false
	}

	roles := []discord.Role{}
	event.Client().Caches().RolesForEach(*event.GuildID(), func(role discord.Role) {
		roles = append(roles, role)
	})
	slices.SortStableFunc(roles, func(a, b discord.Role) int {
		switch {
		case a.Position > b.Position:
			return 1
		case a.Position < b.Position:
			return -1
		}
		return 0
	})
	memberRoleIDs := append(event.Member().RoleIDs, *event.GuildID())
	memberRoles := []discord.Role{}
	for _, role := range roles {
		index := slices.Index(memberRoleIDs, role.ID)
		if index == -1 {
			continue
		}
		memberRoles = append(memberRoles, role)
	}

	ok := false
	for _, r := range memberRoles {
		if g.Permissions[r.ID].Enabled(perm) {
			ok = true
			continue
		}
		if g.Permissions[r.ID].Disabled(perm) {
			ok = false
			continue
		}
	}

	return ok
}
