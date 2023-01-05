package interaction

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
)

func ComponentPanelVoteCreateAdd(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
	_, err := util.ErrorCatch(session.VoteLoad(sessionID))
	if err != nil {
		return
	}
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: product.CommandPanelVoteCreateAddModal + ":" + sessionID,
			Flags:    discordgo.MessageFlagsEphemeral,
			Title:    translate.Message(i.Locale, "command_panel_vote_create_message_add_vote_choice"),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    product.CommandPanelVoteCreateAddModalTitle,
							Label:       translate.Message(i.Locale, "command_panel_vote_create_message_modal_title"),
							Style:       discordgo.TextInputShort,
							MaxLength:   32,
							Placeholder: translate.Message(i.Locale, "command_panel_vote_create_message_modal_title_desc"),
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    product.CommandPanelVoteCreateAddModalDescription,
							Label:       translate.Message(i.Locale, "command_panel_vote_create_message_modal_description"),
							Style:       discordgo.TextInputShort,
							MaxLength:   100,
							Placeholder: translate.Message(i.Locale, "command_panel_vote_create_message_modal_description_desc") + " (" + translate.Message(i.Locale, "command_panel_vote_create_message_modal_description_desc_optional") + ")",
							Required:    false,
						},
					},
				},
			},
		},
	}))
}

func ComponentPanelVoteCreateAddPreview(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	defer util.ErrorCatch("", s.InteractionResponseDelete(i.Interaction))
	data, err := util.ErrorCatch(session.VoteLoad(sessionID))
	if err != nil {
		return
	}
	d := data.Data()
	selections := []types.VoteSelection{}
	util.ErrorCatch("", json.Unmarshal(d.Vote.Selections, &selections))
	for i2, vs := range selections {
		if vs.ID == i.MessageComponentData().Values[0] {
			selections = selections[:i2+copy(selections[i2:], selections[i2+1:])]
		}
	}
	for i2 := range selections {
		selections[i2].Emoji = discordgo.ComponentEmoji{
			Name: util.ToEmojiA(i2 + 1),
			ID:   "",
		}
	}
	buf, _ := util.ErrorCatch(json.Marshal(selections))
	d.Vote.Selections = buf
	session.VoteSaveWithID(&d, data.ID())
	options := []discordgo.SelectMenuOption{}
	for i2, vs := range selections {
		options = append(options, discordgo.SelectMenuOption{
			Label:       vs.Name,
			Value:       vs.ID,
			Description: vs.Description,
			Emoji: discordgo.ComponentEmoji{
				Name: util.ToEmojiA(i2 + 1),
				ID:   "",
			},
		})
	}
	var addable bool
	if len(selections) >= 25 {
		addable = true
	}
	var menu bool
	if len(selections) < 1 {
		menu = true
		options = append(options, discordgo.SelectMenuOption{
			Label: "No selections were added",
			Value: "tmp",
		})
	}
	util.ErrorCatch(s.InteractionResponseEdit(d.InteractionCreate.Interaction, &discordgo.WebhookEdit{
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.StringSelectMenu,
						CustomID:    product.CommandPanelVoteCreatePreview + ":" + data.ID(),
						Disabled:    menu,
						Options:     options,
						Placeholder: translate.Message(i.Locale, "command_panel_vote_create_message_add_vote_choice"),
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: product.CommandPanelVoteCreateAdd + ":" + data.ID(),
						Style:    discordgo.SecondaryButton,
						Disabled: addable,
						Label:    translate.Message(i.Locale, "command_panel_vote_create_message_add"),
					},
					discordgo.Button{
						CustomID: product.CommandPanelVoteCreateDo + ":" + data.ID(),
						Style:    discordgo.PrimaryButton,
						Disabled: menu,
						Label:    translate.Message(i.Locale, "command_panel_vote_create_message_create"),
					},
				},
			},
		},
	}))
}

