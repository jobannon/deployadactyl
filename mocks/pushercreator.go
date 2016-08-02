package mocks

import "github.com/compozed/deployadactyl/interfaces"

// PusherCreator handmade mock for tests.
type PusherCreator struct {
	CreatePusherCall struct {
		TimesCalled int
		Returns     struct {
			Pushers []interfaces.Pusher
			Error   error
		}
	}
}

// CreatePusher mock method.
func (p *PusherCreator) CreatePusher() (interfaces.Pusher, error) {
	defer func() { p.CreatePusherCall.TimesCalled++ }()

	return p.CreatePusherCall.Returns.Pushers[p.CreatePusherCall.TimesCalled], p.CreatePusherCall.Returns.Error
}
