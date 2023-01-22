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
	"github.com/ikafly144/gobot/pkg/lib/logger"
)

type Event struct {
	Operation int             `json:"op"`
	Sequence  int64           `json:"s"`
	Type      string          `json:"t"`
	RawData   json.RawMessage `json:"d"`
	// Structはこのファイルのその他の型の一つを含む
	// TODO:よくわからん
	Struct interface{} `json:"-"`
}

type heartbeatOp struct {
	Op   int   `json:"op"`
	Data int64 `json:"d"`
}

func (a *Api) Open() (err error) {
	logger.Debug("called")

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

	logger.Info("connecting to gateway %s", a.gateway)
	header := http.Header{}
	header.Add("accept-encoding", "zlib")
	a.wsConn, _, err = a.Dialer.Dial(a.gateway, header)
	if err != nil {
		logger.Error("ゲートウェイに接続できませんでした %s, %s", a.gateway, err)
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

	return nil
}

func (a *Api) Close() (err error) {
	return nil
}

func (a *Api) onEvent(messageType int, message []byte) (*Event, error) {
	var reader io.Reader
	reader = bytes.NewBuffer(message)

	if messageType == websocket.BinaryMessage {

		z, err := zlib.NewReader(reader)
		if err != nil {
			logger.Error("ウェブソケットメッセージを展開できませんでした %s", err)
			return nil, err
		}

		defer func() {
			if err := z.Close(); err != nil {
				logger.Warning("zlibをクローズできませんでした %s", err)
			}
		}()

		reader = z
	}

	var e *Event
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&e); err != nil {
		logger.Error("ウェブソケットメッセージをデコードできませんでした %s", err)
		return e, err
	}

	logger.Debug("Op: %d, Seq: %d, Type: %s, Data: %s\n\n", e.Operation, e.Sequence, e.Type, string(e.RawData))

	if e.Operation == 1 {
		logger.Debug("[内部呼び出し] Op1ハートビートの応答を送信します")
		a.wsMutex.Lock()
		err := a.wsConn.WriteJSON(heartbeatOp{1, atomic.LoadInt64(a.sequence)})
		a.wsMutex.Unlock()
		if err != nil {
			logger.Error("[内部呼び出し] Op1ハートビートの応答に失敗")
			return e, err
		}

		return e, nil
	}

	return e, nil
}
