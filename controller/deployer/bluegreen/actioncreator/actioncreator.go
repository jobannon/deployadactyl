package actioncreator

import (
	"io"

	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/startstopper"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
)

type PusherCreator struct {
	Courier      I.Courier
	EventManager I.EventManager
	Logger       I.Logger
}

type StopperCreator struct {
	Courier      I.Courier
	EventManager I.EventManager
	Logger       I.Logger
}

func (a PusherCreator) Create(deploymentInfo S.DeploymentInfo, cfContext I.CFContext, authorization I.Authorization, environment S.Environment, response io.ReadWriter, foundationURL, appPath string) (I.Action, error) {

	p := &pusher.Pusher{
		Courier:        a.Courier,
		DeploymentInfo: deploymentInfo,
		EventManager:   a.EventManager,
		Response:       response,
		Log:            logger.DeploymentLogger{a.Logger, deploymentInfo.UUID},
		FoundationURL:  foundationURL,
		AppPath:        appPath,
		Environment:    environment,
	}

	return p, nil
}

func (a PusherCreator) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (a PusherCreator) ExecuteError(executeErrors []error) error {
	return bluegreen.PushError{PushErrors: executeErrors}
}

func (a PusherCreator) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackError{PushErrors: executeErrors, RollbackErrors: undoErrors}
}

func (a PusherCreator) SuccessError(successErrors []error) error {
	return bluegreen.FinishPushError{FinishPushError: successErrors}
}

func (a StopperCreator) Create(deploymentInfo S.DeploymentInfo, cfContext I.CFContext, authorization I.Authorization, environment S.Environment, response io.ReadWriter, foundationURL, appPath string) (I.Action, error) {

	p := &startstopper.Stopper{
		Courier:       a.Courier,
		CFContext:     cfContext,
		Authorization: authorization,
		EventManager:  a.EventManager,
		Response:      response,
		Log:           logger.DeploymentLogger{a.Logger, deploymentInfo.UUID},
		FoundationURL: foundationURL,
		AppName:       deploymentInfo.AppName,
	}

	return p, nil
}

func (a StopperCreator) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (a StopperCreator) ExecuteError(executeErrors []error) error {
	return bluegreen.StopError{Errors: executeErrors}
}

func (a StopperCreator) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackStopError{StopErrors: executeErrors, RollbackErrors: undoErrors}
}

func (a StopperCreator) SuccessError(successErrors []error) error {
	return bluegreen.FinishStopError{FinishStopErrors: successErrors}
}
