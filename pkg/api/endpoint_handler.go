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
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "400 Bad Request"})
	}

	// キャッシュに保存
	createdGuilds.Set(guildCreate.ID, guildCreate)

	h.Broadcast(func(ws *websocket.Conn) {
		statusUpdate := struct{ Servers int }{Servers: createdGuilds.Len()}
		b, _ := json.Marshal(statusUpdate) //TODO: エラーハンドリング
		logging.Debug("[内部] ステータス更新イベント")
		ws.WriteJSON(Event{Operation: 8, Sequence: h.Seq + 1, Type: "STATUS_UPDATE", RawData: b})
		wh.Seq++
	})

	json.NewEncoder(w).Encode(map[string]interface{}{"status": "200 OK"})
}
