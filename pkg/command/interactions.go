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
package command

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/session"
)

var interactionSessions *session.Session[*discordgo.InteractionCreate] = session.New[*discordgo.InteractionCreate]()

func interactionSave(i *discordgo.InteractionCreate) (id string) {
	return interactionSessions.Add(i)
}

func interactionLoad(id string) (data session.SessionData[*discordgo.InteractionCreate], err error) {
	return interactionSessions.Get(id)
}

func interactionRemove(id string) {
	interactionSessions.Remove(id)
}
