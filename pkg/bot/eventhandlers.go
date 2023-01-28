// "eventhandlers.go"から生成されています; 編集禁止
// events.go を確認

package gobot

// Following are all the event types.
// Event type values are used to match the events returned by Discord.
// EventTypes surrounded by __ are synthetic and are internal to DiscordGo.
const (
	statusUpdateEventType = "STATUS_UPDATE"
)

// StatusUpdate イベントのイベントハンダラを返します
type statusUpdateEventHandler func(*Shard, *StatusUpdate)

// StatusUpdate イベントの型名を返します
func (eh statusUpdateEventHandler) Type() string {
	return statusUpdateEventType
}

// StatusUpdate の新しいインスタンスを返します
func (eh statusUpdateEventHandler) New() any {
	return &StatusUpdate{}
}

// StatusUpdate イベントのハンダラ
func (eh statusUpdateEventHandler) Handle(s *Shard, i any) {
	if t, ok := i.(*StatusUpdate); ok {
		eh(s, t)
	}
}

func handlerForInterface(handler any) EventHandler {
	switch v := handler.(type) {
	case func(*Shard, any):
		return anyEventHandler(v)
	case func(*Shard, *StatusUpdate):
		return statusUpdateEventHandler(v)
	}

	return nil
}

func init() {
	registerInterfaceProvider(statusUpdateEventHandler(nil))
}
