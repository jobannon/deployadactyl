package stop

import (
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/state"
	S "github.com/compozed/deployadactyl/structs"
	"io"
)

type courierCreator interface {
	CreateCourier() (I.Courier, error)
}

type StopManager struct {
	CourierCreator  courierCreator
	EventManager    I.EventManager
	Logger          logger.DeploymentLogger
	DeployEventData S.DeployEventData
}

func (a StopManager) SetUp(environment S.Environment) error {
	return nil
}

func (a StopManager) OnStart() error {
	return nil
}

func (a StopManager) OnFinish(env S.Environment, response io.ReadWriter, err error) I.DeployResponse {
	return I.DeployResponse{}
}

func (a StopManager) CleanUp() {}

func (a StopManager) Create(environment S.Environment, response io.ReadWriter, foundationURL string) (I.Action, error) {
	courier, err := a.CourierCreator.CreateCourier()
	if err != nil {
		a.Logger.Error(err)
		return &Stopper{}, state.CourierCreationError{Err: err}
	}
	p := &Stopper{
		Courier: courier,
		CFContext: I.CFContext{
			Environment:  environment.Name,
			Organization: a.DeployEventData.DeploymentInfo.Org,
			Space:        a.DeployEventData.DeploymentInfo.Space,
			Application:  a.DeployEventData.DeploymentInfo.AppName,
			UUID:         a.DeployEventData.DeploymentInfo.UUID,
			SkipSSL:      a.DeployEventData.DeploymentInfo.SkipSSL,
		},
		Authorization: I.Authorization{
			Username: a.DeployEventData.DeploymentInfo.Username,
			Password: a.DeployEventData.DeploymentInfo.Password,
		},
		EventManager:  a.EventManager,
		Response:      response,
		Log:           a.Logger,
		FoundationURL: foundationURL,
		AppName:       a.DeployEventData.DeploymentInfo.AppName,
	}

	return p, nil
}

func (a StopManager) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (a StopManager) ExecuteError(executeErrors []error) error {
	return bluegreen.StopError{Errors: executeErrors}
}

func (a StopManager) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackStopError{StopErrors: executeErrors, RollbackErrors: undoErrors}
}

func (a StopManager) SuccessError(successErrors []error) error {
	return bluegreen.FinishStopError{FinishStopErrors: successErrors}
}
