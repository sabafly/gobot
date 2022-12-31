/*
	Copyright (C) 2022  ikafly144

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package interaction

import (
	"encoding/json"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
)

var selects map[types.MessageSelect]*discordgo.Message = make(map[types.MessageSelect]*discordgo.Message)

func GetSelectingMessage(uid string, gid string) (mes *discordgo.Message, err error) {
	id := types.MessageSelect{
		MemberID: uid,
		GuildID:  gid,
	}
	if m, ok := selects[id]; ok {
		mes = m
		return
	} else {
		err = errors.New("no message is selected")
		return
	}
}

func MessageSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	data := &discordgo.ApplicationCommandInteractionData{}
	byte, _ := util.ErrorCatch(json.Marshal(i.Interaction.Data))
	util.ErrorCatch("", json.Unmarshal(byte, data))
	mes, _ := util.ErrorCatch(s.ChannelMessage(i.ChannelID, data.TargetID))
	id := types.MessageSelect{
		MemberID: i.Member.User.ID,
		GuildID:  i.GuildID,
	}
	selects[id] = mes
	str := translate.Message(i.Locale, "message_command_select_message")
	util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
	}))
	util.DeferDeleteInteraction(s, i)
}

func RemoveSelect(uid string, gid string) {
	id := types.MessageSelect{
		MemberID: uid,
		GuildID:  gid,
	}
	delete(selects, id)
}
