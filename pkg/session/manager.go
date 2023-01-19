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

// インタラクションセッションを保存
func InteractionSave(i *discordgo.InteractionCreate) (id string) {
	return interactionSessions.set(i)
}

// インタラクションセッションを取得
func InteractionLoad(id string) (data *sessionData[discordgo.InteractionCreate], err error) {
	return interactionSessions.get(id)
}

// インタラクションセッションを削除
func InteractionRemove(id string) {
	interactionSessions.remove(id)
}

var messageSessions *session[types.MessageSessionData[types.MessagePanelConfigEmojiData]] = newSession[types.MessageSessionData[types.MessagePanelConfigEmojiData]]()

// パネル絵文字設定セッションを保存
func MessagePanelConfigEmojiSave(m *types.MessageSessionData[types.MessagePanelConfigEmojiData], id string) {
	messageSessions.setWithID(m, id)
}

// パネル絵文字設定セッションを取得
func MessagePanelConfigEmojiLoad(id string) (data *sessionData[types.MessageSessionData[types.MessagePanelConfigEmojiData]], err error) {
	return messageSessions.get(id)
}

// パネル絵文字設定セッションを削除
func MessagePanelConfigEmojiRemove(id string) {
	messageSessions.remove(id)
}

var voteSessions *session[types.VoteSession] = newSession[types.VoteSession]()

// 投票パネル作成セッションを保存
func VoteSave(d *types.VoteSession) (id string) {
	id = uuid.New().String()
	d.Vote.VoteID = id
	voteSessions.setWithID(d, id)
	return
}

// 投票パネル作成セッションをidを指定して保存
func VoteSaveWithID(d *types.VoteSession, id string) {
	voteSessions.setWithID(d, id)
}

// 投票パネル作成セッションを取得
func VoteLoad(id string) (data *sessionData[types.VoteSession], err error) {
	return voteSessions.get(id)
}

// 投票パネル作成セッションを削除
func VoteRemove(id string) {
	voteSessions.remove(id)
}
