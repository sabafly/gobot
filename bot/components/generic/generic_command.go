package generic

import (
	"fmt"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/bot/components"
)

type CommandHandler EventHandler[*events.ApplicationCommandInteractionCreate]

func (c CommandHandler) Handler() EventHandler[*events.ApplicationCommandInteractionCreate] {
	return EventHandler[*events.ApplicationCommandInteractionCreate](c)
}
func (CommandHandler) PermissionCheck() PEventHandler[*events.ApplicationCommandInteractionCreate] {
	return nil
}

var _ PermissionCommandHandler = (*CommandHandler)(nil)

type PCommandHandler struct {
	CommandHandler  EventHandler[*events.ApplicationCommandInteractionCreate]
	PCommandHandler PEventHandler[*events.ApplicationCommandInteractionCreate]
}

func (c PCommandHandler) Handler() EventHandler[*events.ApplicationCommandInteractionCreate] {
	return EventHandler[*events.ApplicationCommandInteractionCreate](c.CommandHandler)
}
func (c PCommandHandler) PermissionCheck() PEventHandler[*events.ApplicationCommandInteractionCreate] {
	return PEventHandler[*events.ApplicationCommandInteractionCreate](c.PCommandHandler)
}

var _ PermissionCommandHandler = (*PCommandHandler)(nil)

type ComponentHandler func(component *components.Components, event *events.ComponentInteractionCreate) error
type ModalHandler func(component *components.Components, event *events.ModalSubmitInteractionCreate) error
type AutocompleteHandler = func(component *components.Components, event *events.AutocompleteInteractionCreate) error

type PermissionCommandHandler PermissionHandler[*events.ApplicationCommandInteractionCreate]

type EventHandler[E bot.Event] func(*components.Components, E) error
type PEventHandler[E bot.Event] func(*components.Components, E) bool

type PermissionHandler[E bot.Event] interface {
	Handler() EventHandler[E]
	PermissionCheck() PEventHandler[E]
}

var _ components.Command = (*GenericCommand)(nil)

type GenericCommand struct {
	Namespace     string
	CommandCreate []discord.ApplicationCommandCreate
	// /command/subcommand_group/subcommand
	CommandHandlers      map[string]PermissionCommandHandler
	ComponentHandlers    map[string]ComponentHandler
	ModalHandlers        map[string]ModalHandler
	AutocompleteHandlers map[string]AutocompleteHandler
	EventHandler         EventHandler[bot.Event]
	db                   *components.Components
}

func (gc *GenericCommand) SetDB(db *components.Components) *GenericCommand {
	gc.db = db
	return gc
}

func (gc *GenericCommand) Name() string                               { return gc.Namespace }
func (gc *GenericCommand) Create() []discord.ApplicationCommandCreate { return gc.CommandCreate }

func (gc *GenericCommand) CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		cmd, ok := gc.CommandHandlers[event.SlashCommandInteractionData().CommandPath()]
		if !ok {
			return fmt.Errorf("unknown handler: command_path=%s", event.SlashCommandInteractionData().CommandPath())
		}
		if c := cmd.PermissionCheck(); c != nil {
			if !c(gc.db, event) {
				return nil
			}
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: command_path=%s", event.SlashCommandInteractionData().CommandPath())
		}
		return h(gc.db, event)
	}
}

func (gc *GenericCommand) ComponentHandler() func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		cmd, ok := gc.ComponentHandlers[event.Data.CustomID()]
		if !ok {
			return fmt.Errorf("unknown handler: custom_id=%s", event.Data.CustomID())
		}
		return cmd(gc.db, event)
	}
}

func (gc *GenericCommand) ModalHandler() func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		cmd, ok := gc.ModalHandlers[event.Data.CustomID]
		if !ok {
			return fmt.Errorf("unknown handler: custom_id=%s", event.Data.CustomID)
		}
		return cmd(gc.db, event)
	}
}

func (gc *GenericCommand) AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error {
	return func(event *events.AutocompleteInteractionCreate) error {
		cmd, ok := gc.AutocompleteHandlers[event.Data.CommandPath()]
		if !ok {
			return fmt.Errorf("unknown handler: command_path=%s", event.Data.CommandPath())
		}
		return cmd(gc.db, event)
	}
}

func (gc *GenericCommand) OnEvent() func(event bot.Event) error {
	return func(event bot.Event) error {
		return gc.EventHandler(gc.db, event)
	}
}
