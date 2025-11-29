/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package components

import (
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
)

type SchedulerFunc func(c *Components, client *bot.Client) error

type Scheduler struct {
	Duration time.Duration
	Worker   SchedulerFunc
}

func recoverSchedule() {
	if v := recover(); v != nil {
		slog.Error("recovered from panic", slog.Any("value", v))
	}
}

func execSchedule(c *Components, client *bot.Client, s Scheduler) {
	now := time.Now()
	time.Sleep(time.Until(time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())))
	t := time.NewTicker(s.Duration)
	for {
		doSchedule(c, client, s)
		<-t.C
	}
}

func doSchedule(c *Components, client *bot.Client, s Scheduler) {
	defer recoverSchedule()
	slog.Debug("Executing scheduled task", "duration", s.Duration)
	if err := s.Worker(c, client); err != nil {
		slog.Error("コンポーネント処理中にエラーが発生しました", "err", err)
		return
	}
	slog.Debug("Scheduled task completed", "duration", s.Duration)
}
