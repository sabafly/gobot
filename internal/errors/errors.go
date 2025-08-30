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

package errors

import (
	"errors"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/sabafly/gobot/internal/i18n"
	"github.com/sabafly/gobot/internal/translate"
)

var (
	As     = errors.As
	Is     = errors.Is
	Join   = errors.Join
	New    = errors.New
	Unwrap = errors.Unwrap
)

type (
	Config struct {
		Description string
	}

	Option func(*Config)
)

var DefaultConfig = Config{}

func (c *Config) options(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

func WithDescription(s string) Option {
	return func(c *Config) {
		c.Description = s
	}
}

func ErrorMessage(
	key string,
	event interface {
		RespondMessage(messageBuilder discord.MessageBuilder, opts ...rest.RequestOpt) error
		Locale() discord.Locale
	},
	opts ...Option,
) error {
	cfg := Config{}
	cfg.options(opts...)

	var desc string
	if cfg.Description != "" {
		desc = cfg.Description
	} else {
		d, err := translate.Localize(event.Locale(), key+".description", nil, 0)
		if err == nil {
			desc = d
		} else {
			desc = i18n.TranslateText(event.Locale(), key+".description")
		}
	}

	title := translate.Message(event.Locale(), key)
	if title == "" || title == key {
		title = i18n.TranslateText(event.Locale(), key)
	}

	return event.RespondMessage(
		discord.NewMessageBuilder().
			SetEphemeral(true).
			SetIsComponentsV2(true).
			SetComponents(
				discord.ContainerComponent{
					Components: []discord.ContainerSubComponent{
						discord.NewTextDisplayf("## ❗ %s", title),
						discord.NewTextDisplayf("%s", desc),
					},
					AccentColor: 0xff2121,
				},
			),
	)
}
