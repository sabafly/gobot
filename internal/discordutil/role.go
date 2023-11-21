package discordutil

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func GetHighestRolePosition(role map[snowflake.ID]discord.Role) (int, snowflake.ID) {
	var max int
	var id snowflake.ID
	for i, r := range role {
		if max < r.Position {
			max = r.Position
			id = i
		}
	}
	return max, id
}
