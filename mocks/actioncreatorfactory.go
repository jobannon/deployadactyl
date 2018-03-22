package mocks

import (
	"github.com/compozed/deployadactyl/interfaces"
	"io"
)

// PusherCreator handmade mock for tests.
type PusherCreatorFactory struct {
	PusherCreatorCall struct {
		Called   bool
		Received struct {
			Body io.Reader
		}
		Returns struct {
			ActionCreator interfaces.ActionCreator
		}
	}
}

// CreatePusher mock method.

func (p *PusherCreatorFactory) PusherCreator(body io.Reader) interfaces.ActionCreator {
	p.PusherCreatorCall.Called = true
	p.PusherCreatorCall.Received.Body = body
	return p.PusherCreatorCall.Returns.ActionCreator
}
