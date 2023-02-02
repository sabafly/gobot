/*
	Copyright (C) 2022-2023  ikafly144

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
	"fmt"
	"net/http"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
	"github.com/sabafly/gobot/pkg/lib/constants"
	"github.com/sabafly/gobot/pkg/lib/env"
	"github.com/sabafly/gobot/pkg/lib/logging"
)

func init() {
	discordgo.Logger = logging.Logger()
}

// シャードとセッションをまとめる
type Shard struct {
	ShardID int
	*Api
	Session *discordgo.Session
}

type Api struct {
	sync.RWMutex
	Client         http.Client
	MaxRestRetries int
	UserAgent      string

	listening chan any

	handlersMu sync.RWMutex
	handlers   map[string][]*eventHandlerInstance

	Dialer   *websocket.Dialer
	wsConn   *websocket.Conn
	wsMutex  sync.Mutex
	sequence *int64
	gateway  string
}

// ボット接続を管理する
type BotManager struct {
	*Api
	ShardCount int
	Shards     []*Shard
}

// ボットセッションを開始する
func (b *BotManager) Open() (err error) {
	shards := b.Shards

	for i := range shards {
		// 内部APIと接続
		if err := shards[i].ApiOpen(); err != nil {
			return fmt.Errorf("failed open api connection: %w", err)
		}

		s := shards[i].Session

		// セッションを初期化
		s.Identify.Intents = discordgo.IntentsAll
		s.ShardCount = b.ShardCount
		s.ShardID = shards[i].ShardID
		s.UserAgent = constants.UserAgent
		s.StateEnabled = true

		s.LogLevel = env.DLogLevel

		// セッションを開始
		if err := s.Open(); err != nil {
			return fmt.Errorf("failed open session: %w", err)
		}
	}
	b.Shards = shards
	return nil
}

// ボットセッションを終了する
func (b *BotManager) Close() (err error) {
	shards := b.Shards
	for i := range shards {
		s := shards[i].Session

		if err := s.Close(); err != nil {
			return fmt.Errorf("failed close session: %w", err)
		}

		if err := shards[i].ApiClose(); err != nil {
			return fmt.Errorf("failed close api connection: %w", err)
		}
	}
	b.Shards = shards
	return nil
}

// 新規のボット接続を作成する
func New(token string) (bot *BotManager, err error) {
	// セッションを作成
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed create bot: %w", err)
	}

	// シャードの個数を取得
	count, err := shardCount(s)
	if err != nil {
		return nil, fmt.Errorf("failed get shard count: %w", err)
	}

	// シャードを設定
	return validateShards(token, count)
}

// シャード数を取得する
func shardCount(s *discordgo.Session) (count int, err error) {
	gateway, err := s.GatewayBot()
	if err != nil {
		return 0, fmt.Errorf("failed request gateway bot: %w", err)
	}
	count = gateway.Shards
	return count, nil
}

// 指定した数のシャードを用意する
func validateShards(token string, count int) (bot *BotManager, err error) {
	var zero int64 = 0
	bot = &BotManager{
		// API接続関連
		Api: &Api{
			Dialer:         websocket.DefaultDialer,
			MaxRestRetries: 5,
			Client:         http.Client{},
			sequence:       &zero,
		}}

	for i := 0; i < count; i++ {
		s, err := discordgo.New("Bot " + token)
		if err != nil {
			return nil, fmt.Errorf("failed validate shard %v: %w", i, err)
		}
		bot.Shards = append(bot.Shards, &Shard{
			ShardID: i,
			Session: s,
			// API接続関連
			Api: &Api{
				Dialer:         websocket.DefaultDialer,
				MaxRestRetries: 5,
				Client:         http.Client{},
				sequence:       &zero,
			},
		})
	}

	// TODO: 別の場所に移す
	bot.AddHandler(bot.guildCreateHandler)
	bot.AddHandler(bot.guildDeleteHandler)

	return bot, nil
}

// セッションにハンダラを登録する
func (b *BotManager) AddHandler(handler any) {
	for _, s := range b.Shards {
		s.Session.AddHandler(handler)
	}
}

// ギルド作成をデフォルトでハンドルする
func (b *BotManager) guildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	err := b.guildCreateCall(g.ID)
	if err != nil {
		logging.Error("ギルド作成呼び出しに失敗 %s", err)
	}
	logging.Info("ギルドが追加されました %s(%s)", g.Name, g.ID)
}

// ギルド削除をデフォルトでハンドルする
func (b *BotManager) guildDeleteHandler(s *discordgo.Session, g *discordgo.GuildDelete) {
	err := b.guildDeleteCall(g)
	if err != nil {
		logging.Error("ギルド削除呼び出しに失敗 %s", err)
	}
	logging.Info("ギルドが削除されました %s(%s)", g.Name, g.ID)
}

// 内部APIのイベントハンダラを登録する
func (b *BotManager) AddApiHandler(handler any) {
	for _, s := range b.Shards {
		s.AddHandler(handler)
	}
}
