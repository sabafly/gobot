package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func commandBan(locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	var banId string
	var banReason string
	for _, d := range option.Options {
		if d.Name == "target" {
			banId = d.UserValue(s).ID
		} else if d.Name == "reason" {
			banReason = "理由: " + d.StringValue()
		}
	}
	res.Content = "<@" + banId + ">" + "をbanしました"
	if banReason != "" {
		res.Content = res.Content + "\r" + banReason
		err := s.GuildBanCreateWithReason(gid, banId, banReason, 7)
		if err != nil {
			res.Content = "エラーが発生しました\r" + fmt.Sprint(err)
		}
	} else {
		err := s.GuildBanCreate(gid, banId, 7)
		if err != nil {
			res.Content = "エラーが発生しました\r" + fmt.Sprint(err)
		}
	}
	return
}
