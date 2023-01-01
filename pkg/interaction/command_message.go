package interaction

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/util"
)

func CommandMessage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	for _, acido := range i.ApplicationCommandData().Options {
		switch acido.Name {
		case "embed":
			messageEmbed(s, i, acido.Options)
		case "webhook":
			messageWebhook(s, i, acido.Options)
		}
	}
}

func messageEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, option []*discordgo.ApplicationCommandInteractionDataOption) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	var embed_title string
	var embed_content string
	var content string
	var username string = i.Member.User.Username
	var icon string = i.Member.AvatarURL("512")
	var color int64
	for _, acido := range option {
		switch acido.Name {
		case "embed_title":
			embed_title = acido.StringValue()
		case "embed_content":
			embed_content = acido.StringValue()
		case "username":
			username = acido.StringValue()
		case "icon_url":
			icon = acido.StringValue()
		case "content":
			content = acido.StringValue()
		case "color":
			color, _ = util.ErrorCatch(strconv.ParseInt("0x"+acido.StringValue(), 0, 32))
		}
	}
	embed := []*discordgo.MessageEmbed{}
	if embed_title != "" {
		embed = append(embed, &discordgo.MessageEmbed{
			Title:       embed_title,
			Description: embed_content,
			Color:       int(color),
		})
	}
	util.ErrorCatch("", s.InteractionResponseDelete(i.Interaction))
	wid, wt := util.WebhookExec(s, i.ChannelID)
	util.ErrorCatch(s.WebhookExecute(wid, wt, true, &discordgo.WebhookParams{
		Username:  username,
		AvatarURL: icon,
		Content:   content,
		Embeds:    embed,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	}))
}

func messageWebhook(s *discordgo.Session, i *discordgo.InteractionCreate, option []*discordgo.ApplicationCommandInteractionDataOption) {
	messageEmbed(s, i, option)
}
