package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/setup"
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
	RemoveCommands = false

	s, commands, RemoveCommands, GuildID = setup.Setup()
}

func Run() {
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

	defer s.Close()

	go updateStatus()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Ctrl + C で終了")
	<-stop

	if RemoveCommands {
		log.Println("コマンドを登録解除中...")
		// // We need to fetch the commands, since deleting requires the command ID.
		// // We are doing this from the returned commands on line 375, because using
		// // this will delete all the commands, which might not be desirable, so we
		// // are deleting only the commands that we added.
		// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
		// if err != nil {
		// 	log.Fatalf("Could not fetch registered commands: %v", err)
		// }

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, GuildID, v.ID)
			if err != nil {
				log.Panicf("'%v'コマンドを解除できません: %v", v.Name, err)
			}
		}
	}

	log.Println("正常にシャットダウンしました")
}

func updateStatus() {
	for {
		err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status: fmt.Sprintf("Shards %v / %v", s.ShardID+1, s.ShardCount),
			Activities: []*discordgo.Activity{
				{
					Name: fmt.Sprintf("Servers: %v,Shards: %v / %v", len(s.State.Guilds), s.ShardID+1, s.ShardCount),
					Type: discordgo.ActivityTypeWatching,
				},
			},
		})
		if err != nil {
			log.Printf("Error on update status: %v", err)
		}
		time.Sleep(time.Minute * 10)
	}
}
