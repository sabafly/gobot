package components

import (
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
)

type SchedulerFunc func(c *Components, client bot.Client) error

type Scheduler struct {
	Duration time.Duration
	Worker   SchedulerFunc
}

func rec_schedue() {
	if v := recover(); v != nil {
		slog.Error("recovered from panic", slog.Any("value", v))
	}
}

func execSchedule(c *Components, client bot.Client, s Scheduler) {
	for {
		doSchedule(c, client, s)
		time.Sleep(s.Duration)
	}
}

func doSchedule(c *Components, client bot.Client, s Scheduler) {
	defer rec_schedue()
	if err := s.Worker(c, client); err != nil {
		slog.Error("コンポーネント処理中にエラーが発生しました", "err", err)
		return
	}
}
