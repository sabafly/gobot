package generic

import (
	"fmt"
	"log/slog"
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
					SetFooterText(err.ID().String()).
					SetColor(0xff2121).
					Build(),
			).
			SetFlags(discord.MessageFlagEphemeral).
			BuildCreate(),
	)
}

func rec(event interface {
	RespondMessage(messageBuilder discord.MessageBuilder, opts ...rest.RequestOpt) error
	Locale() discord.Locale
}) {
	if v := recover(); v != nil {
		_ = errors.ErrorMessage("errors.panic.message", event, errors.WithDescription(fmt.Sprintf("```\nargs=%v stack=%s```", v, string(debug.Stack()))))
		slog.Error("panic", "args", v, "stack", string(debug.Stack()))
		panic(v)
	}
}

// Command

type CommandHandler EventHandler[*events.ApplicationCommandInteractionCreate]

func (c CommandHandler) Handler() EventHandler[*events.ApplicationCommandInteractionCreate] {
	return EventHandler[*events.ApplicationCommandInteractionCreate](c)
}
func (c CommandHandler) Permissions() []Permission              { return nil }
func (c CommandHandler) DiscordPermission() discord.Permissions { return 0 }

var _ PermissionCommandHandler = (*CommandHandler)(nil)

type PCommandHandler struct {
	CommandHandler EventHandler[*events.ApplicationCommandInteractionCreate]
	Permission     []Permission
	DiscordPerm    discord.Permissions
}

func (c PCommandHandler) Handler() EventHandler[*events.ApplicationCommandInteractionCreate] {
	return c.CommandHandler
}
func (c PCommandHandler) Permissions() []Permission              { return c.Permission }
func (c PCommandHandler) DiscordPermission() discord.Permissions { return c.DiscordPerm }

var _ PermissionCommandHandler = (*PCommandHandler)(nil)

// AutoComplete

type AutocompleteHandler EventHandler[*events.AutocompleteInteractionCreate]

func (c AutocompleteHandler) Handler() EventHandler[*events.AutocompleteInteractionCreate] {
	return EventHandler[*events.AutocompleteInteractionCreate](c)
}
func (c AutocompleteHandler) Permissions() []Permission              { return nil }
func (c AutocompleteHandler) DiscordPermission() discord.Permissions { return 0 }

var _ PermissionAutocompleteHandler = (*AutocompleteHandler)(nil)

type PAutocompleteHandler struct {
	AutocompleteHandler EventHandler[*events.AutocompleteInteractionCreate]
	Permission          []Permission
	DiscordPerm         discord.Permissions
}

func (c PAutocompleteHandler) Handler() EventHandler[*events.AutocompleteInteractionCreate] {
	return c.AutocompleteHandler
}
func (c PAutocompleteHandler) Permissions() []Permission              { return c.Permission }
func (c PAutocompleteHandler) DiscordPermission() discord.Permissions { return c.DiscordPerm }

var _ PermissionAutocompleteHandler = (*PAutocompleteHandler)(nil)

// Component

type ComponentHandler EventHandler[*events.ComponentInteractionCreate]

func (c ComponentHandler) Handler() EventHandler[*events.ComponentInteractionCreate] {
	return EventHandler[*events.ComponentInteractionCreate](c)
}
func (c ComponentHandler) Permissions() []Permission              { return nil }
func (c ComponentHandler) DiscordPermission() discord.Permissions { return 0 }

var _ PermissionComponentHandler = (*ComponentHandler)(nil)

type PComponentHandler struct {
	ComponentHandler EventHandler[*events.ComponentInteractionCreate]
	Permission       []Permission
	DiscordPerm      discord.Permissions
}

func (c PComponentHandler) Handler() EventHandler[*events.ComponentInteractionCreate] {
	return c.ComponentHandler
}
func (c PComponentHandler) Permissions() []Permission              { return c.Permission }
func (c PComponentHandler) DiscordPermission() discord.Permissions { return c.DiscordPerm }

var _ PermissionComponentHandler = (*PComponentHandler)(nil)

// Modal

type ModalHandler func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error

// Permissions

type PermissionCommandHandler PermissionHandler[*events.ApplicationCommandInteractionCreate]
type PermissionAutocompleteHandler PermissionHandler[*events.AutocompleteInteractionCreate]
type PermissionComponentHandler PermissionHandler[*events.ComponentInteractionCreate]

// Generic Types

type EventHandler[E bot.Event] func(c *components.Components, event E) errors.Error

type PermissionHandler[E bot.Event] interface {
	Handler() EventHandler[E]
	Permissions() []Permission
	DiscordPermission() discord.Permissions
}

