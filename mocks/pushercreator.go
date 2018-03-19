package mocks

import (
	"io"

	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
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

func (p *PusherCreator) SetUp(deploymentInfo S.DeploymentInfo) (string, string, uint16, error) {
	return "", "", 0, nil
}

func (p *PusherCreator) Create(deploymentInfo S.DeploymentInfo, cfContext interfaces.CFContext, authorization interfaces.Authorization, environment S.Environment, response io.ReadWriter, foundationURL, appPath string) (interfaces.Action, error) {
	defer func() { p.CreatePusherCall.TimesCalled++ }()

	return p.CreatePusherCall.Returns.Pushers[p.CreatePusherCall.TimesCalled], p.CreatePusherCall.Returns.Error[p.CreatePusherCall.TimesCalled]
}

func (p *PusherCreator) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (p *PusherCreator) ExecuteError(executeErrors []error) error {
	return bluegreen.PushError{PushErrors: executeErrors}
}

func (p *PusherCreator) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackError{PushErrors: executeErrors, RollbackErrors: undoErrors}
}

func (p *PusherCreator) SuccessError(successErrors []error) error {
	return bluegreen.FinishPushError{FinishPushError: successErrors}
}
