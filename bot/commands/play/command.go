package play

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

var (
	slotLocks   = make(map[uuid.UUID]*sync.Mutex)
	slotLocksMu sync.Mutex
)

func getSlotLock(id uuid.UUID) *sync.Mutex {
	slotLocksMu.Lock()
	defer slotLocksMu.Unlock()
	lock, ok := slotLocks[id]
	if !ok {
		lock = &sync.Mutex{}
		slotLocks[id] = lock
	}
	return lock
}

func deleteSlotLock(id uuid.UUID) {
	slotLocksMu.Lock()
	defer slotLocksMu.Unlock()
	delete(slotLocks, id)
}

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
					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					data := NewHALData(event.User().ID)

					hal_values.Set(data.id, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALStartMessage(*data, event.Locale(), point, data.startOptionIndex)...)); err != nil {
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
					if err := event.DeferCreateMessage(false); err != nil {
						return errors.NewError(err)
					}

					data := &SlotData{
						ID:      uuid.New(),
						UserID:  event.User().ID,
						GuildID: *event.GuildID(),
						Status:  SlotStatusNormal,
						LastReels: [3][3]string{
							{"🍇", "🔔", "🍒"},
							{"🔄", "7️⃣", "🍇"},
							{"🤡", "⬛", "🔄"},
						},
						GogoLit:     false,
						LastSpinWin: 0,
						TotalSpins:  0,
						TotalSpent:  0,
						TotalWon:    0,
					}

					reason, err := SlotPlay(c, data)
					if err != nil {
						return err
					}
					if reason == "insufficient_points" {
						const cost = 3
						if err := errors.ErrorMessage("error.play.slot.insufficient_points", event,
							errors.WithMapContext(i18n.BuildContext().WithText("point", strconv.FormatInt(cost, 10)))); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					slot_values.Set(data.ID, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(SlotMessage(c, data, event.Locale())...)); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"play:hal_select_opt": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err1 := HALPrecondition(event)
					if err1 != nil {
						return err1
					}
					if data == nil {
						return nil
					}
					gopoint, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					selectedOption := 0
					if data := event.StringSelectMenuInteractionData(); len(data.Values) > 0 {
						selected, err := strconv.Atoi(data.Values[0])
						if err != nil {
							return errors.NewError(err)
						}
						if selected < 0 || selected >= len(HALStartOptions) {
							return errors.NewError(fmt.Errorf("invalid selected option"))
						}
						selectedOption = selected
					}

					data.SetStartOption(selectedOption, HALStartOptions[selectedOption])

					hal_values.Set(data.id, data)

					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALStartMessage(*data, event.Locale(), gopoint, selectedOption)...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_start": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					data, err1 := HALPrecondition(event)
					if err1 != nil {
						return err1
					}
					if data == nil {
						return nil
					}

					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					if point < data.cost {
						if err := errors.ErrorMessage("error.play.high-and-low.insufficient_points", event,
							errors.WithMapContext(i18n.BuildContext().WithText("point", strconv.FormatInt(data.cost, 10)))); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					if err := gopoint.AddPoint(c, event.User().ID, *event.GuildID(), -int64(data.cost)); err != nil {
						return errors.NewError(err)
					}

					data.Start()

					hal_values.Set(data.id, data)

					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
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
					finish := HALPlay(data, HALResultHigh)
					if finish != HALFinishStateNone {
						return HALFinish(c, *data, finish, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
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
					finish := HALPlay(data, HALResultLow)
					if finish != HALFinishStateNone {
						return HALFinish(c, *data, finish, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_same": generic.PComponentHandler{
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
					finish := HALPlay(data, HALResultSame)
					if finish != HALFinishStateNone {
						return HALFinish(c, *data, finish, event.User().ID, *event.GuildID(), event)
					}
					hal_values.Set(data.id, data)
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALMessage(*data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:hal_retry": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.high-and-low"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					if len(args) < 3 {
						return errors.NewError(fmt.Errorf("invalid custom ID"))
					}
					optionIndex, err := strconv.Atoi(args[2])
					if err != nil {
						return errors.NewError(err)
					}
					if optionIndex < 0 || optionIndex >= len(HALStartOptions) {
						optionIndex = 0 // default to first option if invalid
					}

					point, _, err := gopoint.GetPoint(c, event.User().ID, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}

					data := NewHALData(event.User().ID)
					data.SetStartOption(optionIndex, HALStartOptions[optionIndex])

					hal_values.Set(data.id, data)

					if err := event.RespondMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(HALStartMessage(*data, event.Locale(), point, data.startOptionIndex)...)); err != nil {
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
					return HALFinish(c, *data, HALFinishStatePayout, event.User().ID, *event.GuildID(), event)
				},
			},
			"play:slot_spin": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.slot"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					if len(args) < 3 {
						return errors.NewError(fmt.Errorf("invalid custom ID"))
					}
					id, errParse := uuid.Parse(args[2])
					if errParse != nil {
						return errors.NewError(errParse)
					}
					mu := getSlotLock(id)
					mu.Lock()
					defer mu.Unlock()

					data, err := SlotPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}

					reason, err := SlotPlay(c, data)
					if err != nil {
						return err
					}
					if reason == "insufficient_points" {
						var cost int64 = 3
						if data.Status == SlotStatusBB || data.Status == SlotStatusRB {
							cost = 1
						}
						if err := event.CreateMessage(discord.NewMessageCreateBuilder().
							SetContent(i18n.BuildContext().
								WithText("point", strconv.FormatInt(cost, 10)).
								ReplaceText(i18n.TranslateText(event.Locale(), "error.play.slot.insufficient_points.description"))).
							SetFlags(discord.MessageFlagEphemeral).
							Build()); err != nil {
							return errors.NewError(err)
						}
						return nil
					}

					slot_values.Set(data.ID, data)

					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetIsComponentsV2(true).
						SetComponents(SlotMessage(c, data, event.Locale())...).
						BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"play:slot_quit": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("play.slot"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					if len(args) < 3 {
						return errors.NewError(fmt.Errorf("invalid custom ID"))
					}
					id, errParse := uuid.Parse(args[2])
					if errParse != nil {
						return errors.NewError(errParse)
					}
					mu := getSlotLock(id)
					mu.Lock()
					defer mu.Unlock()

					data, err := SlotPrecondition(event)
					if err != nil {
						return err
					}
					if data == nil {
						return nil
					}
					defer deleteSlotLock(id)
					return SlotFinish(c, data, event)
				},
			},
		},
	}).SetComponent(c)
}
