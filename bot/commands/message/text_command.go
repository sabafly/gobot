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
		if err != nil || diceSize < 1 || diceSize > 10000 {
			return nil, true
		}
		var content strings.Builder
		content.WriteString("Dice Roll: ")
		sum := 0
		for range diceCount {
			roll := diceRoll(diceSize)
			sum += roll
			content.WriteString(strconv.Itoa(roll) + " ")
		}

		content.WriteString("\nSum: " + strconv.Itoa(sum))

		_, err = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageBuilder().
			SetContent(content.String()).
			SetMessageReferenceByID(event.Message.ID).
			BuildCreate(),
		)
		if err != nil {
			return err, false
		}
	case strings.EqualFold(content[0], "slot"):
		role := []string{"NONE", "GRAPES", "WATERMELON", "CHERRIES", "LEMON", "ORANGE", "PLUM", "BELL", "BAR", "SEVEN"}
		roleWeights := []int{1400, 30, 25, 20, 15, 10, 8, 5, 3, 1}
		totalWeight := 0
		for _, w := range roleWeights {
			totalWeight += w
		}

		getRole := func() string {
			r := rand.N(totalWeight)
			accumulatedWeight := 0
			for i, w := range roleWeights {
				accumulatedWeight += w
				if r < accumulatedWeight {
					return role[i]
				}
			}
			return role[len(role)-1]
		}
		genSlot := func(role string) string {
			switch role {
			case "GRAPES":
				return "🍇"
			case "WATERMELON":
				return "🍉"
			case "CHERRIES":
				return "🍒"
			case "LEMON":
				return "🍋"
			case "ORANGE":
				return "🍊"
			case "PLUM":
				return "🍑"
			case "BELL":
				return "🔔"
			case "BAR":
				return "💵"
			case "SEVEN":
				return "7️⃣"
			default:
				return ""
			}
		}

		selectedRole := getRole()
		result := genSlot(selectedRole)
		if result == "" {
			// ja: 全ての絵柄
			// en: All symbols
			symbols := []string{"🍇", "🍉", "🍒", "🍋", "🍊", "🍑", "🔔", "💵", "7️⃣"}
			symbols = append(symbols[:0], symbols...)
			rand.Shuffle(len(symbols), func(i, j int) {
				symbols[i], symbols[j] = symbols[j], symbols[i]
			})
			slotResults := []string{symbols[0], symbols[1], symbols[2]}
			result = strings.Join(slotResults, " | ")
		} else {
			result += " | " + result + " | " + result
		}

		_, err = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageBuilder().
			SetContent("Slots: "+result).
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
