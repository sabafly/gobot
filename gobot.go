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
package gobot

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/bwmarrin/discordgo"
	botlib "github.com/sabafly/gobot-lib/bot"
	"github.com/sabafly/gobot-lib/env"
	"github.com/sabafly/gobot-lib/logging"
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

	bot.FeatureCreate(&botlib.Feature{
		Name:         "Pressure",
		ID:           "TYPING_START_PRESSURE",
		IDType:       botlib.FeatureChannelID,
		ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
		Type:         botlib.FeatureTypingStart,
		Handler: func(s *discordgo.Session, ts *discordgo.TypingStart) {
			user, err := s.GuildMember(ts.GuildID, ts.UserID)
			if err != nil {
				return
			}
			s.ChannelMessageSendComplex(ts.ChannelID, &discordgo.MessageSend{
				Embeds: []*discordgo.MessageEmbed{
					{
						Author: &discordgo.MessageEmbedAuthor{
							IconURL: user.AvatarURL(""),
							Name:    fmt.Sprintf("<@%s>が入力を始めた！", ts.UserID),
						},
					},
				},
			})
		},
	})

	bot.FeatureApplicationCommandSettingsSet(botlib.FeatureApplicationCommandSettings{
		Name:         "feature",
		Description:  "manage feature",
		Permission:   discordgo.PermissionAdministrator,
		DMPermission: false,
	})

	// ハンダラ登録
	bot.AddHandler(bot.FeatureHandler())
	bot.AddApiHandler(func(a *botlib.Shard, s *botlib.StatusUpdate) {
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
	command = append(command, &botlib.ApplicationCommand{
		ApplicationCommand: bot.FeaturesApplicationCommand(),
		Handler:            bot.FeatureApplicationCommandHandler(),
	})
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
