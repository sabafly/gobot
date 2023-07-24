package worker

import (
	"time"

	"github.com/sabafly/gobot/bot/client"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
)

func New() *Worker {
	return &Worker{}
}

type WorkerFunc func(b *botlib.Bot[*client.Client]) error

type WorkerHandler struct {
	Handler WorkerFunc
	Minutes int
}

type Worker struct {
	handler []WorkerHandler
}

func (w *Worker) Add(wf WorkerFunc, minutes int) {
	w.handler = append(w.handler,
		WorkerHandler{
			Handler: wf,
			Minutes: minutes,
		},
	)
}

func (w *Worker) Start(b *botlib.Bot[*client.Client]) {
	b.Logger.Info("Worker Started.")
	for _, wh := range w.handler {
		go scheduleHandler(wh.Handler, wh.Minutes, b)
	}
}

func scheduleHandler(w WorkerFunc, minutes int, b *botlib.Bot[*client.Client]) {
	var now time.Time
	for {
		b.Logger.Debug("working...")
		if err := w(b); err != nil {
			b.Logger.Errorf("error on worker: %s", err.Error())
		}
		now = time.Now()
		time.Sleep(time.Until(time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), (now.Minute()-now.Minute()%minutes)+minutes, 0, 0, time.Local)))
	}
}
