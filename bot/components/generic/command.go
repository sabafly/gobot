/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package generic

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
)

var PrintDebugInfo = false

func CreateErrorMessage(
	err errors.Error,
	event interface {
		Locale() discord.Locale
	},
) {
	key := "errors.generic.message"
	if em, ok := err.(errors.ErrorWithMessage); ok {
		key = em.Key()
	}
	builder := discordutil.NewMessageBuilder().
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplayf("🔥 %s", i18n.TranslateText(event.Locale(), key)),
				discord.NewTextDisplayf("```%s``````%s``````%s```", err.Error(), err.Stack(), err.File()),
				discord.NewTextDisplayf("-# %s", err.ID().String()),
			).WithAccentColor(0xff2121),
		).
		AddFlags(discord.MessageFlagEphemeral)
	if PrintDebugInfo && strings.TrimSpace(err.DebugInfo()) != "" {
		builder.AddComponents(discord.NewTextDisplayf("**[DEBUG]**\n```%s```", err.DebugInfo()))
	}
	if err := discordutil.RespondMessage(event, builder); err != nil {
		slog.Error("failed to respond to message", "error", err)
	}
}

func createErrorMessage(
	err errors.Error,
	event interface {
		Locale() discord.Locale
	},
) {
	CreateErrorMessage(err, event)
}

func rec(event interface {
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

func (gc *Command) getCommand(event *events.ApplicationCommandInteractionCreate) (PermissionCommandHandler, error) {
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
		return nil, fmt.Errorf("unknown handler: command_path=%s", path)
	}
	return cmd, nil
}

func (gc *Command) HasCommand(cmd *events.ApplicationCommandInteractionCreate) bool {
	_, err := gc.getCommand(cmd)
	return err == nil
}
func (gc *Command) CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		defer rec(event)
		cmd, err := gc.getCommand(event)
		if err != nil {
			return err
		}
		if c := permissionCheck(gc.component.Ctx(), event, gc.component, cmd.Permissions(), cmd.DiscordPermission()); !c {
			if err := noPermissionMessage(event, cmd.Permissions()); err != nil {
				createErrorMessage(errors.NewError(err), event)
				return err
			}
			return nil
		}
		h := cmd.Handler()
		if h == nil {
			return fmt.Errorf("nil handler: command=%v", event)
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
		if c := permissionCheck(gc.component.Ctx(), event, gc.component, cmd.Permissions(), cmd.DiscordPermission()); !c {
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
		if c := permissionCheck(gc.component.Ctx(), event, gc.component, cmd.Permissions(), cmd.DiscordPermission()); !c {
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

func (gc *Command) RegisterRoutes(r handler.Router) {
	// 1. CommandHandlers
	for pattern, cmd := range gc.CommandHandlers {
		p := pattern
		c := cmd
		var mw []handler.Middleware
		if len(c.Permissions()) > 0 || c.DiscordPermission() != 0 {
			mw = append(mw, RequirePermissions(gc.component, c.DiscordPermission(), c.Permissions()...))
		}
		router := r
		if len(mw) > 0 {
			router = r.With(mw...)
		}

		if after, ok := strings.CutPrefix(p, "m/"); ok {
			name := after
			router.MessageCommand("/"+name, func(data discord.MessageCommandInteractionData, e *handler.CommandEvent) error {
				if err := c.Handler()(gc.component, e.ApplicationCommandInteractionCreate); err != nil {
					CreateErrorMessage(err, e)
					return err
				}
				return nil
			})
		} else if after, ok := strings.CutPrefix(p, "u/"); ok {
			name := after
			router.UserCommand("/"+name, func(data discord.UserCommandInteractionData, e *handler.CommandEvent) error {
				if err := c.Handler()(gc.component, e.ApplicationCommandInteractionCreate); err != nil {
					CreateErrorMessage(err, e)
					return err
				}
				return nil
			})
		} else {
			cleanPattern := p
			if !strings.HasPrefix(cleanPattern, "/") {
				cleanPattern = "/" + cleanPattern
			}
			router.SlashCommand(cleanPattern, func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
				if err := c.Handler()(gc.component, e.ApplicationCommandInteractionCreate); err != nil {
					CreateErrorMessage(err, e)
					return err
				}
				return nil
			})
		}
	}

	// 2. ComponentHandlers
	for pattern, cmd := range gc.ComponentHandlers {
		p := pattern
		c := cmd
		cleanPattern := p
		if !strings.HasPrefix(cleanPattern, "/") {
			cleanPattern = "/" + strings.ReplaceAll(cleanPattern, ":", "/")
		}
		var mw []handler.Middleware
		if len(c.Permissions()) > 0 || c.DiscordPermission() != 0 {
			mw = append(mw, RequirePermissions(gc.component, c.DiscordPermission(), c.Permissions()...))
		}
		router := r
		if len(mw) > 0 {
			router = r.With(mw...)
		}
		router.Component(cleanPattern, func(e *handler.ComponentEvent) error {
			if err := c.Handler()(gc.component, e.ComponentInteractionCreate); err != nil {
				CreateErrorMessage(err, e)
				return err
			}
			return nil
		})
	}

	// 3. ModalHandlers
	for pattern, handlerFunc := range gc.ModalHandlers {
		p := pattern
		h := handlerFunc
		cleanPattern := p
		if !strings.HasPrefix(cleanPattern, "/") {
			cleanPattern = "/" + strings.ReplaceAll(cleanPattern, ":", "/")
		}
		r.Modal(cleanPattern, func(e *handler.ModalEvent) error {
			if err := h(gc.component, e.ModalSubmitInteractionCreate); err != nil {
				CreateErrorMessage(err, e)
				return err
			}
			return nil
		})
	}

	// 4. AutocompleteHandlers
	for pattern, cmd := range gc.AutocompleteHandlers {
		p := pattern
		c := cmd
		var commandPath, focusedOption string
		if before, after, ok := strings.Cut(p, ":"); ok {
			commandPath = before
			focusedOption = after
		} else {
			commandPath = p
		}
		if !strings.HasPrefix(commandPath, "/") {
			commandPath = "/" + commandPath
		}
		var mw []handler.Middleware
		if len(c.Permissions()) > 0 || c.DiscordPermission() != 0 {
			mw = append(mw, RequirePermissions(gc.component, c.DiscordPermission(), c.Permissions()...))
		}
		router := r
		if len(mw) > 0 {
			router = r.With(mw...)
		}
		router.Autocomplete(commandPath, func(e *handler.AutocompleteEvent) error {
			if focusedOption != "" {
				var focused string
				for _, ao := range e.Data.Options {
					if ao.Focused {
						focused = ao.Name
						break
					}
				}
				if focused != focusedOption {
					return nil
				}
			}
			if err := c.Handler()(gc.component, e.AutocompleteInteractionCreate); err != nil {
				return err
			}
			return nil
		})
	}
}
