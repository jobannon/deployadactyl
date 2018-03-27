package actioncreator

import (
	"encoding/base64"
	"fmt"
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

const deploymentOutput = `Deployment Parameters:
Artifact URL: %s,
Username:     %s,
Environment:  %s,
Org:          %s,
Space:        %s,
AppName:      %s`

type courierCreator interface {
	CreateCourier() (I.Courier, error)
}

type fileSystemCleaner interface {
	RemoveAll(path string) error
}

type PusherCreator struct {
	CourierCreator    courierCreator
	EventManager      I.EventManager
	Logger            logger.DeploymentLogger
	Fetcher           I.Fetcher
	DeployEventData   S.DeployEventData
	FileSystemCleaner fileSystemCleaner
}

type StopperCreator struct {
	CourierCreator  courierCreator
	EventManager    I.EventManager
	Logger          I.Logger
	DeployEventData S.DeployEventData
}

func (a *PusherCreator) SetUp(envInstances uint16) error {
	var (
		manifestString string
		instances      *uint16
		appPath        string
		err            error
	)

	var fetchFn func() (string, error)

	if a.DeployEventData.DeploymentInfo.ContentType == "JSON" {

		if a.DeployEventData.DeploymentInfo.Manifest != "" {
			manifest, err := base64.StdEncoding.DecodeString(a.DeployEventData.DeploymentInfo.Manifest)
			if err != nil {
				return pusher.ManifestError{}
			}
			manifestString = string(manifest)
		}

		instances = manifestro.GetInstances(manifestString)
		if instances == nil {
			instances = &envInstances
		}

		fetchFn = func() (string, error) {
			a.Logger.Debug("deploying from json request")
			appPath, err = a.Fetcher.Fetch(a.DeployEventData.DeploymentInfo.ArtifactURL, manifestString)
			if err != nil {
				return "", pusher.AppPathError{Err: err}
			}
			return appPath, nil
		}
	} else {
		instanceVal := uint16(0)
		instances = &instanceVal

		fetchFn = func() (string, error) {
			a.Logger.Debug("deploying from zip request")
			appPath, err = a.Fetcher.FetchZipFromRequest(a.DeployEventData.DeploymentInfo.Body)
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
		return deployer.EventError{Type: constants.ArtifactRetrievalStart, Err: err}
	}

	appPath, err = fetchFn()
	if err != nil {
		a.Logger.Error(err)
		a.EventManager.Emit(I.Event{Type: constants.ArtifactRetrievalFailure, Data: deployEventData})
		return err
	}

	a.Logger.Debugf("emitting a %s event", constants.ArtifactRetrievalSuccess)
	err = a.EventManager.Emit(I.Event{Type: constants.ArtifactRetrievalSuccess, Data: deployEventData})
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return deployer.EventError{Type: constants.ArtifactRetrievalSuccess, Err: err}
	}

	a.DeployEventData.DeploymentInfo.Manifest = manifestString
	a.DeployEventData.DeploymentInfo.AppPath = appPath
	a.DeployEventData.DeploymentInfo.Instances = *instances

	return nil
}

func (a PusherCreator) OnStart() error {
	info := a.DeployEventData.DeploymentInfo
	deploymentMessage := fmt.Sprintf(deploymentOutput, info.ArtifactURL, info.Username, info.Environment, info.Org, info.Space, info.AppName)

	a.Logger.Info(deploymentMessage)
	fmt.Fprintln(a.DeployEventData.Writer, deploymentMessage)

	err := a.EventManager.Emit(I.Event{Type: constants.PushStartedEvent, Data: a.DeployEventData})
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return deployer.EventError{Type: constants.PushStartedEvent, Err: err}
	}
	return nil
}

func (a PusherCreator) CleanUp() {
	a.FileSystemCleaner.RemoveAll(a.DeployEventData.DeploymentInfo.AppPath)
}

func (a PusherCreator) Create(environment S.Environment, response io.ReadWriter, foundationURL string) (I.Action, error) {

	courier, err := a.CourierCreator.CreateCourier()
	if err != nil {
		a.Logger.Error(err)
		return &pusher.Pusher{}, pusher.CourierCreationError{Err: err}
	}

	p := &pusher.Pusher{
		Courier:        courier,
		DeploymentInfo: *a.DeployEventData.DeploymentInfo,
		EventManager:   a.EventManager,
		Response:       response,
		Log:            logger.DeploymentLogger{a.Logger, a.DeployEventData.DeploymentInfo.UUID},
		FoundationURL:  foundationURL,
		AppPath:        a.DeployEventData.DeploymentInfo.AppPath,
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

func (a StopperCreator) SetUp(envInstances uint16) error {
	return nil
}

func (a StopperCreator) OnStart() error {
	return nil
}

func (a StopperCreator) CleanUp() {}

func (a StopperCreator) Create(environment S.Environment, response io.ReadWriter, foundationURL string) (I.Action, error) {
	courier, err := a.CourierCreator.CreateCourier()
	if err != nil {
		a.Logger.Error(err)
		return &pusher.Pusher{}, pusher.CourierCreationError{Err: err}
	}
	p := &startstopper.Stopper{
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
		Log:           logger.DeploymentLogger{a.Logger, a.DeployEventData.DeploymentInfo.UUID},
		FoundationURL: foundationURL,
		AppName:       a.DeployEventData.DeploymentInfo.AppName,
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
