package handler

import (
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
)

type (
	GuildMemberJoinHandler func(event *events.GuildMemberJoin) error
)
type MemberJoin struct {
	UUID    uuid.UUID
	Check   Check[*events.GuildMemberJoin]
	Handler GuildMemberJoinHandler
}

func (h *Handler) handlerMemberJoin(event *events.GuildMemberJoin) {
	if _, ok := h.ExcludeID[event.GuildID]; ok {
		return
	}
	h.Logger.Debugf("メンバー作成 %d", event.GuildID)
	for _, mj := range h.MemberJoin {
		if mj.Check != nil && !mj.Check(event) {
			h.Logger.Debug("チェックに失敗")
			continue
		}
		if err := mj.Handler(event); err != nil {
			h.Logger.Errorf("Failed to handle member join in \"%s\"", event.GuildID)
		}
	}
}
