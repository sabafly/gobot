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
package api

import (
	"encoding/json"
	"net/http"

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

type WebsocketHandler struct{}

func NewWebSocketHandler() *WebsocketHandler {
	return &WebsocketHandler{}
}

func (h *WebsocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	logger.Debug("called")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Fatal("[内部] アップグレードできませんでした")
	}
	message := Event{
		Operation: 1,
		RawData:   json.RawMessage{},
	}

	b, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{"Hello World"})
	message.RawData = b

	b, _ = json.Marshal(message)

	if err := ws.WriteMessage(1, b); err != nil {
		logger.Error("[内部] WebSocket呼び出しに失敗 %s", err)
	}

	data := Event{}
	if err := ws.ReadJSON(&data); err != nil {
		logger.Error("[内部] JSON読み込みに失敗 %s", err)
	}
	logger.Debug("[内部] 受信 %s", data)
	go h.handlerLoop(ws)
}

func (h *WebsocketHandler) handlerLoop(ws *websocket.Conn) {
	for {
		data := Event{}
		ws.ReadJSON(&data)
		logger.Info("[内部] 受信 %s", data)
	}
}
