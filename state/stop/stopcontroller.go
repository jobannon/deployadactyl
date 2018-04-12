package stop

import (
	"bytes"
	"fmt"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/structs"
	"io"
	"net/http"
)

type StopController struct {
	Deployer           I.Deployer
	Log                I.Logger
	StopManagerFactory I.StopManagerFactory
	Config             config.Config
	EventManager       I.EventManager
	ErrorFinder        I.ErrorFinder
}

func (c *StopController) StopDeployment(deployment *I.Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse I.DeployResponse) {
	cf := deployment.CFContext
	if cf.UUID == "" {
		cf.UUID = randomizer.StringRunes(10)
	}
	deploymentLogger := logger.DeploymentLogger{c.Log, cf.UUID}
	deploymentLogger.Debugf("Preparing to stop %s with UUID %s", cf.Application, cf.UUID)

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
	auth, err := c.resolveAuthorization(deployment.Authorization, environment, deploymentLogger)
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

	defer c.emitStopFinish(response, deploymentLogger, cf, &auth, &environment, data, &deployResponse)
	defer c.emitStopSuccessOrFailure(response, deploymentLogger, cf, &auth, &environment, data, &deployResponse)

	err = c.EventManager.EmitEvent(StopStartedEvent{
		CFContext:   cf,
		Data:        data,
		Environment: environment,
		Auth:        auth,
		Response:    response,
	})
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		return I.DeployResponse{
			StatusCode:     http.StatusInternalServerError,
			Error:          deployer.EventError{Type: "StopStartedEvent", Err: err},
			DeploymentInfo: deploymentInfo,
		}
	}

	deployEventData := structs.DeployEventData{Response: response, DeploymentInfo: deploymentInfo}

	manager := c.StopManagerFactory.StopManager(deployEventData)
	return *c.Deployer.Deploy(deploymentInfo, environment, manager, response)
}

func (c StopController) emitStopFinish(response io.ReadWriter, deploymentLogger logger.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event I.IEvent
	event = StopFinishedEvent{
		CFContext:     cfContext,
		Authorization: *auth,
		Environment:   *environment,
		Data:          data,
	}
	deploymentLogger.Debugf("emitting a %s event", event.Name())
	c.EventManager.EmitEvent(event)
}

func (c StopController) emitStopSuccessOrFailure(response io.ReadWriter, deploymentLogger logger.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event I.IEvent

	if deployResponse.Error != nil {
		c.printErrors(response, &deployResponse.Error)
		event = StopFailureEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
			Error:         deployResponse.Error,
		}

	} else {
		event = StopSuccessEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
		}
	}
	deploymentLogger.Debugf("emitting a %s event", event.Name())
	eventErr := c.EventManager.EmitEvent(event)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", event.Name(), eventErr)
		fmt.Fprintln(response, eventErr)
	}
}

func (c StopController) printErrors(response io.ReadWriter, err *error) {
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

func (c *StopController) resolveAuthorization(auth I.Authorization, envs structs.Environment, deploymentLogger logger.DeploymentLogger) (I.Authorization, error) {
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

func (c *StopController) resolveEnvironment(env string) (structs.Environment, error) {
	config := c.Config
	environment, ok := config.Environments[env]
	if !ok {
		return structs.Environment{}, deployer.EnvironmentNotFoundError{env}
	}
	return environment, nil
}
