package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/geterrors"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type PushControllerConstructor func(log I.DeploymentLogger, deployer, silentDeployer I.Deployer, conf config.Config, eventManager I.EventManager, errorFinder I.ErrorFinder, pushManagerFactory I.PushManagerFactory) I.PushController

func NewPushController(l I.DeploymentLogger, d, sd I.Deployer, c config.Config, em I.EventManager, ef I.ErrorFinder, pmf I.PushManagerFactory) I.PushController {
	return &PushController{
		Deployer:           d,
		SilentDeployer:     sd,
		Config:             c,
		EventManager:       em,
		ErrorFinder:        ef,
		PushManagerFactory: pmf,
		Log:                l,
	}
}

type PushController struct {
	Deployer           I.Deployer
	SilentDeployer     I.Deployer
	Log                I.DeploymentLogger
	Config             config.Config
	EventManager       I.EventManager
	ErrorFinder        I.ErrorFinder
	PushManagerFactory I.PushManagerFactory
}

// PUSH specific
func (c *PushController) RunDeployment(deployment *I.Deployment, response *bytes.Buffer) (deployResponse I.DeployResponse) {
	cf := deployment.CFContext
	deploymentInfo := &structs.DeploymentInfo{
		Org:         cf.Organization,
		Space:       cf.Space,
		AppName:     cf.Application,
		Environment: cf.Environment,
		UUID:        c.Log.UUID,
	}

	c.Log.Debugf("Starting deploy of %s with UUID %s", cf.Application, deploymentInfo.UUID)
	c.Log.Debug("building deploymentInfo")

	body := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	if deployment.Type.JSON {
		c.Log.Debug("deploying from json request")

		deploymentInfo.ContentType = "JSON"
	} else if deployment.Type.ZIP {
		c.Log.Debug("deploying from zip request")
		deploymentInfo.Body = body
		deploymentInfo.ContentType = "ZIP"
	} else {
		return I.DeployResponse{
			StatusCode: http.StatusBadRequest,
			Error:      deployer.InvalidContentTypeError{},
		}
	}
	environment, err := c.resolveEnvironment(cf.Environment)
	if err != nil {
		fmt.Fprintln(response, err.Error())
		return I.DeployResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      err,
		}
	}
	auth, err := c.resolveAuthorization(deployment.Authorization, environment, c.Log)
	if err != nil {
		return I.DeployResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      err,
		}
	}

	deploymentInfo.Username = auth.Username
	deploymentInfo.Password = auth.Password
	deploymentInfo.Domain = environment.Domain
	deploymentInfo.SkipSSL = environment.SkipSSL
	deploymentInfo.CustomParams = environment.CustomParams

	if deployment.Type.JSON {
		deploymentInfo, err = c.getDeploymentInfo(deployment.Body, deploymentInfo)
		if err != nil {
			c.Log.Error(err)
			return I.DeployResponse{
				StatusCode:     http.StatusInternalServerError,
				Error:          err,
				DeploymentInfo: deploymentInfo,
			}
		}
	}

	deployEventData := structs.DeployEventData{Response: response, DeploymentInfo: deploymentInfo, RequestBody: body}
	defer c.emitDeployFinish(&deployEventData, response, cf, auth, environment, &deployResponse, c.Log)
	defer c.emitDeploySuccessOrFailure(&deployEventData, response, cf, auth, environment, &deployResponse, c.Log)

	c.Log.Debugf("emitting a %s event", constants.DeployStartEvent)

	err = c.EventManager.Emit(I.Event{Type: constants.DeployStartEvent, Data: &deployEventData})
	if err != nil {
		c.Log.Error(err)
		err = &bluegreen.InitializationError{err}
		return I.DeployResponse{
			StatusCode:     http.StatusInternalServerError,
			Error:          deployer.EventError{Type: constants.DeployStartEvent, Err: err},
			DeploymentInfo: deploymentInfo,
		}
	}

	err = c.EventManager.EmitEvent(DeployStartedEvent{
		CFContext:   cf,
		Auth:        auth,
		Body:        body,
		ContentType: deploymentInfo.ContentType,
		Environment: environment,
		Response:    response,
		ArtifactURL: deploymentInfo.ArtifactURL,
		Data:        deploymentInfo.Data,
		Log:         c.Log,
	})
	if err != nil {
		c.Log.Error(err)
		err = &bluegreen.InitializationError{err}
		return I.DeployResponse{
			StatusCode:     http.StatusInternalServerError,
			Error:          deployer.EventError{Type: constants.DeployStartEvent, Err: err},
			DeploymentInfo: deploymentInfo,
		}
	}

	pusherCreator := c.PushManagerFactory.PushManager(c.Log, deployEventData, cf, auth, environment, deploymentInfo.EnvironmentVariables)

	reqChannel1 := make(chan *I.DeployResponse)
	reqChannel2 := make(chan *I.DeployResponse)
	defer close(reqChannel1)
	defer close(reqChannel2)

	go func() {
		reqChannel1 <- c.Deployer.Deploy(deploymentInfo, environment, pusherCreator, response)
	}()

	silentResponse := &bytes.Buffer{}
	if cf.Environment == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go func() {
			reqChannel2 <- c.SilentDeployer.Deploy(deploymentInfo, environment, pusherCreator, silentResponse)
		}()
		<-reqChannel2
	}

	deployResponse = *<-reqChannel1

	return deployResponse
}

