package handler

import (
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type (
	MessageHandler func(event *events.MessageCreate) error
)

type Message struct {
	UUID      uuid.UUID
	ChannelID snowflake.ID
	AuthorID  *snowflake.ID
	Check     Check[*events.MessageCreate]
	Handler   MessageHandler
}

func (h *Handler) handleMessage(event *events.MessageCreate) {
	if _, ok := h.ExcludeID[event.ChannelID]; ok {
		return
	}
	h.Logger.Debugf("メッセージ作成 %d", event.ChannelID)
	channelID := event.ChannelID
	for _, m := range h.Message {
		if m.ChannelID != channelID {
			h.Logger.Debug("チャンネルが違います")
			continue
		}
		if m.AuthorID != nil && *m.AuthorID != event.Message.Author.ID {
			h.Logger.Debugf("送信者が違います %d %d", *m.AuthorID, event.Message.Author.ID)
			continue
		}
		if m.Check != nil && !m.Check(event) {
			h.Logger.Debug("チェックに失敗")
			continue
		}
		if err := m.Handler(event); err != nil {
			h.Logger.Errorf("Failed to handle message \"%d\" in \"%s\": %s", event.MessageID, event.GuildID, event.ChannelID)
		}
	}
}
