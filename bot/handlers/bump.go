package handlers

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/sabafly-disgo/discord"
	"github.com/sabafly/sabafly-disgo/events"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
)

func BumpUpMessage(b *botlib.Bot[*client.Client]) handler.Message {
	return handler.Message{
		Handler: bumpUpMessageHandler(b),
	}
}

func bumpUpMessageHandler(b *botlib.Bot[*client.Client]) func(event *events.GuildMessageCreate) error {
	return func(event *events.GuildMessageCreate) error {
		if event.Message.Interaction == nil || event.Message.ApplicationID == nil {
			return nil
		}
		if event.Message.Author.ID != 761562078095867916 && event.Message.Author.ID != 302050872383242240 {
			return nil
		}
		b.Self.GuildDataLock(event.GuildID).Lock()
		defer b.Self.GuildDataLock(event.GuildID).Unlock()
		gd, err := b.Self.DB.GuildData().Get(event.GuildID)
		if err != nil {
			return err
		}
		switch event.Message.Interaction.Name {
		case "bump": // disboard
			if !gd.BumpStatus.BumpEnabled {
				return nil
			}
			if event.Message.Embeds[0].Image.URL != "https://disboard.org/images/bot-command-image-bump.png" {
				return nil
			}
			gd.BumpStatus.LastBump = event.Message.CreatedAt
			gd.BumpStatus.LastBumpChannel = &event.ChannelID
			channelID := event.Message.ChannelID
			gd.BumpStatus.BumpCountMap[event.Message.Interaction.User.ID]++
			if gd.BumpStatus.BumpChannel != nil {
				channelID = *gd.BumpStatus.BumpChannel
			}

			ns := db.NewNoticeScheduleBump(false, event.GuildID, channelID, time.Now().Add(2*time.Hour))
			if err := b.Self.DB.NoticeSchedule().Set(ns.ID(), ns); err != nil {
				return err
			}

			message := discord.NewMessageCreateBuilder()
			embed := discord.NewEmbedBuilder()
			embed.SetTitle(gd.BumpStatus.BumpMessage[0])
			embed.SetDescription(gd.BumpStatus.BumpMessage[1])
			embed.Embed = botlib.SetEmbedProperties(embed.Embed)
			message.AddEmbeds(embed.Build())
			if _, err := b.Client.Rest().CreateMessage(event.Message.ChannelID, message.Build()); err != nil {
				b.Logger.Errorf("error on bump message: %s", err)
			}
		case "dissoku up": // dissoku
			if !gd.BumpStatus.UpEnabled {
				return nil
			}
			if len(event.Message.Embeds) < 1 || event.Message.Embeds[0].Color != 7506394 {
				return nil
			}
			gd.BumpStatus.LastUp = event.Message.CreatedAt
			gd.BumpStatus.LastUpChannel = &event.ChannelID
			channelID := event.Message.ChannelID
			gd.BumpStatus.UpCountMap[event.Message.Interaction.User.ID]++
			if gd.BumpStatus.UpChannel != nil {
				channelID = *gd.BumpStatus.UpChannel
			}

			ns := db.NewNoticeScheduleBump(true, event.GuildID, channelID, time.Now().Add(time.Hour))
			if err := b.Self.DB.NoticeSchedule().Set(ns.ID(), ns); err != nil {
				return err
			}

			message := discord.NewMessageCreateBuilder()
			embed := discord.NewEmbedBuilder()
			embed.SetTitle(gd.BumpStatus.UpMessage[0])
			embed.SetDescription(gd.BumpStatus.UpMessage[1])
			embed.Embed = botlib.SetEmbedProperties(embed.Embed)
			message.AddEmbeds(embed.Build())
			if _, err := b.Client.Rest().CreateMessage(event.Message.ChannelID, message.Build()); err != nil {
				b.Logger.Errorf("error on bump message: %s", err)
			}
		}
		if err := b.Self.DB.GuildData().Set(gd.ID, gd); err != nil {
			return err
		}
		return nil
	}
}

func BumpUpdateMessage(b *botlib.Bot[*client.Client]) handler.MessageUpdate {
	return handler.MessageUpdate{
		Handler: bumpUpdateHandler(b),
	}
}

func bumpUpdateHandler(b *botlib.Bot[*client.Client]) handler.MessageUpdateHandler {
	return func(event *events.GuildMessageUpdate) error {
		return bumpUpMessageHandler(b)(&events.GuildMessageCreate{GenericGuildMessage: event.GenericGuildMessage})
	}
}

func ScheduleBump(b *botlib.Bot[*client.Client], bp db.NoticeScheduleBump) error {
	b.Self.GuildDataLock(bp.GuildID).Lock()
	defer b.Self.GuildDataLock(bp.GuildID).Unlock()
	gd, err := b.Self.DB.GuildData().Get(bp.GuildID)
	if err != nil {
		return err
	}
	if !bp.IsUp {
		channelID := bp.ChannelID
		if gd.BumpStatus.BumpChannel != nil {
			channelID = *gd.BumpStatus.BumpChannel
		}
		go bumpScheduler(b, channelID, gd.BumpStatus.BumpRole, gd.BumpStatus.BumpRemind[0], gd.BumpStatus.BumpRemind[1], bp.ScheduledTime)
	} else {
		channelID := bp.ChannelID
		if gd.BumpStatus.UpChannel != nil {
			channelID = *gd.BumpStatus.UpChannel
		}
		go bumpScheduler(b, channelID, gd.BumpStatus.UpRole, gd.BumpStatus.UpRemind[0], gd.BumpStatus.UpRemind[1], bp.ScheduledTime)
	}
	return nil
}

func bumpScheduler(b *botlib.Bot[*client.Client], channelID snowflake.ID, roleID *snowflake.ID, title, desc string, tm time.Time) {
	time.Sleep(time.Until(tm))
	var mention string
	if roleID != nil {
		mention = discord.RoleMention(*roleID)
	}
	message := discord.NewMessageCreateBuilder()
	message.SetContent(mention)
	embed := discord.NewEmbedBuilder()
	embed.SetTitle(title)
	embed.SetDescription(desc)
	embed.Embed = botlib.SetEmbedProperties(embed.Embed)
	message.AddEmbeds(embed.Build())
	if _, err := b.Client.Rest().CreateMessage(channelID, message.Build()); err != nil {
		b.Logger.Errorf("error on bump message: %s", err)
	}
}
