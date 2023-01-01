package interaction

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
)

func MessagePin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	defer util.ErrorCatch("", s.InteractionResponseDelete(i.Interaction))
	data := &discordgo.ApplicationCommandInteractionData{}
	byte, _ := util.ErrorCatch(json.Marshal(i.Interaction.Data))
	util.ErrorCatch("", json.Unmarshal(byte, data))
	m, _ := util.ErrorCatch(s.ChannelMessage(i.ChannelID, data.TargetID))
	res, err := util.ErrorCatch(api.ReqAPI(http.MethodGet, "/api/message/pin?channel="+m.ChannelID, http.NoBody))
	if err != nil {
		return
	}
	b, _ := io.ReadAll(res.Body)
	r := types.Res{}
	json.Unmarshal(b, &r)
	b, _ = json.Marshal(r.Content)
	d := types.MessagePin{}
	json.Unmarshal(b, &d)
	webhookID, webhookToken := util.WebhookExec(s, m.ChannelID)
	util.ErrorCatch("", s.WebhookMessageDelete(webhookID, webhookToken, d.MessageID))
	b, _ = json.Marshal(m.Embeds)
	pin := types.MessagePin{
		ChannelID: m.ChannelID,
		UserID:    m.Author.ID,
		UserName:  m.Author.Username,
		UserIcon:  m.Author.AvatarURL("512"),
		Content:   m.Content,
		Embeds:    b,
	}
	embed := []*discordgo.MessageEmbed{}
	json.Unmarshal(pin.Embeds, &embed)
	if m.Content == "" && len(embed) == 0 {
		pin.Content = "`" + translate.Message(i.Locale, "error_cannot_read_content") + "`"
	}
	ms, err := util.ErrorCatch(s.WebhookExecute(webhookID, webhookToken, true, &discordgo.WebhookParams{
		Username:  pin.UserName,
		AvatarURL: pin.UserIcon,
		Content:   pin.Content,
		Embeds:    embed,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		},
	}))
	if err != nil {
		return
	}
	pin.MessageID = ms.ID
	buf, _ := util.ErrorCatch(json.Marshal(pin))
	messagePinDelete(m.ChannelID)
	api.ReqAPI(http.MethodPost, "/api/message/pin", bytes.NewReader(buf))
}

func MessagePinExec(s *discordgo.Session, m *discordgo.MessageCreate) {
	res, err := util.ErrorCatch(api.ReqAPI(http.MethodGet, "/api/message/pin?channel="+m.ChannelID, http.NoBody))
	if err != nil {
		return
	}
	b, _ := io.ReadAll(res.Body)
	r := types.Res{}
	json.Unmarshal(b, &r)
	b, _ = json.Marshal(r.Content)
	data := types.MessagePin{}
	json.Unmarshal(b, &data)
	if data.MessageID != m.ID {
		webhookID, webhookToken := util.WebhookExec(s, data.ChannelID)
		go util.ErrorCatch("", s.WebhookMessageDelete(webhookID, webhookToken, data.MessageID))
		embed := []*discordgo.MessageEmbed{}
		json.Unmarshal(data.Embeds, &embed)
		ms, err := util.ErrorCatch(s.WebhookExecute(webhookID, webhookToken, true, &discordgo.WebhookParams{
			Content:   data.Content,
			Username:  data.UserName,
			AvatarURL: data.UserIcon,
			Embeds:    embed,
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{},
			},
		}))
		if err != nil {
			return
		}
		data.MessageID = ms.ID
		buf, _ := util.ErrorCatch(json.Marshal(data))
		messagePinDelete(data.ChannelID)
		api.ReqAPI(http.MethodPost, "/api/message/pin", bytes.NewReader(buf))
	}
}

func MessagePinDelete(channelID string, messageID ...string) {
	res, err := util.ErrorCatch(api.ReqAPI(http.MethodGet, "/api/message/pin?channel="+channelID, http.NoBody))
	if err != nil {
		return
	}
	b, _ := io.ReadAll(res.Body)
	r := types.Res{}
	json.Unmarshal(b, &r)
	b, _ = json.Marshal(r.Content)
	data := types.MessagePin{}
	json.Unmarshal(b, &data)
	for _, v := range messageID {
		if data.MessageID == v {
			messagePinDelete(channelID)
		}
	}
}

func messagePinDelete(channelID string) {
	util.ErrorCatch(api.ReqAPI(http.MethodDelete, "/api/message/pin?channel="+channelID, http.NoBody))
}
