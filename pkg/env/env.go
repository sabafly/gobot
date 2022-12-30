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
