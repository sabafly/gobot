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
	"time"

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
	"github.com/sabafly/gobot/internal/admin"
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

	startedAt := time.Now().UTC()
	logHub := admin.NewLogHub(0)
	slog.SetDefault(slog.New(admin.NewTeeHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     LogLevel{},
	}), logHub)))
	_ = godotenv.Load()

	config, err := components.LoadConfig("gobot.yml")
	if err != nil {
		return fmt.Errorf("設定ファイルを読み込めません: %w", err)
	}
	if config.Admin.LogBufferSize > 0 {
		logHub.SetCapacity(config.Admin.LogBufferSize)
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

	bbsSys, err := initBBSSystem(component, config, gormDB)
	if err != nil {
		slog.Error("BBSシステムの初期化に失敗したためBBSコマンドを無効化します", slog.Any("error", err))
	}

	cmds := []components.Command{
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
	}
	if bbsSys != nil {
		cmds = append(cmds, bbsCmd.Command(bbsSys), bbsCmd.AdminCommand(bbsSys))
	}
	component.AddCommands(cmds...)

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

	var adminDone chan struct{}
	if config.Admin.Enabled {
		adminSrv, adminErr := initAdminServer(component, config, gormDB, client, bbsSys, logHub, version, startedAt)
		if adminErr != nil {
			slog.Error("管理APIサーバーの初期化に失敗したため管理APIを無効化します", slog.Any("error", adminErr))
		} else {
			adminDone = make(chan struct{})
			go func() {
				defer close(adminDone)
				if serveErr := adminSrv.ListenAndServe(ctx); serveErr != nil {
					slog.Error("管理APIサーバーでエラーが発生しました", slog.Any("error", serveErr))
				}
			}()
		}
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
	if adminDone != nil {
		<-adminDone
	}

	return nil
}

func initAdminServer(
	component *components.Components,
	config *components.Config,
	gormDB *database.DB,
	client *bot.Client,
	bbsSys *bbsCmd.System,
	logHub *admin.LogHub,
	ver string,
	startedAt time.Time,
) (*admin.Server, error) {
	if config == nil {
		return nil, fmt.Errorf("設定が空です")
	}
	var bbsSvc *bbs.Service
	var bbsSync *discordsync.Syncer
	if bbsSys != nil {
		bbsSvc = bbsSys.Service
		bbsSync = bbsSys.Syncer
	}
	return admin.NewServer(admin.ServerDeps{
		Config:      *config,
		AdminConfig: config.Admin,
		DB:          gormDB,
		Components:  component,
		Client:      client,
		BBSService:  bbsSvc,
		BBSSyncer:   bbsSync,
		LogHub:      logHub,
		Version:     ver,
		StartedAt:   startedAt,
	})
}

func initBBSSystem(component *components.Components, config *components.Config, gormDB *database.DB) (*bbsCmd.System, error) {
	if gormDB == nil || gormDB.DB == nil {
		return nil, fmt.Errorf("BBS用のデータベース接続が初期化されていません")
	}
	store := gormstore.New(gormDB.DB)
	secret := os.Getenv("BBS_SECRET_KEY")
	if secret == "" {
		secret = config.BBS.SecretKey
	}
	var secretBytes []byte
	if secret != "" {
		secretBytes = []byte(secret)
	} else {
		var rnd [32]byte
		if _, err := rand.Read(rnd[:]); err != nil {
			return nil, fmt.Errorf("BBSシステムシークレットの乱数生成に失敗しました: %w", err)
		}
		persisted, err := store.GetOrCreateSystemSecret(context.Background(), "default_hmac_secret", rnd[:])
		if err != nil {
			return nil, fmt.Errorf("BBSシステムシークレットの取得・保存に失敗しました: %w", err)
		}
		if len(persisted) == 0 {
			return nil, fmt.Errorf("BBSシステムシークレットが空です")
		}
		secretBytes = persisted
		slog.Warn("BBS_SECRET_KEY and config.bbs.secret_key are empty; using DB-persisted auto-generated key (recommend configuring fixed secret)",
			slog.String("hint", hex.EncodeToString(secretBytes[:min(4, len(secretBytes))])))
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

	policyCfg := bbs.PolicyConfig{
		Version:          config.BBS.PolicyVersion,
		TermsURL:         config.BBS.TermsURL,
		GuidelinesURL:    config.BBS.GuidelinesURL,
		PrivacyPolicyURL: config.BBS.PrivacyPolicyURL,
	}
	if err := bbs.ValidatePolicyConfig(policyCfg); err != nil {
		return nil, fmt.Errorf("BBSポリシー設定が不正です: %w", err)
	}
	registry := bbs.NewProviderRegistry(bbs.DefaultProviderKey)
	registry.Register(store)
	identity := bbs.NewIdentityGenerator(secretBytes)
	policy := bbs.NewPolicyManager(policyCfg)
	svc := bbs.NewService(registry, store, identity, policy, operators)
	capDefs := make([]bbs.CapDefinition, 0, len(config.BBS.Caps))
	for _, c := range config.BBS.Caps {
		capDefs = append(capDefs, bbs.CapDefinition{
			ID:             c.ID,
			Name:           c.Name,
			Description:    c.Description,
			Users:          c.Users,
			GrantOperators: c.GrantOperators,
		})
	}
	if err := svc.SetAdminCaps(capDefs); err != nil {
		return nil, fmt.Errorf("BBSキャップ設定が不正です: %w", err)
	}
	slipEvents := make([]bbs.SlipEventDefinition, 0, len(config.BBS.SlipEvents))
	for _, ev := range config.BBS.SlipEvents {
		slipEvents = append(slipEvents, bbs.SlipEventDefinition{
			ID:        ev.ID,
			Label:     ev.Label,
			StartDate: ev.StartDate,
			EndDate:   ev.EndDate,
			Mode:      bbs.SlipEventMode(ev.Mode),
		})
	}
	if err := svc.SetSlipEvents(slipEvents); err != nil {
		return nil, fmt.Errorf("BBS SLIPイベント設定が不正です: %w", err)
	}
	syncer := discordsync.NewSyncer(svc)
	syncer.CanSendDM = func(ctx context.Context, uid snowflake.ID) bool {
		return discordsync.CanSendDMToUser(ctx, gormDB, uid)
	}
	syncer.UserLocale = func(ctx context.Context, uid snowflake.ID) discord.Locale {
		return discordsync.ResolveUserLocaleFromDB(ctx, gormDB, uid)
	}
	if component != nil {
		syncer.GuildLocale = component.GuildLocaleOverride
		component.RegisterGuildLocaleChangeHook(func(ctx context.Context, client *bot.Client, guildID snowflake.ID, resolvedLocale discord.Locale) {
			if client == nil {
				return
			}
			go syncer.RefreshGuildBoardGuides(context.Background(), client, guildID, resolvedLocale)
		})
	}
	return bbsCmd.NewSystem(component, svc, syncer, config.BBS.OperatorLogChannel), nil
}
