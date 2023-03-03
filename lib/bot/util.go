/*
	Copyright (C) 2022-2023  sabafly

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
package botlib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"sort"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/lib/constants"
	"github.com/sabafly/gobot/lib/logging"
	"github.com/sabafly/gobot/lib/translate"
)

// 埋め込みの色、フッター、タイムスタンプを設定する
func SetEmbedProperties(embeds []discord.Embed) []discord.Embed {
	now := time.Now()
	for i := range embeds {
		if embeds[i].Color == 0 {
			embeds[i].Color = constants.Color
		}
		if i == len(embeds)-1 {
			embeds[i].Footer = &discord.EmbedFooter{
				Text: constants.BotName,
			}
			embeds[i].Timestamp = &now
		}
	}
	return embeds
}

// エラーメッセージ埋め込みを作成する
func ErrorMessageEmbed(locale discord.Locale, t string) []discord.Embed {
	embeds := []discord.Embed{
		{
			Title:       translate.Message(locale, t+"_title"),
			Description: translate.Message(locale, t+"_message"),
			Color:       0xff0000,
		},
	}
	embeds = SetEmbedProperties(embeds)
	return embeds
}

// エラートレース埋め込みを作成する
func ErrorTraceEmbed(locale discord.Locale, err error) []discord.Embed {
	stack := debug.Stack()
	embeds := []discord.Embed{
		{
			Title:       "💥" + translate.Message(locale, "error_occurred_embed_message"),
			Description: fmt.Sprintf("%s\r```%s```", err, string(stack)),
			Color:       0xff0000,
		},
	}
	embeds = SetEmbedProperties(embeds)
	return embeds
}

// エラーが発生したことを返すレスポンスを作成する
func ErrorRespond(locale discord.Locale, err error) *discord.Message {
	return &discord.Message{
		Embeds: ErrorTraceEmbed(locale, err),
	}
}

// 渡されたステータスの絵文字を返す
func StatusString(status discord.OnlineStatus) (str string) {
	switch status {
	case discord.OnlineStatusOnline:
		return "<:online:1055430359363354644>"
	case discord.OnlineStatusDND:
		return "<:dnd:1055434290629980220>"
	case discord.OnlineStatusIdle:
		return "<:idle:1055433789020586035> "
	case discord.OnlineStatusInvisible:
		return "<:offline:1055434315514785792>"
	case discord.OnlineStatusOffline:
		return "<:offline:1055434315514785792>"
	}
	return ""
}

// アクティビティ名をアクティビティの種類によって渡された言語に翻訳して返す
func ActivitiesNameString(locale discord.Locale, activity discord.Activity) (str string) {
	switch activity.Type {
	case discord.ActivityTypeGame:
		str = translate.Translate(locale, "activity_game_name", map[string]any{"Name": activity.Name})
	case discord.ActivityTypeStreaming:
		str = translate.Translate(locale, "activity_streaming_name", map[string]any{"Details": activity.Details, "URL": activity.URL})
	case discord.ActivityTypeListening:
		str = translate.Translate(locale, "activity_listening_name", map[string]any{"Name": activity.Name})
	case discord.ActivityTypeWatching:
		str = translate.Translate(locale, "activity_watching_name", map[string]any{"Name": activity.Name})
	case discord.ActivityTypeCustom:
		if activity.Emoji != nil {
			return
		}
		str = discord.EmojiMention(*activity.Emoji.ID, activity.Emoji.Name) + " " + activity.Name
	case discord.ActivityTypeCompeting:
		str = translate.Translate(locale, "activity_competing_name", map[string]any{"Name": activity.Name})
	}
	return str
}

func MessageLogDetails(m []MessageLog) (day, week, all int, channelID snowflake.ID) {
	var inDay, inWeek []MessageLog
	channelCount := map[snowflake.ID]int{}
	for _, ml := range m {
		channelCount[ml.ChannelID]++
		timestamp := ml.ID.Time()
		if timestamp.After(time.Now().Add(-time.Hour * 24 * 7)) {
			inWeek = append(inWeek, ml)
		}
	}
	for _, ml := range inWeek {
		timestamp := ml.ID.Time()
		if timestamp.After(time.Now().Add(-time.Hour * 24)) {
			inDay = append(inDay, ml)
		}
	}
	count := []struct {
		ID    snowflake.ID
		Count int
	}{}
	for k, v := range channelCount {
		count = append(count, struct {
			ID    snowflake.ID
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

func WaitModalSubmit(i *events.ApplicationCommandInteractionCreate, title string, container []discord.TextInputComponent) (st *events.ModalSubmitInteractionCreate, err error) {
	customID := uuid.NewString()
	ctx := context.Background()
	ctx, cancelCtx := context.WithDeadline(ctx, time.Now().Add(time.Minute*3)) //TODO: タイムアウトをカスタマイズ可能に
	defer cancelCtx()
	handler := func(m *events.ModalSubmitInteractionCreate) {
		if m.Data.CustomID != customID {
			return
		}
		st = m
		cancelCtx()
	}
	listener := bot.NewListenerFunc(handler)
	i.Client().AddEventListeners(listener)
	i.Client().RemoveEventListeners(listener)
	builder := discord.NewModalCreateBuilder().
		SetTitle(title).
		SetCustomID(customID)
	for _, tic := range container {
		builder = builder.AddActionRow(tic)
	}
	err = i.CreateModal(builder.Build())
	if err != nil {
		return nil, err
	}
	<-ctx.Done()
	if ctx.Err() != nil {
		return nil, err
	}
	return st, nil
}

func SendWebhook(client bot.Client, channelID snowflake.ID, data discord.WebhookMessageCreate) (st *discord.Message, err error) {
	webhooks, err := client.Rest().GetWebhooks(channelID)
	if err != nil {
		return nil, err
	}
	me, ok := client.Caches().SelfUser()
	if !ok {
		logging.Error("セルフ取得に失敗しました %s", err)
		return
	}
	var token string
	var webhook discord.Webhook = nil
	for _, w := range webhooks {
		if w.Type() != discord.WebhookTypeIncoming {
			continue
		}
		buf, err := w.MarshalJSON()
		if err != nil {
			continue
		}
		data := discord.IncomingWebhook{}
		json.Unmarshal(buf, &data)
		if data.User.ID == client.ID() {
			token = data.Token
			webhook = data
		}
	}
	if webhook == nil {
		var buf []byte
		if avatarURL := me.EffectiveAvatarURL(discord.WithFormat(discord.ImageFormatPNG)); avatarURL != "" {
			resp, err := http.Get(avatarURL)
			if err != nil {
				return nil, err
			}
			buf, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
		}
		data, err := client.Rest().CreateWebhook(channelID, discord.WebhookCreate{
			Name:   "gobot-webhook",
			Avatar: discord.NewIconRaw(discord.IconTypePNG, buf),
		})
		if err != nil {
			return nil, err
		}
		token = data.Token
		webhook = data
	}
	if data.Username == "" {
		data.Username = me.Username
	}
	if data.AvatarURL == "" {
		data.AvatarURL = me.EffectiveAvatarURL(discord.WithFormat(discord.ImageFormatPNG))
	}
	st, err = client.Rest().CreateWebhookMessage(webhook.ID(), token, data, true, snowflake.ID(0))
	if err != nil {
		return nil, err
	}
	return st, nil
}
