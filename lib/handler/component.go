package handler

import (
	"strings"

	"github.com/disgoorg/disgo/events"
)

type ComponentHandler func(event *events.ComponentInteractionCreate) error

type Component struct {
	Name    string
	Check   Check[*events.ComponentInteractionCreate]
	Handler map[string]ComponentHandler
}

func (h *Handler) handleComponent(event *events.ComponentInteractionCreate) {
	customID := event.Data.CustomID()
	h.Logger.Debugf("コンポーネントインタラクション呼び出し %s", customID)
	if !strings.HasPrefix(customID, "handler:") {
		return
	}

	var subName string
	if strings.Count(customID, ":") >= 2 {
		subName = strings.Split(customID, ":")[2]
	}

	componentName := strings.Split(customID, ":")[1]
	component, ok := h.Components[componentName]
	if !ok || component.Handler == nil {
		h.Logger.Errorf("No component handler for \"%s\" found", componentName)
	}

	if component.Check != nil && !component.Check(event) {
		return
	}

	handler, ok := component.Handler[subName]
	if !ok {
		h.Logger.Debugf("不明なハンダラ %s", subName)
		err := event.DeferUpdateMessage()
		if err != nil {
			h.Logger.Errorf("Failed to handle unknown handler interaction for \"%s\" : %s", customID, err)
		}
		return
	}
	if err := handler(event); err != nil {
		h.Logger.Errorf("Failed to handle component interaction for \"%s\" : %s", componentName, err)
	}
}
