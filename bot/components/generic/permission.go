package generic

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/internal/translate"
)

func PermissionCommandCheck(perm string, perms ...discord.Permissions) PEventHandler[*events.ApplicationCommandInteractionCreate] {
	return func(c *components.Components, event *events.ApplicationCommandInteractionCreate) bool {
		shouldReturn, returnValue := permissionCheck(event, perms, c, perm)
		if shouldReturn {
			return returnValue
		}

		noPermissionMessage(event, perm)

		return false
	}
}

func PermissionAutocompleteCheck(perm string, perms ...discord.Permissions) PEventHandler[*events.AutocompleteInteractionCreate] {
	return func(c *components.Components, event *events.AutocompleteInteractionCreate) bool {
		shouldReturn, returnValue := permissionCheck(event, perms, c, perm)
		if shouldReturn {
			return returnValue
		}

		return false
	}
}

func PermissionComponentCheck(perm string, perms ...discord.Permissions) PEventHandler[*events.ComponentInteractionCreate] {
	return func(c *components.Components, event *events.ComponentInteractionCreate) bool {
		shouldReturn, returnValue := permissionCheck(event, perms, c, perm)
		if shouldReturn {
			return returnValue
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
}, perms []discord.Permissions, c *components.Components, perm string) (bool, bool) {
	if event.Member().Permissions.Has(perms...) {
		return true, true
	}

	if g := c.DB().Guild.Query().Where(guild.ID(*event.GuildID())).FirstX(event).QueryMembers().Where(member.UserID(event.User().ID)).FirstX(event); g != nil && g.Permission.Has(perm) {
		return true, true
	}
	return false, false
}
