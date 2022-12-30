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
package init

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/command/handler"
	"github.com/ikafly144/gobot/pkg/env"
	"github.com/ikafly144/gobot/pkg/session"
	"github.com/ikafly144/gobot/pkg/translate"
)

var s *discordgo.Session

func init() {
	var err error
	s, err = discordgo.New("Bot " + *env.BotToken)
	if err != nil {
		log.Fatalf("無効なbotパラメータ: %v", err)
	}
	s.Identify.Intents = discordgo.IntentsAll

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		p, err := s.State.UserChannelPermissions(s.State.User.ID, i.ChannelID)
		if err == nil && p&int64(discordgo.PermissionAdministrator) != 0 {
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := handler.CommandHandler()[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
				return
			case discordgo.InteractionMessageComponent:
				ids := strings.Split(i.MessageComponentData().CustomID, ":")
				var customID string
				var sessionID string
				for i2, v := range ids {
					switch i2 {
					case 0:
						customID = v
					case 1:
						sessionID = v
					}
				}
				if c, ok := handler.MessageComponentHandler()[customID]; ok {
					c(s, i, sessionID)
				}
				return
			case discordgo.InteractionModalSubmit:
				ids := strings.Split(i.ModalSubmitData().CustomID, ":")
				var customID string
				var mid string
				for i2, v := range ids {
					switch i2 {
					case 0:
						customID = v
					case 1:
						mid = v
					}
				}
				if m, ok := handler.ModalSubmitHandler()[customID]; ok {
					m(s, i, mid)
				}
				return
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: translate.Message(i.Locale, "error_unknown_command"),
				},
			})
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: translate.Message(i.Locale, "error_bot_does_not_have_permissions"),
				},
			})
		}
	})

	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		str, err := m.ContentWithMoreMentionsReplaced(s)
		if err != nil {
			str = m.Content
		}
		g, _ := s.Guild(m.GuildID)
		c, _ := s.Channel(m.ChannelID)
		log.Printf("[Message Created] : %v(%v) #%v(%v) <%v#%v>\n                 >> %v", g.Name, g.ID, c.Name, c.ID, m.Author.Username, m.Author.Discriminator, str)
		p, err := s.State.UserChannelPermissions(s.State.User.ID, m.ChannelID)
		if err == nil && p&int64(discordgo.PermissionAdministrator) == 0 {
			data, err := session.MessagePanelConfigEmojiLoad(m.Author.ID)
			if err != nil {
				log.Print(err)
				return
			} else {
				d := data.Data()
				data.Data().Handler(d, s, m)
			}
		}
	})
}

func Session() *discordgo.Session { return s }