func (c *PushController) getDeploymentInfo(body *[]byte, deploymentInfo *structs.DeploymentInfo) (*structs.DeploymentInfo, error) {
	reader := ioutil.NopCloser(bytes.NewBuffer(*body))
	err := json.NewDecoder(reader).Decode(deploymentInfo)
	if err != nil {
		return deploymentInfo, err
	}

	getter := geterrors.WrapFunc(func(key string) string {
		if key == "artifact_url" {
			return deploymentInfo.ArtifactURL
		}
		return ""
	})

	getter.Get("artifact_url")

	err = getter.Err("The following properties are missing")
	if err != nil {
		return &structs.DeploymentInfo{}, err
	}
	return deploymentInfo, nil
}

func (c *PushController) resolveAuthorization(auth I.Authorization, envs structs.Environment, deploymentLogger I.DeploymentLogger) (I.Authorization, error) {
	config := c.Config
	deploymentLogger.Debug("checking for basic auth")
	if auth.Username == "" && auth.Password == "" {
		if envs.Authenticate {
			return I.Authorization{}, deployer.BasicAuthError{}

		}
		auth.Username = config.Username
		auth.Password = config.Password
	}

	return auth, nil
}

func (c *PushController) resolveEnvironment(env string) (structs.Environment, error) {
	config := c.Config
	environment, ok := config.Environments[env]
	if !ok {
		return structs.Environment{}, deployer.EnvironmentNotFoundError{env}
	}
	return environment, nil
}

func (c *PushController) emitDeployFinish(deployEventData *structs.DeployEventData, response io.ReadWriter, cf I.CFContext, auth I.Authorization, environment structs.Environment, deployResponse *I.DeployResponse, deploymentLogger I.DeploymentLogger) {
	deploymentLogger.Debugf("emitting a %s event", constants.DeployFinishEvent)
	finishErr := c.EventManager.Emit(I.Event{Type: constants.DeployFinishEvent, Data: deployEventData})
	if finishErr != nil {
		fmt.Fprintln(response, finishErr)
		err := bluegreen.FinishDeployError{Err: fmt.Errorf("%s: %s", deployResponse.Error, deployer.EventError{constants.DeployFinishEvent, finishErr})}
		deployResponse.Error = err
		deployResponse.StatusCode = http.StatusInternalServerError
	}

	finishErr = c.EventManager.EmitEvent(DeployFinishedEvent{
		CFContext:   cf,
		Auth:        auth,
		Body:        deployEventData.RequestBody,
		ContentType: deployEventData.DeploymentInfo.ContentType,
		Environment: environment,
		Response:    deployEventData.Response,
		Data:        deployEventData.DeploymentInfo.Data,
		Log:         c.Log,
	})
	if finishErr != nil {
		fmt.Fprintln(response, finishErr)
		if finishErr != nil {
			fmt.Fprintln(response, finishErr)
			err := bluegreen.FinishDeployError{Err: fmt.Errorf("%s: %s", deployResponse.Error, deployer.EventError{constants.DeployFinishEvent, finishErr})}
			deployResponse.Error = err
			deployResponse.StatusCode = http.StatusInternalServerError
		}
	}
}

