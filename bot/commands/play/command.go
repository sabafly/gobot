package play

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

var (
	hal_values = database.NewMemoryValues[uuid.UUID, HALData](time.Minute * 10)
)

type HALData struct {
	id            uuid.UUID
	userID        snowflake.ID
	currentPoint  int64
	currentNumber int
	lastChoice    HALChoice
	turn          int
}

func (h HALData) OnDelete() error {
	return nil
}

func (h *HALData) Roll() HALChoice {
	prev := h.currentNumber
	h.currentNumber = 1 + rand.N(10)

	if h.currentNumber > prev {
		return HALChoiceHigh
	} else if h.currentNumber < prev {
		return HALChoiceLow
	}
	return HALChoiceNone
}

type HALChoice uint8

const (
	HALChoiceNone HALChoice = iota
	HALChoiceHigh
	HALChoiceLow
)

func (c HALChoice) String() string {
	switch c {
	case HALChoiceHigh:
		return "HIGH"
	case HALChoiceLow:
		return "LOW"
	default:
		return "NONE"
	}
}

func HALPrecondition(event *events.ComponentInteractionCreate) (*HALData, errors.Error) {
	args := strings.Split(event.Data.CustomID(), ":")
	id := uuid.MustParse(args[2])
	v, ok := hal_values.Get(id)
	if !ok {
		if err := errors.ErrorMessage("error.play.high-and-low.expired", event); err != nil {
			return nil, errors.NewError(err)
		}
		return nil, nil
	}
	if v.userID != event.User().ID {
		if err := errors.ErrorMessage("error.play.high-and-low.not_your_game", event); err != nil {
			return nil, errors.NewError(err)
		}
		return nil, nil
	}
	return &v, nil
}

func HALPlay(data *HALData, choice HALChoice) (success, equal bool) {
	data.lastChoice = choice
	data.turn++
	result := data.Roll()
	if result == HALChoiceNone {
		return false, true
	}
	if result == choice {
		// Handle the correct choice
		return true, false
	} else {
		data.currentPoint = 0
		return false, false
	}
}

func HALFinish(c *components.Components, data HALData, userID, guildID snowflake.ID, event *events.ComponentInteractionCreate) errors.Error {
	hal_values.Delete(data.id)
	if err := gopoint.AddPoint(c, userID, guildID, data.currentPoint); err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(i18n.BuildContext().
			WithText("point", strconv.FormatInt(data.currentPoint, 10)).
			WithText("current_card", strconv.Itoa(data.currentNumber)).
			WithText("turn", strconv.Itoa(data.turn)).
			WithText("last_choice", data.lastChoice.String()).
			Translate(i18n.TranslateLayout(event.Locale(), "command.play.high-and-low.result"))...,
		).BuildUpdate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func HALMessage(data HALData, locale discord.Locale, equal bool) []discord.LayoutComponent {
	ctx := i18n.BuildContext()
	if equal {
		ctx.WithText("message", i18n.TranslateText(locale, "command.play.high-and-low.equal-retry"))
	} else {
		ctx.WithText("message", i18n.TranslateText(locale, "command.play.high-and-low.guess-next"))
	}
	return ctx.
		WithText("point", strconv.FormatInt(data.currentPoint, 10)).
		WithText("current_card", strconv.Itoa(data.currentNumber)).
		WithText("turn", strconv.Itoa(data.turn)).
		WithText("last_choice", data.lastChoice.String()).
		WithCustomID("uuid", data.id.String()).
		Translate(i18n.TranslateLayout(locale, "command.play.high-and-low.game"))
}

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
					if success {
						data.currentPoint *= 2
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
					if success {
						data.currentPoint *= 2
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
