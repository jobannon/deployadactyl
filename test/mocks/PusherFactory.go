package mocks

import "github.com/compozed/deployadactyl/interfaces"

type PusherFactory struct {
	CreatePusherCall struct {
		TimesCalled int
		Returns     struct {
			Pushers []interfaces.Pusher
			Error   error
		}
	}
}

func (p *PusherFactory) CreatePusher() (interfaces.Pusher, error) {
	defer func() { p.CreatePusherCall.TimesCalled++ }()

	return p.CreatePusherCall.Returns.Pushers[p.CreatePusherCall.TimesCalled], p.CreatePusherCall.Returns.Error
}
