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

package bbs_test

import (
	"strings"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"

	"github.com/sabafly/gobot/internal/bbs"
)

func TestGenerateSlip_BaseNicknameOverrideAndEvents(t *testing.T) {
	idGen := bbs.NewIdentityGenerator([]byte("test-secret-key-32-bytes-long!!!"))
	boardID := uuid.New()
	userID := snowflake.ID(123456789012345678)
	sc := bbs.SlipContext{
		BaseNicknameOverride: "ｶﾞｲﾌﾞ",
		Username:             "alice",
		GlobalName:           "Alice",
	}

	normalDay := time.Date(2026, 6, 15, 12, 0, 0, 0, bbs.JSTLocation)
	if got := idGen.GenerateSlip(bbs.SLIPVVV, boardID, bbs.ScopeGlobal, 0, userID, sc, normalDay); got != "ｶﾞｲﾌﾞ" {
		t.Fatalf("expected SLIPVVV override 'ｶﾞｲﾌﾞ', got %q", got)
	}
	vvv5 := idGen.GenerateSlip(bbs.SLIPVVVVV, boardID, bbs.ScopeGlobal, 0, userID, sc, normalDay)
	if !strings.HasPrefix(vvv5, "ｶﾞｲﾌﾞ ") || len(strings.Fields(vvv5)) != 2 {
		t.Fatalf("expected SLIPVVVVV 'ｶﾞｲﾌﾞ MMMM-XXXX', got %q", vvv5)
	}

	// Built-in append event (Christmas 12-25) appends to override base.
	xmas := time.Date(2026, 12, 25, 12, 0, 0, 0, bbs.JSTLocation)
	if got := idGen.GenerateSlip(bbs.SLIPVVV, boardID, bbs.ScopeGlobal, 0, userID, sc, xmas); got != "ﾒﾘｸﾘ ｶﾞｲﾌﾞ" {
		t.Fatalf("expected append event 'ﾒﾘｸﾘ ｶﾞｲﾌﾞ', got %q", got)
	}

	// Built-in override event (April Fools 04-01) overrides base.
	aprilFools := time.Date(2026, 4, 1, 12, 0, 0, 0, bbs.JSTLocation)
	if got := idGen.GenerateSlip(bbs.SLIPVVV, boardID, bbs.ScopeGlobal, 0, userID, sc, aprilFools); got != "ｳｿﾂｷ" {
		t.Fatalf("expected override event 'ｳｿﾂｷ', got %q", got)
	}
}
