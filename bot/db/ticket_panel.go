package db

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/disgo/discord"
)

type TicketPanel interface {
	ID() uuid.UUID
	Type() TicketPanelType
	GuildID() snowflake.ID
	Name() string
	Description() *string
	Emoji() *discord.ComponentEmoji
	Addons() []TicketPanelAddon
}

type TicketPanelType int

const (
	TicketPanelTypeDefault TicketPanelType = iota
	TicketPanelTypeSubmitForm
	TicketPanelTypeIssueTemplate
)

type TicketPanelField struct {
	Label       string `json:"label"`
	Placeholder string `json:"place_holder"`
}

type TicketPanelAddonHandler interface {
	TicketPanel() TicketPanel
	Ticket() *Ticket
	Next(TicketPanelAddonHandler)
	DB() DB
}

type TicketPanelAddon interface {
	Type() TicketPanelAddonType
	Handle(TicketPanelAddonHandler) error // パネルからチケットが作成されたときに実行される
}

type TicketPanelAddonType int

const (
	TicketPanelAddonTypeTask = iota + 1
	TicketPanelAddonTypeDiscussion
	TicketPanelAddonTypeParliament
)

func NewTicketPanelTaskAddon(timezone time.Location, deadline time.Time, routine *TicketPanelTaskAddonRoutine) (*TicketPanelTaskAddon, error) {
	tz, err := NewDataLocation(timezone.String())
	if err != nil {
		return nil, err
	}
	addon := &TicketPanelTaskAddon{
		Deadline: deadline,
		Timezone: tz,
		Routine:  routine,
	}
	return addon, nil
}

type TicketPanelTaskAddon struct {
	Deadline time.Time                    `json:"deadline"`
	Timezone DataLocation                 `json:"timezone"`
	Routine  *TicketPanelTaskAddonRoutine `json:"routine"`
}

type TicketPanelTaskAddonRoutine struct {
	Weekday  time.Weekday `json:"weekday"`
	Interval int          `json:"interval"`
}

func (t TicketPanelTaskAddon) Type() TicketPanelAddonType { return TicketPanelAddonTypeTask }

func (t TicketPanelTaskAddon) Handle(ctx TicketPanelAddonHandler) error {
	year, month, day := t.Deadline.Date()
	// 通知をスケジュールする
	notice := NewNoticeScheduleTicketTask(ctx.Ticket().GuildID(), ctx.Ticket().ID(), time.Date(year, month, day, 3, 0, 0, 0, t.Timezone.Location))
	if err := ctx.DB().NoticeSchedule().Set(notice.id, notice); err != nil {
		return err
	}
	if t.Routine != nil {
		ctx.Ticket().Routine = t.Routine
	}
	return nil
}