func (c PushController) emitDeploySuccessOrFailure(deployEventData *structs.DeployEventData, response io.ReadWriter, cf I.CFContext, auth I.Authorization, environment structs.Environment, deployResponse *I.DeployResponse, deploymentLogger I.DeploymentLogger) {
	deployEvent := I.Event{Type: constants.DeploySuccessEvent, Data: deployEventData}
	if deployResponse.Error != nil {
		c.printErrors(response, &deployResponse.Error)

		deployEvent.Type = constants.DeployFailureEvent
		deployEvent.Error = deployResponse.Error
	}
	deploymentLogger.Debug(fmt.Sprintf("emitting a %s event", deployEvent.Name()))
	eventErr := c.EventManager.Emit(deployEvent)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", deployEvent.Name(), eventErr)
		fmt.Fprintln(response, eventErr)
		return
	}

	var event I.IEvent
	if deployResponse.Error != nil {
		event = DeployFailureEvent{
			CFContext:   cf,
			Auth:        auth,
			Body:        deployEventData.RequestBody,
			ContentType: deployEventData.DeploymentInfo.ContentType,
			Environment: environment,
			Response:    deployEventData.Response,
			Data:        deployEventData.DeploymentInfo.Data,
			Error:       deployResponse.Error,
			Log:         c.Log,
		}
	} else {
		event = DeploySuccessEvent{
			CFContext:           cf,
			Auth:                auth,
			Body:                deployEventData.RequestBody,
			ContentType:         deployEventData.DeploymentInfo.ContentType,
			Environment:         environment,
			Response:            deployEventData.Response,
			Data:                deployEventData.DeploymentInfo.Data,
			HealthCheckEndpoint: deployEventData.DeploymentInfo.HealthCheckEndpoint,
			ArtifactURL:         deployEventData.DeploymentInfo.ArtifactURL,
			Log:                 c.Log,
		}
	}
	deploymentLogger.Debug(fmt.Sprintf("emitting a %s event", event.Name()))
	eventErr = c.EventManager.EmitEvent(event)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", event.Name(), eventErr)
		fmt.Fprintln(response, eventErr)
	}

}

func (c PushController) printErrors(response io.ReadWriter, err *error) {
	tempBuffer := bytes.Buffer{}
	tempBuffer.ReadFrom(response)
	fmt.Fprint(response, tempBuffer.String())

	errors := c.ErrorFinder.FindErrors(tempBuffer.String())
	if len(errors) > 0 {
		*err = errors[0]
		for _, error := range errors {
			fmt.Fprintln(response)
			fmt.Fprintln(response, "*******************")
			fmt.Fprintln(response)
			fmt.Fprintln(response, "The following error was found in the above logs: "+error.Error())
			fmt.Fprintln(response)
			fmt.Fprintln(response, "Error: "+error.Details()[0])
			fmt.Fprintln(response)
			fmt.Fprintln(response, "Potential solution: "+error.Solution())
			fmt.Fprintln(response)
			fmt.Fprintln(response, "*******************")
		}
	}
}
