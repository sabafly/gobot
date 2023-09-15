package client

import (
	"fmt"
	"os"
	"sync"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/logging"
)

func New(cfg *Config, db *db.DB) (*Client, error) {
	if err := os.MkdirAll("./logs/messages", 0755); err != nil {
		return nil, err
	}
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
	Config          *Config
	DB              *db.DB
	MessagePin      map[snowflake.ID]*db.GuildMessagePins
	Logger          *Logger
	ResourceManager *ResourceManager
}

// Deprecated: Use DB.GuildData().Mu()
func (c *Client) GuildDataLock(gid snowflake.ID) *sync.Mutex {
	return c.DB.GuildData().Mu(gid)
}

// Deprecated: Use DB.UserData().Mu()
func (c *Client) UserDataLock(uid snowflake.ID) *sync.Mutex {
	return c.DB.UserData().Mu(uid)
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
