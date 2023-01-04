package interaction

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
)

func ModalPanelVoteCreateAdd(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	defer util.ErrorCatch("", s.InteractionResponseDelete(i.Interaction))
	data, err := util.ErrorCatch(session.VoteLoad(sessionID))
	if err != nil {
		return
	}
	var title string
	var description string
	ms := i.ModalSubmitData()
	for _, mc := range ms.Components {
		if mc.Type() == discordgo.ActionsRowComponent {
			bytes, _ := util.ErrorCatch(mc.MarshalJSON())
			data := &discordgo.ActionsRow{}
			util.ErrorCatch("", json.Unmarshal(bytes, data))
			bytes, _ = util.ErrorCatch(data.Components[0].MarshalJSON())
			text := &discordgo.TextInput{}
			util.ErrorCatch("", json.Unmarshal(bytes, text))
			switch text.CustomID {
			case product.CommandPanelVoteCreateAddModalTitle:
				title = text.Value
			case product.CommandPanelVoteCreateAddModalDescription:
				description = text.Value
			}
		}
	}
	d := data.Data()
	selections := []types.VoteSelection{}
	util.ErrorCatch("", json.Unmarshal(d.Vote.Selections, &selections))
	selections = append(selections, types.VoteSelection{
		ID: uuid.New().String(),
		Emoji: discordgo.ComponentEmoji{
			Name: util.ToEmojiA(len(selections) + 1),
			ID:   "",
		},
		Name:        title,
		Description: description,
		Users:       []string{},
	})
	buf, _ := util.ErrorCatch(json.Marshal(selections))
	d.Vote.Selections = buf
	session.VoteSaveWithID(&d, data.ID())
	options := []discordgo.SelectMenuOption{}
	for _, vs := range selections {
		options = append(options, discordgo.SelectMenuOption{
			Label:       vs.Name,
			Value:       vs.ID,
			Description: vs.Description,
			Emoji:       vs.Emoji,
		})
	}
	var addable bool
	if len(selections) >= 25 {
		addable = true
	}
	util.ErrorCatch(s.InteractionResponseEdit(d.InteractionCreate.Interaction, &discordgo.WebhookEdit{
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.StringSelectMenu,
						CustomID:    product.CommandPanelVoteCreatePreview + ":" + data.ID(),
						Options:     options,
						Placeholder: "Add choices",
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: product.CommandPanelVoteCreateAdd + ":" + data.ID(),
						Style:    discordgo.SecondaryButton,
						Disabled: addable,
						Label:    "Add",
					},
					discordgo.Button{
						CustomID: product.CommandPanelVoteCreateDo + ":" + data.ID(),
						Style:    discordgo.PrimaryButton,
						Label:    "Create",
					},
				},
			},
		},
	}))
}
