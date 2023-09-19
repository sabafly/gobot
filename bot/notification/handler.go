package notification

import (
	"time"

	"github.com/sabafly/gobot/bot/client"
	"github.com/sabafly/gobot/bot/db"
	"github.com/sabafly/gobot/bot/handlers"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
)

func Handler(b *botlib.Bot[*client.Client]) error {
	ns, err := b.Self.DB.NoticeSchedule().GetAll()
	if err != nil {
		return err
	}
	for _, s := range ns {
		switch s.Type() {
		case db.NoticeScheduleTypeBump:
			s, ok := s.(db.NoticeScheduleBump)
			if !ok {
				b.Logger.Warn("failed to convert")
				break
			}
			if !time.Now().After(s.ScheduledTime.Add(-time.Minute * 15)) {
				continue
			}
			if err := handlers.ScheduleBump(b, s); err != nil {
				b.Logger.Errorf("error on worker notice schedule: %s", err)
				continue
			}
			if err := b.Self.DB.NoticeSchedule().Del(s.ID()); err != nil {
				b.Logger.Errorf("error on worker notice schedule delete: %s", err)
				continue
			}
		}
	}
	return nil
}