type Permission interface {
	PermString() string
	Default() bool
}

type PermissionString string

func (p PermissionString) PermString() string { return string(p) }
func (p PermissionString) Default() bool      { return false }

type PermissionDefaultString string

func (p PermissionDefaultString) PermString() string { return string(p) }
func (p PermissionDefaultString) Default() bool      { return true }

var _ components.Command = (*Command)(nil)

type Command struct {
	Namespace     string
	Private       bool
	CommandCreate []discord.ApplicationCommandCreate
	// /command/subcommand_group/subcommand
	CommandHandlers      map[string]PermissionCommandHandler
	ComponentHandlers    map[string]PermissionComponentHandler
	ModalHandlers        map[string]ModalHandler
	AutocompleteHandlers map[string]PermissionAutocompleteHandler
	EventHandler         EventHandler[bot.Event]
	Schedulers           []components.Scheduler
	component            *components.Components
}

func (gc *Command) Scheduler() []components.Scheduler { return gc.Schedulers }

func (gc *Command) SetComponent(c *components.Components) *Command {
	gc.component = c
	return gc
}

func (gc *Command) Name() string                               { return gc.Namespace }
func (gc *Command) Create() []discord.ApplicationCommandCreate { return gc.CommandCreate }
func (gc *Command) IsPrivate() bool                            { return gc.Private }
func (gc *Command) CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		defer rec(event)
		var path string
		switch event.Data.Type() {
		case discord.ApplicationCommandTypeSlash:
			path = event.SlashCommandInteractionData().CommandPath()
		case discord.ApplicationCommandTypeMessage:
			path = "m/" + event.MessageCommandInteractionData().CommandName()
		case discord.ApplicationCommandTypeUser:
			path = "u/" + event.UserCommandInteractionData().CommandName()
		}
		cmd, ok := gc.CommandHandlers[path]
		if !ok {
			return fmt.Errorf("unknown handler: command_path=%s", path)
		}
		if c := permissionCheck(event, gc.component, cmd.Permissions(), cmd.DiscordPermission()); !c {
			if err := noPermissionMessage(event, cmd.Permissions()); err != nil {
				createErrorMessage(errors.NewError(err), event)
				return err
			}
			return nil
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: command_path=%s", path)
		}
		if err := h(gc.component, event); err != nil {
			createErrorMessage(err, event)
			return err
		}
		return nil
	}
}

func (gc *Command) ComponentHandler() func(event *events.ComponentInteractionCreate) error {
	return func(event *events.ComponentInteractionCreate) error {
		defer rec(event)
		customID := strings.Split(event.Data.CustomID(), ":")
		cmd, ok := gc.ComponentHandlers[strings.Join(customID[:2], ":")]
		if !ok {
			return fmt.Errorf("unknown handler: custom_id=%s", event.Data.CustomID())
		}
		if c := permissionCheck(event, gc.component, cmd.Permissions(), cmd.DiscordPermission()); !c {
			if err := noPermissionMessage(event, cmd.Permissions()); err != nil {
				createErrorMessage(errors.NewError(err), event)
				return err
			}
			return nil
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: custom_id=%s", event.Data.CustomID())
		}
		if err := h(gc.component, event); err != nil {
			createErrorMessage(err, event)
			return err
		}
		return nil
	}
}

func (gc *Command) ModalHandler() func(event *events.ModalSubmitInteractionCreate) error {
	return func(event *events.ModalSubmitInteractionCreate) error {
		defer rec(event)
		customID := strings.Split(event.Data.CustomID, ":")
		cmd, ok := gc.ModalHandlers[strings.Join(customID[:2], ":")]
		if !ok {
			return fmt.Errorf("unknown handler: custom_id=%s", event.Data.CustomID)
		}
		if err := cmd(gc.component, event); err != nil {
			createErrorMessage(err, event)
			return err
		}
		return nil
	}
}

func (gc *Command) AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error {
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
		if c := permissionCheck(event, gc.component, cmd.Permissions(), cmd.DiscordPermission()); !c {
			if err := event.AutocompleteResult(make([]discord.AutocompleteChoice, 0)); err != nil {
				return err
			}
			return nil
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: command_path=%s", path)
		}
		if err := h(gc.component, event); err != nil {
			return err
		}
		return nil
	}
}

func (gc *Command) OnEvent() func(event bot.Event) error {
	return func(event bot.Event) error {
		if gc.EventHandler == nil {
			return nil
		}
		if err := gc.EventHandler(gc.component, event); err != nil {
			return err
		}
		return nil
	}
}
