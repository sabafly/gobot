/*
	Copyright (C) 2022-2023  ikafly144

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
package main

import (
	"fmt"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/sabafly/gobot/pkg/lib/logging"
	"github.com/sabafly/gobot/pkg/lib/translate"
)

// ----------------------------------------------------------------
// テキストコマンド
// ----------------------------------------------------------------

func CommandTextPing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: setEmbedProperties([]*discordgo.MessageEmbed{
				{
					Title: "🏓 pong!",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Discord API",
							Value: s.HeartbeatLatency().String(),
						},
					},
				},
			}),
		},
	})
	if err != nil {
		logging.Error("インタラクション応答に失敗 %s", err)
	}
}

// ----------------------------------------------------------------
// ユーザーコマンド
// ----------------------------------------------------------------

func CommandUserInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// ユーザー情報を取得
	user, err := s.GuildMember(i.GuildID, i.ApplicationCommandData().TargetID)
	if err != nil {
		err := s.InteractionRespond(i.Interaction, ErrorRespond(i, err))
		if err != nil {
			logging.Error("インタラクションレスポンスに失敗 %s", err)
		}
		return
	}

	// 返信メッセージを用意
	var response *discordgo.InteractionResponseData
	var embeds []*discordgo.MessageEmbed
	var fields []*discordgo.MessageEmbedField

	//ステータス取得
	var statusStr string
	status, err := s.State.Presence(i.GuildID, i.ApplicationCommandData().TargetID)
	if err != nil {
		logging.Warning("ユーザーステータスの取得に失敗 %s", err)
		statusStr = translate.Message(i.Locale, "online_status") + ": " + StatusString(discordgo.StatusOffline)
	} else {
		if status.Status != discordgo.StatusOffline {
			if str := StatusString(status.ClientStatus.Web); str != "" {
				statusStr += translate.Message(i.Locale, "client_web") + ": " + str + "\r"
			}
			if str := StatusString(status.ClientStatus.Desktop); str != "" {
				statusStr += translate.Message(i.Locale, "client_desktop") + ": " + str + "\r"
			}
			if str := StatusString(status.ClientStatus.Mobile); str != "" {
				statusStr += translate.Message(i.Locale, "client_mobile") + ": " + str + "\r"
			}
		}
		if statusStr == "" {
			statusStr += translate.Message(i.Locale, "online_status") + ": " + StatusString(status.Status) + "\r"
		}
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   translate.Message(i.Locale, "command_user_info_status"),
		Value:  statusStr,
		Inline: true,
	})

	// ユーザーアバターを取得
	avatarURL := user.AvatarURL("512")

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   translate.Message(i.Locale, "command_user_info_avatar"),
		Value:  avatarURL,
		Inline: true,
	})

	// ユーザーロールを取得
	var roles []*discordgo.Role
	for _, roleID := range user.Roles {
		role, err := s.State.Role(i.GuildID, roleID)
		if err != nil {
			err := s.InteractionRespond(i.Interaction, ErrorRespond(i, err))
			if err != nil {
				logging.Error("インタラクションレスポンスに失敗 %s", err)
			}
			return
		}
		roles = append(roles, role)
	}

	// ロールをソート
	sort.Slice(roles, func(i2, j int) bool {
		return roles[i2].Position > roles[j].Position
	})

	// ロール一覧
	var roleStr string
	for _, r := range roles {
		roleStr += r.Mention()
	}
	roleStr += fmt.Sprintf("<@&%s>", i.GuildID)

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   translate.Message(i.Locale, "command_user_info_role"),
		Value:  roleStr,
		Inline: true,
	})

	// ロールカラー取得
	var color int
	for i2, j := 0, len(roles)-1; i2 < j; i2, j = i2+1, j-1 {
		roles[i2], roles[j] = roles[j], roles[i2]
	}
	for _, r := range roles {
		if r.Color != 0 {
			color = r.Color
		}
	}
	colorStr := strconv.FormatInt(int64(color), 16)
	for utf8.RuneCountInString(colorStr) < 6 {
		colorStr = "0" + colorStr
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   translate.Message(i.Locale, "command_user_info_color"),
		Value:  colorStr,
		Inline: true,
	})

	// ユーザー名
	title := fmt.Sprintf("%s#%s (%s)", user.User.Username, user.User.Discriminator, user.User.ID)
	if user.Nick != "" {
		title += fmt.Sprintf("\r%s %s", translate.Message(i.Locale, "command_user_info_nick"), user.Nick)
	}

	// ユーザーの日時情報を取得
	created, err := discordgo.SnowflakeTimestamp(user.User.ID)
	if err != nil {
		err := s.InteractionRespond(i.Interaction, ErrorRespond(i, err))
		if err != nil {
			logging.Error("インタラクションレスポンスに失敗 %s", err)
		}
		return
	}
	joined := user.JoinedAt
	description := fmt.Sprintf("%s\r - %s: <t:%d:F> (<t:%d:R>)\r - %s: <t:%d:F> (<t:%d:R>)",
		translate.Message(i.Locale, "command_user_info_time"),
		translate.Message(i.Locale, "command_user_info_time_created"),
		created.Unix(), created.Unix(),
		translate.Message(i.Locale, "command_user_info_time_joined"),
		joined.Unix(), joined.Unix())

	// 埋め込み組み立て
	embeds = append(embeds, &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: avatarURL},
		Fields:      fields,
	})
	embeds = setEmbedProperties(embeds)

	// 応答送信
	response = &discordgo.InteractionResponseData{
		Embeds: embeds,
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: response,
	}); err != nil {
		err := s.InteractionRespond(i.Interaction, ErrorRespond(i, err))
		if err != nil {
			logging.Error("インタラクションレスポンスに失敗 %s", err)
		}
		return
	}
}
