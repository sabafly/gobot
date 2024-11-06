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

package ratelimit

import (
	"time"
)

type Rule struct {
	Limit int
	Unit  time.Duration
}

// CheckLimit はセーフならtrueを返す
func CheckLimit(t []time.Time, r []Rule) ([]time.Time, bool) {
	m := 0
	for _, v := range r {
		if !check(t, v.Limit, v.Unit) {
			return t, false
		}
		m = max(v.Limit, m)
	}
	t = append([]time.Time{time.Now()}, (t)[0:min(m, len(t))]...)
	return t, true
}

func check(times []time.Time, limit int, unit time.Duration) bool {
	return len(times) < limit || time.Since(times[limit-1]) >= unit
}
