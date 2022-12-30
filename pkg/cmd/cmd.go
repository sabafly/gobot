/*
	Copyright (C) 2022  ikafly144

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
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command/reg"
	"github.com/ikafly144/gobot/pkg/env"
	session "github.com/ikafly144/gobot/pkg/init"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/worker"
)

var VERSION = "Development Version"

var (
	s                  *discordgo.Session              = session.Session()
	commands           []*discordgo.ApplicationCommand = reg.Commands()
	GuildID            string                          = *env.GuildID
	RemoveCommands     bool                            = *env.RemoveCommands
	registeredCommands []*discordgo.ApplicationCommand
)

func Run() {
	s.UserAgent = "DiscordBot(https://github.com/ikafly144/gobot, " + VERSION + ")"
	s.Identify.Properties.Browser = product.ProductName + " " + VERSION
	fmt.Printf("\n<%v>: Version: %v\n", product.ProductName, VERSION)
	s.ShardID = 0
	s.ShardCount = 1
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("%v#%v としてログインしました", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("セッションを開始できません: %v", err)
	}
	s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{
			{
				Name: fmt.Sprintf("起動準備 | %v Servers | Shard %v/%v | %v", len(s.State.Guilds), s.ShardID+1, s.ShardCount, VERSION),
				Type: discordgo.ActivityTypeGame,
			},
		},
		Status: string(discordgo.StatusDoNotDisturb),
	})

	go regCommand()

	defer end()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	log.Println("Ctrl+Cで終了")

	s := <-sigCh

	log.Printf("受信: %v\n", s.String())
}

func end() {
	if RemoveCommands {
		log.Println("コマンドを登録解除中...")

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, GuildID, v.ID)
			if err != nil {
				log.Panicf("'%v'コマンドを解除できません: %v", v.Name, err)
			}
		}
		c, _ := s.ApplicationCommands(s.State.User.ID, "")
		for _, v := range c {
			err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
			if err != nil {
				log.Panicf("'%v'コマンドを解除できません: %v", v.Name, err)
			}
		}

		cs, _ := s.ApplicationCommands(s.State.User.ID, *env.SupportGuildID)
		for _, v := range cs {
			s.ApplicationCommandDelete(s.State.User.ID, v.GuildID, v.ID)
		}

	}
	s.Close()
	log.Println("正常にシャットダウンしました")
	os.Exit(0)
}

func regCommand() {

	log.Println("コマンドを追加中...")
	registeredCommands = make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, GuildID, v)
		if err != nil {
			log.Panicf("'%v'コマンドを追加できません: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}
	var (
		dmPermission                 = false
		PermissionAdminMembers int64 = discordgo.PermissionAdministrator
	)
	s.ApplicationCommandCreate(s.State.User.ID, *env.SupportGuildID, &discordgo.ApplicationCommand{
		Name:                     "admin",
		Description:              "only for bot admins",
		DMPermission:             &dmPermission,
		DefaultMemberPermissions: &PermissionAdminMembers,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "ban",
				Description: "only for bot admins",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "add",
						Description: "only for bot admins",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandOption{
							{
								Name:        "target",
								Description: "only for bot admins",
								Type:        discordgo.ApplicationCommandOptionString,
								Required:    true,
							},
							{
								Name:        "reason",
								Description: "only for bot admins",
								Type:        discordgo.ApplicationCommandOptionString,
								Required:    false,
							},
						},
					},
					{
						Name:        "remove",
						Description: "only for bot admins",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandOption{
							{
								Name:        "target",
								Description: "only for bot admins",
								Type:        discordgo.ApplicationCommandOptionString,
								Required:    true,
							},
						},
					},
					{
						Name:        "get",
						Description: "only for admins",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
				},
			},
			{
				Name:        "servers",
				Description: "only for admins",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "get",
						Description: "only for admins",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
				},
			},
		},
		Version: "1",
	})

	log.Print("完了")

	go updateStatus()
	go autoBans()
}

func updateStatus() {
	for {
		err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{
				{
					Name: fmt.Sprintf("WIP | %v Servers | Shard %v/%v | %v", len(s.State.Guilds), s.ShardID+1, s.ShardCount, VERSION),
					Type: discordgo.ActivityTypeGame,
				},
			},
			Status: string(discordgo.StatusOnline),
		})
		if err != nil {
			log.Printf("Error on update status: %v", err)
		}
		time.Sleep(time.Second * 30)
	}
}

func autoBans() {
	go worker.DeleteBanListener()
	for {
		worker.MakeBan(s)
		time.Sleep(time.Minute)
	}
}
