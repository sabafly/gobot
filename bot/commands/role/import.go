package role

import (
	"regexp"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent/schema"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/emoji"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func ImportCommand(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "import-rolepanel",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.MessageCommandCreate{
				Name:              "import-rolepanel",
				NameLocalizations: translate.MessageMap("components.role.panel.import.name", false),
				DMPermission:      builtin.Ptr(false),
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"m/import-rolepanel": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionString("role.panel.import"),
				},
				DiscordPerm: discord.PermissionManageRoles,
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if !check(event) {
						return errors.NewError(errors.ErrorMessage("errors.unsupported", event))
					}
					message := event.MessageCommandInteractionData().TargetMessage()
					lines := strings.Split(message.Embeds[0].Description, "\n")
					var roles []schema.Role
					roleCount := 0
					for _, v := range lines {
						if !roleRegexp.MatchString(v) {
							continue
						}
						var emojis []string
						if !emoji.MatchString(v) {
							emojis = append(emojis, discordutil.Number2Emoji(roleCount+1))
						} else {
							emojis = emoji.FindAllString(v)
						}
						componentEmoji := discordutil.ParseComponentEmoji(emojis[0])
						if _, ok := event.Client().Caches().Emoji(*event.GuildID(), componentEmoji.ID); !ok && componentEmoji.ID != 0 {
							componentEmoji = discordutil.ParseComponentEmoji(discordutil.Number2Emoji(roleCount + 1))
						}
						roleID, err := snowflake.Parse(roleIDRegexp.FindString(roleRegexp.FindString(v)))
						if err != nil {
							continue
						}
						role, ok := event.Client().Caches().Role(*event.GuildID(), roleID)
						if !ok {
							rolePtr, err := event.Client().Rest().GetRole(*event.GuildID(), roleID)
							if err != nil {
								continue
							}
							role = *rolePtr
						}
						roleCount++
						roles = append(roles, schema.Role{
							ID:    role.ID,
							Name:  role.Name,
							Emoji: &componentEmoji,
						})
					}
					if len(roles) < 1 {
						return errors.NewError(errors.ErrorMessage("errors.unsupported", event))
					}

					g, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					panel := c.DB().RolePanel.Create().
						SetName(builtin.Or(message.Embeds[0].Title != "", message.Embeds[0].Title, translate.Message(event.Locale(), "components.role.panel.default_name"))).
						SetDescription("").
						SetRoles(roles).
						SetGuild(g).
						SaveX(event)

					place := c.DB().RolePanelPlaced.Create().
						SetGuild(g).
						SetChannelID(event.Channel().ID()).
						SetRolePanel(panel).
						SetName(panel.Name).
						SetDescription(panel.Description).
						SetRoles(panel.Roles).
						SetUpdatedAt(time.Now()).
						SaveX(event)

					if err := event.CreateMessage(
						rpPlaceBaseMenu(place, event.Locale()).
							SetFlags(discord.MessageFlagEphemeral).
							BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}

					return nil
				},
			},
		},
	}).SetComponent(c)
}

var roleRegexp = regexp.MustCompile("<@&([0-9]{18,20})>")
var roleIDRegexp = regexp.MustCompile("[0-9]{18,20}")

func check(event *events.ApplicationCommandInteractionCreate) bool {
	message := event.MessageCommandInteractionData().TargetMessage()
	var wid snowflake.ID
	if message.WebhookID != nil {
		wh, err := event.Client().Rest().GetWebhook(*message.WebhookID)
		if err != nil {
			return false
		} else if wh.Type() == discord.WebhookTypeIncoming && wh.(discord.IncomingWebhook).User.ID == 716496407212589087 {
			wid = message.Author.ID
		}
	}
	switch message.Author.ID {
	case 895912135039803402, 1138119538190340146, 1137367652482957313, 971523089550671953 /*役職パネルv3*/, 682774762837377045 /*役職パネルv2*/, 917780792032251904 /*役職ボット*/, 669817785932578826 /*陽菜*/, 716496407212589087, wid /*RT*/, 718760319207473152 /*SevenBot*/, 832614051514417202 /*Glow-bot*/ :
		if len(message.Embeds) < 1 {
			return false
		}
		lines := strings.Split(message.Embeds[0].Description, "\r")
		validLines := 0
		for _, v := range lines {
			if !roleRegexp.MatchString(v) {
				continue
			}
			validLines++
		}
		return validLines > 0
	default:
		return false
	}
}
