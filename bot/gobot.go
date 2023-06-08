/*
	Copyright (C) 2022-2023  sabafly

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package gobot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/disgoorg/dislog"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"github.com/sabafly/disgo/bot"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/commands"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/gobot/bot/handlers"
	"github.com/sirupsen/logrus"

	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/logging"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

var (
	version = "dev"
)

func init() {
	botlib.BotName = "gobot"
	botlib.Color = 0x89d53c
}

func Run(file_path, lang_path, gobot_path string) {
	if _, err := translate.LoadTranslations(lang_path); err != nil {
		panic(err)
	}
	cfg, err := botlib.LoadConfig(file_path)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	logger := logrus.New()
	logger.ReportCaller = cfg.DevMode
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (function string, file string) {
			return "", fmt.Sprintf("%s:%d", f.File, f.Line)
		},
	})
	logger.SetOutput(colorable.NewColorableStdout())
	lvl, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	logger.SetLevel(lvl)
	l, err := logging.New(logging.Config{
		LogPath:   "./logs",
		LogLevels: logrus.AllLevels,
	})
	if err != nil {
		panic(err)
	}
	logger.AddHook(l)
	dlog, err := dislog.New(
		dislog.WithLogLevels(dislog.TraceLevelAndAbove...),
		dislog.WithWebhookIDToken(cfg.Dislog.WebhookID, cfg.Dislog.WebhookToken),
	)
	if err != nil {
		logger.Fatal("error initializing dislog: ", err)
	}
	defer dlog.Close(context.TODO())
	logger.AddHook(dlog)
	logger.Infof("Starting bot version: %s", version)
	logger.Infof("Syncing commands? %t", cfg.ShouldSyncCommands)

	b := botlib.New[*client.Client](logger, version, *cfg)

	gobot_cfg, err := client.LoadConfig(gobot_path)
	if err != nil {
		panic(err)
	}
	d, err := db.SetupDatabase(gobot_cfg.DBConfig)
	if err != nil {
		panic(err)
	}
	cl, err := client.New(gobot_cfg, d)
	if err != nil {
		panic(err)
	}

	b.Self = cl

	b.Self.DB, err = db.SetupDatabase(b.Self.Config.DBConfig)
	if err != nil {
		panic(err)
	}

	b.Handler.AddExclude(b.Config.Dislog.WebhookChannel)

	b.Logger.Infof("dev guilds %v", b.Config.DevGuildIDs)
	b.Handler.DevGuildID = b.Config.DevGuildIDs
	b.Handler.IsDebug = b.Config.DevMode
	b.Handler.IsLogEvent = true

	b.Handler.AddCommands(
		commands.Ping(b),
		commands.Poll(b),
		commands.Role(b),
		commands.RolePanel(b),
		commands.Util(b),
		commands.Admin(b),
		commands.About(b),
		commands.Message(b),
	)

	b.Handler.AddComponents(
		commands.PollComponent(b),
		commands.RolePanelComponent(b),
		commands.UtilCalcComponent(b),
		commands.MessageComponent(b),

		handlers.EmbedDialogComponent(b),
	)

	b.Handler.AddModals(
		commands.PollModal(b),
		commands.RolePanelModal(b),
		commands.MessageModal(b),

		handlers.EmbedDialogModal(b),
	)

	b.Handler.AddMessages(
		commands.MessagePinMessageCreate(b),
	)

	b.Handler.AddReady(func(r *events.Ready) {
		b.Logger.Info("Ready!")
		polls, err := b.Self.DB.Poll().GetAll()
		if err == nil {
			for _, p := range polls {
				go commands.End(b, p)
			}
		}
		mp, err := b.Self.DB.MessagePin().GetAll()
		if err == nil {
			b.Self.MessagePin = mp
		}
	})

	b.Handler.AddMemberJoins(
		handler.MemberJoin{
			UUID: uuid.New(),
			Handler: func(event *events.GuildMemberJoin) error {
				b.OnGuildMemberJoin(event)
				return nil
			},
		},
	)

	b.Handler.AddMemberLeaves(
		handler.MemberLeave{
			UUID: uuid.New(),
			Handler: func(event *events.GuildMemberLeave) error {
				b.OnGuildMemberLeave(event)
				return nil
			},
		},
	)

	b.SetupBot(bot.NewListenerFunc(b.Handler.OnEvent))
	b.Client.EventManager().AddEventListeners(&events.ListenerAdapter{
		OnGuildJoin:  b.OnGuildJoin,
		OnGuildLeave: b.OnGuildLeave,
		OnGuildReady: func(event *events.GuildReady) {
			b.Logger.Infof("guild ready: %s", event.GuildID)
		},
		OnGuildsReady: func(event *events.GuildsReady) {
			b.Logger.Infof("guilds on shard %d ready", event.ShardID())
		},
	})

	if cfg.ShouldSyncCommands {
		var guilds []snowflake.ID
		if cfg.DevOnly {
			guilds = b.Config.DevGuildIDs
		}
		b.Handler.SyncCommands(b.Client, guilds...)
	}

	if err := b.Client.OpenShardManager(context.TODO()); err != nil {
		b.Logger.Fatalf("failed to open shard manager: %s", err)
	}
	defer b.Client.ShardManager().Close(context.TODO())

	b.Logger.Infof("shard count: %d", len(b.Client.ShardManager().Shards()))
	b.Logger.Info("Bot is running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
	b.Logger.Info("Shutting down...")
}
