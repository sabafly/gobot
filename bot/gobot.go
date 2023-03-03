/*
	Copyright (C) 2022-2023  sabafly

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
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	botlib "github.com/sabafly/gobot/lib/bot"
	"github.com/sabafly/gobot/lib/env"
	"github.com/sabafly/gobot/lib/logging"
)

func Run() {
	// ----------------------------------------------------------------
	// ボット
	// ----------------------------------------------------------------

	// ボットを用意
	bot, err := botlib.New(env.Token)
	if err != nil {
		logging.Fatal("failed create bot: %s", err)
	}

	bot.Api.AddHandler(func(a *botlib.Api, s *botlib.StatusUpdate) {
		if err := bot.Client.SetPresence(context.TODO(),
			gateway.WithOnlineStatus(discord.OnlineStatusOnline),
			gateway.WithPlayingActivity(fmt.Sprintf("/help | %d Servers", s.Servers)),
		); err != nil {
			logging.Error("ステータス更新に失敗 %s", err)
		}
	})

	// ボットを開始
	if err := bot.Open(); err != nil {
		logging.Fatal("failed open bot: %s", err)
	}
	defer bot.Close()

	// シグナル待機
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	logging.Info("Ctrl+Cで終了")

	sig := <-sigCh

	logging.Info("受信: %v\n", sig.String())
}
