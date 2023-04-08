package botlib

import "github.com/disgoorg/snowflake/v2"

func (b *Bot) CheckDev(id snowflake.ID) bool {
	for _, i := range b.Config.DevUserIDs {
		if id == i {
			return true
		}
	}
	for _, i := range b.Config.DevGuildIDs {
		if id == i {
			return true
		}
	}
	return false
}
