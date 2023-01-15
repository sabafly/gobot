module github.com/ikafly144/gobot

go 1.19

require (
	github.com/BurntSushi/toml v1.2.1
	github.com/Tnze/go-mc v1.19.2
	github.com/bwmarrin/discordgo v0.26.1
	github.com/dlclark/regexp2 v1.8.0
	github.com/google/uuid v1.3.0
	github.com/joho/godotenv v1.4.0
	github.com/millkhan/mcstatusgo/v2 v2.2.0
	github.com/nicksnyder/go-i18n/v2 v2.2.1
	golang.org/x/text v0.6.0
	gorm.io/gorm v1.24.3
)

require (
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
)

replace github.com/bwmarrin/discordgo => ./discordgo
