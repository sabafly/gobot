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
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/sabafly/gobot/pkg/lib/logging"
)

type Event struct {
	Operation int             `json:"op"`
	Sequence  int64           `json:"s"`
	Type      string          `json:"t"`
	RawData   json.RawMessage `json:"d"`
	// Structはこのファイルのその他の型の一つを含む
	// TODO:よくわからん
	Struct any `json:"-"`
}

type heartbeatOp struct {
	Op   int   `json:"op"`
	Data int64 `json:"d"`
}

func (a *Shard) ApiOpen() (err error) {
	logging.Debug("called")

	a.Lock()
	defer a.Unlock()

	if a.wsConn != nil {
		return fmt.Errorf("websocket already opened")
	}

	if a.gateway == "" {
		a.gateway, err = a.Gateway()
		if err != nil {
			return err
		}
	}

	logging.Info("connecting to gateway %s", a.gateway)
	header := http.Header{}
	header.Add("accept-encoding", "zlib")
	a.wsConn, _, err = a.Dialer.Dial(a.gateway, header)
	if err != nil {
		logging.Error("ゲートウェイに接続できませんでした %s, %s", a.gateway, err)
		a.gateway = ""
		a.wsConn = nil
		return err
	}

	a.wsConn.SetCloseHandler(func(code int, text string) error {
		return nil
	})

	defer func() {
		if err != nil {
			a.wsConn.Close()
			a.wsConn = nil
		}
	}()

	mt, m, err := a.wsConn.ReadMessage()
	if err != nil {
		return err
	}
	_, err = a.onEvent(mt, m)
	if err != nil {
		return err
	}

	a.listening = make(chan any)

	go a.listen(a.wsConn, a.listening)

	return nil
}

func (s *Shard) listen(wsConn *websocket.Conn, listening <-chan any) {

	logging.Info("内部呼び出し")

	for {

		messageType, message, err := wsConn.ReadMessage()

		if err != nil {
			s.RLock()
			sameConn := s.wsConn == wsConn
			s.RUnlock()

			if sameConn {
				logging.Warning("ゲートウェイ %s からメッセージを読み込めませんでした %s", s.gateway, err)

				err := s.ApiClose()
				if err != nil {
					logging.Warning("API接続をクローズできません %s", err)
				}

				//TODO: 再読み込み
			}

			return
		}

		select {
		case <-listening:
			return

		default:
			_, err := s.onEvent(messageType, message)
			if err != nil {
				logging.Error("イベント呼び出しに失敗 %s", err)
			}
		}
	}
}

// 内部APIとの接続を閉じる
//
// TODO:実装する
func (a *Shard) ApiClose() (err error) {
	if a.wsConn != nil {
		err := a.wsConn.Close()
		if err != nil {
			logging.Fatal("ウェブソケット接続のクローズに失敗 %s", err)
		}
		a.wsConn = nil
	}
	if a.listening != nil {
		close(a.listening)
		a.listening = nil
	}
	return nil
}

// Websocketイベント呼び出し
func (a *Shard) onEvent(messageType int, message []byte) (*Event, error) {
	var reader io.Reader
	reader = bytes.NewBuffer(message)

	if messageType == websocket.BinaryMessage {

		z, err := zlib.NewReader(reader)
		if err != nil {
			logging.Error("ウェブソケットメッセージを展開できませんでした %s", err)
			return nil, err
		}

		defer func() {
			if err := z.Close(); err != nil {
				logging.Warning("zlibをクローズできませんでした %s", err)
			}
		}()

		reader = z
	}

	var e *Event
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&e); err != nil {
		logging.Error("ウェブソケットメッセージをデコードできませんでした %s", err)
		return e, err
	}

	logging.Debug("Op: %d, Seq: %d, Type: %s, Data: %s\n\n", e.Operation, e.Sequence, e.Type, string(e.RawData))

	if e.Operation == 1 {
		logging.Debug("[内部呼び出し] Op1ハートビートの応答を送信します")
		a.wsMutex.Lock()
		err := a.wsConn.WriteJSON(heartbeatOp{1, atomic.LoadInt64(a.sequence)})
		a.wsMutex.Unlock()
		if err != nil {
			logging.Error("[内部呼び出し] Op1ハートビートの応答に失敗")
			return e, err
		}

		return e, nil
	}

	if e.Operation != 8 {
		logging.Warning("不明なイベント Op: %d, Seq: %d, Type: %s, Data: %s, message: %s", e.Operation, e.Sequence, e.Type, string(e.RawData), string(message))
	}

	atomic.StoreInt64(a.sequence, e.Sequence)

	logging.Debug("[内部] イベント呼び出し") //TODO: いらない

	if eh, ok := registeredInterfaceProviders[e.Type]; ok {
		e.Struct = eh.New()

		if err := json.Unmarshal(e.RawData, e.Struct); err != nil {
			logging.Error("%s イベントをアンマーシャルできませんでした %s", e.Type, err)
		}

		a.handleEvent(e.Type, e.Struct)
	} else {
		logging.Warning("不明なイベント Op: %d, Seq: %d, Type: %s, Data: %s", e.Operation, e.Sequence, e.Type, string(e.RawData))
	}

	return e, nil
}
