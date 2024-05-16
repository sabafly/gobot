package userinfo

import (
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
	"slices"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "userinfo",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.UserCommandCreate{
				Name:              "userinfo",
				NameLocalizations: translate.MessageMap("components.user.info.name", false),
				DMPermission:      builtin.Ptr(false),
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"u/userinfo": generic.CommandHandler(func(_ *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				var roleString string
				{ // ロールを取得する
					event.Member().RoleIDs = append(event.Member().RoleIDs, *event.GuildID())
					roles := event.Client().Caches().MemberRoles(event.Member().Member)
					slices.SortStableFunc(roles, func(a, b discord.Role) int {
						return a.Compare(b)
					})
					for i, r := range roles {
						roleString += fmt.Sprintf("%d %s\n", i+1, discord.RoleMention(r.ID))
					}
				}
				if err := event.CreateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							embeds.SetEmbedProperties(
								discord.NewEmbedBuilder().
									SetFields(
										discord.EmbedField{
											Name:  translate.Message(event.Locale(), "userinfo.roles"),
											Value: roleString,
										},
									).
									Build(),
							),
						).
						BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
		},
	}).SetComponent(c)
}
