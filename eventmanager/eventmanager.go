// Package eventmanager emits events.
package eventmanager

import (
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// EventManager has handlers for each registered event type.
type EventManager struct {
	handlers map[string][]I.Handler
	Log      I.Logger
}

// NewEventManager returns an EventManager.
func NewEventManager(l I.Logger) *EventManager {
	return &EventManager{
		handlers: make(map[string][]I.Handler),
		Log:      l,
	}
}

// AddHandler takes a handler and eventType and returns an error if a handler is not provided.
func (e *EventManager) AddHandler(handler I.Handler, eventType string) error {
	if handler == nil {
		return InvalidArgumentError{}
	}
	e.handlers[eventType] = append(e.handlers[eventType], handler)
	return nil
}

// Emit emits an event.
func (e *EventManager) Emit(event S.Event) error {
	for _, handler := range e.handlers[event.Type] {
		err := handler.OnEvent(event)
		if err != nil {
			return err
		}
		e.Log.Debugf("a %s event has been emitted", event.Type)
	}
	return nil
}
