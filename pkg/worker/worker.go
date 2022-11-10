package worker

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
	"github.com/ikafly144/gobot/pkg/command"
	"github.com/ikafly144/gobot/pkg/setup"
	"github.com/ikafly144/gobot/pkg/translate"
	"gorm.io/gorm"
)

type FeedMCServer struct {
	gorm.Model
	Hash      string `gorm:"uniqueIndex"`
	GuildID   string
	ChannelID string
	RoleID    string
	Name      string
	Locale    discordgo.Locale
}

type FeedMCServers []FeedMCServer

func MakeBan(s *discordgo.Session) {
	resp, err := api.GetApi("/api/ban", http.NoBody)
	if err == nil {
		b, _ := io.ReadAll(resp.Body)
		j := ([]byte)(b)
		data := &command.GlobalBan{}
		json.Unmarshal(j, data)
		for _, v := range s.State.Guilds {
			for _, d := range data.Content {
				s.GuildBanCreateWithReason(v.ID, strconv.Itoa(int(d.ID)), "GoBot Global Ban | Reason "+d.Reason, 7)
			}
			time.Sleep(time.Second)
		}
	}
}

func deleteBan(id string) {
	s := setup.GetSession()
	for _, v := range s.State.Guilds {
		s.GuildBanDelete(v.ID, id)
		time.Sleep(time.Second)
	}
}

func DeleteBanListener() {
	http.HandleFunc("/ban/delete", deleteBanHandler)
	http.HandleFunc("/feed/mc", feedMinecraftHandler)
	go log.Print(http.ListenAndServe(":8192", nil))
}

func deleteBanHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	if r.URL.Query().Has("id") {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{"Status": "200 OK"})
		deleteBan(r.URL.Query().Get("id"))
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{"Status": "400 Bad Request", "Content": "missing id"})
	}
}

func feedMinecraftHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("OK")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"Status": "200 OK"})
	body, _ := io.ReadAll(r.Body)
	data := FeedMCServers{}
	json.Unmarshal(body, &data)
	s := setup.GetSession()
	for _, v := range data {
		if v.Locale == "" {
			v.Locale = discordgo.Japanese
		}
		ws, _ := s.ChannelWebhooks(v.ChannelID)
		var wid string
		var wToken string
		var hasWB bool
		for _, w := range ws {
			if w.User.ID == s.State.User.ID {
				wid = w.ID
				wToken = w.Token
				hasWB = true
				break
			}
		}
		if !hasWB {
			w, err := s.WebhookCreate(v.ChannelID, "gobot-webhook", s.State.User.AvatarURL("1024"))
			if err != nil {
				log.Print(err)
				return
			}
			wid = w.ID
			wToken = w.Token
		}
		var ctx string
		var embed []*discordgo.MessageEmbed
		online, err := strconv.ParseBool(r.URL.Query().Get("online"))
		if err == nil && online {
			if v.RoleID != "" {
				ctx = "<@&" + v.RoleID + ">"
			}
			embed = []*discordgo.MessageEmbed{
				{
					Title: translate.Translate(v.Locale, "panel_minecraft_message_boot", map[string]interface{}{
						"Name": v.Name,
					}),
					Color: 0x00ff00,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "gobot",
					},
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				},
			}
		} else if err == nil && !online {
			embed = []*discordgo.MessageEmbed{
				{
					Title: translate.Translate(v.Locale, "panel_minecraft_message_stop", map[string]interface{}{
						"Name": v.Name,
					}),
					Color: 0xff0000,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "gobot",
					},
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				},
			}
		} else {
			log.Print(err)
			return
		}
		_, err = s.WebhookExecute(wid, wToken, true, &discordgo.WebhookParams{
			Content:   ctx,
			Username:  translate.Message(v.Locale, "feed_minecraft_webhook_name"),
			AvatarURL: "",
			Embeds:    embed,
		})
		if err != nil {
			log.Print(err)
		}
	}
}
