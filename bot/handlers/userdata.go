package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
)

func UserDataMessage(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		Handler: userDataMessageHandler(b),
	}
}

func userDataMessageHandler(b *botlib.Bot[*client.Client]) func(event *events.GuildMessageCreate) error {
	return func(event *events.GuildMessageCreate) error {
		if event.Message.Author.System || event.Message.Author.Bot {
			return nil
		}
		if !b.Self.UserDataLock(event.Message.Author.ID).TryLock() {
			return nil
		}
		defer b.Self.UserDataLock(event.Message.Author.ID).Unlock()
		u, err := b.Self.DB.UserData().Get(event.Message.Author.ID)
		if err != nil {
			return err
		}
		if !u.LastMessageTime.After(event.Message.CreatedAt.Add(-time.Duration((time.Now().Minute() % 5) * int(time.Minute)))) {
			u.GlobalLevel.AddRandom()
			u.LastMessageTime = event.Message.CreatedAt
			u.GlobalMessageLevel.AddRandom()
			if err := b.Self.DB.UserData().Set(u.ID, u); err != nil {
				return err
			}
		}
		if b.Self.GuildDataLock(event.GuildID).TryLock() {
			defer b.Self.GuildDataLock(event.GuildID).Unlock()
			gd, err := b.Self.DB.GuildData().Get(event.GuildID)
			if err != nil {
				return err
			}
			if !gd.UserLevels[event.Message.Author.ID].LastMessageTime.After(event.Message.CreatedAt.Add(-time.Duration((time.Now().Minute() % 5) * int(time.Minute)))) {
				ul := gd.UserLevels[event.Message.Author.ID]
				before := ul.Level()
				ul.AddRandom()
				after := ul.Level()
				if before < after {
					// メッセージを送る処理
					mes := strings.ReplaceAll(gd.Config.LevelUpMessage, "{mention}", discord.UserMention(event.Message.Author.ID))
					mes = strings.ReplaceAll(mes, "{username}", event.Message.Author.EffectiveName())
					mes = strings.ReplaceAll(mes, "{level}", strconv.FormatUint(after, 10))
					mes = strings.ReplaceAll(mes, "{level_before}", strconv.FormatUint(before, 10))
					channelID := event.ChannelID
					if gd.Config.LevelUpMessageChannel != nil {
						channelID = *gd.Config.LevelUpMessageChannel
					}
					if _, err := event.Client().Rest().CreateMessage(channelID, discord.MessageCreate{Content: mes}); err != nil {
						return err
					}
				}
				ul.LastMessageTime = event.Message.CreatedAt
				gd.UserLevels[event.Message.Author.ID] = ul
			}
			if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
				return err
			}
		}
		return nil
	}
}