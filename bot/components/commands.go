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

package components

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
)

var (
	DefaultCommands []Command
)

func (c *Components) AddCommands(cmds ...Command) {
	for _, cmd := range cmds {
		c.commandsRegistry[cmd.Name()] = cmd
	}
}

func (c *Components) AddCommand(cmd Command) {
	c.commandsRegistry[cmd.Name()] = cmd
}

func NormalizeCustomID(customID string) string {
	if strings.HasPrefix(customID, "/") {
		return customID
	}
	return "/" + strings.ReplaceAll(customID, ":", "/")
}

func recoveryMiddleware() handler.Middleware {
	return func(next handler.Handler) handler.Handler {
		return func(event *handler.InteractionEvent) (err error) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("パニックが発生しました", "panic", r)
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			return next(event)
		}
	}
}

func (c *Components) Initialize(client *bot.Client) error {
	for _, cmd := range DefaultCommands {
		c.AddCommand(cmd)
	}

	c.router = handler.New()
	c.router.DefaultContext(func() context.Context {
		return c.Ctx()
	})
	c.router.Use(recoveryMiddleware())
	c.router.Error(func(event *handler.InteractionEvent, err error) {
		slog.Error("インタラクションエラー", "err", err)
	})

	var commands, privateCommands []discord.ApplicationCommandCreate
	for name, cmd := range c.commandsRegistry {
		if len(cmd.Scheduler()) > 0 {
			slog.Info("コマンドスケジューラ―を登録します", "name", name, "count", len(cmd.Create()))
			for _, s := range cmd.Scheduler() {
				go execSchedule(c, client, s)
			}
		}
		if len(cmd.Create()) > 0 {
			slog.Info("コマンドを登録します", "name", name, "count", len(cmd.Create()), "is_private", cmd.IsPrivate())
			if cmd.IsPrivate() {
				privateCommands = append(privateCommands, cmd.Create()...)
			} else {
				commands = append(commands, cmd.Create()...)
			}
		}
		cmd.RegisterRoutes(c.router)
	}

	if _, err := client.Rest.SetGlobalCommands(client.ApplicationID, commands); err != nil {
		slog.Error("コマンドの登録に失敗", slog.Any("err", err), slog.String("body", string(err.(*rest.Error).RsBody)))
		return err
	}

	for _, id := range c.Config().Debug.DebugGuilds {
		if _, err := client.Rest.SetGuildCommands(client.ApplicationID, id, privateCommands); err != nil {
			slog.Error("プライベートコマンドの登録に失敗", "err", err, "guild", id)
			return err
		}
	}

	client.EventManager.AddEventListeners(
		bot.NewListenerFunc(c.OnEvent()),
		&events.ListenerAdapter{
			OnGuildReady:  c.OnGuildReady(),
			OnGuildJoin:   c.OnGuildJoin(),
			OnGuildLeave:  c.OnGuildLeave(),
			OnGuildUpdate: c.OnGuildUpdate(),
			OnUserUpdate:  c.OnUserUpdate(),
		},
	)
	return nil
}

func (c *Components) OnEvent() func(bot bot.Event) {
	return func(event bot.Event) {
		switch e := event.(type) {
		case *events.InteractionCreate:
			c.dispatchInteraction(e)
		default:
			for name, cmd := range c.commandsRegistry {
				if h := cmd.OnEvent(); h != nil {
					if err := h(event); err != nil {
						slog.Error("イベント処理中にエラーが発生しました", "err", err, "cmd_name", name)
					}
				}
			}
		}
	}
}

func (c *Components) dispatchInteraction(e *events.InteractionCreate) {
	var path string
	switch i := e.Interaction.(type) {
	case discord.ApplicationCommandInteraction:
		if sci, ok := i.Data.(discord.SlashCommandInteractionData); ok {
			path = sci.CommandPath()
		} else {
			path = "/" + i.Data.CommandName()
		}
	case discord.AutocompleteInteraction:
		path = i.Data.CommandPath()
	case discord.ComponentInteraction:
		path = NormalizeCustomID(i.Data.CustomID())
	case discord.ModalSubmitInteraction:
		path = NormalizeCustomID(i.Data.CustomID)
	}

	if path == "" {
		return
	}

	ctx := c.Ctx()
	ic := *e
	ie := &handler.InteractionEvent{
		InteractionCreate: &ic,
		Ctx:               ctx,
		Vars:              make(map[string]string),
	}
	ic.Respond = func(responseType discord.InteractionResponseType, data discord.InteractionResponseData, opts ...rest.RequestOpt) error {
		return e.Respond(responseType, data, append([]rest.RequestOpt{rest.WithCtx(ie.Ctx)}, opts...)...)
	}
	if err := c.router.Handle(path, ie); err != nil {
		slog.Error("インタラクション処理中にエラーが発生しました", "err", err, "path", path)
	}
}

type Command interface {
	Name() string

	Create() []discord.ApplicationCommandCreate
	IsPrivate() bool
	RegisterRoutes(r handler.Router)
	CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error
	ComponentHandler() func(event *events.ComponentInteractionCreate) error
	ModalHandler() func(event *events.ModalSubmitInteractionCreate) error
	AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error
	OnEvent() func(event bot.Event) error
	Scheduler() []Scheduler
	HasCommand(cmd *events.ApplicationCommandInteractionCreate) bool
}
