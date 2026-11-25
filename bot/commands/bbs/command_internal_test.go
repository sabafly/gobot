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
 */

package bbs

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/bot/components"
)

func TestPendingActionTokensAndCombineNameAndMail(t *testing.T) {
	sys := NewSystem(&components.Components{}, nil, nil, nil)
	u1 := snowflake.ID(101)
	u2 := snowflake.ID(202)
	target := uuid.New()

	tok := sys.savePending(u1, pendingOpenReader, target, "latest")
	if tok == "" {
		t.Fatal("expected non-empty token")
	}
	if _, ok := sys.popPending(tok, u2); ok {
		t.Fatal("another user must not consume u1's pending token")
	}
	act, ok := sys.popPending(tok, u1)
	if !ok {
		t.Fatal("expected u1 to consume pending token")
	}
	if act.Kind != pendingOpenReader || act.TargetID != target || act.Page != "latest" {
		t.Fatalf("unexpected pending action: %+v", act)
	}
	if _, ok := sys.popPending(tok, u1); ok {
		t.Fatal("consumed token must be single-use")
	}

	if got := combineNameAndMail("", "sage"); got != "sage" {
		t.Fatalf("expected sage, got %q", got)
	}
	if got := combineNameAndMail("名無し#secret", "SAGE"); got != "!sage 名無し#secret" {
		t.Fatalf("expected !sage prefix before trip, got %q", got)
	}
}
