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

package xppoint_test

import (
	"testing"

	"golang.org/x/exp/slog"

	"github.com/sabafly/gobot/internal/xppoint"
)

func TestXPSum(t *testing.T) {
	for i := range 1000 {
		tp := xppoint.TotalPoint(uint64(i))
		xp := xppoint.XP(tp)
		if xp.Level() != uint64(i) {
			slog.Warn("failed 1", "xp", xp, "i", i, "level", xp.Level())
			t.Fail()
		}
		rp := xppoint.RequiredPoint(uint64(i))
		xp.Add(rp)
		if xp.Level() != uint64(i)+1 {
			slog.Warn("failed 2", "xp", xp, "i", i, "level", xp.Level())
			t.Fail()
		}
	}
}
