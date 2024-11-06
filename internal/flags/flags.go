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

package flags

import "golang.org/x/exp/constraints"

// Add allows you to add multiple bits together, producing a new bit
func Add[T constraints.Integer](f T, bits ...T) T {
	for _, bit := range bits {
		f |= bit
	}
	return f
}

// Remove allows you to subtract multiple bits from the first, producing a new bit
func Remove[T constraints.Integer](f T, bits ...T) T {
	for _, bit := range bits {
		f &^= bit
	}
	return f
}

// Has will ensure that the bit includes all the bits entered
func Has[T constraints.Integer](f T, bits ...T) bool {
	for _, bit := range bits {
		if (f & bit) != bit {
			return false
		}
	}
	return true
}

// Missing will check whether the bit is missing any one of the bits
func Missing[T constraints.Integer](f T, bits ...T) bool {
	for _, bit := range bits {
		if (f & bit) != bit {
			return true
		}
	}
	return false
}
