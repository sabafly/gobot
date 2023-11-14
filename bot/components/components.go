package components

import (
	"github.com/sabafly/gobot/ent"
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

	Version string
}

func (c *Components) DB() *ent.Client { return c.db }
