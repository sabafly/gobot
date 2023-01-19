/*
	Copyright (C) 2022  ikafly144

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
	"github.com/ikafly144/gobot/pkg/interaction"
	"github.com/ikafly144/gobot/pkg/product"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
)

func MakeBan(s *discordgo.Session) {
	resp, err := api.GetApi("/api/ban", http.NoBody)
	if err == nil {
		b, _ := io.ReadAll(resp.Body)
		j := ([]byte)(b)
		data := &types.GlobalBan{}
		json.Unmarshal(j, data)
		for _, v := range s.State.Guilds {
			for _, d := range data.Content {
				s.GuildBanCreateWithReason(v.ID, strconv.Itoa(int(d.ID)), product.ProductName+" Global Ban | Reason "+d.Reason, 7)
			}
			time.Sleep(time.Second)
		}
	}
}

func deleteBan(s *discordgo.Session, id string) {
	for _, v := range s.State.Guilds {
		s.GuildBanDelete(v.ID, id)
		time.Sleep(time.Second)
	}
}

var s *discordgo.Session

func Listener(sl *discordgo.Session) {
	s = sl
	log.Print("start web server")
	http.HandleFunc("/ban/delete", deleteBanHandler)
	http.HandleFunc("/feed/mc", feedMinecraftHandler)
	http.HandleFunc("/panel/vote", panelVote)
	go log.Print(http.ListenAndServe(":8192", nil))
}

func deleteBanHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	if r.URL.Query().Has("id") {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{"Status": "200 OK"})
		deleteBan(s, r.URL.Query().Get("id"))
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{"Status": "400 Bad Request", "Content": "missing id"})
	}
}

func feedMinecraftHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("OK")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"Status": "200 OK"})
	body, _ := io.ReadAll(r.Body)
	data := types.FeedMCServers{}
	json.Unmarshal(body, &data)
	for _, v := range data {
		var locale discordgo.Locale
		if v.Locale == "" {
			locale = discordgo.Japanese
		}
		if l, ok := types.StL[string(v.Locale)]; ok {
			locale = l
		}
		wid, wToken := util.WebhookExec(s, v.ChannelID)
		var ctx string
		var embed []*discordgo.MessageEmbed
		online, err := strconv.ParseBool(r.URL.Query().Get("online"))
		if err == nil && online {
			if v.RoleID != "" {
				ctx = "<@&" + v.RoleID + ">"
			}
			embed = []*discordgo.MessageEmbed{
				{
					Title: translate.Translate(locale, "panel_minecraft_message_boot", map[string]interface{}{
						"Name": v.Name,
					}),
					Color: 0x00ff00,
					Footer: &discordgo.MessageEmbedFooter{
						Text: product.ProductName,
					},
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				},
			}
		} else if err == nil && !online {
			embed = []*discordgo.MessageEmbed{
				{
					Title: translate.Translate(locale, "panel_minecraft_message_stop", map[string]interface{}{
						"Name": v.Name,
					}),
					Color: 0xff0000,
					Footer: &discordgo.MessageEmbedFooter{
						Text: product.ProductName,
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
			Username:  translate.Message(locale, "feed_minecraft_webhook_name"),
			AvatarURL: "",
			Embeds:    embed,
		})
		if err != nil {
			log.Print(err)
		}
	}
}

func panelVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"Status": "200 OK"})
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
		return
	}
	data := []types.VoteObject{}
	json.Unmarshal(b, &data)
	for _, vo := range data {
		go interaction.PanelVoteRemove(s, vo)
	}
}
