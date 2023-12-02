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
