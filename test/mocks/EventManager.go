package mocks

import (
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

type EventManager struct {
	AddHandlerCall struct {
		Received struct {
			Handler   I.Handler
			EventType string
		}
		Returns struct {
			Error error
		}
	}
	EmitCall struct {
		TimesCalled int
		Received    struct {
			Event S.Event
		}
		Returns struct {
			Error error
		}
	}
}

func (e *EventManager) AddHandler(handler I.Handler, eventType string) error {
	e.AddHandlerCall.Received.Handler = handler
	e.AddHandlerCall.Received.EventType = eventType

	return e.AddHandlerCall.Returns.Error
}

func (e *EventManager) Emit(event S.Event) error {
	defer func() { e.EmitCall.TimesCalled++ }()
	e.EmitCall.Received.Event = event

	return e.EmitCall.Returns.Error
}
