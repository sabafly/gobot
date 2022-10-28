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
)

func MakeBan(s *discordgo.Session) {
	resp, err := api.GetApi("/api/ban")
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
