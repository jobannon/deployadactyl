// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"encoding/json"
	I "github.com/compozed/deployadactyl/interfaces"

	"github.com/compozed/deployadactyl/config"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer        I.Deployer
	SilentDeployer  I.Deployer
	Log             I.Logger
	PushController  I.PushController
	StopController  I.StopController
	StartController I.StartController
	Config          config.Config
	EventManager    I.EventManager
	ErrorFinder     I.ErrorFinder
}

type PutRequest struct {
	State string                 `json:"state"`
	Data  map[string]interface{} `json:"data"`
}

// RunDeploymentViaHttp checks the request content type and passes it to the Deployer.
func (c *Controller) RunDeploymentViaHttp(g *gin.Context) {
	c.Log.Debugf("Request originated from: %+v", g.Request.RemoteAddr)

	cfContext := I.CFContext{
		Environment:  g.Param("environment"),
		Organization: g.Param("org"),
		Space:        g.Param("space"),
		Application:  g.Param("appName"),
	}

	user, pwd, _ := g.Request.BasicAuth()
	authorization := I.Authorization{
		Username: user,
		Password: pwd,
	}

	deploymentType := I.DeploymentType{
		JSON: isJSON(g.Request.Header.Get("Content-Type")),
		ZIP:  isZip(g.Request.Header.Get("Content-Type")),
	}
	response := &bytes.Buffer{}

	deployment := I.Deployment{
		Authorization: authorization,
		CFContext:     cfContext,
		Type:          deploymentType,
	}
	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)
	g.Request.Body.Close()
	deployment.Body = &bodyBuffer

	deployResponse := c.PushController.RunDeployment(&deployment, response)

	defer io.Copy(g.Writer, response)

	if deployResponse.Error != nil {
		g.Writer.WriteHeader(deployResponse.StatusCode)
		fmt.Fprintf(response, "cannot deploy application: %s\n", deployResponse.Error)
		return
	}

	g.Writer.WriteHeader(deployResponse.StatusCode)
}

func (c *Controller) PutRequestHandler(g *gin.Context) {
	c.Log.Debugf("PUT Request originated from: %+v", g.Request.RemoteAddr)

	cfContext := I.CFContext{
		Environment:  g.Param("environment"),
		Organization: g.Param("org"),
		Space:        g.Param("space"),
		Application:  g.Param("appName"),
	}

	user, pwd, _ := g.Request.BasicAuth()
	authorization := I.Authorization{
		Username: user,
		Password: pwd,
	}
	response := &bytes.Buffer{}

	deployment := I.Deployment{
		Authorization: authorization,
		CFContext:     cfContext,
	}

	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)
	g.Request.Body.Close()

	putRequest := &PutRequest{}
	json.Unmarshal(bodyBuffer, putRequest)

	if putRequest.State == "stopped" {
		c.StopController.StopDeployment(&deployment, putRequest.Data, response)
	} else if putRequest.State == "started" {
		c.StartController.StartDeployment(&deployment, putRequest.Data, response)
	}

	defer io.Copy(g.Writer, response)

	g.Writer.WriteHeader(http.StatusOK)
}

func isZip(contentType string) bool {
	return contentType == "application/zip"
}

func isJSON(contentType string) bool {
	return contentType == "application/json"
}

//func (c *Controller) getDeploymentInfo(body *[]byte, deploymentInfo *structs.DeploymentInfo) (*structs.DeploymentInfo, error) {
//	reader := ioutil.NopCloser(bytes.NewBuffer(*body))
//	err := json.NewDecoder(reader).Decode(deploymentInfo)
//	if err != nil {
//		return deploymentInfo, err
//	}
//
//	getter := geterrors.WrapFunc(func(key string) string {
//		if key == "artifact_url" {
//			return deploymentInfo.ArtifactURL
//		}
//		return ""
//	})
//
//	getter.Get("artifact_url")
//
//	err = getter.Err("The following properties are missing")
//	if err != nil {
//		return &structs.DeploymentInfo{}, err
//	}
//	return deploymentInfo, nil
//}
//
//func (c *Controller) resolveAuthorization(auth I.Authorization, envs structs.Environment, deploymentLogger logger.DeploymentLogger) (I.Authorization, error) {
//	config := c.Config
//	deploymentLogger.Debug("checking for basic auth")
//	if auth.Username == "" && auth.Password == "" {
//		if envs.Authenticate {
//			return I.Authorization{}, deployer.BasicAuthError{}
//
//		}
//		auth.Username = config.Username
//		auth.Password = config.Password
//	}
//
//	return auth, nil
//}
//
//func (c *Controller) resolveEnvironment(env string) (structs.Environment, error) {
//	config := c.Config
//	environment, ok := config.Environments[env]
//	if !ok {
//		return structs.Environment{}, deployer.EnvironmentNotFoundError{env}
//	}
//	return environment, nil
//}

