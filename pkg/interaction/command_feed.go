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
package interaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/api"
	"github.com/ikafly144/gobot/pkg/translate"
	"github.com/ikafly144/gobot/pkg/types"
	"github.com/ikafly144/gobot/pkg/util"
)

func Feed(s *discordgo.Session, i *discordgo.InteractionCreate) {
	util.ErrorCatch("", s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}))
	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "minecraft":
		options = options[0].Options
		switch options[0].Name {
		case "create":
			feedMinecraftCreate(s, i, options)
		case "get":
			feedMinecraftGet(s, i)
		case "remove":
			feedMinecraftRemove(s, i, options)
		}
	}
}

func feedMinecraftCreate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	gid := i.GuildID
	cid := i.ChannelID
	var name string
	var address string
	var port int
	var role discordgo.Role
	options = options[0].Options
	for _, v := range options {
		switch v.Name {
		case "name":
			name = v.StringValue()
		case "address":
			address = v.StringValue()
		case "port":
			port = int(v.IntValue())
		case "role":
			role = *v.RoleValue(s, gid)
		}
	}
	hash := sha256.New()
	util.ErrorCatch(io.WriteString(hash, address+":"+strconv.Itoa(port)))
	st := hash.Sum(nil)
	code := hex.EncodeToString(st)
	data := &types.TransMCServer{
		Address: address,
		Port:    uint16(port),
		FeedMCServer: types.FeedMCServer{
			Hash:      code,
			Name:      name,
			GuildID:   gid,
			ChannelID: cid,
			RoleID:    role.ID,
			Locale:    i.Locale,
		},
	}
	log.Print(data.Address, data.Port, i.Locale)
	body, _ := util.ErrorCatch(json.Marshal(data))
	util.ErrorCatch(api.GetApi("/api/feed/mc/add", bytes.NewBuffer(body)))
	str := "OK"
	util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &str,
	}))
}

func feedMinecraftGet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	resp, err := util.ErrorCatch(api.GetApi("/api/feed/mc", http.NoBody))
	if err != nil {
		embed := translate.ErrorEmbed(i.Locale, "error_failed_to_connect_api")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embed,
		})
		return
	}
	body, _ := util.ErrorCatch(io.ReadAll(resp.Body))
	content := types.Res{}
	data := types.FeedMCServers{}
	util.ErrorCatch("", json.Unmarshal(body, &content))
	b, _ := util.ErrorCatch(json.Marshal(content.Content))
	util.ErrorCatch("", json.Unmarshal(b, &data))
	array := []*discordgo.MessageEmbed{}
	var server types.FeedMCServers
	var locales []discordgo.Locale
	for _, v := range data {
		var locale discordgo.Locale
		if v.Locale == "" {
			locale = discordgo.Japanese
		}
		if l, ok := types.StL[string(v.Locale)]; ok {
			locale = l
		}
		if v.GuildID == i.GuildID {
			server = append(server, v)
			locales = append(locales, locale)
		}
	}
	resp2, err := util.ErrorCatch(api.GetApi("/api/feed/mc/hash", http.NoBody))
	if err != nil {
		embed := translate.ErrorEmbed(i.Locale, "error_failed_to_connect_api")
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embed,
		})
		return
	}
	body, _ = util.ErrorCatch(io.ReadAll(resp2.Body))
	content2 := types.Res{}
	util.ErrorCatch("", json.Unmarshal(body, &content2))
	b, _ = util.ErrorCatch(json.Marshal(content2.Content))
	hash := types.MCServers{}
	util.ErrorCatch("", json.Unmarshal(b, &hash))
	for n, v := range server {
		var address string
		var port uint16
		for _, v2 := range hash {
			if v2.Hash == v.Hash {
				address = v2.Address
				port = v2.Port
				break
			}
		}
		array = append(array, &discordgo.MessageEmbed{
			Title: v.Name,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   translate.Message(locales[n], "address"),
					Value:  address,
					Inline: true,
				},
				{
					Name:   translate.Message(locales[n], "port"),
					Value:  strconv.Itoa(int(port)),
					Inline: true,
				},
				{
					Name:  translate.Message(locales[n], "channel"),
					Value: "<#" + v.ChannelID + ">",
				},
			},
		})
	}
	if len(array) != 0 {
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &array,
		}))
	} else {
		str := "no data"
		util.ErrorCatch(s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		}))
	}
}

func feedMinecraftRemove(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name string
	options = options[0].Options
	for _, v := range options {
		switch v.Name {
		case "name":
			name = v.StringValue()
		}
		data := &types.FeedMCServer{
			Name:    name,
			GuildID: i.GuildID,
		}
		body, _ := util.ErrorCatch(json.Marshal(data))
		util.ErrorCatch(api.GetApi("/api/feed/mc/remove", bytes.NewBuffer(body)))
		str := "OK"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &str,
		})
	}
}
