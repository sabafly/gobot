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

package discordutil

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
)

type MessageResponder interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
	UpdateInteractionResponse(messageUpdate discord.MessageUpdate, opts ...rest.RequestOpt) (*discord.Message, error)
}

func RespondMessage(event any, builder MessageBuilder, opts ...rest.RequestOpt) error {
	switch e := event.(type) {
	case *handler.InteractionEvent:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.UpdateInteractionResponse(builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	case *handler.CommandEvent:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.UpdateInteractionResponse(builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	case *handler.ComponentEvent:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.UpdateInteractionResponse(builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	case *handler.ModalEvent:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.UpdateInteractionResponse(builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	case *events.ApplicationCommandInteractionCreate:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.Client().Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(), builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	case *events.ComponentInteractionCreate:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.Client().Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(), builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	case *events.ModalSubmitInteractionCreate:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.Client().Rest.UpdateInteractionResponse(e.ApplicationID(), e.Token(), builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	case MessageResponder:
		err := e.CreateMessage(builder.BuildCreate(), opts...)
		if err != nil {
			_, errUpdate := e.UpdateInteractionResponse(builder.BuildUpdate(), opts...)
			return errUpdate
		}
		return nil
	default:
		return fmt.Errorf("unsupported event type for RespondMessage: %T", event)
	}
}

func UpdateMessage(event any, message any, opts ...rest.RequestOpt) error {
	var update discord.MessageUpdate
	switch m := message.(type) {
	case MessageBuilder:
		update = m.BuildUpdate()
	case discord.MessageUpdate:
		update = m
	default:
		return fmt.Errorf("unsupported message type for UpdateMessage: %T", message)
	}

	switch e := event.(type) {
	case *handler.InteractionEvent:
		return e.UpdateMessage(update, opts...)
	case *handler.ComponentEvent:
		return e.UpdateMessage(update, opts...)
	case *handler.ModalEvent:
		return e.UpdateMessage(update, opts...)
	case *events.ComponentInteractionCreate:
		return e.UpdateMessage(update, opts...)
	case *events.ModalSubmitInteractionCreate:
		return e.UpdateMessage(update, opts...)
	default:
		return fmt.Errorf("unsupported event type for UpdateMessage: %T", event)
	}
}
