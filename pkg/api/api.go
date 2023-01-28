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
package apinternal

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sabafly/gobot/pkg/lib/database"
	"github.com/sabafly/gobot/pkg/lib/env"
	"github.com/sabafly/gobot/pkg/lib/logging"
)

// TODO: mainパッケージで組み立てるべき

var (
	g        = gin.New()
	address  = "localhost"
	port     = "8686"
	basePath = ""
	path     = "/api/v0"
)

var db *database.DatabaseManager
var wh = NewWebSocketHandler()

func init() {
	db = database.NewDatabase()
	if err := db.Connect(env.DBHost, env.DBPort, env.DBUser, env.DBPass, env.DBName, 2); err != nil {
		logging.Fatal("データベースに接続できませんでした %s", err)
	}
}

// APIサーバーを開始する
func Serve() {
	// ginを初期化
	g.RouterGroup = *g.Group(path)

	// ハンダラを登録
	g.Handle(http.MethodGet, "/gateway", func(ctx *gin.Context) {
		json.NewEncoder(ctx.Writer).Encode(map[string]interface{}{"URL": "ws://" + address + ":" + port + basePath + path + "/gateway/ws"})
	})
	g.Handle("GET", "/gateway/ws", func(ctx *gin.Context) { wh.Handle(ctx.Writer, ctx.Request) })
	g.Handle("POST", "/guild/create", func(ctx *gin.Context) { wh.HandlerGuildCreate(ctx.Writer, ctx.Request) })

	// サーバー開始
	go func() {
		if err := g.Run(":8686"); err != nil {
			logging.Fatal("[内部] APIを開始できませんでした %s", err)
		}
	}()
}
