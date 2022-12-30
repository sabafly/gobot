package env

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	BotToken       = flag.String("token", "", "botアクセストークン")
	RemoveCommands = flag.Bool("rm", true, "停止時にコマンドを登録解除するか")
	GuildID        = flag.String("guild", "", "コマンドを追加するギルドのID（空白でグローバル）")
	SupportGuildID = flag.String("support", "", "管理ギルドのID")
	APIServer      = flag.String("api", "", "APIサーバーのip")
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Failed to load .env:%v", err)
	}
	*BotToken = os.Getenv("TOKEN")
	*GuildID = os.Getenv("GUILD_ID")
	*RemoveCommands, err = strconv.ParseBool(os.Getenv("REMOVE_COMMANDS"))
	if err != nil {
		*RemoveCommands = true
	}
	*APIServer = os.Getenv("API_SERVER")
	*SupportGuildID = os.Getenv("SUPPORT_ID")

	flag.Parse()
}
