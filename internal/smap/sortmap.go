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

package smap

import (
	"slices"

	"golang.org/x/exp/maps"
)

func MakeSortMap[M map[K]V, K comparable, V any](m map[K]V) SortMap[M, K, V] {
	return SortMap[M, K, V]{
		m: m,
	}
}

type SortMap[M map[K]V, K comparable, V any] struct {
	m M
}

func (m SortMap[M, K, V]) SortKey(f func(a, b K) int) ([]K, []V) {
	k := maps.Keys(m.m)
	slices.SortStableFunc(k, f)
	v := make([]V, len(k))
	for i, k := range k {
		v[i] = m.m[k]
	}
	return k, v
}

func (m SortMap[M, K, V]) Range(s func(a, b K) int, f func(k K, v V)) {
	k, v := m.SortKey(s)
	for i, k := range k {
		f(k, v[i])
	}
}
