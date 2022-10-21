package util

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/translate"
)

func ErrorMessage(locale discordgo.Locale, err error) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	res.Content = ""
	res.Embeds = append(res.Embeds, &discordgo.MessageEmbed{
		Title:       translate.Message(locale, "error.message"),
		Description: err.Error(),
		Color:       0xff0000,
	})
	res.Flags = discordgo.MessageFlagsEphemeral
	return
}

func DeepcopyJson(src interface{}, dst interface{}) (err error) {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, dst)
	if err != nil {
		return err
	}
	return nil
}
