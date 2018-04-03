package start

import (
	"reflect"

	"github.com/compozed/deployadactyl/eventmanager"
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
)

type IEvent interface {
	Type() string
}

type InvalidEventType struct {
	error
}

type eventBinding struct {
	etype   reflect.Type
	handler func(event interface{}) error
}

func (s eventBinding) Accepts(event interface{}) bool {
	return reflect.TypeOf(event) == s.etype
}

func (b eventBinding) Emit(event interface{}) error {
	return b.handler(event)
}

type StartFailureEvent struct {
	CFContext     interfaces.CFContext
	Data          map[string]interface{}
	Authorization interfaces.Authorization
	Environment   structs.Environment
	Error         error
}

func (e StartFailureEvent) Type() string {
	return "StartFailureEvent"
}

func NewStartFailureEventBinding(handler func(event StartFailureEvent) error) eventmanager.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StartFailureEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StartFailureEvent)
			if ok {
				return handler(event)
			} else {
				return InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}

type StartSuccessEvent struct {
	CFContext     interfaces.CFContext
	Data          map[string]interface{}
	Authorization interfaces.Authorization
	Environment   structs.Environment
}

func (e StartSuccessEvent) Type() string {
	return "StartSuccessEvent"
}

func NewStartSuccessEventBinding(handler func(event StartSuccessEvent) error) eventmanager.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StartSuccessEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StartSuccessEvent)
			if ok {
				return handler(event)
			} else {
				return InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}

type StartStartedEvent struct {
	CFContext interfaces.CFContext
	Data      map[string]interface{}
}

func (e StartStartedEvent) Type() string {
	return "StartStartedEvent"
}

func NewStartStartedEventBinding(handler func(event StartStartedEvent) error) eventmanager.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StartStartedEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StartStartedEvent)
			if ok {
				return handler(event)
			} else {
				return InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}

type StartFinishedEvent struct {
	CFContext     interfaces.CFContext
	Data          map[string]interface{}
	Authorization interfaces.Authorization
	Environment   structs.Environment
}

func (e StartFinishedEvent) Type() string {
	return "StartFinishedEvent"
}

func NewStartFinishedEventBinding(handler func(event StartFinishedEvent) error) eventmanager.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StartFinishedEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StartFinishedEvent)
			if ok {
				return handler(event)
			} else {
				return InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}
