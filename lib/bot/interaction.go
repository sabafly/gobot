/*
	Copyright (C) 2022-2023  ikafly144

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
package botlib

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/sabafly/gobot/lib/logging"
)

// アプリケーションコマンドとそのハンダラを備えた構造体
type ApplicationCommand struct {
	discord.ApplicationCommandCreate
	Handler func(*events.ApplicationCommandInteractionCreate)
}

// アプリケーションコマンドのスライス型
type ApplicationCommands []ApplicationCommand

// アプリケーションコマンドを解析してハンダラを返す
func (a ApplicationCommands) Parse() func(*events.ApplicationCommandInteractionCreate) {
	handler := map[string]func(*events.ApplicationCommandInteractionCreate){}
	for _, ac := range a {
		if ac.Handler != nil {
			handler[ac.CommandName()] = ac.Handler
		}
	}
	return func(aci *events.ApplicationCommandInteractionCreate) {
		if aci.Type() != discord.InteractionTypeApplicationCommand {
			return
		}

		if f, ok := handler[aci.Data.CommandName()]; ok {
			f(aci)
		} else {
			logging.Warning("不明なコマンド要求")
		}
	}
}
