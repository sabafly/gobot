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
	"runtime/debug"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	gobot "github.com/sabafly/gobot/pkg/bot"
	"github.com/sabafly/gobot/pkg/lib/constants"
	"github.com/sabafly/gobot/pkg/lib/translate"
)

// 埋め込みの色、フッター、タイムスタンプを設定する
func setEmbedProperties(embeds []*discordgo.MessageEmbed) []*discordgo.MessageEmbed {
	for i := range embeds {
		if embeds[i].Color == 0 {
			embeds[i].Color = constants.Color
		}
		if i == len(embeds)-1 {
			embeds[i].Footer = &discordgo.MessageEmbedFooter{
				Text: constants.BotName,
			}
			embeds[i].Timestamp = time.Now().Format(time.RFC3339)
		}
	}
	return embeds
}

// エラーメッセージ埋め込みを作成する
func ErrorTraceEmbed(locale discordgo.Locale, err error) []*discordgo.MessageEmbed {
	stack := debug.Stack()
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "💥" + translate.Message(locale, "error_occurred_embed_message"),
			Description: fmt.Sprintf("%s\r```%s```", err, string(stack)),
			Color:       0xff0000,
		},
	}
	embeds = setEmbedProperties(embeds)
	return embeds
}

// エラーが発生したことを返すレスポンスを作成する
func ErrorRespond(i *discordgo.InteractionCreate, err error) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: ErrorTraceEmbed(i.Locale, err),
		},
	}
}

// 渡されたステータスの絵文字を返す
func StatusString(status discordgo.Status) (str string) {
	switch status {
	case discordgo.StatusOnline:
		return "<:online:1055430359363354644>"
	case discordgo.StatusDoNotDisturb:
		return "<:dnd:1055434290629980220>"
	case discordgo.StatusIdle:
		return "<:idle:1055433789020586035> "
	case discordgo.StatusInvisible:
		return "<:offline:1055434315514785792>"
	case discordgo.StatusOffline:
		return "<:offline:1055434315514785792>"
	}
	return ""
}

// アクティビティ名をアクティビティの種類によって渡された言語に翻訳して返す
func ActivitiesNameString(locale discordgo.Locale, activity *discordgo.Activity) (str string) {
	switch activity.Type {
	case discordgo.ActivityTypeGame:
		str = translate.Translate(locale, "activity_game_name", map[string]any{"Name": activity.Name})
	case discordgo.ActivityTypeStreaming:
		str = translate.Translate(locale, "activity_streaming_name", map[string]any{"Details": activity.Details, "URL": activity.URL})
	case discordgo.ActivityTypeListening:
		str = translate.Translate(locale, "activity_listening_name", map[string]any{"Name": activity.Name})
	case discordgo.ActivityTypeWatching:
		str = translate.Translate(locale, "activity_watching_name", map[string]any{"Name": activity.Name})
	case discordgo.ActivityTypeCustom:
		str = activity.Emoji.MessageFormat() + " " + activity.Name
	case discordgo.ActivityTypeCompeting:
		str = translate.Translate(locale, "activity_competing_name", map[string]any{"Name": activity.Name})
	}
	return str
}

func MessageLogDetails(m []gobot.MessageLog) (day, week, all int, channelID string) {
	var inDay, inWeek []gobot.MessageLog
	channelCount := map[string]int{}
	for _, ml := range m {
		channelCount[ml.ChannelID]++
		timestamp, err := discordgo.SnowflakeTimestamp(ml.ID)
		if err != nil {
			continue
		}
		if timestamp.After(time.Now().Add(-time.Hour * 24 * 7)) {
			inWeek = append(inWeek, ml)
		}
	}
	for _, ml := range inWeek {
		timestamp, err := discordgo.SnowflakeTimestamp(ml.ID)
		if err != nil {
			continue
		}
		if timestamp.After(time.Now().Add(-time.Hour * 24)) {
			inDay = append(inDay, ml)
		}
	}
	count := []struct {
		ID    string
		Count int
	}{}
	for k, v := range channelCount {
		count = append(count, struct {
			ID    string
			Count int
		}{ID: k, Count: v})
	}
	sort.Slice(count, func(i, j int) bool {
		return count[i].Count > count[j].Count
	})
	if len(count) != 0 {
		channelID = count[0].ID
	}
	return len(inDay), len(inWeek), len(m), channelID
}
