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

	"github.com/ikafly144/gobot/pkg/lib/logger"
)

func Serve() {
	http.HandleFunc("/api/v0/gateway", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"URL": "ws://localhost:8686/api/v0/gateway/ws"})
	})
	http.HandleFunc("/api/v0/gateway/ws", NewWebSocketHandler().Handle)
	go func() {
		if err := http.ListenAndServe(":8686", nil); err != nil {
			logger.Fatal("[内部] APIを開始できませんでした %s", err)
		}
	}()
}
