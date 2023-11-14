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
package bot

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/sabafly/gobot/bot/commands/debug"
	"github.com/sabafly/gobot/bot/commands/message"
	"github.com/sabafly/gobot/bot/commands/ping"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var cmd = &cobra.Command{
	Use:   "bot",
	Short: "botを起動する",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func Command() *cobra.Command { return cmd }

var (
	version = "v1.0"
)

func run() error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	})))
	if err := godotenv.Load(); err != nil {
		slog.Error(".envファイルを読み込めませんでした", "err", err)
		return err
	}

	db, err := ent.Open("mysql", "root:admin@tcp(localhost:3306)/gobot_dev?parseTime=True")
	if err != nil {
		slog.Error("mysqlとの接続を開けません", "err", err)
		return err
	}
	defer db.Close()

	if err := db.Schema.Create(context.Background()); err != nil {
		slog.Error("スキーマを定義できません", "err", err)
		return err
	}

	f, err := os.Open("gobot.yml")
	if err != nil {
		slog.Error("設定ファイルが見つかりません", "err", err)
		return err
	}
	defer f.Close()

	var config components.Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		slog.Error("設定ファイルが読み込めません", "err", err)
		return err
	}

	if _, err := translate.LoadDir(config.TranslateDir); err != nil {
		slog.Error("翻訳ファイルが読み込めません", "err", err, "path", config.TranslateDir)
		return err
	}

	component := components.New(db, config)
	component.Version = version

	component.AddCommands(
		debug.Command(component),
		ping.Command(),
		message.Command(component),
	)

	token := os.Getenv("TOKEN")
	if token == "" {
		slog.Error("TOKEN が空です")
		return errors.New("empty token")
	}
	client, err := disgo.New(token,
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagsAll)),
		bot.WithShardManagerConfigOpts(
			sharding.WithAutoScaling(true),
			sharding.WithGatewayConfigOpts(
				gateway.WithAutoReconnect(true),
				gateway.WithIntents(gateway.IntentsGuild, gateway.IntentsPrivileged),
			),
		),
		bot.WithEventManagerConfigOpts(
			bot.WithAsyncEventsEnabled(),
		),
	)
	if err != nil {
		slog.Error("クライアントを作成できません", "err", err)
		return err
	}

	if err := component.Initialize(client); err != nil {
		slog.Error("コンポーネントを初期化できません", "err", err)
		return err
	}

	if err := client.OpenShardManager(context.Background()); err != nil {
		slog.Error("Discord ゲートウェイを開けません", "err", err)
		return err
	}
	defer client.Close(context.Background())

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.Signal(0x13), syscall.Signal(0x14))
	<-s

	return nil
}