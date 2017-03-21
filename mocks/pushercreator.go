package mocks

import (
	"io"

	"github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// PusherCreator handmade mock for tests.
type PusherCreator struct {
	CreatePusherCall struct {
		TimesCalled int
		Returns     struct {
			Pushers []interfaces.Pusher
			Error   []error
		}
	}
}

// CreatePusher mock method.
func (p *PusherCreator) CreatePusher(deploymentInfo S.DeploymentInfo, response io.ReadWriter) (interfaces.Pusher, error) {
	defer func() { p.CreatePusherCall.TimesCalled++ }()

	return p.CreatePusherCall.Returns.Pushers[p.CreatePusherCall.TimesCalled], p.CreatePusherCall.Returns.Error[p.CreatePusherCall.TimesCalled]
}
