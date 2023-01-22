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
	"github.com/ikafly144/gobot/pkg/lib/constants"
	"github.com/ikafly144/gobot/pkg/lib/logger"
)

// シャードとセッションをまとめる
type Shard struct {
	ShardID int
	Api     *Api
	session *discordgo.Session
}

type Api struct {
	sync.Mutex
	Client         http.Client
	MaxRestRetries int
	UserAgent      string

	Dialer   *websocket.Dialer
	wsConn   *websocket.Conn
	wsMutex  sync.Mutex
	sequence *int64
	gateway  string
}

// ボット接続を管理する
type BotManager struct {
	ShardCount int
	Shards     []*Shard
}

// ボットセッションを開始する
func (b *BotManager) Open() (err error) {
	shards := b.Shards
	for i := range shards {
		s := shards[i].session

		// セッションを初期化
		s.ShardCount = b.ShardCount
		s.ShardID = shards[i].ShardID
		s.UserAgent = constants.UserAgent

		s.LogLevel = logger.SetLogLevel()

		// セッションを開始
		if err := s.Open(); err != nil {
			return fmt.Errorf("failed open session: %w", err)
		}

		api := shards[i].Api

		// 内部APIと接続
		if err := api.Open(); err != nil {
			return fmt.Errorf("failed open api connection: %w", err)
		}
	}
	b.Shards = shards
	return nil
}

// ボットセッションを終了する
func (b *BotManager) Close() (err error) {
	shards := b.Shards
	for i := range shards {
		s := shards[i].session

		if err := s.Close(); err != nil {
			return fmt.Errorf("failed close session: %w", err)
		}

		api := shards[i].Api

		if err := api.Close(); err != nil {
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
	bot = &BotManager{}

	for i := 0; i < count; i++ {
		s, err := discordgo.New("Bot " + token)
		if err != nil {
			return nil, fmt.Errorf("failed validate shard %v: %w", i, err)
		}
		var zero int64 = 0
		bot.Shards = append(bot.Shards, &Shard{
			ShardID: i,
			session: s,
			Api: &Api{
				Dialer:         websocket.DefaultDialer,
				MaxRestRetries: 5,
				Client:         http.Client{},
				sequence:       &zero,
			},
		})
	}
	return bot, nil
}
