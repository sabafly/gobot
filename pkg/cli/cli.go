package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/setup"
	"github.com/ikafly144/gobot/pkg/worker"
)

var (
	s              *discordgo.Session
	commands       = []*discordgo.ApplicationCommand{}
	GuildID        string
	RemoveCommands bool
)

func init() {
	s = &discordgo.Session{}
	commands = []*discordgo.ApplicationCommand{}
	GuildID = ""
	RemoveCommands = true

	s, commands, RemoveCommands, GuildID = setup.Setup()
}

func Run() {
	s.ShardID = 0
	s.ShardCount = 1
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("%v#%v としてログインしました", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("セッションを開始できません: %v", err)
	}

	log.Println("コマンドを追加中...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
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
	s.ApplicationCommandCreate(s.State.User.ID, *setup.SupportGuildID, &discordgo.ApplicationCommand{
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
		},
		Version: "1",
	})

	defer end(registeredCommands)

	go updateStatus()
	go autoBans()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Ctrl + C で終了")
	<-stop

	log.Println("正常にシャットダウンしました")
}

func end(registeredCommands []*discordgo.ApplicationCommand) {
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

	}
	s.Close()
}

func updateStatus() {
	for {
		err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{
				{
					Name: fmt.Sprintf("WIP | %v Servers | Shard %v/%v", len(s.State.Guilds), s.ShardID+1, s.ShardCount),
					Type: discordgo.ActivityTypeGame,
				},
			},
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
