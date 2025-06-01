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

package message

import (
	"context"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func doTextCommand(ctx context.Context, event *events.GuildMessageCreate) (err error, shouldContinue bool) {
	if event.Message.Author.Bot || event.Message.WebhookID != nil {
		return nil, true
	}

	c, ok := strings.CutPrefix(event.Message.Content, discord.UserMention(event.Client().ApplicationID))
	if !ok {
		return nil, true
	}
	content := strings.Split(strings.TrimSpace(c), " ")

	switch {
	case diceRollRegex.MatchString(content[0]):
		subMatch := diceRollRegex.FindStringSubmatch(content[0])
		diceCount, err := strconv.Atoi(subMatch[1])
		if err != nil || diceCount < 1 || diceCount > 10 {
			return nil, true
		}
		diceSize, err := strconv.Atoi(subMatch[2])
		if err != nil || diceSize < 1 || diceSize > 1000 {
			return nil, true
		}
		content := "Dice Roll: "
		sum := 0
		for i := 0; i < diceCount; i++ {
			roll := diceRoll(diceSize)
			sum += roll
			content += strconv.Itoa(roll) + " "
		}

		content += "\nSum: " + strconv.Itoa(sum)

		_, err = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageBuilder().
			SetContent(content).
			SetMessageReferenceByID(event.Message.ID).
			BuildCreate(),
		)
		if err != nil {
			return err, false
		}
	}
	return nil, true
}

func diceRoll(size int) int {
	return rand.N(size) + 1
}

var (
	diceRollRegex = regexp.MustCompile(`^(\d+)[dｄ](\d+)$`)
)
