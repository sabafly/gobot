package components

import (
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/smap"
)

func New(db *ent.Client, conf Config) *Components {
	return &Components{
		db:               db,
		commandsRegistry: make(map[string]Command),
		config:           conf,
	}
}

type Components struct {
	db *ent.Client

	config Config

	commandsRegistry map[string]Command

	l smap.SyncedMap[string, *Mu]

	Version string
}

func (c *Components) DB() *ent.Client { return c.db }
