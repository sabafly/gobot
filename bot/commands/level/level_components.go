/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package level

import (
	"crypto/sha1"
	"encoding/hex"
	"log/slog"
	"math/rand/v2"
	"slices"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/internal/embeds"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
)

func upMessageModalHandler(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	g, err := c.GuildCreateID(event, *event.GuildID())
	if err != nil {
		return errors.NewError(err)
	}
	g.LevelUpMessage = event.Data.Text("message")
	if err := c.GormDB().Save(g).Error; err != nil {
		return errors.NewError(err)
	}

	embed := discord.NewEmbedBuilder().
		SetTitle(translate.Message(event.Locale(), "components.level.up.message.message")).
		SetDescription(g.LevelUpMessage).
		Build()

	if err := event.CreateMessage(
		discord.NewMessageBuilder().
			SetEmbeds(embeds.SetEmbedProperties(embed)).
			BuildCreate(),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func eventHandler(c *components.Components, event bot.Event) errors.Error {
	switch event := event.(type) {
	case *events.GuildMessageCreate:
		if event.Message.Author.Bot || event.Message.Author.System || event.Message.Type.System() {
			return nil
		}
		if event.Message.Type != discord.MessageTypeDefault && event.Message.Type != discord.MessageTypeReply {
			return nil
		}
		g, err := c.GuildCreateID(event, event.GuildID)
		if err != nil || g.LevelingDisabled || slices.Contains(g.LevelUpExcludeChannel, event.ChannelID) {
			return nil
		}
		var channel discord.GuildChannel
		ch, ok := event.Channel()
		if !ok {
			ch2, _ := event.Client().Rest.GetChannel(event.ChannelID)
			channel, _ = ch2.(discord.GuildChannel)
		} else {
			channel = ch
		}
		if channel != nil && channel.ParentID() != nil && slices.Contains(g.LevelUpExcludeChannel, *channel.ParentID()) {
			return nil
		}
		m, err := c.MemberCreate(event, event.Message.Author, event.GuildID)
		if err != nil {
			return errors.NewError(err)
		}
		hash := sha1.Sum([]byte(event.Message.Content))
		hashStr := hex.EncodeToString(hash[:])
		if slices.Contains(m.LastMessageHashes, hashStr) {
			return nil
		}
		if len(m.LastMessageHashes) >= 10 {
			m.LastMessageHashes = slices.Delete(m.LastMessageHashes, 0, 1)
		}
		m.LastMessageHashes = append(m.LastMessageHashes, hashStr)
		if err := c.GormDB().Save(m).Error; err != nil {
			return errors.NewError(err)
		}

		if _, err = addXp(event, rand.N[uint64](16)+15, event.Client(), m, g, event.ChannelID, event.Message.Author.EffectiveName(), false, c.GormDB()); err != nil {
			return errors.NewError(err)
		}

		if err := gopoint.AddPoint(c, m.UserID, g.ID, rand.Int64N(2)*50); err != nil {
			slog.Error("ポイント追加に失敗", slog.Any("err", err), slog.Any("user_id", m.UserID), slog.Any("guild_id", g.ID))
		}

	}
	return nil
}
