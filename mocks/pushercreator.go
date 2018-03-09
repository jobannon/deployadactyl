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
			Pushers []interfaces.Action
			Error   []error
		}
	}
}

// CreatePusher mock method.
func (p *PusherCreator) Create(deploymentInfo S.DeploymentInfo, cfContext interfaces.CFContext, authorization interfaces.Authorization, response io.ReadWriter, foundationURL, appPath string) (interfaces.Action, error) {
	defer func() { p.CreatePusherCall.TimesCalled++ }()

	return p.CreatePusherCall.Returns.Pushers[p.CreatePusherCall.TimesCalled], p.CreatePusherCall.Returns.Error[p.CreatePusherCall.TimesCalled]
}
