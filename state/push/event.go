package push

import (
	"bytes"
	"errors"
	"github.com/compozed/deployadactyl/eventmanager"
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
	"io"
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

type DeployStartedEvent struct {
	CFContext   interfaces.CFContext
	Body        io.ReadCloser
	ContentType string
	Environment structs.Environment
	Auth        interfaces.Authorization
	Response    *bytes.Buffer
}

func (d DeployStartedEvent) Name() string {
	return "DeployStartedEvent"
}

func NewDeployStartEventBinding(handler func(event DeployStartedEvent) error) interfaces.Binding {
	return eventBinding{
		etype: reflect.TypeOf(DeployStartedEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(DeployStartedEvent)
			if ok {
				return handler(event)
			} else {
				return eventmanager.InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}

type DeployFinishedEvent struct {
	CFContext   interfaces.CFContext
	Body        io.ReadCloser
	ContentType string
	Environment structs.Environment
	Auth        interfaces.Authorization
	Response    *bytes.Buffer
}

func (d DeployFinishedEvent) Name() string {
	return "DeployFinishEvent"
}

func NewDeployFinishedEventBinding(handler func(event DeployFinishedEvent) error) interfaces.Binding {
	return eventBinding{
		etype: reflect.TypeOf(DeployFinishedEvent{}),
		handler: func(gevent interface{}) error {
			event, ok := gevent.(DeployFinishedEvent)
			if ok {
				return handler(event)
			} else {
				return eventmanager.InvalidEventType{errors.New("invalid event type")}
			}
		},
	}
}
