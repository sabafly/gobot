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
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/util"
)

func UserInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	gid := i.GuildID
	uid := i.ApplicationCommandData().TargetID
	m, err := util.ErrorCatch(s.State.Member(i.GuildID, uid))
	if err != nil {
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		}))
		return
	}
	var status string
	p, err := util.ErrorCatch(s.State.Presence(gid, uid))
	if err != nil {
		status = "Status: " + util.StatusString(discordgo.StatusOffline)
	} else {
		if p.Status != discordgo.StatusOffline {
			if str := util.StatusString(p.ClientStatus.Web); str != "" {
				status += translate.Message(i.Locale, "client_web") + ": " + str + "\r"
			}
			if str := util.StatusString(p.ClientStatus.Desktop); str != "" {
				status += translate.Message(i.Locale, "client_desktop") + ": " + str + "\r"
			}
			if str := util.StatusString(p.ClientStatus.Mobile); str != "" {
				status += translate.Message(i.Locale, "client_mobile") + ": " + str + "\r"
			}
		}
		if status == "" {
			status += translate.Message(i.Locale, "online_status") + ": " + util.StatusString(p.Status) + "\r"
		}
	}
	u, err := util.ErrorCatch(s.User(uid))
	if err != nil {
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				util.ErrorMessage(i.Locale, err).Embeds[0],
			},
		}))
		return
	}
	var roles string
	var color int = 0x000000
	role, _ := util.ErrorCatch(s.GuildRoles(i.GuildID))
	me, _ := util.ErrorCatch(s.GuildMember(i.GuildID, uid))
	var highestPosition int
	for _, v := range me.Roles {
		r, _ := s.State.Role(i.GuildID, v)
		if r.Position >= highestPosition {
			highestPosition = r.Position
		}
	}
	var r []*discordgo.Role
	for _, r2 := range role {
		for _, v := range me.Roles {
			if r2.ID == v {
				r = append(r, r2)
			}
		}
	}
	sort.Slice(r, func(i2, j int) bool {
		return r[i2].Position < r[j].Position
	})
	for _, v := range r {
		if v.Color != 0x000000 {
			color = v.Color
		}
	}
	for i2, j := 0, len(r)-1; i2 < j; i2, j = i2+1, j-1 {
		r[i2], r[j] = r[j], r[i2]
	}

	for _, v := range r {
		roles += v.Mention()
	}
	if roles == "" {
		roles = "`" + translate.Message(i.Locale, "message_command_user_info_none") + "`"
	}
	sColor := strconv.FormatInt(int64(color), 16)
	for utf8.RuneCountInString(sColor) < 6 {
		sColor = "0" + sColor
	}

	if m.Nick == "" {
		m.Nick = "`" + translate.Message(i.Locale, "message_command_user_info_none") + "`"
	}
	embed := discordgo.MessageEmbed{
		Title: u.Username + "#" + u.Discriminator,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:    m.AvatarURL("512"),
			Width:  512,
			Height: 512,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_nick"),
				Value:  m.Nick,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_id"),
				Value:  m.User.ID,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_roles"),
				Value:  roles,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_joined_at"),
				Value:  "<t:" + strconv.FormatInt(m.JoinedAt.Unix(), 10) + ":F>",
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_color_code"),
				Value:  sColor,
				Inline: true,
			},
			{
				Name:   translate.Message(i.Locale, "message_command_user_info_status"),
				Value:  status,
				Inline: true,
			},
		},
	}
	util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{&embed},
	}))
}
