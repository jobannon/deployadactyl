package eventmanager

import (
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
)

type EventManager struct {
	handlers map[string][]I.Handler
}

func NewEventManager() *EventManager {
	return &EventManager{
		handlers: make(map[string][]I.Handler),
	}
}

func (e *EventManager) AddHandler(handler I.Handler, eventType string) error {
	if handler == nil {
		return errors.Errorf("Invalid argument: error handler does not exist")
	}
	e.handlers[eventType] = append(e.handlers[eventType], handler)
	return nil
}

func (e *EventManager) Emit(event S.Event) error {
	for _, handler := range e.handlers[event.Type] {
		err := handler.OnEvent(event)
		if err != nil {
			return err
		}
	}
	return nil
}
