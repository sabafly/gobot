package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func commandBan(locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	defaultLocalizer = i18n.NewLocalizer(translations, locale.String())
	var banId string
	var banReason string
	for _, d := range option.Options {
		if d.Name == "target" {
			banId = d.UserValue(s).ID
		} else if d.Name == "reason" {
			banReason = translates(*locale, "command.ban.reason", map[string]interface{}{"Reason": d.StringValue()}, 1)
		}
	}

	// メッセージ&banの処理
	if banId != *ApplicationId {
		res.Content = translate(*locale, "command.ban.message", map[string]interface{}{
			"Target": "<@" + banId + ">",
		})
		if banReason != "" {
			res.Content += "\r" + banReason
			err := s.GuildBanCreateWithReason(gid, banId, banReason, 7)
			if err != nil {
				res.Content = translate(*locale, "error.0", map[string]interface{}{
					"Error": err,
				})
			}
		} else {
			err := s.GuildBanCreate(gid, banId, 7)
			if err != nil {
				res.Content = translate(*locale, "error.0", map[string]interface{}{
					"Error": err,
				})
			}
		}
	} else {
		res.Content = translate(*locale, "error.TargetIsBot", map[string]interface{}{})
	}
	return
}
