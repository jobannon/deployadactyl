package actioncreator

import (
	"encoding/base64"
	"github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/startstopper"
	"github.com/compozed/deployadactyl/controller/deployer/manifestro"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
	"io"
)

type courierCreator interface {
	CreateCourier() (I.Courier, error)
}

type PusherCreator struct {
	CourierCreator  courierCreator
	EventManager    I.EventManager
	Logger          logger.DeploymentLogger
	Fetcher         I.Fetcher
	DeployEventData S.DeployEventData
}

type StopperCreator struct {
	CourierCreator courierCreator
	EventManager   I.EventManager
	Logger         I.Logger
}

func (a PusherCreator) SetUp(deploymentInfo S.DeploymentInfo, envInstances uint16) (string, string, uint16, error) {
	var (
		manifestString string
		instances      *uint16
		appPath        string
		err            error
	)

	var fetchFn func() (string, error)
	if deploymentInfo.ContentType == "JSON" {

		if deploymentInfo.Manifest != "" {
			manifest, err := base64.StdEncoding.DecodeString(deploymentInfo.Manifest)
			if err != nil {
				return "", "", 0, pusher.ManifestError{}
			}
			manifestString = string(manifest)
		}

		instances = manifestro.GetInstances(manifestString)
		if instances == nil {
			instances = &envInstances
		}

		fetchFn = func() (string, error) {
			appPath, err = a.Fetcher.Fetch(deploymentInfo.ArtifactURL, manifestString)
			if err != nil {
				return "", pusher.AppPathError{Err: err}
			}
			return appPath, nil
		}
	} else {
		instanceVal := uint16(0)
		instances = &instanceVal

		fetchFn = func() (string, error) {
			appPath, err = a.Fetcher.FetchZipFromRequest(deploymentInfo.Body)
			if err != nil {
				return "", pusher.UnzippingError{Err: err}
			}
			return appPath, nil
		}
	}

	deployEventData := a.DeployEventData

	a.Logger.Debugf("emitting a %s event", constants.ArtifactRetrievalStart)
	err = a.EventManager.Emit(I.Event{Type: constants.ArtifactRetrievalStart, Data: deployEventData})
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return "", "", 0, deployer.EventError{Type: constants.ArtifactRetrievalStart, Err: err}

	}

	appPath, err = fetchFn()
	if err != nil {
		a.Logger.Error(err)
		a.EventManager.Emit(I.Event{Type: constants.ArtifactRetrievalFailure, Data: deployEventData})
		return "", "", 0, err
	}

	a.Logger.Debugf("emitting a %s event", constants.ArtifactRetrievalSuccess)
	err = a.EventManager.Emit(I.Event{Type: constants.ArtifactRetrievalSuccess, Data: deployEventData})
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return "", "", 0, deployer.EventError{Type: constants.ArtifactRetrievalSuccess, Err: err}
	}

	return appPath, manifestString, *instances, err
}

func (a PusherCreator) Create(deploymentInfo S.DeploymentInfo, cfContext I.CFContext, authorization I.Authorization, environment S.Environment, response io.ReadWriter, foundationURL, appPath string) (I.Action, error) {

	courier, err := a.CourierCreator.CreateCourier()
	if err != nil {
		a.Logger.Error(err)
		return &pusher.Pusher{}, pusher.CourierCreationError{Err: err}
	}

	p := &pusher.Pusher{
		Courier:        courier,
		DeploymentInfo: deploymentInfo,
		EventManager:   a.EventManager,
		Response:       response,
		Log:            logger.DeploymentLogger{a.Logger, deploymentInfo.UUID},
		FoundationURL:  foundationURL,
		AppPath:        appPath,
		Environment:    environment,
		Fetcher:        a.Fetcher,
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

func (a StopperCreator) SetUp(deploymentInfo S.DeploymentInfo, envInstances uint16) (string, string, uint16, error) {
	return "", "", 0, nil
}

func (a StopperCreator) Create(deploymentInfo S.DeploymentInfo, cfContext I.CFContext, authorization I.Authorization, environment S.Environment, response io.ReadWriter, foundationURL, appPath string) (I.Action, error) {
	courier, err := a.CourierCreator.CreateCourier()
	if err != nil {
		a.Logger.Error(err)
		return &pusher.Pusher{}, pusher.CourierCreationError{Err: err}
	}
	p := &startstopper.Stopper{
		Courier:       courier,
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
