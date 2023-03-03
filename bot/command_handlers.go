/*
	Copyright (C) 2022-2023  sabafly

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
package gobot

import (
	"github.com/disgoorg/disgo/events"
)

// ----------------------------------------------------------------
// テキストコマンド
// ----------------------------------------------------------------

// 疎通確認用コマンド
// Discord API とのレスポンスを返す
func CommandTextPing(event *events.ApplicationCommandInteractionCreate) {
	// TODO: ピングコマンド
}

// 投票コマンド
func CommandTextVote(event *events.ApplicationCommandInteractionCreate) {
	// TODO: 投票コマンド
}

// ----------------------------------------------------------------
// ユーザーコマンド
// ----------------------------------------------------------------

// コマンド対象のユーザーデータを返すコマンド
// TODO: 統計を追加
func CommandUserInfo(event *events.ApplicationCommandInteractionCreate) {
	// TODO: インフォコマンド
}
