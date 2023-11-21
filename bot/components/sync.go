package components

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/smap"
)

type Mu struct {
	m smap.SyncedMap[snowflake.ID, *sync.Mutex]
}

func (m *Mu) Mutex(id snowflake.ID) *sync.Mutex {
	mu, _ := m.m.LoadOrStore(id, &sync.Mutex{})
	return mu
}

func (c *Components) GetLock(namespace string) *Mu {
	m, _ := c.l.LoadOrStore(namespace, &Mu{})
	return m
}
