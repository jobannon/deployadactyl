package start

import (
	"io"

	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/state"
	S "github.com/compozed/deployadactyl/structs"
)

type courierCreator interface {
	CreateCourier() (I.Courier, error)
}

type StartManager struct {
	CourierCreator  courierCreator
	EventManager    I.EventManager
	Logger          logger.DeploymentLogger
	DeployEventData S.DeployEventData
}

func (a StartManager) SetUp() error {
	return nil
}

func (a StartManager) OnStart() error {
	return nil
}

func (a StartManager) OnFinish(env S.Environment, response io.ReadWriter, err error) I.DeployResponse {
	return I.DeployResponse{}
}

func (a StartManager) CleanUp() {}

func (a StartManager) Create(environment S.Environment, response io.ReadWriter, foundationURL string) (I.Action, error) {
	courier, err := a.CourierCreator.CreateCourier()
	if err != nil {
		a.Logger.Error(err)
		return &Starter{}, state.CourierCreationError{Err: err}
	}
	p := &Starter{
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
		Data:          a.DeployEventData.DeploymentInfo.Data,
	}

	return p, nil
}

func (a StartManager) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (a StartManager) ExecuteError(executeErrors []error) error {
	return bluegreen.StartError{Errors: executeErrors}
}

func (a StartManager) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackStartError{StartErrors: executeErrors, RollbackErrors: undoErrors}
}

func (a StartManager) SuccessError(successErrors []error) error {
	return bluegreen.FinishStartError{FinishStartErrors: successErrors}
}
