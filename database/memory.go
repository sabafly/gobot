package database

import (
	"sync"
	"sync/atomic"
	"time"
)

type DeleteHandler interface {
	OnDelete() error
}

type Expires[T any] struct {
	Value     T
	ExpiresAt time.Time
}

func (e *Expires[T]) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpiresAt)
}

func (e *Expires[T]) SetExpiration(duration time.Duration) {
	e.ExpiresAt = time.Now().Add(duration)
}

func NewMemoryValues[K comparable, T any](timeout time.Duration) *MemoryValues[K, T] {
	v := &MemoryValues[K, T]{
		values:  make(map[K]Expires[T]),
		cond:    sync.NewCond(&sync.Mutex{}),
		timeout: timeout,
	}
	go func() {
		for {
			v.checkExpiration()
			if v.abort.Load() {
				break
			}
		}
	}()
	return v
}

type MemoryValues[K comparable, T any] struct {
	values  map[K]Expires[T]
	timeout time.Duration
	cond    *sync.Cond
	abort   atomic.Bool
}

func (m *MemoryValues[K, T]) Get(key K) (T, bool) {
	m.check()
	value, ok := m.values[key]
	if !ok {
		var zero T
		return zero, false
	}
	m.cond.Broadcast()
	if value.IsExpired() {
		m.delete(key)
		var zero T
		return zero, false
	}
	return value.Value, true
}

func (m *MemoryValues[K, T]) Set(key K, value T) {
	m.check()
	var expires time.Time
	if m.timeout > 0 {
		expires = time.Now().Add(m.timeout)
	}
	m.values[key] = Expires[T]{Value: value, ExpiresAt: expires}
	m.cond.Broadcast()
}

func (m *MemoryValues[K, T]) Delete(key K) {
	m.check()
	m.delete(key)
	m.cond.Broadcast()
}

func (m *MemoryValues[K, T]) Close() {
	m.check()
	m.abort.Store(true)
	m.cond.Broadcast()
}

func (m *MemoryValues[K, T]) delete(key K) {
	v, ok := m.values[key]
	if !ok {
		return
	}
	delete(m.values, key)
	if h, ok := any(v.Value).(DeleteHandler); ok {
		_ = h.OnDelete()
	}
}

func (m *MemoryValues[K, T]) check() {
	if m.cond == nil {
		panic("nil cond")
	}
	if m.abort.Load() {
		panic("memory values is closed")
	}
}

func (m *MemoryValues[K, T]) checkExpiration() {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	m.cond.Wait()

	for key, value := range m.values {
		if value.IsExpired() {
			m.delete(key)
		}
	}
}
