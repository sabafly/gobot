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
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/dislog"
	"github.com/disgoorg/snowflake/v2"
	"github.com/mattn/go-colorable"
	"github.com/sabafly/gobot/bot/commands"
	"github.com/sirupsen/logrus"

	botlib "github.com/sabafly/gobot/lib/bot"
)

var (
	version = "dev"
)

func Run() {

	cfg, err := botlib.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// logger := log.New(log.Ldate | log.Ltime | log.Lshortfile)
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

	b := botlib.New(logger, version, *cfg)

	b.Handler.AddExclude(b.Config.Dislog.WebhookChannel)

	b.Handler.AddCommands(
		commands.Ping(b),
		commands.Poll(b),
		commands.Role(b),
		commands.RolePanel(b),
	)

	b.Handler.AddComponents(
		commands.PollComponent(b),
		commands.RolePanelComponent(b),
	)

	b.Handler.AddModals(
		commands.PollModal(b),
		commands.RolePanelModal(b),
	)

	b.Handler.AddReady(func(r *events.Ready) {
		polls, err := b.DB.Poll().GetAll()
		if err != nil {
			logger.Fatal(err)
		}
		for _, p := range polls {
			go commands.End(b, p)
		}
	})

	b.SetupBot(bot.NewListenerFunc(b.Handler.OnEvent))
	b.Client.EventManager().AddEventListeners(&events.ListenerAdapter{
		OnGuildJoin:  b.OnGuildJoin,
		OnGuildLeave: b.OnGuildLeave,
	})

	if cfg.ShouldSyncCommands {
		var guilds []snowflake.ID
		if cfg.DevMode {
			guilds = b.Config.DevGuildIDs
		}
		b.Handler.SyncCommands(b.Client, guilds...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = b.Client.OpenGateway(ctx); err != nil {
		b.Logger.Errorf("Failed to connect to gateway: %s", err)
	}
	defer b.Client.Close(context.TODO())

	b.Logger.Info("Bot is running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
	b.Logger.Info("Shutting down...")
}