func ComponentPanelVoteCreateDo(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	defer util.ErrorCatch("", s.InteractionResponseDelete(i.Interaction))
	data, err := util.ErrorCatch(session.VoteLoad(sessionID))
	if err != nil {
		return
	}
	d := data.Data()

	selections := []types.VoteSelection{}
	json.Unmarshal(d.Vote.Selections, &selections)
	var fields []*discordgo.MessageEmbedField
	for _, v := range selections {
		f := &discordgo.MessageEmbedField{
			Name:   util.EmojiFormat(&v.Emoji) + " | " + v.Name,
			Value:  v.Description,
			Inline: true,
		}
		if f.Value == "" {
			f.Value = "`" + translate.Message(i.Locale, "command_panel_vote_create_message_no_desc") + "`"
		}
		if d.Vote.ShowCount {
			f.Name += " - " + translate.Translates(i.Locale, "command_panel_vote_votes", map[string]interface{}{"Vote": "0"}, 0)
		}
		fields = append(fields, f)
	}
	d.Vote.StartAt = time.Now()
	d.Vote.EndAt = time.Now().Add(d.Vote.Duration)
	options := []discordgo.SelectMenuOption{}
	for _, vs := range selections {
		options = append(options, discordgo.SelectMenuOption{
			Label:       vs.Name,
			Value:       vs.ID,
			Description: vs.Description,
			Emoji:       vs.Emoji,
		})
	}
	for len(options) < d.Vote.MaxSelection {
		d.Vote.MaxSelection--
	}
	for d.Vote.MaxSelection < d.Vote.MinSelection {
		d.Vote.MinSelection--
	}
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       d.Vote.Title,
			Description: d.Vote.Description,
			Fields:      fields,
		},
		{
			Title:       translate.Message(i.Locale, "command_panel_vote_create_message_vote_info"),
			Description: translate.Message(i.Locale, "command_panel_vote_create_message_min") + " " + strconv.Itoa(d.Vote.MinSelection) + " " + translate.Message(i.Locale, "command_panel_vote_create_message_max") + " " + strconv.Itoa(d.Vote.MaxSelection) + "\r" + translate.Message(i.Locale, "command_panel_vote_create_message_end_at") + " " + "<t:" + strconv.FormatInt(d.Vote.EndAt.Unix(), 10) + ":F>" + "(" + "<t:" + strconv.FormatInt(d.Vote.EndAt.Unix(), 10) + ":R>" + ")",
		},
	}
	s.InteractionResponseDelete(d.InteractionCreate.Interaction)
	m, err := util.ErrorCatch(s.ChannelMessageSendComplex(d.Vote.ChannelID, &discordgo.MessageSend{
		Embeds: embeds,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:  discordgo.StringSelectMenu,
						CustomID:  product.CommandPanelVote + ":" + d.Vote.VoteID,
						MinValues: &d.Vote.MinSelection,
						MaxValues: d.Vote.MaxSelection,
						Options:   options,
					},
				},
			},
		},
	}))
	if err != nil {
		return
	}
	d.Vote.MessageID = m.ID
	d.Vote.Locale = i.Locale.String()

	session.VoteRemove(sessionID)
	buf, err := util.ErrorCatch(json.Marshal(*d.Vote))
	if err != nil {
		return
	}
	_, err = util.ErrorCatch(api.ReqAPI(http.MethodPost, "/api/panel/vote", bytes.NewBuffer(buf)))
	if err != nil {
		return
	}
	createdVotePanel[d.Vote.MessageID] = d.Vote.VoteID
	if time.Until(d.Vote.EndAt) < time.Minute*30 {
		PanelVoteRemove(s, *d.Vote)
	}
}

var createdVotePanel map[string]string = map[string]string{}

