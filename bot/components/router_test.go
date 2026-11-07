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

package components_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"

	"github.com/sabafly/gobot/bot/components"
)

func TestNormalizeCustomID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"role:panel_edit_component:12345", "/role/panel_edit_component/12345"},
		{"play:fx_order_modal:abc-def", "/play/fx_order_modal/abc-def"},
		{"/role/panel_edit_component/12345", "/role/panel_edit_component/12345"},
		{"ping", "/ping"},
		{"currency:wallet:user:6789", "/currency/wallet/user/6789"},
	}

	for _, tc := range tests {
		got := components.NormalizeCustomID(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeCustomID(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRouterCustomIDDispatch(t *testing.T) {
	r := handler.New()

	var called bool
	var capturedID string
	var capturedCtx context.Context

	type ctxKey string
	const testKey ctxKey = "test_key"
	expectedCtx := context.WithValue(context.Background(), testKey, "test_value")

	r.Route("/role", func(sub handler.Router) {
		sub.Component("/panel_edit/{id}", func(e *handler.ComponentEvent) error {
			called = true
			capturedID = e.Vars["id"]
			capturedCtx = e.Ctx
			return nil
		})
	})

	// Simulate incoming ComponentInteraction with legacy colon custom_id
	customID := "role:panel_edit:999"
	normalizedPath := components.NormalizeCustomID(customID)

	var interaction discord.ComponentInteraction
	jsonBlob := `{
		"type": 3,
		"data": {
			"custom_id": "` + customID + `",
			"component_type": 2
		}
	}`
	if err := json.Unmarshal([]byte(jsonBlob), &interaction); err != nil {
		t.Fatalf("failed to unmarshal test component interaction: %v", err)
	}

	ie := &handler.InteractionEvent{
		InteractionCreate: &events.InteractionCreate{
			Interaction: interaction,
		},
		Ctx:  expectedCtx,
		Vars: make(map[string]string),
	}

	err := r.Handle(normalizedPath, ie)
	if err != nil {
		t.Fatalf("unexpected error handling route: %v", err)
	}

	if !called {
		t.Errorf("expected handler to be called, but was not")
	}
	if capturedID != "999" {
		t.Errorf("expected capturedID to be '999', got %q", capturedID)
	}
	if capturedCtx.Value(testKey) != "test_value" {
		t.Errorf("expected context to propagate with value 'test_value', got %v", capturedCtx.Value(testKey))
	}
}

func TestRouterSlashCommandDispatch(t *testing.T) {
	r := handler.New()

	var called bool
	var capturedUser snowflake.ID

	r.SlashCommand("/ping", func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		called = true
		capturedUser = e.User().ID
		return nil
	})

	var interaction discord.ApplicationCommandInteraction
	jsonBlob := `{
		"type": 2,
		"data": {
			"type": 1,
			"name": "ping",
			"id": "12345"
		},
		"user": {
			"id": "123456789",
			"username": "testuser"
		}
	}`
	if err := json.Unmarshal([]byte(jsonBlob), &interaction); err != nil {
		t.Fatalf("failed to unmarshal test slash interaction: %v", err)
	}

	ie := &handler.InteractionEvent{
		InteractionCreate: &events.InteractionCreate{
			Interaction: interaction,
		},
		Ctx:  context.Background(),
		Vars: make(map[string]string),
	}

	err := r.Handle("/ping", ie)
	if err != nil {
		t.Fatalf("unexpected error handling slash command: %v", err)
	}

	if !called {
		t.Errorf("expected slash command handler to be called")
	}
	if capturedUser != snowflake.ID(123456789) {
		t.Errorf("expected capturedUser to be 123456789, got %v", capturedUser)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	r := handler.New()
	r.Use(func(next handler.Handler) handler.Handler {
		return func(event *handler.InteractionEvent) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					err = errors.New("recovered from panic")
				}
			}()
			return next(event)
		}
	})

	r.Component("/panic", func(e *handler.ComponentEvent) error {
		panic("something went wrong")
	})

	var interaction discord.ComponentInteraction
	jsonBlob := `{
		"type": 3,
		"data": {
			"custom_id": "/panic",
			"component_type": 2
		}
	}`
	if err := json.Unmarshal([]byte(jsonBlob), &interaction); err != nil {
		t.Fatalf("failed to unmarshal test panic component interaction: %v", err)
	}

	ie := &handler.InteractionEvent{
		InteractionCreate: &events.InteractionCreate{
			Interaction: interaction,
		},
		Ctx:  context.Background(),
		Vars: make(map[string]string),
	}

	err := r.Handle("/panic", ie)
	if err == nil {
		t.Fatalf("expected error from recovered panic, got nil")
	}
	if err.Error() != "recovered from panic" {
		t.Errorf("expected error message 'recovered from panic', got %q", err.Error())
	}
}
