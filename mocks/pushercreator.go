package mocks

import "github.com/compozed/deployadactyl/interfaces"

type PusherCreator struct {
	CreatePusherCall struct {
		TimesCalled int
		Returns     struct {
			Pushers      []interfaces.Pusher
			Error        error
			ResponseLogs []byte
		}
	}
}

func (p *PusherCreator) CreatePusher() (interfaces.Pusher, error, []byte) {
	defer func() { p.CreatePusherCall.TimesCalled++ }()

	return p.CreatePusherCall.Returns.Pushers[p.CreatePusherCall.TimesCalled], p.CreatePusherCall.Returns.Error, p.CreatePusherCall.Returns.ResponseLogs
}
