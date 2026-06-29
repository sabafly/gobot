package play

import (
	"math"
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
	hal_values = database.NewMemoryValues[uuid.UUID, *HALData](time.Minute * 10)
)

type HALData struct {
	id           uuid.UUID
	userID       snowflake.ID
	currentPoint float64
	previousCard Card
	currentCard  Card
	multiplier   float64
	lastChoice   HALChoice
	lastResult   HALChoice
	turn         int
}

func (h HALData) OnDelete() error {
	return nil
}

func (h *HALData) Roll() HALChoice {
	prev := h.currentCard
	h.previousCard = h.currentCard
	if h.currentCard == CardNone {
		h.currentCard = RandomCardWithoutJoker()
		return HALChoiceNone
	}
	h.currentCard = RandomCard()

	if h.currentCard.Number() > prev.Number() {
		return HALChoiceHigh
	} else if h.currentCard.Number() < prev.Number() {
		return HALChoiceLow
	}
	return HALChoiceSame
}

type HALChoice uint8

const (
	HALChoiceNone HALChoice = iota
	HALChoiceHigh
	HALChoiceLow
	HALChoiceSame
)

func (c HALChoice) String() string {
	switch c {
	case HALChoiceHigh:
		return "HIGH"
	case HALChoiceLow:
		return "LOW"
	case HALChoiceSame:
		return "SAME"
	default:
		return "NONE"
	}
}

func HALPrecondition(event *events.ComponentInteractionCreate) (*HALData, errors.Error) {
	args := strings.Split(event.Data.CustomID(), ":")
	id := uuid.MustParse(args[2])
	data, ok := hal_values.Get(id)
	if !ok {
		if err := errors.ErrorMessage("error.play.high-and-low.expired", event); err != nil {
			return nil, errors.NewError(err)
		}
		return nil, nil
	}
	if data.userID != event.User().ID {
		if err := errors.ErrorMessage("error.play.high-and-low.not_your_game", event); err != nil {
			return nil, errors.NewError(err)
		}
		return nil, nil
	}
	return data, nil
}

func HALPlay(data *HALData, choice HALChoice) (success, equal bool) {
	data.lastChoice = choice
	data.lastResult = HALChoiceNone // Initialize lastResult
	data.turn++
	result := data.Roll()
	data.lastResult = result // Update lastResult
	if data.currentCard.IsJoker() {
		data.currentPoint = 0
		return false, false
	}
	if result == HALChoiceSame {
		if choice != HALChoiceSame {
			return false, true
		}
		data.multiplier += data.multiplier * (float64(data.currentCard.Number()))
		if data.previousCard.Suit() == data.currentCard.Suit() {
			data.currentPoint += data.currentPoint * (float64(data.currentCard.Number()) * 1.5)
		}
		return true, true
	}
	if result == choice {
		data.currentPoint = data.currentPoint * data.multiplier
		return true, false
	} else {
		data.currentPoint = 0
		return false, false
	}
}

func HALFinish(c *components.Components, data HALData, userID, guildID snowflake.ID, event *events.ComponentInteractionCreate) errors.Error {
	hal_values.Delete(data.id)
	if err := gopoint.AddPoint(c, userID, guildID, int64(math.Floor(data.currentPoint))); err != nil {
		return errors.NewError(err)
	}

	if err := event.UpdateMessage(discord.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(i18n.BuildContext().
			WithText("point", strconv.FormatInt(int64(math.Floor(data.currentPoint)), 10)).
			WithText("current_card", data.currentCard.String()).
			WithText("last_card", data.previousCard.String()).
			WithText("turn", strconv.Itoa(data.turn)).
			WithText("last_choice", data.lastChoice.String()).
			WithText("last_result", data.lastResult.String()).
			WithText("multiplier", strconv.FormatFloat(data.multiplier, 'f', 2, 64)).
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
	previousCardText := "N/A"
	if data.previousCard != CardNone {
		previousCardText = data.previousCard.String()
	}
	return ctx.
		WithText("point", strconv.FormatInt(int64(math.Floor(data.currentPoint)), 10)).
		WithText("current_card", data.currentCard.String()).
		WithText("last_card", previousCardText).
		WithText("turn", strconv.Itoa(data.turn)).
		WithText("last_choice", data.lastChoice.String()).
		WithText("last_result", data.lastResult.String()).
		WithText("multiplier", strconv.FormatFloat(data.multiplier, 'f', 2, 64)).
		WithDisabled("play:hal_same:{uuid}", data.lastResult != HALChoiceSame).
		WithCustomID("uuid", data.id.String()).
		Translate(i18n.TranslateLayout(locale, "command.play.high-and-low.game"))
}
