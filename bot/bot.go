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
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/sharding"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/sabafly/gobot/bot/commands/debug"
	"github.com/sabafly/gobot/bot/commands/level"
	"github.com/sabafly/gobot/bot/commands/message"
	"github.com/sabafly/gobot/bot/commands/ping"
	"github.com/sabafly/gobot/bot/commands/role"
	userinfo "github.com/sabafly/gobot/bot/commands/user_info"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/internal/translate"
	"github.com/spf13/cobra"
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
	version = "v1.0.0-alpha.0"
)

func run() error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	})))
	_ = godotenv.Load()

	config, err := components.Load("gobot.yml")
	if err != nil {
		return fmt.Errorf("設定ファイルを読み込めません: %w", err)
	}

	db, err := ent.Open("mysql", config.MySQL)
	if err != nil {
		return fmt.Errorf("mysqlとの接続を開けません: %w", err)
	}
	defer db.Close()

	// c, err := caches.Open(config.Redis...)
	// if err != nil {
	// 	return fmt.Errorf("cacheを開けません: %w", err)
	// }

	if err := db.Schema.Create(context.Background()); err != nil {
		return fmt.Errorf("スキーマを定義できません: %w", err)
	}

	if _, err := translate.LoadDir(config.TranslateDir); err != nil {
		return fmt.Errorf("翻訳ファイルが読み込めません path=%s: %w", config.TranslateDir, err)
	}

	component := components.New(db, *config)
	component.Version = version

	component.AddCommands(
		debug.Command(component),
		ping.Command(),
		message.Command(component),
		role.Command(component),
		level.Command(component),
		userinfo.Command(component),
	)

	token := os.Getenv("TOKEN")
	if token == "" {
		return fmt.Errorf("TOKEN が空です")
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
		bot.WithRestClientConfigOpts(
			rest.WithUserAgent(fmt.Sprintf("DiscordBot (%s, %s)", disgo.GitHub, disgo.Version)),
		),
		bot.WithEventManagerConfigOpts(
			bot.WithAsyncEventsEnabled(),
		),
	)
	if err != nil {
		return fmt.Errorf("クライアントを作成できません: %w", err)
	}

	if err := client.OpenShardManager(context.Background()); err != nil {
		return fmt.Errorf("Discord ゲートウェイを開けません: %w", err)
	}
	defer client.Close(context.Background())

	if err := component.Initialize(client); err != nil {
		return fmt.Errorf("コンポーネントを初期化できません: %w", err)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.Signal(0x13), syscall.Signal(0x14))
	<-s

	return nil
}
