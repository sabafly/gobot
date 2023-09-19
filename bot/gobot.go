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
	"github.com/mattn/go-colorable"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/commands"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/gobot/bot/handlers"
	"github.com/sabafly/gobot/bot/notification"
	"github.com/sabafly/gobot/bot/worker"
	"github.com/sabafly/sabafly-disgo/bot"
	"github.com/sabafly/sabafly-disgo/events"
	"github.com/sirupsen/logrus"

	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/logging"
	"github.com/sabafly/sabafly-lib/v2/translate"
)

var (
	version = "v0.12.2"
)

func init() {
	botlib.BotName = "gobot"
	botlib.Color = 0x00AED9
}

func Run(file_path, lang_path, gobot_path string) error {
	if _, err := translate.LoadTranslations(lang_path); err != nil {
		return err
	}
	cfg, err := botlib.LoadConfig(file_path)
	if err != nil {
		return err
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
		return err
	}
	logger.SetLevel(lvl)
	if err := os.MkdirAll("./logs", 0755); err != nil {
		return err
	}
	l, err := logging.New(logging.Config{
		LogPath:   "./logs",
		LogLevels: logrus.AllLevels,
	})
	if err != nil {
		return err
	}
	defer l.Close()
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
		return err
	}
	d, err := db.SetupDatabase(gobot_cfg.DBConfig)
	if err != nil {
		return err
	}
	defer d.Close()
	cl, err := client.New(gobot_cfg, d)
	if err != nil {
		return err
	}
	defer cl.Close()

	b.Self = cl

	b.Handler.AddExclude(b.Config.Dislog.WebhookChannel)

	b.Logger.Infof("dev guilds %v", b.Config.DevGuildIDs)
	b.Handler.DevGuildID = b.Config.DevGuildIDs
	b.Handler.IsDebug = b.Config.DevMode
	b.Handler.IsLogEvent = true

	b.Handler.AddCommands(
		commands.Ping(b),
		commands.Poll(b),
		commands.Role(b),
		commands.Util(b),
		commands.Admin(b),
		commands.About(b),
		commands.Message(b),
		commands.Level(b),
		commands.Permission(b),
		commands.Config(b),
		commands.Minecraft(b),
		commands.User(b),
		// commands.Ticket(b), // TODO: v0.13までに実装

		commands.UserInfo(b),

		commands.MessageOther(b),
	)

	b.Handler.AddComponents(
		commands.PollComponent(b),
		commands.UtilCalcComponent(b),
		commands.MessageComponent(b),
		commands.MinecraftComponent(b),
		commands.RolePanelV2Component(b),

		handlers.EmbedDialogComponent(b),
	)

	b.Handler.AddModals(
		commands.PollModal(b),
		commands.MessageModal(b),
		commands.LevelModal(b),
		commands.ConfigModal(b),
		commands.RolePanelV2Modal(b),
		commands.TicketModal(b),

		handlers.EmbedDialogModal(b),
	)

	b.Handler.AddMessages(
		commands.MessagePinMessageCreateHandler(b),
		commands.MessageSuffixMessageCreateHandler(b),
		commands.RolePanelV2Message(b),

		handlers.LogMessage(b),
		handlers.UserDataMessage(b),
		handlers.BumpUpMessage(b),
		handlers.MentionMessage(b),
	)

	b.Handler.AddMessageUpdates(
		handlers.BumpUpdateMessage(b),
	)

	b.Handler.AddMessageDelete(
		commands.RolePanelV2MessageDelete(b),
	)

	b.Handler.MessageReactionAdd.Adds(
		commands.RolePanelV2MessageReaction(b),
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

	b.Handler.MemberJoin.Adds(handler.Generics[events.GuildMemberJoin]{
		Handler: func(event *events.GuildMemberJoin) error {
			onGuildMemberJoin(event, b)
			return nil
		},
	})

	b.Handler.MemberLeave.Adds(handler.Generics[events.GuildMemberLeave]{
		Handler: func(event *events.GuildMemberLeave) error {
			onGuildMemberLeave(event, b)
			return nil
		},
	})

	b.Handler.AddEvent(
		handlers.LogEvent(b),
	)

	b.SetupBot(bot.NewListenerFunc(b.Handler.OnEvent))

	b.Self.ResourceManager = client.NewResourceManager(b.Client, b.Self)

	b.Client.EventManager().AddEventListeners(&events.ListenerAdapter{
		OnGuildJoin:  onGuildJoin(b),
		OnGuildLeave: onGuildLeave(b),
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

	w := worker.New()
	w.Add(
		notification.Handler,
		5,
	)
	w.Start(b)

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
	return nil
}
