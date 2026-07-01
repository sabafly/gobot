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

func NewHALData(userID snowflake.ID) *HALData {
	return &HALData{
		id:               uuid.New(),
		userID:           userID,
		started:          false,
		startOptionIndex: 0,
		maxTurns:         HALStartOptions[0].MaxTurns,
		currentPoint:     float64(HALStartOptions[0].StartPoint),
		multiplier:       HALStartOptions[0].Multiplier,
	}
}

type HALData struct {
	id     uuid.UUID
	userID snowflake.ID

	startOptionIndex int
	cost             int64
	maxTurns         int
	started          bool

	currentPoint float64
	previousCard Card
	currentCard  Card
	multiplier   float64
	lastChoice   HALResult
	lastResult   HALResult
	turn         int
}

func (h *HALData) SetStartOption(index int, option HALStartOption) {
	if h.started {
		return
	}
	h.startOptionIndex = index
	h.maxTurns = option.MaxTurns
	h.currentPoint = option.StartPoint
	h.multiplier = option.Multiplier
	h.cost = option.Cost
}

func (h *HALData) Start() {
	if h.started {
		return
	}
	h.started = true
	h.Roll()
}

func (h HALData) OnDelete() error {
	return nil
}

func (h *HALData) Roll() HALResult {
	prev := h.currentCard
	h.previousCard = h.currentCard
	if h.currentCard == CardNone {
		h.currentCard = RandomCardWithoutJoker()
		return HALResultNone
	}
	h.currentCard = RandomCard()

	if h.currentCard.Number() > prev.Number() {
		return HALResultHigh
	} else if h.currentCard.Number() < prev.Number() {
		return HALResultLow
	}
	return HALResultSame
}

type HALResult uint8

const (
	HALResultNone HALResult = iota
	HALResultHigh
	HALResultLow
	HALResultSame
	HALResultJoker
)

func (c HALResult) String() string {
	switch c {
	case HALResultHigh:
		return "HIGH"
	case HALResultLow:
		return "LOW"
	case HALResultSame:
		return "SAME"
	case HALResultJoker:
		return "JOKER"
	default:
		return "NONE"
	}
}

type HALFinishState uint8

const (
	HALFinishStateNone HALFinishState = iota
	HALFinishStateWin
	HALFinishStateLose
	HALFinishStatePayout
	HALFinishStateJoker
)

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

func HALPlay(data *HALData, choice HALResult) (finishState HALFinishState) {
	data.lastChoice = choice
	data.lastResult = HALResultNone // Initialize lastResult
	data.turn++
	result := data.Roll()
	data.lastResult = result // Update lastResult
	if data.currentCard.IsJoker() {
		data.lastResult = HALResultJoker
		finishState = HALFinishStateJoker
		return
	}
	defer func() {
		if data.turn >= data.maxTurns && finishState == HALFinishStateNone {
			finishState = HALFinishStateWin
		}
	}()
	if result == HALResultSame {
		if choice != HALResultSame {
			finishState = HALFinishStateNone
			return
		}
		data.multiplier += float64(data.currentCard.Number())
		if data.previousCard.Suit() == data.currentCard.Suit() {
			data.currentPoint *= float64(data.currentCard.Number()) * 1.5
		}
		data.maxTurns = min(data.turn+5, data.maxTurns)
		finishState = HALFinishStateNone
		return
	}
	if result == choice {
		data.currentPoint += data.currentPoint * data.multiplier
		finishState = HALFinishStateNone
		return
	} else {
		data.currentPoint = 0
		finishState = HALFinishStateLose
		return
	}
}

