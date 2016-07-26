package mocks

import "github.com/compozed/deployadactyl/interfaces"

type PusherFactory struct {
	Iterator         int
	CreatePusherCall struct {
		Returns struct {
			Pushers []interfaces.Pusher
			Error   error
		}
	}
}

func (p *PusherFactory) CreatePusher() (interfaces.Pusher, error) {
	defer func() { p.Iterator++ }()
	return p.CreatePusherCall.Returns.Pushers[p.Iterator], p.CreatePusherCall.Returns.Error
}
