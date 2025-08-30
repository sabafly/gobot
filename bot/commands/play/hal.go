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
		data.currentPoint *= 2
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
