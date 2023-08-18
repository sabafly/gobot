package client

import (
	"fmt"
	"sync"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/db"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
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
		userDataLocks:  make(map[snowflake.ID]*sync.Mutex),
		guildDataLocks: make(map[snowflake.ID]*sync.Mutex),
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
	Config         *Config
	DB             db.DB
	MessagePin     map[snowflake.ID]*db.GuildMessagePins
	MessagePinSync sync.Mutex
	Logger         *Logger
	userDataLock   sync.Mutex
	userDataLocks  map[snowflake.ID]*sync.Mutex
	guildDataLock  sync.Mutex
	guildDataLocks map[snowflake.ID]*sync.Mutex
}

func (c *Client) GuildDataLock(gid snowflake.ID) *sync.Mutex {
	c.guildDataLock.Lock()
	defer c.guildDataLock.Unlock()
	if c.guildDataLocks[gid] == nil {
		c.guildDataLocks[gid] = new(sync.Mutex)
	}
	return c.guildDataLocks[gid]
}

func (c *Client) UserDataLock(uid snowflake.ID) *sync.Mutex {
	c.userDataLock.Lock()
	defer c.userDataLock.Unlock()
	if c.userDataLocks[uid] == nil {
		c.userDataLocks[uid] = new(sync.Mutex)
	}
	return c.userDataLocks[uid]
}

func (c *Client) CheckAutoCompletePermission(b *botlib.Bot[*Client], perm string, alt_perm discord.Permissions) handler.Check[*events.AutocompleteInteractionCreate] {
	return func(ctx *events.AutocompleteInteractionCreate) bool {
		if b.CheckDev(ctx.User().ID) {
			return true
		}
		if ctx.Member() != nil && ctx.Member().Permissions.Has(alt_perm) {
			return true
		}
		gd, err := c.DB.GuildData().Get(*ctx.GuildID())
		if err == nil {
			if gd.UserPermissions[ctx.User().ID].Has(perm) {
				return true
			}
			for _, id := range ctx.Member().RoleIDs {
				if gd.RolePermissions[id].Has(perm) {
					return true
				}
			}
		}
		return false
	}
}

func (c *Client) CheckCommandPermission(b *botlib.Bot[*Client], perm string, alt_perm discord.Permissions) handler.Check[*events.ApplicationCommandInteractionCreate] {
	return func(ctx *events.ApplicationCommandInteractionCreate) bool {
		if b.CheckDev(ctx.User().ID) {
			return true
		}
		if ctx.Member() != nil && ctx.Member().Permissions.Has(alt_perm) {
			return true
		}
		gd, err := c.DB.GuildData().Get(*ctx.GuildID())
		if err == nil {
			if gd.UserPermissions[ctx.User().ID].Has(perm) {
				return true
			}
			for _, id := range ctx.Member().RoleIDs {
				if gd.RolePermissions[id].Has(perm) {
					return true
				}
			}
		}
		_ = botlib.ReturnErrMessage(ctx, "error_no_permission", botlib.WithTranslateData(map[string]any{"Name": perm}))
		return false
	}
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
