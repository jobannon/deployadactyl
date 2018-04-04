package stop

import (
	"github.com/compozed/deployadactyl/eventmanager"
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"reflect"
)

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

type StopFailureEvent struct {
	CFContext     interfaces.CFContext
	Data          map[string]interface{}
	Authorization interfaces.Authorization
	Environment   structs.Environment
	Error         error
}

func (e StopFailureEvent) Name() string {
	return "StopFailureEvent"
}

func NewStopFailureEventBinding(handler func(event StopFailureEvent) error) interfaces.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StopFailureEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StopFailureEvent)
			if ok {
				return handler(event)
			} else {
				return eventmanager.InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}

type StopSuccessEvent struct {
	CFContext     interfaces.CFContext
	Data          map[string]interface{}
	Authorization interfaces.Authorization
	Environment   structs.Environment
}

func (e StopSuccessEvent) Name() string {
	return "StopSuccessEvent"
}

func NewStopSuccessEventBinding(handler func(event StopSuccessEvent) error) interfaces.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StopSuccessEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StopSuccessEvent)
			if ok {
				return handler(event)
			} else {
				return eventmanager.InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}

type StopStartedEvent struct {
	CFContext interfaces.CFContext
	Data      map[string]interface{}
}

func (e StopStartedEvent) Name() string {
	return "StopStartedEvent"
}

func NewStopStartedEventBinding(handler func(event StopStartedEvent) error) interfaces.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StopStartedEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StopStartedEvent)
			if ok {
				return handler(event)
			} else {
				return eventmanager.InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}

type StopFinishedEvent struct {
	CFContext     interfaces.CFContext
	Data          map[string]interface{}
	Authorization interfaces.Authorization
	Environment   structs.Environment
}

func (e StopFinishedEvent) Name() string {
	return "StopFinishedEvent"
}

func NewStopFinishedEventBinding(handler func(event StopFinishedEvent) error) interfaces.Binding {
	return eventBinding{
		etype: reflect.TypeOf(StopFinishedEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(StopFinishedEvent)
			if ok {
				return handler(event)
			} else {
				return eventmanager.InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}
