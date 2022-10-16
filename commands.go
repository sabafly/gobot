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
	var err error
	for _, d := range option.Options {
		if d.Name == "target" {
			banId = d.UserValue(s).ID
		} else if d.Name == "reason" {
			banReason, err = defaultLocalizer.Localize(&i18n.LocalizeConfig{
				MessageID: "command.ban.reason",
				TemplateData: map[string]interface{}{
					"Reason": d.StringValue(),
				},
				PluralCount: 1,
			})
			if err != nil {
				res.Content, _ = defaultLocalizer.Localize(&i18n.LocalizeConfig{
					MessageID: "error.0",
					TemplateData: map[string]interface{}{
						"Error": err,
					},
				})
			}
		}
	}
	res.Content, err = defaultLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID: "command.ban.message",
		TemplateData: map[string]interface{}{
			"Target": "<@" + banId + ">",
		},
	})
	if err != nil {
		res.Content, _ = defaultLocalizer.Localize(&i18n.LocalizeConfig{
			MessageID: "error.0",
			TemplateData: map[string]interface{}{
				"Error": err,
			},
		})
	}
	if banReason != "" {
		res.Content += "\r" + banReason
		err := s.GuildBanCreateWithReason(gid, banId, banReason, 7)
		if err != nil {
			res.Content, _ = defaultLocalizer.Localize(&i18n.LocalizeConfig{
				MessageID: "error.0",
				TemplateData: map[string]interface{}{
					"Error": err,
				},
			})
		}
	} else {
		err := s.GuildBanCreate(gid, banId, 7)
		if err != nil {
			res.Content, _ = defaultLocalizer.Localize(&i18n.LocalizeConfig{
				MessageID: "error.0",
				TemplateData: map[string]interface{}{
					"Error": err,
				},
			})
		}
	}
	return
}
