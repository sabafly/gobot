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
package env

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	// ボットトークン
	Token string

	// アプリケーションコマンドを追加するギルドのID (空白でグローバルに追加)
	GuildID string

	// 管理ギルド用コマンドを追加するギルドのID
	AdminID string

	// 停止時にアプリケーションコマンドを削除
	RemoveCommands bool

	// ログレベル (ERROR, WARN, INFO, DEBUG)
	LogLevel  string
	DLogLevel int

	// DB接続オプション
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Panic(fmt.Errorf("failed load .env file: %W", err))
	}
	Token = os.Getenv("TOKEN")
	GuildID = os.Getenv("GUILD_ID")
	AdminID = os.Getenv("ADMIN_ID")
	var err error
	RemoveCommands, err = strconv.ParseBool(os.Getenv("REMOVE_COMMANDS"))
	if err != nil {
		log.Panic(fmt.Errorf("failed load REMOVE_COMMANDS in .env file: %w", err))
	}
	LogLevel = os.Getenv("LOG_LEVEL")

	// DiscordGo用のログレベル
	l := os.Getenv("D_LOG_LEVEL")
	switch l {
	case "INFO", "info":
		DLogLevel = 2
	case "DEBUG", "debug":
		DLogLevel = 3
	case "ERROR", "error":
		DLogLevel = 0
	case "WARNING", "warning":
		DLogLevel = 1
	default:
		DLogLevel = 2
	}

	DBHost = os.Getenv("DB_HOST")
	DBPort = os.Getenv("DB_PORT")
	DBUser = os.Getenv("DB_USER")
	DBPass = os.Getenv("DB_PASS")
	DBName = os.Getenv("DB_NAME")
}
