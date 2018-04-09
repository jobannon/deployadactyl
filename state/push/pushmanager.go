package push

import (
	"encoding/base64"
	"fmt"
	"github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/manifestro"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/state"
	S "github.com/compozed/deployadactyl/structs"
	"io"
	"net/http"
	"regexp"
)

const deploymentOutput = `Deployment Parameters:
Artifact URL: %s,
Username:     %s,
Environment:  %s,
Org:          %s,
Space:        %s,
AppName:      %s`

const successfulDeploy = `Your deploy was successful! (^_^)b
If you experience any problems after this point, check that you can manually push your application to Cloud Foundry on a lower environment.
It is likely that it is an error with your application and not with Deployadactyl.
Thanks for using Deployadactyl! Please push down pull up on your lap bar and exit to your left.

`

type courierCreator interface {
	CreateCourier() (I.Courier, error)
}

type fileSystemCleaner interface {
	RemoveAll(path string) error
}

type PushManager struct {
	CourierCreator       courierCreator
	EventManager         I.EventManager
	Logger               logger.DeploymentLogger
	Fetcher              I.Fetcher
	DeployEventData      S.DeployEventData
	FileSystemCleaner    fileSystemCleaner
	CFContext            I.CFContext
	Auth                 I.Authorization
	Environment          S.Environment
	EnvironmentVariables map[string]string
}

func (a *PushManager) SetUp() error {
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
				return state.ManifestError{}
			}
			manifestString = string(manifest)
		}

		instances = manifestro.GetInstances(manifestString)
		if instances == nil {
			instances = &a.Environment.Instances
		}

		fetchFn = func() (string, error) {
			a.Logger.Debug("deploying from json request")
			appPath, err = a.Fetcher.Fetch(a.DeployEventData.DeploymentInfo.ArtifactURL, manifestString)
			if err != nil {
				return "", state.AppPathError{Err: err}
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
				return "", state.UnzippingError{Err: err}
			}
			return appPath, nil
		}
	}

	var event I.IEvent
	event = ArtifactRetrievalStartEvent{
		CFContext:   a.CFContext,
		Auth:        a.Auth,
		Environment: a.Environment,
		Response:    a.DeployEventData.Response,
		Data:        a.DeployEventData.DeploymentInfo.Data,
		Manifest:    manifestString,
		ArtifactURL: a.DeployEventData.DeploymentInfo.ArtifactURL,
	}
	a.Logger.Debugf("emitting a %s event", event.Name())

	err = a.EventManager.EmitEvent(event)
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return deployer.EventError{Type: event.Name(), Err: err}
	}

	appPath, err = fetchFn()
	if err != nil {
		a.Logger.Error(err)
		event = ArtifactRetrievalFailureEvent{
			CFContext:   a.CFContext,
			Auth:        a.Auth,
			Environment: a.Environment,
			Response:    a.DeployEventData.Response,
			Data:        a.DeployEventData.DeploymentInfo.Data,
			Manifest:    manifestString,
			ArtifactURL: a.DeployEventData.DeploymentInfo.ArtifactURL,
		}
		a.EventManager.EmitEvent(event)
		return err
	}

	event = ArtifactRetrievalSuccessEvent{
		CFContext:            a.CFContext,
		Auth:                 a.Auth,
		Environment:          a.Environment,
		Response:             a.DeployEventData.Response,
		Data:                 a.DeployEventData.DeploymentInfo.Data,
		Manifest:             manifestString,
		ArtifactURL:          a.DeployEventData.DeploymentInfo.ArtifactURL,
		AppPath:              appPath,
		EnvironmentVariables: a.EnvironmentVariables,
	}
	a.Logger.Debugf("emitting a %s event", event.Name())
	err = a.EventManager.EmitEvent(event)
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return deployer.EventError{Type: event.Name(), Err: err}
	}

	a.DeployEventData.DeploymentInfo.Manifest = manifestString
	a.DeployEventData.DeploymentInfo.AppPath = appPath
	a.DeployEventData.DeploymentInfo.Instances = *instances

	return nil
}

func (a PushManager) OnStart() error {
	info := a.DeployEventData.DeploymentInfo
	deploymentMessage := fmt.Sprintf(deploymentOutput, info.ArtifactURL, info.Username, info.Environment, info.Org, info.Space, info.AppName)

	a.Logger.Info(deploymentMessage)
	fmt.Fprintln(a.DeployEventData.Response, deploymentMessage)

	err := a.EventManager.Emit(I.Event{Type: constants.PushStartedEvent, Data: &a.DeployEventData})
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return deployer.EventError{Type: constants.PushStartedEvent, Err: err}
	}

	event := PushStartedEvent{
		CFContext:   a.CFContext,
		Auth:        a.Auth,
		Environment: a.Environment,
		Body:        info.Body,
		Response:    a.DeployEventData.Response,
		ContentType: info.ContentType,
		Data:        info.Data,
		Instances:   info.Instances,
	}
	err = a.EventManager.EmitEvent(event)
	if err != nil {
		a.Logger.Error(err)
		err = &bluegreen.InitializationError{err}
		return deployer.EventError{Type: event.Name(), Err: err}
	}
	return nil
}

func (a PushManager) OnFinish(env S.Environment, response io.ReadWriter, err error) I.DeployResponse {
	if err != nil {
		if !env.EnableRollback {
			a.Logger.Errorf("EnableRollback %t, returning status %d and err %s", env.EnableRollback, http.StatusOK, err)
			return I.DeployResponse{
				StatusCode: http.StatusOK,
				Error:      err,
			}
		}

		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			return I.DeployResponse{
				StatusCode: http.StatusBadRequest,
				Error:      err,
			}
		}

		return I.DeployResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      err,
		}
	}
	a.Logger.Infof("successfully deployed application %s", a.DeployEventData.DeploymentInfo.AppName)
	fmt.Fprintf(response, "\n%s", successfulDeploy)

	return I.DeployResponse{StatusCode: http.StatusOK}
}

func (a PushManager) CleanUp() {
	a.FileSystemCleaner.RemoveAll(a.DeployEventData.DeploymentInfo.AppPath)
}

func (a PushManager) Create(environment S.Environment, response io.ReadWriter, foundationURL string) (I.Action, error) {

	courier, err := a.CourierCreator.CreateCourier()
	if err != nil {
		a.Logger.Error(err)
		return &Pusher{}, state.CourierCreationError{Err: err}
	}

	p := &Pusher{
		Courier:        courier,
		DeploymentInfo: *a.DeployEventData.DeploymentInfo,
		EventManager:   a.EventManager,
		Response:       response,
		Log:            a.Logger,
		FoundationURL:  foundationURL,
		AppPath:        a.DeployEventData.DeploymentInfo.AppPath,
		Environment:    environment,
		Fetcher:        a.Fetcher,
		CFContext:      a.CFContext,
		Auth:           a.Auth,
	}

	return p, nil
}

func (a PushManager) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (a PushManager) ExecuteError(executeErrors []error) error {
	return bluegreen.PushError{PushErrors: executeErrors}
}

func (a PushManager) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackError{PushErrors: executeErrors, RollbackErrors: undoErrors}
}

func (a PushManager) SuccessError(successErrors []error) error {
	return bluegreen.FinishPushError{FinishPushError: successErrors}
}
