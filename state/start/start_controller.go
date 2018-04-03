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
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/structs"
	"github.com/compozed/randomizer"
)

type startManagerFactory interface {
	StartManager(deployEventData structs.DeployEventData) I.ActionCreator
}

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Log                 I.Logger
	StartManagerFactory startManagerFactory
	Deployer            I.Deployer
	Config              config.Config
	EventManager        I.EventManager
	ErrorFinder         I.ErrorFinder
}

func (c *Controller) StartDeployment(deployment *I.Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse I.DeployResponse) {
	auth := &I.Authorization{}
	environment := &structs.Environment{}

	cf := deployment.CFContext

	deploymentInfo := &structs.DeploymentInfo{
		Org:         cf.Organization,
		Space:       cf.Space,
		AppName:     cf.Application,
		Environment: cf.Environment,
		UUID:        cf.UUID,
	}
	if deploymentInfo.UUID == "" {
		deploymentInfo.UUID = randomizer.StringRunes(10)
		cf.UUID = deploymentInfo.UUID
	}
	deploymentLogger := logger.DeploymentLogger{c.Log, deploymentInfo.UUID}
	deploymentLogger.Debugf("Preparing to start %s with UUID %s", cf.Application, deploymentInfo.UUID)

	event := StartStartedEvent{CFContext: cf, Data: data}
	defer c.emitStartFinish(response, deploymentLogger, cf, auth, environment, data, &deployResponse)
	defer c.emitStartSuccessOrFailure(response, deploymentLogger, cf, auth, environment, data, &deployResponse)

	err := c.EventManager.EmitEvent(event)
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		return I.DeployResponse{
			StatusCode:     http.StatusInternalServerError,
			Error:          deployer.EventError{Type: "StartStartedEvent", Err: err},
			DeploymentInfo: deploymentInfo,
		}
	}

	*environment, err = c.resolveEnvironment(cf.Environment)
	if err != nil {
		fmt.Fprintln(response, err.Error())
		return I.DeployResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      err,
		}
	}
	*auth, err = c.resolveAuthorization(deployment.Authorization, *environment, deploymentLogger)
	if err != nil {
		return I.DeployResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      err,
		}
	}
	deploymentInfo.Domain = environment.Domain
	deploymentInfo.SkipSSL = environment.SkipSSL
	deploymentInfo.CustomParams = environment.CustomParams
	deploymentInfo.Username = auth.Username
	deploymentInfo.Password = auth.Password

	if data != nil {
		deploymentInfo.Data = data
	} else {
		deploymentInfo.Data = make(map[string]interface{})
	}

	deployEventData := structs.DeployEventData{Response: response, DeploymentInfo: deploymentInfo}

	manager := c.StartManagerFactory.StartManager(deployEventData)
	deployResponse = *c.Deployer.Deploy(deploymentInfo, *environment, manager, response)
	return deployResponse
}

func (c *Controller) resolveAuthorization(auth I.Authorization, envs structs.Environment, deploymentLogger logger.DeploymentLogger) (I.Authorization, error) {
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

func (c *Controller) resolveEnvironment(env string) (structs.Environment, error) {
	config := c.Config
	environment, ok := config.Environments[env]
	if !ok {
		return structs.Environment{}, deployer.EnvironmentNotFoundError{env}
	}
	return environment, nil
}

func (c Controller) emitStartFinish(response io.ReadWriter, deploymentLogger logger.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event IEvent
	event = StartFinishedEvent{
		CFContext:     cfContext,
		Authorization: *auth,
		Environment:   *environment,
		Data:          data,
	}
	deploymentLogger.Debugf("emitting a %s event", event.Type())
	c.EventManager.EmitEvent(event)
}

func (c Controller) emitStartSuccessOrFailure(response io.ReadWriter, deploymentLogger logger.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event IEvent

	if deployResponse.Error != nil {
		c.printErrors(response, &deployResponse.Error)
		event = StartFailureEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
			Error:         deployResponse.Error,
		}

	} else {
		event = StartSuccessEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
		}
	}
	eventErr := c.EventManager.EmitEvent(event)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", event.Type(), eventErr)
		fmt.Fprintln(response, eventErr)
	}
}

func (c Controller) printErrors(response io.ReadWriter, err *error) {
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
