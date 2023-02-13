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
	"os"
	"os/signal"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	apinternal "github.com/sabafly/gobot/pkg/api"
	gobot "github.com/sabafly/gobot/pkg/bot"
	"github.com/sabafly/gobot/pkg/lib/env"
	"github.com/sabafly/gobot/pkg/lib/logging"
)

var (
	address  = "localhost"
	port     = "8686"
	basePath = ""
	path     = "/api/v0"
)

func main() {

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

						Child: []*apinternal.Page{
							{
								Path: "feature",
								Handlers: []*apinternal.Handler{
									{
										Method:  "POST",
										Handler: func(ctx *gin.Context) {},
									},
								},
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
	go func() {
		if err := server.Serve(":8686"); err != nil {
			logging.Fatal("[内部] APIを開始できませんでした %s", err)
		}
	}()

	// ----------------------------------------------------------------
	// ボット
	// ----------------------------------------------------------------

	// ボットを用意
	bot, err := gobot.New(env.Token)
	if err != nil {
		logging.Fatal("failed create bot: %s", err)
	}

	// ハンダラ登録
	bot.AddApiHandler(func(a *gobot.Shard, s *gobot.StatusUpdate) {
		err := a.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{
				{
					Name: "/help | " + strconv.Itoa(s.Servers) + " Servers",
					Type: discordgo.ActivityTypeGame,
				},
			},
			Status: "online",
		})
		if err != nil {
			logging.Error("ステータス更新に失敗 %s", err)
		}
	})

	// コマンドハンダラ
	command := commands()
	bot.AddHandler(command.Parse())

	// ボットを開始
	if err := bot.Open(); err != nil {
		logging.Fatal("failed open bot: %s", err)
	}
	defer bot.Close()

	// コマンド登録
	registeredCommands, err := bot.ApplicationCommandCreate(command)
	if err != nil {
		panic(err)
	}

	if env.RemoveCommands {
		// コマンド削除
		defer func() {
			err := bot.ApplicationCommandDelete(registeredCommands)
			if err != nil {
				logging.Error("コマンド削除に失敗 %s", err)
			}
			err = bot.LocalApplicationCommandDelete()
			if err != nil {
				logging.Error("コマンド削除に失敗 %s", err)
			}
		}()
	}

	// シグナル待機
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	logging.Info("Ctrl+Cで終了")

	sig := <-sigCh

	logging.Info("受信: %v\n", sig.String())
}
