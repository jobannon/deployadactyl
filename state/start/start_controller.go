package start

import (
	"bytes"
	"fmt"
	"net/http"

	"io"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
)

type StartControllerConstructor func(log I.DeploymentLogger, deployer I.Deployer, conf config.Config, eventManager I.EventManager, errorFinder I.ErrorFinder, startManagerFactory I.StartManagerFactory) I.StartController

func NewStartController(l I.DeploymentLogger, d I.Deployer, c config.Config, em I.EventManager, ef I.ErrorFinder, smf I.StartManagerFactory) I.StartController {
	return &StartController{
		Deployer: d,
		Config: c,
		EventManager: em,
		ErrorFinder: ef,
		StartManagerFactory: smf,
		Log: l,
	}
}


// StartController is used to determine the type of request and process it accordingly.
type StartController struct {
	Log                 I.DeploymentLogger
	StartManagerFactory I.StartManagerFactory
	Deployer            I.Deployer
	Config              config.Config
	EventManager        I.EventManager
	ErrorFinder         I.ErrorFinder
}

func (c *StartController) StartDeployment(deployment *I.Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse I.DeployResponse) {
	cf := deployment.CFContext
	if cf.UUID == "" {
		cf.UUID = c.Log.UUID
	}
	c.Log.Debugf("Preparing to start %s with UUID %s", cf.Application, cf.UUID)

	if data == nil {
		data = make(map[string]interface{})
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

	deploymentInfo := &structs.DeploymentInfo{
		Org:          cf.Organization,
		Space:        cf.Space,
		AppName:      cf.Application,
		Environment:  cf.Environment,
		UUID:         cf.UUID,
		Domain:       environment.Domain,
		SkipSSL:      environment.SkipSSL,
		CustomParams: environment.CustomParams,
		Username:     auth.Username,
		Password:     auth.Password,
		Data:         data,
	}

	defer c.emitStartFinish(response, c.Log, cf, &auth, &environment, data, &deployResponse)
	defer c.emitStartSuccessOrFailure(response, c.Log, cf, &auth, &environment, data, &deployResponse)

	err = c.EventManager.EmitEvent(StartStartedEvent{
		CFContext:     cf,
		Authorization: auth,
		Environment:   environment,
		Data:          data,
		Response:      response,
	})
	if err != nil {
		c.Log.Error(err)
		err = &bluegreen.InitializationError{err}
		return I.DeployResponse{
			StatusCode:     http.StatusInternalServerError,
			Error:          deployer.EventError{Type: "StartStartedEvent", Err: err},
			DeploymentInfo: deploymentInfo,
		}
	}

	deployEventData := structs.DeployEventData{Response: response, DeploymentInfo: deploymentInfo}

	manager := c.StartManagerFactory.StartManager(c.Log, deployEventData)
	deployResponse = *c.Deployer.Deploy(deploymentInfo, environment, manager, response)
	return deployResponse
}

func (c *StartController) resolveAuthorization(auth I.Authorization, envs structs.Environment, deploymentLogger I.DeploymentLogger) (I.Authorization, error) {
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

func (c *StartController) resolveEnvironment(env string) (structs.Environment, error) {
	config := c.Config
	environment, ok := config.Environments[env]
	if !ok {
		return structs.Environment{}, deployer.EnvironmentNotFoundError{env}
	}
	return environment, nil
}

func (c StartController) emitStartFinish(response io.ReadWriter, deploymentLogger I.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event I.IEvent
	event = StartFinishedEvent{
		CFContext:     cfContext,
		Authorization: *auth,
		Data:          data,
		Environment:   *environment,
	}
	deploymentLogger.Debugf("emitting a %s event", event.Name())
	c.EventManager.EmitEvent(event)
}

func (c StartController) emitStartSuccessOrFailure(response io.ReadWriter, deploymentLogger I.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event I.IEvent

	if deployResponse.Error != nil {
		c.printErrors(response, &deployResponse.Error)
		event = StartFailureEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
			Response:      response,
			Error:         deployResponse.Error,
		}

	} else {
		event = StartSuccessEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
			Response:      response,
		}
	}
	deploymentLogger.Debugf("emitting a %s event", event.Name())
	eventErr := c.EventManager.EmitEvent(event)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", event.Name(), eventErr)
		fmt.Fprintln(response, eventErr)
	}
}

func (c StartController) printErrors(response io.ReadWriter, err *error) {
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
