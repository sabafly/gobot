package client

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/db"
)

func New(cfg *Config, db db.DB) (*Client, error) {
	return &Client{
		Config: cfg,
		DB:     db,
	}, nil
}

type Client struct {
	Config     *Config
	DB         db.DB
	MessagePin map[snowflake.ID]db.GuildMessagePins
}
