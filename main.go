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
	"os"
	"os/signal"
	"strconv"

	"github.com/bwmarrin/discordgo"
	apinternal "github.com/sabafly/gobot/pkg/api"
	gobot "github.com/sabafly/gobot/pkg/bot"
	"github.com/sabafly/gobot/pkg/lib/env"
	"github.com/sabafly/gobot/pkg/lib/logging"
)

func main() {
	apinternal.Serve()

	// ボットを用意
	bot, err := gobot.New(env.Token)
	if err != nil {
		logging.Fatal("failed create bot: %s", err)
	}

	bot.AddApiHandler(func(a *gobot.Shard, s *gobot.StatusUpdate) {
		a.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{
				{
					Name: "/help | " + strconv.Itoa(s.Servers) + " Servers",
					Type: discordgo.ActivityTypeGame,
				},
			},
			Status: "online",
		})
	})

	// ボットを開始
	if err := bot.Open(); err != nil {
		logging.Fatal("failed open bot: %s", err)
	}
	defer bot.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	logging.Info("Ctrl+Cで終了")

	sig := <-sigCh

	logging.Info("受信: %v\n", sig.String())
}
