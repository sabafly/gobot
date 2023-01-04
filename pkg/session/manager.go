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
package session

import (
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/ikafly144/gobot/pkg/types"
)

var interactionSessions *session[discordgo.InteractionCreate] = newSession[discordgo.InteractionCreate]()

func InteractionSave(i *discordgo.InteractionCreate) (id string) {
	return interactionSessions.add(i)
}

func InteractionLoad(id string) (data *sessionData[discordgo.InteractionCreate], err error) {
	return interactionSessions.get(id)
}

func InteractionRemove(id string) {
	interactionSessions.remove(id)
}

var messageSessions *session[types.MessageSessionData[types.MessagePanelConfigEmojiData]] = newSession[types.MessageSessionData[types.MessagePanelConfigEmojiData]]()

func MessagePanelConfigEmojiSave(m *types.MessageSessionData[types.MessagePanelConfigEmojiData], id string) {
	messageSessions.addWithID(m, id)
}

func MessagePanelConfigEmojiLoad(id string) (data *sessionData[types.MessageSessionData[types.MessagePanelConfigEmojiData]], err error) {
	return messageSessions.get(id)
}

func MessagePanelConfigEmojiRemove(id string) {
	messageSessions.remove(id)
}

var voteSessions *session[types.VoteSession] = newSession[types.VoteSession]()

func VoteSave(d *types.VoteSession) (id string) {
	id = uuid.New().String()
	d.Vote.VoteID = id
	voteSessions.addWithID(d, id)
	return
}

func VoteSaveWithID(d *types.VoteSession, id string) {
	voteSessions.addWithID(d, id)
}

func VoteLoad(id string) (data *sessionData[types.VoteSession], err error) {
	return voteSessions.get(id)
}

func VoteRemove(id string) {
	voteSessions.remove(id)
}
