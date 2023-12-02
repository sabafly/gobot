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
		return *new(V), false
	}
	r, ok := v.(V)
	return r, ok
}

func (s *SyncedMap[K, V]) LoadAndDelete(key K) (V, bool) {
	v, ok := s.m.LoadAndDelete(key)
	if !ok {
		return *new(V), false
	}
	r, ok := v.(V)
	return r, ok
}

func (s *SyncedMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	v, ok := s.m.LoadOrStore(key, value)
	if !ok {
		return v.(V), ok
	}
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