func ComponentPanelVote(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	r, err := util.ErrorCatch(api.ReqAPI(http.MethodGet, "/api/panel/vote?id="+sessionID, http.NoBody))
	if err != nil {
		return
	}
	b, err := util.ErrorCatch(io.ReadAll(r.Body))
	if err != nil {
		return
	}
	res := types.Res{}
	util.ErrorCatch("", json.Unmarshal(b, &res))
	b, _ = util.ErrorCatch(json.Marshal(res.Content))
	d := types.VoteObject{}
	util.ErrorCatch("", json.Unmarshal(b, &d))
	selections := []types.VoteSelection{}
	util.ErrorCatch("", json.Unmarshal(d.Selections, &selections))
	var choice string
	for i2 := range selections {
		users := selections[i2].Users
		selections[i2].Users = []string{}
		for _, v := range users {
			if v != i.Member.User.ID {
				selections[i2].Users = append(selections[i2].Users, v)
			}
		}
		for _, v := range i.MessageComponentData().Values {
			if selections[i2].ID == v {
				selections[i2].Users = append(selections[i2].Users, i.Member.User.ID)
				choice += util.EmojiFormat(&selections[i2].Emoji) + " | " + selections[i2].Name + "\r"
			}
		}
	}
	locale := types.StL[d.Locale]
	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: d.ChannelID,
		ID:      d.MessageID,
	})
	var fields []*discordgo.MessageEmbedField
	for i2 := range selections {
		f := &discordgo.MessageEmbedField{
			Name:   util.EmojiFormat(&selections[i2].Emoji) + " | " + selections[i2].Name,
			Value:  selections[i2].Description,
			Inline: true,
		}
		if f.Value == "" {
			f.Value = "`" + translate.Message(locale, "command_panel_vote_create_message_no_desc") + "`"
		}
		if d.ShowCount {
			f.Name += " - " + translate.Translates(locale, "command_panel_vote_votes", map[string]interface{}{"Vote": strconv.Itoa(len(selections[i2].Users))}, len(selections[i2].Users))
		}
		fields = append(fields, f)
	}
	d.StartAt = time.Now()
	d.EndAt = time.Now().Add(d.Duration)
	d.Selections, _ = util.ErrorCatch(json.Marshal(selections))
	buf, _ := util.ErrorCatch(json.Marshal(d))
	api.ReqAPI(http.MethodPost, "/api/panel/vote", bytes.NewBuffer(buf))
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       d.Title,
			Description: d.Description,
			Fields:      fields,
		},
		{
			Title:       translate.Message(locale, "command_panel_vote_create_message_vote_info"),
			Description: translate.Message(locale, "command_panel_vote_create_message_min") + " " + strconv.Itoa(d.MinSelection) + " " + translate.Message(locale, "command_panel_vote_create_message_max") + " " + strconv.Itoa(d.MaxSelection) + "\r" + translate.Message(locale, "command_panel_vote_create_message_end_at") + " " + "<t:" + strconv.FormatInt(d.EndAt.Unix(), 10) + ":F>" + "(" + "<t:" + strconv.FormatInt(d.EndAt.Unix(), 10) + ":R>" + ")",
		},
	}
	util.ErrorCatch(s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      d.MessageID,
		Channel: d.ChannelID,
		Embeds:  embeds,
	}))
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       translate.Message(i.Locale, "command_panel_vote_message_voted"),
				Description: choice,
			},
		},
	})
}

var removeList []string

func PanelVoteRemove(s *discordgo.Session, v types.VoteObject) {
	for _, v2 := range removeList {
		if v2 == v.VoteID {
			return
		}
	}
	removeList = append(removeList, v.VoteID)
	time.Sleep(time.Until(v.EndAt))
	r, err := util.ErrorCatch(api.ReqAPI(http.MethodGet, "/api/panel/vote?id="+v.VoteID, http.NoBody))
	if err != nil {
		return
	}
	b, _ := util.ErrorCatch(io.ReadAll(r.Body))
	res := types.Res{}
	util.ErrorCatch("", json.Unmarshal(b, &res))
	b, _ = util.ErrorCatch(json.Marshal(res.Content))
	data := types.VoteObject{}
	_, err = util.ErrorCatch("", json.Unmarshal(b, &data))
	if err != nil {
		return
	}
	locale := types.StL[data.Locale]
	selections := []types.VoteSelection{}
	util.ErrorCatch("", json.Unmarshal(data.Selections, &selections))
	sort.Slice(selections, func(i, j int) bool {
		return len(selections[i].Users) > len(selections[j].Users)
	})
	field := []*discordgo.MessageEmbedField{}
	for i, vs := range selections {
		switch i {
		case 0:
			field = append(field, &discordgo.MessageEmbedField{
				Name:  translate.Message(locale, "command_panel_vote_message_winner"),
				Value: util.EmojiFormat(&vs.Emoji) + " | " + vs.Name + " - " + translate.Translates(locale, "command_panel_vote_votes", map[string]interface{}{"Vote": strconv.Itoa(len(vs.Users))}, len(vs.Users)),
			})
		case 1:
			var v string
			for _, vs := range selections {
				v += util.EmojiFormat(&vs.Emoji) + " | " + vs.Name + " - " + translate.Translates(locale, "command_panel_vote_votes", map[string]interface{}{"Vote": strconv.Itoa(len(vs.Users))}, len(vs.Users)) + "\r"
			}
			field = append(field, &discordgo.MessageEmbedField{
				Name:  translate.Message(locale, "command_panel_vote_message_result"),
				Value: v,
			})
		}
	}
	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    data.ChannelID,
		ID:         data.MessageID,
		Components: []discordgo.MessageComponent{},
	})
	s.ChannelMessageSendComplex(data.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:  translate.Message(locale, "command_panel_vote_message_finished"),
				Fields: field,
			},
		},
		Reference: &discordgo.MessageReference{
			MessageID: data.MessageID,
		},
	})
	delete(createdVotePanel, data.VoteID)
	util.ErrorCatch(api.ReqAPI(http.MethodDelete, "/api/panel/vote?id="+v.VoteID, http.NoBody))
}
