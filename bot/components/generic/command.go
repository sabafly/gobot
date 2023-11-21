package generic

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func createErrorMessage(
	err errors.Error,
	event interface {
		CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
		Locale() discord.Locale
	},
) {
	key := "errors.generic.message"
	if em, ok := err.(errors.ErrorWithMessage); ok {
		key = em.Key()
	}
	_ = event.CreateMessage(
		discord.NewMessageBuilder().
			SetEmbeds(
				discord.NewEmbedBuilder().
					SetTitlef("🔥 %s", translate.Message(event.Locale(), key)).
					SetDescriptionf("```%s``````%s``````%s```", err.Error(), err.Stack(), err.File()).
					SetColor(0xff2121).
					Build(),
			).
			SetFlags(discord.MessageFlagEphemeral).
			Create(),
	)
}

func rec(event interface {
	RespondMessage(messageBuilder discord.MessageBuilder, opts ...rest.RequestOpt) error
	Locale() discord.Locale
}) {
	if v := recover(); v != nil {
		_ = errors.ErrorMessage("errors.panic.message", event, errors.WithDescription(fmt.Sprintf("```\nargs=%v stack=%s```", v, string(debug.Stack()))))
		panic(v)
	}
}

// Command

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

// AutoComplete

type AutocompleteHandler EventHandler[*events.AutocompleteInteractionCreate]

func (c AutocompleteHandler) Handler() EventHandler[*events.AutocompleteInteractionCreate] {
	return EventHandler[*events.AutocompleteInteractionCreate](c)
}
func (AutocompleteHandler) PermissionCheck() PEventHandler[*events.AutocompleteInteractionCreate] {
	return nil
}

var _ PermissionAutocompleteHandler = (*AutocompleteHandler)(nil)

type PAutocompleteHandler struct {
	AutocompleteHandler  EventHandler[*events.AutocompleteInteractionCreate]
	PAutocompleteHandler PEventHandler[*events.AutocompleteInteractionCreate]
}

func (c PAutocompleteHandler) Handler() EventHandler[*events.AutocompleteInteractionCreate] {
	return EventHandler[*events.AutocompleteInteractionCreate](c.AutocompleteHandler)
}
func (c PAutocompleteHandler) PermissionCheck() PEventHandler[*events.AutocompleteInteractionCreate] {
	return PEventHandler[*events.AutocompleteInteractionCreate](c.PAutocompleteHandler)
}

var _ PermissionAutocompleteHandler = (*PAutocompleteHandler)(nil)

// Component

type ComponentHandler EventHandler[*events.ComponentInteractionCreate]

func (c ComponentHandler) Handler() EventHandler[*events.ComponentInteractionCreate] {
	return EventHandler[*events.ComponentInteractionCreate](c)
}
func (ComponentHandler) PermissionCheck() PEventHandler[*events.ComponentInteractionCreate] {
	return nil
}

var _ PermissionComponentHandler = (*ComponentHandler)(nil)

type PComponentHandler struct {
	ComponentHandler  EventHandler[*events.ComponentInteractionCreate]
	PComponentHandler PEventHandler[*events.ComponentInteractionCreate]
}

func (c PComponentHandler) Handler() EventHandler[*events.ComponentInteractionCreate] {
	return EventHandler[*events.ComponentInteractionCreate](c.ComponentHandler)
}
func (c PComponentHandler) PermissionCheck() PEventHandler[*events.ComponentInteractionCreate] {
	return PEventHandler[*events.ComponentInteractionCreate](c.PComponentHandler)
}

var _ PermissionComponentHandler = (*PComponentHandler)(nil)

// Modal

type ModalHandler func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error

// Permissions

type PermissionCommandHandler PermissionHandler[*events.ApplicationCommandInteractionCreate]
type PermissionAutocompleteHandler PermissionHandler[*events.AutocompleteInteractionCreate]
type PermissionComponentHandler PermissionHandler[*events.ComponentInteractionCreate]

// Generic Types

