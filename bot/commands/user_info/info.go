package userinfo

import (
	"fmt"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
)

func Command(c *components.Components) components.Command {
	return (&generic.GenericCommand{
		Namespace: "userinfo",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.UserCommandCreate{
				Name:         "userinfo",
				DMPermission: builtin.Ptr(false),
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"u/userinfo": generic.CommandHandler(func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
				var roleString string
				{ // ロールを取得する
					roles, err := event.Client().Rest().GetRoles(*event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					slices.SortStableFunc(roles, func(a, b discord.Role) int {
						switch {
						case a.Position > b.Position:
							return -1
						case a.Position < b.Position:
							return 1
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
					for i, r := range memberRoles {
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
											Name:  "role",
											Value: roleString,
										},
									).
									Build(),
							),
						).
						Create(),
				); err != nil {
					return errors.NewError(err)
				}
				return nil
			}),
		},
	}).SetComponent(c)
}
