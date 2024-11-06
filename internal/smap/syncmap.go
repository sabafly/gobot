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

import "sync"

type SyncedMap[K, V any] struct {
	m sync.Map
}

func (s *SyncedMap[K, V]) Delete(key K) {
	s.m.Delete(key)
}

func (s *SyncedMap[K, V]) Load(key K) (V, bool) {
	v, ok := s.m.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	r, ok := v.(V)
	return r, ok
}

func (s *SyncedMap[K, V]) LoadAndDelete(key K) (V, bool) {
	v, ok := s.m.LoadAndDelete(key)
	if !ok {
		var zero V
		return zero, false
	}
	r, ok := v.(V)
	return r, ok
}

func (s *SyncedMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	v, ok := s.m.LoadOrStore(key, value)
	return v.(V), ok
}

func (s *SyncedMap[K, V]) Range(f func(k K, v V) bool) {
	s.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (s *SyncedMap[K, V]) Store(key K, value V) {
	s.m.Store(key, value)
}

func (s *SyncedMap[K, V]) Swap(key K, value V) (V, bool) {
	pr, loaded := s.m.Swap(key, value)
	return pr.(V), loaded
}
