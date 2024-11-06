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

package builtin

func Or[T any](ok bool, a, b T) T {
	if ok {
		return a
	}
	return b
}

func Ptr[T any](v T) *T { return &v }

func RequireNonNil[T any](v *T) T {
	if v == nil {
		panic("unexpected nil")
	}
	return *v
}

func NonNil[T any](v *T) T {
	if v != nil {
		return *v
	}
	return Zero[T]()
}

func NonNilMap[T map[K]V, K comparable, V any](v T) T {
	if v != nil {
		return v
	}
	return make(T)
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func NonNilOrDefault[T any](v *T, def T) T {
	if v != nil {
		return *v
	}
	return def
}

func Zero[T any]() T {
	var zero T
	return zero
}
