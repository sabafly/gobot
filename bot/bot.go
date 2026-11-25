/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package bot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/sabafly/gobot/database"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/sharding"
	"github.com/disgoorg/snowflake/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	bbsCmd "github.com/sabafly/gobot/bot/commands/bbs"
	"github.com/sabafly/gobot/bot/commands/bet"
	"github.com/sabafly/gobot/bot/commands/currency"
	"github.com/sabafly/gobot/bot/commands/debug"
	"github.com/sabafly/gobot/bot/commands/level"
	"github.com/sabafly/gobot/bot/commands/message"
	"github.com/sabafly/gobot/bot/commands/permission"
	"github.com/sabafly/gobot/bot/commands/ping"
	"github.com/sabafly/gobot/bot/commands/play"
	"github.com/sabafly/gobot/bot/commands/role"
	"github.com/sabafly/gobot/bot/commands/setting"
	"github.com/sabafly/gobot/bot/commands/user"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/internal/bbs"
	"github.com/sabafly/gobot/internal/bbs/discordsync"
	"github.com/sabafly/gobot/internal/bbs/gormstore"
	"github.com/sabafly/gobot/internal/discordutil"
	"github.com/sabafly/gobot/internal/emoji"
	"github.com/sabafly/gobot/internal/i18n"

	_ "net/http/pprof"
)

var cmd = &cobra.Command{
	Use:   "bot",
	Short: "botを起動する",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	cmd.Flags().BoolVarP(&usePprof, "pprof", "p", false, "pprofを有効にする")
	cmd.Flags().BoolVarP(&isDebug, "debug", "d", false, "デバッグモードを有効にする")
}

var (
	usePprof bool
	isDebug  bool
)

func Command() *cobra.Command {
	return cmd
}

var (
	version = "v1.0.0-alpha.0"
)

type LogLevel struct{}

func (l LogLevel) Level() slog.Level {
	if isDebug {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func run() error {
	if usePprof {
		slog.Info("pprofが有効になっています")
		go func() {
			if err := http.ListenAndServe(":6060", nil); err != nil {
				slog.Error("pprofを起動できません", slog.Any("error", err))
			}
		}()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     LogLevel{},
	})))
	_ = godotenv.Load()

	config, err := components.LoadConfig("gobot.yml")
	if err != nil {
		return fmt.Errorf("設定ファイルを読み込めません: %w", err)
	}

	if config.Debug.Trace {
		slog.Info("トレースモードが有効になっています")
		generic.PrintDebugInfo = true
	}

	gormDB, err := database.NewDB(config.GormDSN)
	if err != nil {
		return fmt.Errorf("gormとの接続を開けません: %w", err)
	}

	// c, err := caches.Open(config.Redis...)
	// if err != nil {
	// 	return fmt.Errorf("cacheを開けません: %w", err)
	// }

	// ent schema migration is removed as we use GORM migration in database.NewDB

	if err := i18n.LoadLocales(config.LocaleDir); err != nil {
		return fmt.Errorf("ロケールファイルが読み込めません path=%s: %w", config.LocaleDir, err)
	}

	reg, err := emoji.LoadRegistry(config.EmojiRegistryFile)
	if err != nil {
		return fmt.Errorf("絵文字レジストリを読み込めません path=%s: %w", config.EmojiRegistryFile, err)
	}
	emoji.SetDefaultRegistry(reg)

	component := components.New(ctx, *config, gormDB)
	component.Version = version

	bbsSys := initBBSSystem(component, config, gormDB)

	component.AddCommands(
		debug.Command(component),
		ping.Command(component),
		message.Command(component),
		role.Command(component),
		level.Command(component),
		permission.Command(component),
		setting.Command(component),
		role.ImportCommand(component),
		currency.Command(component),
		play.Command(component),
		bet.Command(component),
		user.Command(component),
		bbsCmd.Command(bbsSys),
		bbsCmd.AdminCommand(bbsSys),
	)

	ready := make(chan *events.Ready)

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return fmt.Errorf("DISCORD_TOKEN が空です")
	}
	client, err := disgo.New(token,
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagsAll)),
		bot.WithShardManagerConfigOpts(
			sharding.WithAutoScaling(true),
			sharding.WithGatewayConfigOpts(
				gateway.WithAutoReconnect(true),
				gateway.WithIntents(gateway.IntentsGuild.Remove(gateway.IntentGuildPresences), gateway.IntentsPrivileged.Remove(gateway.IntentGuildPresences)),
				gateway.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					AddSource: true,
					Level:     slog.LevelInfo,
				}))),
			),
		),
		bot.WithRestClientConfigOpts(
			rest.WithUserAgent(fmt.Sprintf("DiscordBot (%s, %s)", disgo.GitHub, disgo.Version)),
			rest.WithRateLimiterConfigOpts(
				rest.WithRateLimiterLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					AddSource: true,
					Level:     slog.LevelInfo,
				}))),
			),
		),
		bot.WithEventManagerConfigOpts(
			bot.WithAsyncEventsEnabled(),
			bot.WithListeners(
				bot.NewListenerChan(ready),
			),
		),
	)
	if err != nil {
		return fmt.Errorf("クライアントを作成できません: %w", err)
	}

	if err := component.Initialize(client); err != nil {
		return fmt.Errorf("コンポーネントを初期化できません: %w", err)
	}

	if err := client.OpenShardManager(ctx); err != nil {
		return fmt.Errorf("discord ゲートウェイを開けません: %w", err)
	}
	defer client.Close(ctx)

	<-ready
	slog.Info("bots are now ready")

	// set default webhook
	discordutil.WebhookDefaultName = "gobot-webhook"
	self, ok := client.Caches.SelfUser()
	if !ok {
		return fmt.Errorf("cannot cache self user")
	}
	if avatarURL, err := url.Parse(self.EffectiveAvatarURL(discord.WithFormat(discord.FileFormatPNG))); err == nil {
		resp, err := http.Get(avatarURL.String())
		if err != nil {
			return fmt.Errorf("error on get: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				slog.Error("error on close response body", slog.Any("error", err))
			}
		}()
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error on read all: %w", err)
		}
		discordutil.WebhookDefaultAvatar = discord.NewIconRaw(discord.IconTypePNG, buf)
	}

	slog.Info("Bot is now running")
	<-ctx.Done()
	slog.Info("Bot is shutting down")

	return nil
}