//func (c *Controller) emitDeployFinish(deployEventData *structs.DeployEventData, response io.ReadWriter, deployResponse *I.DeployResponse, deploymentLogger logger.DeploymentLogger) {
//	deploymentLogger.Debugf("emitting a %s event", constants.DeployFinishEvent)
//	finishErr := c.EventManager.Emit(I.Event{Type: constants.DeployFinishEvent, Data: deployEventData})
//	if finishErr != nil {
//		fmt.Fprintln(response, finishErr)
//		err := bluegreen.FinishDeployError{Err: fmt.Errorf("%s: %s", deployResponse.Error, deployer.EventError{constants.DeployFinishEvent, finishErr})}
//		deployResponse.Error = err
//		deployResponse.StatusCode = http.StatusInternalServerError
//	}
//}
//
//func (c Controller) emitDeploySuccessOrFailure(deployEventData *structs.DeployEventData, response io.ReadWriter, deployResponse *I.DeployResponse, deploymentLogger logger.DeploymentLogger) {
//	deployEvent := I.Event{Type: constants.DeploySuccessEvent, Data: deployEventData}
//	if deployResponse.Error != nil {
//		c.printErrors(response, &deployResponse.Error)
//
//		deployEvent.Type = constants.DeployFailureEvent
//		deployEvent.Error = deployResponse.Error
//	}
//
//	deploymentLogger.Debug(fmt.Sprintf("emitting a %s event", deployEvent.Type))
//	eventErr := c.EventManager.Emit(deployEvent)
//	if eventErr != nil {
//		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", deployEvent.Type, eventErr)
//		fmt.Fprintln(response, eventErr)
//	}
//}
//func (c Controller) emitStopFinish(response io.ReadWriter, deploymentLogger logger.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
//	var event I.IEvent
//	event = stop.StopFinishedEvent{
//		CFContext:     cfContext,
//		Authorization: *auth,
//		Environment:   *environment,
//		Data:          data,
//	}
//	deploymentLogger.Debugf("emitting a %s event", event.Name())
//	c.EventManager.EmitEvent(event)
//}
//func (c Controller) emitStopSuccessOrFailure(response io.ReadWriter, deploymentLogger logger.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
//	var event I.IEvent
//
//	if deployResponse.Error != nil {
//		c.printErrors(response, &deployResponse.Error)
//		event = stop.StopFailureEvent{
//			CFContext:     cfContext,
//			Authorization: *auth,
//			Environment:   *environment,
//			Data:          data,
//			Error:         deployResponse.Error,
//		}
//
//	} else {
//		event = stop.StopSuccessEvent{
//			CFContext:     cfContext,
//			Authorization: *auth,
//			Environment:   *environment,
//			Data:          data,
//		}
//	}
//	eventErr := c.EventManager.EmitEvent(event)
//	if eventErr != nil {
//		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", event.Name(), eventErr)
//		fmt.Fprintln(response, eventErr)
//	}
//}

//func (c Controller) printErrors(response io.ReadWriter, err *error) {
//	tempBuffer := bytes.Buffer{}
//	tempBuffer.ReadFrom(response)
//	fmt.Fprint(response, tempBuffer.String())
//
//	errors := c.ErrorFinder.FindErrors(tempBuffer.String())
//	if len(errors) > 0 {
//		*err = errors[0]
//		for _, error := range errors {
//			fmt.Fprintln(response)
//			fmt.Fprintln(response, "*******************")
//			fmt.Fprintln(response)
//			fmt.Fprintln(response, "The following error was found in the above logs: "+error.Error())
//			fmt.Fprintln(response)
//			fmt.Fprintln(response, "Error: "+error.Details()[0])
//			fmt.Fprintln(response)
//			fmt.Fprintln(response, "Potential solution: "+error.Solution())
//			fmt.Fprintln(response)
//			fmt.Fprintln(response, "*******************")
//		}
//	}
//}
