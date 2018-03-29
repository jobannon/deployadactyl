package mocks

import (
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
)

// PusherCreator handmade mock for tests.
type PusherCreatorFactory struct {
	PusherCreatorCall struct {
		Called   bool
		Received struct {
			DeployEventData structs.DeployEventData
		}
		Returns struct {
			ActionCreator interfaces.ActionCreator
		}
	}
}

// CreatePusher mock method.

func (p *PusherCreatorFactory) PusherCreator(deployEventData structs.DeployEventData) interfaces.ActionCreator {
	p.PusherCreatorCall.Called = true
	p.PusherCreatorCall.Received.DeployEventData = deployEventData

	return p.PusherCreatorCall.Returns.ActionCreator
}