func initBBSSystem(component *components.Components, config *components.Config, gormDB *database.DB) *bbsCmd.System {
	secret := os.Getenv("BBS_SECRET_KEY")
	if secret == "" {
		secret = config.BBS.SecretKey
	}
	var secretBytes []byte
	if secret != "" {
		secretBytes = []byte(secret)
	} else {
		var rnd [32]byte
		_, _ = rand.Read(rnd[:])
		secretBytes = rnd[:]
		slog.Warn("BBS_SECRET_KEY and config.bbs.secret_key are empty; using ephemeral random key (IDs/hashes will rotate on restart)",
			slog.String("hint", hex.EncodeToString(rnd[:4])))
	}

	seenOps := make(map[snowflake.ID]struct{}, len(config.BBS.Operators)+len(config.Debug.DebugUsers))
	var operators []snowflake.ID
	for _, id := range append(append([]snowflake.ID(nil), config.BBS.Operators...), config.Debug.DebugUsers...) {
		if id == 0 {
			continue
		}
		if _, ok := seenOps[id]; !ok {
			seenOps[id] = struct{}{}
			operators = append(operators, id)
		}
	}

	store := gormstore.New(gormDB.DB)
	registry := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	registry.Register(store)
	identity := bbs.NewIdentityGenerator(secretBytes)
	policy := bbs.NewPolicyManager(bbs.PolicyConfig{
		Version:          config.BBS.PolicyVersion,
		TermsURL:         config.BBS.TermsURL,
		GuidelinesURL:    config.BBS.GuidelinesURL,
		PrivacyPolicyURL: config.BBS.PrivacyPolicyURL,
	})
	svc := bbs.NewService(registry, store, identity, policy, operators)
	syncer := discordsync.NewSyncer(svc)
	syncer.CanSendDM = func(ctx context.Context, uid snowflake.ID) bool {
		return discordsync.CanSendDMToUser(ctx, gormDB, uid)
	}
	return bbsCmd.NewSystem(component, svc, syncer, config.BBS.OperatorLogChannel)
}
