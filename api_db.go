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
package main

import (
	"github.com/sabafly/gobot/pkg/lib/database"
	"github.com/sabafly/gobot/pkg/lib/env"
	"gorm.io/gorm/logger"
)

func db() {
	db := database.NewDatabase()
	err := db.Connect(database.DSN{Host: env.DBHost, Port: env.DBPort, User: env.DBUser, Pass: env.DBPass, Name: env.DBName, LogLevel: logger.Info})
	if err != nil {
		panic(err)
	}
}
