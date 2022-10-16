package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	GuildID        = flag.String("guild", "", "テストサーバーID")
	BotToken       = flag.String("Token", "", "botアクセストークン")
	RemoveCommands = flag.Bool("rmcmd", true, "停止時にコマンドを登録解除するか")
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env:%v", err)
	}
	*BotToken = os.Getenv("TOKEN")
}

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("無効なbotパラメータ: %v", err)
	}
}

var (
	// integerOptionMinValue          = 1.0
	// dmPermission                   = false
	// defaultMemberPermissions int64 = discordgo.PermissionManageServer

	commands = []*discordgo.ApplicationCommand{}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
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
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("'%v'コマンドを追加できません: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Ctrl + C で終了")
	<-stop

	if *RemoveCommands {
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
			err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("'%v'コマンドを解除できません: %v", v.Name, err)
			}
		}
	}

	log.Println("正常にシャットダウンしました")
}
