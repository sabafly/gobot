package client

import (
	"fmt"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-lib/v2/logging"
)

func New(cfg *Config, db db.DB) (*Client, error) {
	ml, err := logging.New(logging.Config{
		LogPath: "./logs/messages",
		LogName: "message.log",
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		Config: cfg,
		DB:     db,
		Logger: &Logger{
			Message:      ml,
			DebugChannel: map[snowflake.ID]*DebugLog{},
			DebugGuild:   map[snowflake.ID]*DebugLog{},
		},
	}, nil
}

func (c *Client) Close() (err error) {
	defer func() {
		e := c.Logger.Message.Close()
		if e != nil {
			err = fmt.Errorf("%w: %w", err, e)
		}
	}()
	defer func() {
		for _, l := range c.Logger.DebugChannel {
			if l.Logger != nil {
				e := l.Logger.Close()
				if e != nil {
					err = fmt.Errorf("%w: %w", err, e)
				}
			}
		}
	}()
	defer func() {
		for _, l := range c.Logger.DebugGuild {
			if l.Logger != nil {
				e := l.Logger.Close()
				if e != nil {
					err = fmt.Errorf("%w: %w", err, e)
				}
			}
		}
	}()
	return
}

type Client struct {
	Config     *Config
	DB         db.DB
	MessagePin map[snowflake.ID]db.GuildMessagePins
	Logger     *Logger
}

type Logger struct {
	Message      *logging.Logging
	DebugChannel map[snowflake.ID]*DebugLog
	DebugGuild   map[snowflake.ID]*DebugLog
}

type DebugLog struct {
	Logger     *logging.Logging
	LogChannel *snowflake.ID
}
