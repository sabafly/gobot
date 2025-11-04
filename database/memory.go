package database

import (
	"sync"
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

// NewMemoryValues creates a new MemoryValues instance that periodically removes
// expired items at the specified cleanup interval.
// The cleanup interval is automatically determined based on the timeout:
// - For timeout > 1 minute: cleanup every minute
// - For timeout > 0: cleanup every timeout/2 (minimum 10ms)
// - For timeout = 0: cleanup every minute (items never expire)
// The timeout parameter specifies the expiration duration for items.
func NewMemoryValues[K comparable, T any](timeout time.Duration) *MemoryValues[K, T] {
	// Default cleanup interval is 1 minute
	cleanupInterval := time.Minute
	if timeout > 0 && timeout < cleanupInterval {
		// Use shorter cleanup interval if timeout is shorter
		cleanupInterval = timeout / 2
		if cleanupInterval < 10*time.Millisecond {
			cleanupInterval = 10 * time.Millisecond
		}
	}

	m := &MemoryValues[K, T]{
		values:   make(map[K]Expires[T]),
		timeout:  timeout,
		stopChan: make(chan struct{}),
	}

	go m.startCleanupTicker(cleanupInterval)

	return m
}

type MemoryValues[K comparable, T any] struct {
	values   map[K]Expires[T]
	timeout  time.Duration
	mu       sync.RWMutex
	stopChan chan struct{}
	stopOnce sync.Once
}

func (m *MemoryValues[K, T]) Get(key K) (T, bool) {
	m.mu.RLock()
	value, ok := m.values[key]
	if !ok {
		m.mu.RUnlock()
		var zero T
		return zero, false
	}

	if !value.IsExpired() {
		m.mu.RUnlock()
		return value.Value, true
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	value, ok = m.values[key]
	if !ok {
		var zero T
		return zero, false
	}
	if value.IsExpired() {
		m.delete(key)
		var zero T
		return zero, false
	}

	return value.Value, true
}

func (m *MemoryValues[K, T]) Set(key K, value T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expires time.Time
	if m.timeout > 0 {
		expires = time.Now().Add(m.timeout)
	}
	m.values[key] = Expires[T]{Value: value, ExpiresAt: expires}
}

func (m *MemoryValues[K, T]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delete(key)
}

func (m *MemoryValues[K, T]) Close() {
	m.stopOnce.Do(func() {
		close(m.stopChan)
	})
}

// Len returns the number of items currently stored (including expired items).
// This is primarily useful for testing and monitoring.
func (m *MemoryValues[K, T]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.values)
}

// delete removes a key from the map and calls OnDelete if the value implements DeleteHandler.
// Must be called with write lock held.
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

func (m *MemoryValues[K, T]) startCleanupTicker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkExpiration()
		case <-m.stopChan:
			return
		}
	}
}

func (m *MemoryValues[K, T]) checkExpiration() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, value := range m.values {
		if value.IsExpired() {
			m.delete(key)
		}
	}
}
