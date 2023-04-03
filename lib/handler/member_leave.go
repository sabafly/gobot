package handler

import (
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
)

type (
	GuildMemberLeaveHandler func(event *events.GuildMemberLeave) error
)
type MemberLeave struct {
	UUID    uuid.UUID
	Check   Check[*events.GuildMemberLeave]
	Handler GuildMemberLeaveHandler
}

func (h *Handler) handlerMemberLeave(event *events.GuildMemberLeave) {
	if _, ok := h.ExcludeID[event.GuildID]; ok {
		return
	}
	h.Logger.Debugf("メンバー作成 %d", event.GuildID)
	for _, mj := range h.MemberLeave {
		if mj.Check != nil && !mj.Check(event) {
			h.Logger.Debug("チェックに失敗")
			continue
		}
		if err := mj.Handler(event); err != nil {
			h.Logger.Errorf("Failed to handle member join in \"%s\"", event.GuildID)
		}
	}
}