func HALFinish(c *components.Components, data HALData, finishState HALFinishState, userID, guildID snowflake.ID, event *events.ComponentInteractionCreate) errors.Error {
	if err := gopoint.AddPoint(c, userID, guildID, int64(math.Floor(data.currentPoint))); err != nil {
		return errors.NewError(err)
	}
	hal_values.Delete(data.id)

	var finalStateStr string
	switch finishState {
	case HALFinishStateWin:
		finalStateStr = i18n.TranslateText(event.Locale(), "command.play.high-and-low.finish.win")
	case HALFinishStateLose:
		finalStateStr = i18n.TranslateText(event.Locale(), "command.play.high-and-low.finish.lose")
	case HALFinishStatePayout:
		finalStateStr = i18n.TranslateText(event.Locale(), "command.play.high-and-low.finish.payout")
	case HALFinishStateJoker:
		finalStateStr = i18n.TranslateText(event.Locale(), "command.play.high-and-low.finish.joker")
	default:
		finalStateStr = i18n.TranslateText(event.Locale(), "command.play.high-and-low.finish.unknown")
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
			WithText("multiplier", strconv.FormatFloat(data.multiplier*100, 'f', 0, 64)).
			WithText("final_state", finalStateStr).
			WithCustomID("option_index", strconv.Itoa(data.startOptionIndex)).
			Translate(i18n.TranslateLayout(event.Locale(), "command.play.high-and-low.result"))...,
		).BuildUpdate()); err != nil {
		return errors.NewError(err)
	}
	return nil
}

var HALStartOptions = []HALStartOption{
	{Cost: 15, StartPoint: 1, Multiplier: 0.5, MaxTurns: 99},
	{Cost: 30, StartPoint: 2, Multiplier: 0.8, MaxTurns: 27},
	{Cost: 50, StartPoint: 3, Multiplier: 0.8, MaxTurns: 27},
	{Cost: 80, StartPoint: 3, Multiplier: 1.0, MaxTurns: 28},
	{Cost: 150, StartPoint: 5, Multiplier: 1.0, MaxTurns: 28},
	{Cost: 10000, StartPoint: 3, Multiplier: 2.0, MaxTurns: 30},
	{Cost: 1000000, StartPoint: 4, Multiplier: 3.0, MaxTurns: 30},
	{Cost: 1000000000, StartPoint: 5, Multiplier: 4.0, MaxTurns: 30},
	{Cost: 1000000000000, StartPoint: 6, Multiplier: 5.0, MaxTurns: 30},
}

type HALStartOption struct {
	Cost       int64
	StartPoint float64
	Multiplier float64
	MaxTurns   int
}

func HALStartMessage(data HALData, locale discord.Locale, gopoint int64, selectedOptionIndex int) []discord.LayoutComponent {
	var startOptions []discord.StringSelectMenuOption
	for i, option := range HALStartOptions {
		startOptions = append(startOptions, discord.StringSelectMenuOption{
			Label:       i18n.TranslateText(locale, "command.play.high-and-low.start_option_entry.label", option),
			Description: i18n.TranslateText(locale, "command.play.high-and-low.start_option_entry.description", option),
			Value:       strconv.Itoa(i),
			Default:     i == selectedOptionIndex,
		})
	}

	return i18n.BuildContext().
		WithText("gopoint", strconv.FormatInt(gopoint, 10)).
		WithText("selected_option", i18n.TranslateText(locale, "command.play.high-and-low.start_option_entry.selected", HALStartOptions[selectedOptionIndex])).
		WithStringOptions("hal_start_options", startOptions).
		WithCustomID("uuid", data.id.String()).
		Translate(i18n.TranslateLayout(locale, "command.play.high-and-low.start"))
}

func HALMessage(data HALData, locale discord.Locale) []discord.LayoutComponent {
	ctx := i18n.BuildContext()
	if data.lastResult == HALResultSame {
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
		WithText("multiplier", strconv.FormatFloat(data.multiplier*100, 'f', 0, 64)).
		WithDisabled("play:hal_same:{uuid}", data.lastResult != HALResultSame).
		WithCustomID("uuid", data.id.String()).
		Translate(i18n.TranslateLayout(locale, "command.play.high-and-low.game"))
}
