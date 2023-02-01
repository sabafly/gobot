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
package main

import (
	"encoding/json"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
	"github.com/sabafly/gobot/pkg/lib/caches"
	"github.com/sabafly/gobot/pkg/lib/logging"
	"github.com/sabafly/gobot/pkg/lib/requests"
)

var createdGuilds *caches.CacheManager[discordgo.GuildCreate] = caches.NewCacheManager[discordgo.GuildCreate](nil)

// ギルド作成ハンダラ
func (h *WebsocketHandler) HandlerGuildCreate(w http.ResponseWriter, r *http.Request) {
	// データを取り出す
	guildCreate := discordgo.GuildCreate{}
	if err := requests.Unmarshal(r, &guildCreate); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
	}

	// キャッシュに保存
	createdGuilds.Set(guildCreate.ID, guildCreate)

	h.Broadcast(func(ws *websocket.Conn) {
		statusUpdate := struct{ Servers int }{Servers: createdGuilds.Len()}
		b, _ := json.Marshal(statusUpdate) //TODO: エラーハンドリング
		logging.Debug("[内部] ステータス更新イベント")
		err := ws.WriteJSON(Event{Operation: 8, Sequence: h.Seq + 1, Type: "STATUS_UPDATE", RawData: b})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		h.Seq++
	})

	err := json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}

func (h *WebsocketHandler) HandlerGuildDelete(w http.ResponseWriter, r *http.Request) {
	// データを取り出す
	guildDelete := discordgo.GuildDelete{}
	if err := requests.Unmarshal(r, &guildDelete); err != nil {
		logging.Error("[内部] [REST] アンマーシャルできませんでした %s", err)
		w.WriteHeader(400)
		err := json.NewEncoder(w).Encode(map[string]any{"status": "400 Bad Request"})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
	}

	// キャッシュに保存
	createdGuilds.Delete(guildDelete.ID)

	h.Broadcast(func(ws *websocket.Conn) {
		statusUpdate := struct{ Servers int }{Servers: createdGuilds.Len()}
		b, _ := json.Marshal(statusUpdate) //TODO: エラーハンドリング
		logging.Debug("[内部] ステータス更新イベント")
		err := ws.WriteJSON(Event{Operation: 8, Sequence: h.Seq + 1, Type: "STATUS_UPDATE", RawData: b})
		if err != nil {
			logging.Error("応答に失敗 %s", err)
		}
		h.Seq++
	})

	err := json.NewEncoder(w).Encode(map[string]any{"status": "200 OK"})
	if err != nil {
		logging.Error("応答に失敗 %s", err)
	}
}
