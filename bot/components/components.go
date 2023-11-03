package components

import "github.com/sabafly/gobot/ent"

func New(db *ent.Client) *Components {
	return &Components{
		db: db,
	}
}

type Components struct {
	db *ent.Client

	commandsRegistry map[string]Command

	Version string
}

func (c *Components) DB() *ent.Client { return c.db }
