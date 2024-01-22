package message

import (
	"context"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

func doTextCommand(ctx context.Context, event *events.GuildMessageCreate) (err error, shouldContinue bool) {
	content := strings.Split(event.Message.Content, " ")
	if content[0] != discord.UserMention(event.Client().ApplicationID()) {
		return nil, true
	}

	switch {
	case diceRollRegex.MatchString(content[1]):
		subMatch := diceRollRegex.FindStringSubmatch(content[1])
		diceCount, err := strconv.Atoi(subMatch[1])
		if err != nil || diceCount < 1 {
			return nil, true
		}
		diceSize, err := strconv.Atoi(subMatch[2])
		if err != nil || diceSize < 1 {
			return nil, true
		}
		content := "Dice Roll: "
		sum := 0
		for i := 0; i < diceCount; i++ {
			roll := diceRoll(diceSize)
			sum += roll
			content += strconv.Itoa(roll) + " "
		}

		content += "\nSum:" + strconv.Itoa(sum)

		_, err = event.Client().Rest().CreateMessage(event.ChannelID, discord.NewMessageBuilder().
			SetContent(content).
			SetMessageReferenceByID(event.Message.ID).
			Create(),
		)
		if err != nil {
			return err, false
		}

	}

	return nil, true

}

func diceRoll(size int) int {
	return rand.Intn(size) + 1
}

var (
	diceRollRegex = regexp.MustCompile(`^(\d+)d(\d+)$`)
)
