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
package botlib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/gorilla/websocket"
	"github.com/sabafly/gobot/lib/logging"
)

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
	Client     bot.Client
}

// ボットセッションを開始する
func (b *BotManager) Open() (err error) {
	// 内部APIと接続
	if err := b.Api.ApiOpen(); err != nil {
		return fmt.Errorf("failed open api connection: %w", err)
	}

	// セッションを開始
	if err := b.Client.OpenGateway(context.TODO()); err != nil {
		return fmt.Errorf("failed open session: %w", err)
	}
	return nil
}

// ボットセッションを終了する
func (b *BotManager) Close() (err error) {
	b.Client.Close(context.TODO())

	if err := b.ApiClose(); err != nil {
		return fmt.Errorf("failed close api connection: %w", err)
	}
	return nil
}

// 新規のボット接続を作成する
func New(token string) (b *BotManager, err error) {
	b = &BotManager{
		// API接続関連
		Api: NewApi(),
	}

	client, err := disgo.New(token,
		bot.WithDefaultGateway(),
		bot.WithCaches(cache.New()),
		bot.WithDefaultShardManager(),
		bot.WithEventListeners(&events.ListenerAdapter{OnRaw: b.interfaceHandler}),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentsAll),
			gateway.WithEnableRawEvents(true),
		),
	)
	if err != nil {
		return nil, err
	}

	b.Client = client

	return b, nil
}

func NewApi() *Api {
	var zero int64 = 0
	return &Api{
		Dialer:         websocket.DefaultDialer,
		MaxRestRetries: 5,
		Client:         http.Client{},
		sequence:       &zero,
	}
}

func (b *BotManager) interfaceHandler(event *events.Raw) {
	buf, err := io.ReadAll(event.Payload)
	if err != nil {
		logging.Error("イベントバッファ読み込みに失敗 %s", err)
	}
	switch event.EventType {
	case gateway.EventTypeGuildCreate:
		data := gateway.EventGuildCreate{}
		err := json.Unmarshal(buf, &data)
		if err != nil {
			logging.Error("イベントアンマーシャルに失敗 %s", err)
		}
		b.guildCreateCall(data.ID)
	case gateway.EventTypeGuildDelete:
		data := gateway.EventGuildDelete{}
		err := json.Unmarshal(buf, &data)
		if err != nil {
			logging.Error("イベントアンマーシャルに失敗 %s", err)
		}
		b.guildDeleteCall(data)
	case gateway.EventTypeMessageCreate:
		data := gateway.EventMessageCreate{}
		err := json.Unmarshal(buf, &data)
		if err != nil {
			logging.Error("イベントアンマーシャルに失敗 %s", err)
		}
		b.messageCreateCall(data)
	}
}
