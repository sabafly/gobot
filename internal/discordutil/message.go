package discordutil

import (
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
)

func DeleteMessageAfter(client bot.Client, channelID, messageID snowflake.ID, after time.Duration) error {
	time.Sleep(after)
	return client.Rest().DeleteMessage(channelID, messageID)
}
