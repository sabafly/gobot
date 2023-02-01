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
	"sync"

	"github.com/andersfylling/snowflake/v5"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// TODO: コメントを書く

// APIサーバーの構造体
type Server struct {
	sync.RWMutex

	Conn []*Connection

	gin   *gin.Engine
	Pages []*Page
}

// ウェブソケット接続を扱う
type Connection struct {
	*websocket.Conn
	ID snowflake.Snowflake
}

// ページの構造を表す構造体
//
// ハンダラがnilだった場合、Parse時に無視される
type Page struct {
	Method  string
	Path    string
	Handler func(*gin.Context)

	Child []*Page
}

// 新たなサーバー構造を生成する
func NewServer() *Server {
	return &Server{
		gin: gin.New(),
	}
}

// ページを解析してサーバーを起動する
func (s *Server) Serve(addr ...string) (err error) {
	for _, p := range s.Pages {
		p.Parse(s.gin)
	}
	return s.gin.Run(addr...)
}

// ページ構造を解析してginエンジンに登録する
func (p *Page) Parse(g *gin.Engine) {
	p.parse(p.Method, p.Path, p.Handler, g)
	for _, p2 := range p.Child {
		p2.parse(p2.Method, p2.Path, p.Handler, g)
	}
}

// ginエンジンにページハンダラを登録する
func (p *Page) parse(method, path string, handler func(*gin.Context), g *gin.Engine) {
	if handler != nil {
		g.Handle(method, path, handler)
	}
	for _, p2 := range p.Child {
		p2.parse(p2.Method, path+p2.Path, p2.Handler, g)
	}
}
