package command

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/util"
)

func Ban(s *discordgo.Session, locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	var banId string
	var banReason string
	for _, d := range option.Options {
		if d.Name == "target" {
			banId = d.UserValue(s).ID
		} else if d.Name == "reason" {
			banReason = translate.Translates(*locale, "command.ban.reason", map[string]interface{}{"Reason": d.StringValue()}, 1)
		}
	}

	// メッセージ&banの処理
	if banId != s.State.User.ID {
		res.Content = translate.Translate(*locale, "command.ban.message", map[string]interface{}{
			"Target": "<@" + banId + ">",
		})
		if banReason != "" {
			res.Content += "\r" + banReason
			err := s.GuildBanCreateWithReason(gid, banId, banReason, 7)
			if err != nil {
				res = util.ErrorMessage(*locale, err)
			}
		} else {
			err := s.GuildBanCreate(gid, banId, 7)
			if err != nil {
				res = util.ErrorMessage(*locale, err)
			}
		}
	} else {
		res.Content = translate.Message(*locale, "error.TargetIsBot")
		res.Flags = discordgo.MessageFlagsEphemeral
	}
	return
}

func UnBan(s *discordgo.Session, locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	var kickId string
	for _, d := range option.Options {
		if d.Name == "target" {
			kickId = d.UserValue(s).ID
		}
	}

	if kickId != s.State.User.ID {
		res.Content = translate.Translate(*locale, "command.unban.message", map[string]interface{}{
			"Target": "<@" + kickId + ">",
		})
		err := s.GuildBanDelete(gid, kickId)
		if err != nil {
			res = util.ErrorMessage(*locale, err)
		}
	} else {
		res.Content = translate.Message(*locale, "error.TargetIsBot")
		res.Flags = discordgo.MessageFlagsEphemeral
	}
	return
}

func Kick(s *discordgo.Session, locale *discordgo.Locale, option discordgo.ApplicationCommandInteractionData, gid string) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{Content: "ERR"}
	var kickId string
	for _, d := range option.Options {
		if d.Name == "target" {
			kickId = d.UserValue(s).ID
		}
	}

	if kickId != s.State.User.ID {
		res.Content = translate.Translate(*locale, "command.kick.message", map[string]interface{}{
			"Target": "<@" + kickId + ">",
		})
		err := s.GuildMemberDelete(gid, kickId)
		if err != nil {
			res = util.ErrorMessage(*locale, err)
		}
	} else {
		res.Content = translate.Message(*locale, "error.TargetIsBot")
		res.Flags = discordgo.MessageFlagsEphemeral
	}
	return
}
