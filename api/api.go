package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	apinternal "github.com/sabafly/gobot/lib/api"
	"github.com/sabafly/gobot/lib/logging"
)

var (
	// TODO: コマンド引数で指定する
	address  = "localhost"
	port     = "8686"
	basePath = ""
	path     = "/api/v0"
)

func Run() {

	// ----------------------------------------------------------------
	// 内部API
	// ----------------------------------------------------------------

	// 内部APIを用意
	wh := NewWebSocketHandler()
	server := apinternal.NewServer()
	server.PageTree = &apinternal.Page{
		Path: "/api",
		Child: []*apinternal.Page{
			{
				Path: "/v0/",
				Child: []*apinternal.Page{
					{
						Path: "gateway",
						Handlers: []*apinternal.Handler{{
							Method: "GET",
							Handler: func(ctx *gin.Context) {
								err := json.NewEncoder(ctx.Writer).Encode(map[string]any{"URL": "ws://" + address + ":" + port + basePath + path + "/gateway/ws"})
								if err != nil {
									logging.Error("応答に失敗 %s", err)
								}
							},
						}},

						Child: []*apinternal.Page{
							{
								Path: "/ws",
								Handlers: []*apinternal.Handler{
									{
										Method:  "GET",
										Handler: func(ctx *gin.Context) { wh.Handle(ctx.Writer, ctx.Request) },
									},
								},
							},
						},
					},
					{
						Path: "guild",
						Handlers: []*apinternal.Handler{
							{
								Method:  "POST",
								Handler: func(ctx *gin.Context) { wh.HandlerGuildCreate(ctx.Writer, ctx.Request) },
							},
							{
								Method:  "DELETE",
								Handler: func(ctx *gin.Context) { wh.HandlerGuildDelete(ctx.Writer, ctx.Request) },
							},
						},
					},
					{
						Path: "message",
						Handlers: []*apinternal.Handler{
							{
								Method:  "POST",
								Handler: func(ctx *gin.Context) { wh.HandlerMessageCreate(ctx.Writer, ctx.Request) },
							},
						},
					},
					{
						Path: "statics/",

						Child: []*apinternal.Page{
							{
								Path: "user",

								Child: []*apinternal.Page{
									{
										Path: "/message",
										Handlers: []*apinternal.Handler{
											{
												Method:  "GET",
												Handler: func(ctx *gin.Context) { wh.HandlerStaticsUserMessage(ctx.Writer, ctx.Request) },
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// サーバー開始
	if err := server.Serve(":8686"); err != nil {
		logging.Fatal("[内部] APIを開始できませんでした %s", err)
	}
}
