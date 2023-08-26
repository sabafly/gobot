package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type NoticeScheduleDB interface {
	Set(id uuid.UUID, data NoticeSchedule) error
	Get(id uuid.UUID) (NoticeSchedule, error)
	GetAll() ([]NoticeSchedule, error)
	GetByType(t NoticeScheduleType) ([]NoticeSchedule, error)
	Del(id uuid.UUID) error
}

type noticeScheduleDBImpl struct {
	db *redis.Client
}

func (n *noticeScheduleDBImpl) Set(id uuid.UUID, data NoticeSchedule) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := n.db.HSet(context.TODO(), "notice-schedule", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (n *noticeScheduleDBImpl) Get(id uuid.UUID) (NoticeSchedule, error) {
	res := n.db.HGet(context.TODO(), "notice-schedule", id.String())
	if err := res.Err(); err != nil {
		return nil, err
	}
	var v noticeScheduleUnmarshal
	if err := json.Unmarshal([]byte(res.Val()), &v); err != nil {
		return nil, err
	}
	return v.NoticeSchedule, nil
}

func (n *noticeScheduleDBImpl) GetAll() ([]NoticeSchedule, error) {
	res := n.db.HGetAll(context.TODO(), "notice-schedule")
	if err := res.Err(); err != nil {
		return nil, err
	}
	var r []NoticeSchedule
	for _, data := range res.Val() {
		var v noticeScheduleUnmarshal
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		r = append(r, v.NoticeSchedule)
	}
	return r, nil
}

func (n *noticeScheduleDBImpl) GetByType(t NoticeScheduleType) ([]NoticeSchedule, error) {
	res := n.db.HGetAll(context.TODO(), "notice-schedule")
	if err := res.Err(); err != nil {
		return nil, err
	}
	var r []NoticeSchedule
	for _, data := range res.Val() {
		var v noticeScheduleUnmarshal
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		if v.Type() != t {
			continue
		}
		r = append(r, v)
	}
	return r, nil
}

func (n *noticeScheduleDBImpl) Del(id uuid.UUID) error {
	res := n.db.HDel(context.TODO(), "notice-schedule", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

type noticeScheduleUnmarshal struct {
	NoticeSchedule
}

func (n *noticeScheduleUnmarshal) UnmarshalJSON(data []byte) error {
	var nType struct {
		Type NoticeScheduleType `json:"type"`
	}

	if err := json.Unmarshal(data, &nType); err != nil {
		return err
	}

	var (
		noticeSchedule NoticeSchedule
		err            error
	)

	switch nType.Type {
	case NoticeScheduleTypeBump:
		var v NoticeScheduleBump
		err = json.Unmarshal(data, &v)
		noticeSchedule = v
	default:
		err = fmt.Errorf("unknown notice schedule with type %d received", nType.Type)
	}

	if err != nil {
		return err
	}

	n.NoticeSchedule = noticeSchedule
	return nil
}

type NoticeSchedule interface {
	Type() NoticeScheduleType
	ID() uuid.UUID
}

type NoticeScheduleType int

const (
	NoticeScheduleTypeBump = iota + 1
	NoticeScheduleTypeTicketTask
)

func (t NoticeScheduleType) String() string {
	switch t {
	case NoticeScheduleTypeBump:
		return "bump"
	case NoticeScheduleTypeTicketTask:
		return "ticket_task"
	default:
		return "unknown"
	}
}

func NewNoticeScheduleBump(is_up bool, guildID, channelID snowflake.ID, schedule_time time.Time) *NoticeScheduleBump {
	return &NoticeScheduleBump{
		id:            uuid.New(),
		IsUp:          is_up,
		GuildID:       guildID,
		ChannelID:     channelID,
		ScheduledTime: schedule_time,
	}
}

type NoticeScheduleBump struct {
	id            uuid.UUID
	IsUp          bool         `json:"is_up"`
	GuildID       snowflake.ID `json:"guild_id"`
	ChannelID     snowflake.ID `json:"channel_id"`
	ScheduledTime time.Time    `json:"scheduled_time"`
}

func (n NoticeScheduleBump) Type() NoticeScheduleType { return NoticeScheduleTypeBump }
func (n NoticeScheduleBump) ID() uuid.UUID            { return n.id }

func (n *NoticeScheduleBump) UnmarshalJSON(data []byte) error {
	type noticeScheduleBump NoticeScheduleBump
	var v struct {
		ID uuid.UUID
		noticeScheduleBump
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*n = NoticeScheduleBump(v.noticeScheduleBump)
	n.id = v.ID
	return nil
}

func (n NoticeScheduleBump) MarshalJSON() ([]byte, error) {
	type noticeScheduleBump NoticeScheduleBump
	return json.Marshal(struct {
		Type NoticeScheduleType `json:"type"`
		ID   uuid.UUID          `json:"id"`
		noticeScheduleBump
	}{
		Type:               n.Type(),
		ID:                 n.id,
		noticeScheduleBump: noticeScheduleBump(n),
	},
	)
}

func NewNoticeScheduleTicketTask(guildID snowflake.ID, ticketID uuid.UUID, scheduledTime time.Time) *NoticeScheduleTicketTask {
	return &NoticeScheduleTicketTask{
		id:            uuid.New(),
		GuildID:       guildID,
		TicketID:      ticketID,
		ScheduledTime: scheduledTime,
	}
}

type NoticeScheduleTicketTask struct {
	id            uuid.UUID
	TicketID      uuid.UUID    `json:"ticket_id"`
	GuildID       snowflake.ID `json:"guild_id"`
	ScheduledTime time.Time    `json:"scheduled_time"`
	IsDeadline    bool         `json:"is_dead_line"`
}

func (n NoticeScheduleTicketTask) Type() NoticeScheduleType { return NoticeScheduleTypeTicketTask }
func (n NoticeScheduleTicketTask) ID() uuid.UUID            { return n.id }

func (n *NoticeScheduleTicketTask) UnmarshalJSON(data []byte) error {
	type noticeScheduleTicketTask NoticeScheduleTicketTask
	var v struct {
		ID uuid.UUID
		noticeScheduleTicketTask
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*n = NoticeScheduleTicketTask(v.noticeScheduleTicketTask)
	n.id = v.ID
	return nil
}

func (n NoticeScheduleTicketTask) MarshalJSON() ([]byte, error) {
	type noticeScheduleTicketTask NoticeScheduleTicketTask
	return json.Marshal(struct {
		Type NoticeScheduleType `json:"type"`
		ID   uuid.UUID          `json:"id"`
		noticeScheduleTicketTask
	}{
		Type:                     n.Type(),
		ID:                       n.id,
		noticeScheduleTicketTask: noticeScheduleTicketTask(n),
	},
	)
}
