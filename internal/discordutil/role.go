package discordutil

import (
	"github.com/disgoorg/disgo/discord"
	"slices"
)

func GetHighestRole(roles []discord.Role) *discord.Role {
	slices.SortStableFunc(roles, func(a, b discord.Role) int {
		return a.Compare(b)
	})
	if len(roles) < 1 {
		return nil
	}
	return &roles[0]
}
