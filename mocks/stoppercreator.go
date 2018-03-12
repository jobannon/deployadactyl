package mocks

import (
	"github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"

	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"io"
)

type receivedCall struct {
	DeploymentInfo S.DeploymentInfo
	Response       io.ReadWriter
}

type StopperCreator struct {
	CreateStopperCall struct {
		TimesCalled int
		Received    []receivedCall
		Returns     struct {
			Stoppers []interfaces.Action
			Error    []error
		}
	}
}

func (s *StopperCreator) Create(deploymentInfo S.DeploymentInfo, cfContext interfaces.CFContext, authorization interfaces.Authorization, environment S.Environment, response io.ReadWriter, foundationURL, appPath string) (interfaces.Action, error) {
	defer func() { s.CreateStopperCall.TimesCalled++ }()

	received := receivedCall{
		DeploymentInfo: deploymentInfo,
		Response:       response,
	}
	s.CreateStopperCall.Received = append(s.CreateStopperCall.Received, received)

	return s.CreateStopperCall.Returns.Stoppers[s.CreateStopperCall.TimesCalled], s.CreateStopperCall.Returns.Error[s.CreateStopperCall.TimesCalled]
}

func (s *StopperCreator) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (s *StopperCreator) ExecuteError(executeErrors []error) error {
	return bluegreen.StopError{Errors: executeErrors}
}

func (s *StopperCreator) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackStopError{StopErrors: executeErrors, RollbackErrors: undoErrors}
}

func (s *StopperCreator) SuccessError(successErrors []error) error {
	return bluegreen.FinishStopError{FinishStopErrors: successErrors}
}
