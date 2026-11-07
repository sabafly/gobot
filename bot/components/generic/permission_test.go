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

package generic_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
)

func TestRequirePermissions_DebugUser(t *testing.T) {
	conf := components.Config{
		Debug: components.ConfigDebug{
			DebugUsers: []snowflake.ID{snowflake.ID(999)},
		},
	}
	c := components.New(context.Background(), conf, nil)

	var called bool
	h := generic.RequirePermissions(c, 0, generic.PermissionString("admin"))(func(event *handler.InteractionEvent) error {
		called = true
		return nil
	})

	var interaction discord.ApplicationCommandInteraction
	jsonBlob := `{
		"type": 2,
		"data": {
			"type": 1,
			"name": "admin_cmd",
			"id": "123"
		},
		"user": {
			"id": "999",
			"username": "debuguser"
		}
	}`
	if err := json.Unmarshal([]byte(jsonBlob), &interaction); err != nil {
		t.Fatalf("failed to unmarshal interaction: %v", err)
	}

	ie := &handler.InteractionEvent{
		InteractionCreate: &events.InteractionCreate{
			Interaction: interaction,
		},
		Ctx:  context.Background(),
		Vars: make(map[string]string),
	}

	if err := h(ie); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Errorf("expected handler to be called for debug user")
	}
}

func TestRequirePermissions_NoPerms(t *testing.T) {
	c := components.New(context.Background(), components.Config{}, nil)

	var called bool
	h := generic.RequirePermissions(c, 0)(func(event *handler.InteractionEvent) error {
		called = true
		return nil
	})

	var interaction discord.ApplicationCommandInteraction
	jsonBlob := `{
		"type": 2,
		"data": {
			"type": 1,
			"name": "public_cmd",
			"id": "123"
		},
		"user": {
			"id": "111",
			"username": "regularuser"
		}
	}`
	if err := json.Unmarshal([]byte(jsonBlob), &interaction); err != nil {
		t.Fatalf("failed to unmarshal interaction: %v", err)
	}

	ie := &handler.InteractionEvent{
		InteractionCreate: &events.InteractionCreate{
			Interaction: interaction,
		},
		Ctx:  context.Background(),
		Vars: make(map[string]string),
	}

	if err := h(ie); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Errorf("expected handler to be called when no perms required")
	}
}
