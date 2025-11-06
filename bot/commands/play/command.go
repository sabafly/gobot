package play

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

func ptr[T any](v T) *T {
	return &v
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
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionInt{
								Name:                     "extra_bet",
								NameLocalizations:        i18n.TranslateCommandOptionMap("command.play.high-and-low.option.extra_bet.name"),
								Description:              "Place an extra bet to increase your winnings",
								DescriptionLocalizations: i18n.TranslateTextMap("command.play.high-and-low.option.extra_bet.description"),
								MinValue:                 ptr(1),
								MaxValue:                 ptr(10),
								Required:                 false,
							},
						},
					},
					discord.ApplicationCommandOptionSubCommand{
						Name:                     "slot",
						Description:              "Play a slot machine game",
						DescriptionLocalizations: i18n.TranslateTextMap("command.play.slot.description"),
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
					cost := int64(15)

					// Lookup table for power of 10 costs (more efficient than math.Pow10 for integer calculations)
					costTable := []int64{0, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000, 10000000000, 100000000000}
					extra := int64(event.SlashCommandInteractionData().Int("extra_bet"))
					if extra > 0 && int(extra+1) < len(costTable) {
						cost += costTable[extra+1]
					}
					extraMultiplier := int64(0)
					if extra > 3 {
						extraMultiplier = extra - 3
					}

					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if point < cost {
						if err := errors.ErrorMessage("error.play.high-and-low.insufficient_points", event,
							errors.WithMapContext(i18n.BuildContext().WithText("point", strconv.FormatInt(cost, 10)))); err != nil {
							return errors.NewError(err)
						}
						return nil
					}
					if err = gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -cost); err != nil {
						return errors.NewError(err)
					}
					data := &HALData{
						id:           uuid.New(),
						userID:       event.User().ID,
						currentPoint: 3 + extra,
						multiplier:   2 + extraMultiplier,
					}
					data.Roll()

					hal_values.Set(data.id, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale(), false)...)); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"/play/slot": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.slot"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					const cost = 8

					if err := event.DeferCreateMessage(false); err != nil {
						return errors.NewError(err)
					}
					userPoint, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					if userPoint < cost {
						if err := errors.ErrorMessage("error.play.slot.insufficient_points", event,
							errors.WithMapContext(i18n.BuildContext().WithText("point", strconv.FormatInt(cost, 10)))); err != nil {
							return errors.NewError(err)
						}
						return nil
					}
					if err = gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -cost); err != nil {
						return errors.NewError(err)
					}

					role := []string{"NONE", "GRAPES", "WATERMELON", "CHERRIES", "LEMON", "ORANGE", "PLUM", "BELL", "BAR", "SEVEN"}
					roleWeights := []int{1200, 120, 100, 30, 15, 10, 8, 5, 1, 1}
					rolePoints := []int64{0, 8, 10, 12, 15, 20, 25, 100, 500, 1000}
					totalWeight := 0
					for _, w := range roleWeights {
						totalWeight += w
					}

					getRole := func() (string, int64) {
						r := rand.N(totalWeight)
						accumulatedWeight := 0
						for i, w := range roleWeights {
							accumulatedWeight += w
							if r < accumulatedWeight {
								return role[i], rolePoints[i]
							}
						}
						return role[len(role)-1], rolePoints[len(role)-1]
					}
					genSlot := func(role string) string {
						switch role {
						case "GRAPES":
							return "🍇"
						case "WATERMELON":
							return "🍉"
						case "CHERRIES":
							return "🍒"
						case "LEMON":
							return "🍋"
						case "ORANGE":
							return "🍊"
						case "PLUM":
							return "🍑"
						case "BELL":
							return "🔔"
						case "BAR":
							return "💵"
						case "SEVEN":
							return "7️⃣"
						default:
							return ""
						}
					}

					selectedRole, point := getRole()
					result := genSlot(selectedRole)
					if result == "" {
						// ja: 全ての絵柄
						// en: All symbols
						symbols := []string{"🍇", "🍉", "🍒", "🍋", "🍊", "🍑", "🔔", "💵", "7️⃣"}
						symbols = append(symbols[:0], symbols...)
						rand.Shuffle(len(symbols), func(i, j int) {
							symbols[i], symbols[j] = symbols[j], symbols[i]
						})
						slotResults := []string{symbols[0], symbols[1], symbols[2]}
						result = strings.Join(slotResults, " | ")
					} else {
						result += " | " + result + " | " + result
					}

					if point > 0 {
						if err = gopoint.AddPoint(c, event.User().ID, *event.GuildID(), point); err != nil {
							return errors.NewError(err)
						}
					}

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetContentf("Slots: %s\nResult Point: %d", result, point),
					); err != nil {
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
					hal_values.Set(data.id, data)
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
					hal_values.Set(data.id, data)
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
