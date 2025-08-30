package play

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

func Command(c *components.Components) components.Command {
	return (&generic.Command{
		Namespace: "play",
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "play",
				Description: "Play some games with your friends!",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "high-and-low",
						Description:              "Play a high and low card game",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.high-and-low.description"),
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/play/high-and-low": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					const cost = 25

					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if point < cost {
						if err := errors.ErrorMessage("error.play.high-and-low.insufficient_points", event); err != nil {
							return errors.NewError(err)
						}
						return nil
					}
					if err = gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -cost); err != nil {
						return errors.NewError(err)
					}
					data := HALData{
						id:           uuid.New(),
						userID:       event.User().ID,
						currentPoint: 3,
					}
					data.Roll()

					hal_values.Set(data.id, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(data, event.Locale(), false)...)); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"play:hal_high": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err := HALPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					success, equal := HALPlay(data, HALChoiceHigh)
					if !success && !equal {
						return HALFinish(c, *data, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, *data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale(), equal)...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_low": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err := HALPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					success, equal := HALPlay(data, HALChoiceLow)
					if !success && !equal {
						return HALFinish(c, *data, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, *data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale(), equal)...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_cancel": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err := HALPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					return HALFinish(c, *data, event.User().ID, *event.GuildID(), event)
				},
			},
		},
	}).SetComponent(c)
}
