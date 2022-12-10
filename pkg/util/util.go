package util

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/translate"
)

func ErrorMessage(locale discordgo.Locale, err error) (res *discordgo.InteractionResponseData) {
	res = &discordgo.InteractionResponseData{}
	res.Content = ""
	res.Embeds = append(res.Embeds, &discordgo.MessageEmbed{
		Title:       translate.Message(locale, "error.message"),
		Description: err.Error(),
		Color:       0xff0000,
	})
	res.Flags = discordgo.MessageFlagsEphemeral
	return
}

func DeepcopyJson(src interface{}, dst interface{}) (err error) {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, dst)
	if err != nil {
		return err
	}
	return nil
}

func LogResp(resp *http.Response) {
	log.Printf("succeed %v %v %v", resp.Request.Method, resp.StatusCode, resp.Request.URL)
}

func MessageResp(resp *http.Response) string {
	defer resp.Body.Close()
	byteArray, _ := io.ReadAll(resp.Body)
	jsonBytes := ([]byte)(byteArray)
	str := fmt.Sprintf("succeed %v %v ```json\r%v```", resp.Request.Method, resp.StatusCode, string(jsonBytes))
	return str
}

func ToEmojiA(i int) string {
	return string(rune('🇦' - 1 + i))
}

var (
	EmojiRegex = regexp.MustCompile(`<(a|):[A-z0-9_~]+:[0-9].*>`)
)

func GetCustomEmojis(m *discordgo.MessageCreate) []*discordgo.Emoji {
	var toReturn []*discordgo.Emoji
	emojis := EmojiRegex.FindAllString(m.Content, -1)
	if len(emojis) < 1 {
		return toReturn
	}
	for _, em := range emojis {
		parts := strings.Split(em, ":")
		toReturn = append(toReturn, &discordgo.Emoji{
			ID:       parts[2][:len(parts[2])-1],
			Name:     parts[1],
			Animated: strings.HasPrefix(em, "<a:"),
		})
	}
	return toReturn
}

func EmojiFormat(e *discordgo.ComponentEmoji) string {
	if e.ID != "" && e.Name != "" {
		if e.Animated {
			return "<a:" + APIName(e) + ">"
		}

		return "<:" + APIName(e) + ">"
	}

	return APIName(e)
}

func APIName(e *discordgo.ComponentEmoji) string {
	if e.ID != "" && e.Name != "" {
		return e.Name + ":" + e.ID
	}
	if e.Name != "" {
		return e.Name
	}
	return e.ID
}
