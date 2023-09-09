package db

import (
	"sync"
	"sync/atomic"
)

type UMutex struct {
	rwmu sync.RWMutex
	u    int32
}

func (m *UMutex) RLock() {
	m.rwmu.RLock()
}

func (m *UMutex) RUnlock() {
	m.rwmu.RUnlock()
}

func (m *UMutex) Lock() {
lock:
	m.rwmu.Lock()
	if atomic.LoadInt32(&m.u) > 0 {
		m.rwmu.Unlock()
		goto lock
	}
}

func (m *UMutex) Unlock() {
	m.rwmu.Unlock()
}

func (m *UMutex) Upgrade() bool {
	success := atomic.AddInt32(&m.u, 1) == 1
	if success {
		m.rwmu.RUnlock()
		m.rwmu.Lock()
	}
	atomic.AddInt32(&m.u, -1)
	return success
}