type EventHandler[E bot.Event] func(c *components.Components, event E) errors.Error
type PEventHandler[E bot.Event] func(c *components.Components, event E) bool

type PermissionHandler[E bot.Event] interface {
	Handler() EventHandler[E]
	PermissionCheck() PEventHandler[E]
}

var _ components.Command = (*GenericCommand)(nil)

type GenericCommand struct {
	Namespace     string
	Private       bool
	CommandCreate []discord.ApplicationCommandCreate
	// /command/subcommand_group/subcommand
	CommandHandlers      map[string]PermissionCommandHandler
	ComponentHandlers    map[string]PermissionComponentHandler
	ModalHandlers        map[string]ModalHandler
	AutocompleteHandlers map[string]PermissionAutocompleteHandler
	EventHandler         EventHandler[bot.Event]
	db                   *components.Components
}

func (gc *GenericCommand) SetDB(db *components.Components) *GenericCommand {
	gc.db = db
	return gc
}

func (gc *GenericCommand) Name() string                               { return gc.Namespace }
func (gc *GenericCommand) Create() []discord.ApplicationCommandCreate { return gc.CommandCreate }
func (gc *GenericCommand) IsPrivate() bool                            { return gc.Private }
func (gc *GenericCommand) CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		defer rec(event)
		path := event.SlashCommandInteractionData().CommandPath()
		switch event.Data.Type() {
		case discord.ApplicationCommandTypeMessage:
			path = "m" + path
		case discord.ApplicationCommandTypeUser:
			path = "u" + path
		}
		cmd, ok := gc.CommandHandlers[path]
		if !ok {
			return fmt.Errorf("unknown handler: command_path=%s", path)
		}
		if c := cmd.PermissionCheck(); c != nil {
			if !c(gc.db, event) {
				return nil
			}
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: command_path=%s", path)
		}
		if err := h(gc.db, event); err != nil {
			createErrorMessage(err, event)
			return err
		}
		return nil
	}
}

func (gc *GenericCommand) ComponentHandler() func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		defer rec(event)
		customID := strings.Split(event.Data.CustomID(), ":")
		cmd, ok := gc.ComponentHandlers[strings.Join(customID[:2], ":")]
		if !ok {
			return fmt.Errorf("unknown handler: custom_id=%s", event.Data.CustomID())
		}
		if c := cmd.PermissionCheck(); c != nil {
			if !c(gc.db, event) {
				return nil
			}
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: custom_id=%s", event.Data.CustomID())
		}
		if err := h(gc.db, event); err != nil {
			createErrorMessage(err, event)
			return err
		}
		return nil
	}
}

func (gc *GenericCommand) ModalHandler() func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		defer rec(event)
		customID := strings.Split(event.Data.CustomID, ":")
		cmd, ok := gc.ModalHandlers[strings.Join(customID[:2], ":")]
		if !ok {
			return fmt.Errorf("unknown handler: custom_id=%s", event.Data.CustomID)
		}
		if err := cmd(gc.db, event); err != nil {
			createErrorMessage(err, event)
			return err
		}
		return nil
	}
}

func (gc *GenericCommand) AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error {
	return func(event *events.AutocompleteInteractionCreate) error {
		var focused string
		for _, ao := range event.Data.Options {
			if ao.Focused {
				focused = ao.Name
			}
		}
		path := event.Data.CommandPath() + ":" + focused
		cmd, ok := gc.AutocompleteHandlers[path]
		if !ok {
			return fmt.Errorf("unknown handler: command_path=%s", path)
		}
		if c := cmd.PermissionCheck(); c != nil {
			if !c(gc.db, event) {
				return nil
			}
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: command_path=%s", path)
		}
		if err := h(gc.db, event); err != nil {
			return err
		}
		return nil
	}
}

func (gc *GenericCommand) OnEvent() func(event bot.Event) error {
	return func(event bot.Event) error {
		if gc.EventHandler == nil {
			return nil
		}
		if err := gc.EventHandler(gc.db, event); err != nil {
			return err
		}
		return nil
	}
}
