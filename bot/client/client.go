package client

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-lib/v2/logging"
)

func New(cfg *Config, db db.DB) (*Client, error) {
	ml, err := logging.New(logging.Config{
		LogPath: "./logs/messages",
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		Config:        cfg,
		DB:            db,
		MessageLogger: ml,
	}, nil
}

type Client struct {
	Config        *Config
	DB            db.DB
	MessagePin    map[snowflake.ID]db.GuildMessagePins
	MessageLogger *logging.Logging
}
