/*
	Copyright (C) 2022-2023  ikafly144

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package botlib

import (
	"github.com/sabafly/gobot/lib/logging"
)

// TODO: 仕様を決定、実装する

type eventHandlerInstance struct {
	eventHandler EventHandler
}

type EventHandler interface {
	Type() string

	Handle(*Shard, any)
}

type EventInterfaceProvider interface {
	// Type is the type of event this handler belongs to.
	Type() string

	// New returns a new instance of the struct this event handler handles.
	// This is called once per event.
	// The struct is provided to all handlers of the same Type().
	New() any
}

// anyEventType is the event handler type for any events.
const anyEventType = "__ANY__"

// anyEventHandler is an event handler for any events.
type anyEventHandler func(*Shard, any)

// Type returns the event type for any events.
func (eh anyEventHandler) Type() string {
	return anyEventType
}

// Handle is the handler for an any event.
func (eh anyEventHandler) Handle(s *Shard, i any) {
	eh(s, i)
}

var registeredInterfaceProviders = map[string]EventInterfaceProvider{}

// registerInterfaceProvider registers a provider so that DiscordGo can
// access it's New() method.
func registerInterfaceProvider(eh EventInterfaceProvider) {
	if _, ok := registeredInterfaceProviders[eh.Type()]; ok {
		return
		// XXX:
		// if we should error here, we need to do something with it.
		// fmt.Errorf("event %s already registered", eh.Type())
	}
	registeredInterfaceProviders[eh.Type()] = eh
}

// APIイベントのハンダラを登録する
func (a *Shard) AddHandler(handler any) func() {
	eh := handlerForInterface(handler)

	if eh == nil {
		logging.Error("無効なハンダラタイプ このハンダラは呼び出されません")
		return func() {}
	}

	return a.addEventHandler(eh)
}

func (a *Shard) addEventHandler(eventHandler EventHandler) func() {
	a.handlersMu.Lock()
	defer a.handlersMu.Unlock()

	if a.handlers == nil {
		a.handlers = map[string][]*eventHandlerInstance{}
	}

	ehi := &eventHandlerInstance{eventHandler: eventHandler}
	a.handlers[eventHandler.Type()] = append(a.handlers[eventHandler.Type()], ehi)

	return func() {
		a.removeEventHandlerInstance(eventHandler.Type(), ehi)
	}
}

func (a *Shard) removeEventHandlerInstance(t string, ehi *eventHandlerInstance) {
	a.handlersMu.Lock()
	defer a.handlersMu.Unlock()

	handlers := a.handlers[t]
	for i := range handlers {
		if handlers[i] == ehi {
			a.handlers[t] = append(a.handlers[t], handlers[i+1:]...)
		}
	}
}

func (s *Shard) handle(t string, i any) {
	for _, eh := range s.handlers[t] {
		go eh.eventHandler.Handle(s, i)
	}
}

func (a *Shard) handleEvent(t string, i any) {
	a.handlersMu.Lock()
	defer a.handlersMu.Unlock()

	a.onInterface(i)

	a.handle(anyEventType, i)

	a.handle(t, i)
}

func (s *Shard) onInterface(i any) {
	// TODO: なんか実装する
	switch i.(type) {
	}
}
