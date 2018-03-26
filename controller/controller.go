// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"os"

	"encoding/json"
	I "github.com/compozed/deployadactyl/interfaces"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/geterrors"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	"net/http"
)

type pusherCreatorFactory interface {
	PusherCreator(deployEventData structs.DeployEventData) I.ActionCreator
}

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer             I.Deployer
	SilentDeployer       I.Deployer
	Log                  I.Logger
	PusherCreatorFactory pusherCreatorFactory
	Config               config.Config
	EventManager         I.EventManager
	ErrorFinder          I.ErrorFinder
}

// PUSH specific
func (c *Controller) RunDeployment(deployment *I.Deployment, response *bytes.Buffer) (deployResponse I.DeployResponse) {
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
	}
	deploymentLogger := logger.DeploymentLogger{c.Log, deploymentInfo.UUID}
	deploymentLogger.Debugf("Starting deploy of %s with UUID %s", cf.Application, deploymentInfo.UUID)
	deploymentLogger.Debug("building deploymentInfo")

	bodyNotSilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	bodySilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	if deployment.Type.JSON {
		deploymentLogger.Debug("deploying from json request")

		deploymentInfo.ContentType = "JSON"
	} else if deployment.Type.ZIP {
		deploymentLogger.Debug("deploying from zip request")
		deploymentInfo.Body = bodyNotSilent
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

	auth, err := c.resolveAuthorization(deployment.Authorization, environment, deploymentLogger)
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
			deploymentLogger.Error(err)
			return I.DeployResponse{
				StatusCode:     http.StatusInternalServerError,
				Error:          err,
				DeploymentInfo: deploymentInfo,
			}
		}
	}

	deployEventData := structs.DeployEventData{Response: response, DeploymentInfo: deploymentInfo, RequestBody: bodyNotSilent}
	defer c.emitDeployFinish(&deployEventData, response, &deployResponse, deploymentLogger)
	defer c.emitDeploySuccessOrFailure(&deployEventData, response, &deployResponse, deploymentLogger)

	deploymentLogger.Debugf("emitting a %s event", constants.DeployStartEvent)
	err = c.EventManager.Emit(I.Event{Type: constants.DeployStartEvent, Data: &deployEventData})
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		return I.DeployResponse{
			StatusCode:     http.StatusInternalServerError,
			Error:          deployer.EventError{Type: constants.DeployStartEvent, Err: err},
			DeploymentInfo: deploymentInfo,
		}

	}
	pusherCreator := c.PusherCreatorFactory.PusherCreator(deployEventData)

	reqChannel1 := make(chan *I.DeployResponse)
	reqChannel2 := make(chan *I.DeployResponse)
	defer close(reqChannel1)
	defer close(reqChannel2)

	go func() {
		reqChannel1 <- c.Deployer.Deploy(deploymentInfo, environment, deployment.Authorization, bodyNotSilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, response)
	}()

	silentResponse := &bytes.Buffer{}
	if cf.Environment == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go func() {
			reqChannel2 <- c.SilentDeployer.Deploy(deploymentInfo, environment, deployment.Authorization, bodySilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, silentResponse)
		}()
		<-reqChannel2
	}

	deployResponse = *<-reqChannel1

	return deployResponse
}

func (c *Controller) StopDeployment(deployment *I.Deployment, response *bytes.Buffer) I.DeployResponse {
	cf := deployment.CFContext
	environment, err := c.resolveEnvironment(cf.Environment)
	if err != nil {
		fmt.Fprintln(response, err.Error())
		return I.DeployResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      err,
		}
	}

	bodyNotSilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	bodySilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))

	deploymentInfo := structs.DeploymentInfo{}
	deployEventData := &structs.DeployEventData{Response: response, DeploymentInfo: &deploymentInfo, RequestBody: bodyNotSilent}

	pusherCreator := c.PusherCreatorFactory.PusherCreator(*deployEventData)

	reqChannel1 := make(chan I.DeployResponse)
	reqChannel2 := make(chan I.DeployResponse)
	defer close(reqChannel1)
	defer close(reqChannel2)

	go c.Deployer.Deploy(&deploymentInfo, environment, deployment.Authorization, bodyNotSilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, response)

	silentResponse := &bytes.Buffer{}
	if cf.Environment == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go c.SilentDeployer.Deploy(&deploymentInfo, environment, deployment.Authorization, bodySilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, silentResponse)
		<-reqChannel2
	}

	deployResponse := <-reqChannel1

	return deployResponse
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

	deployResponse := c.RunDeployment(&deployment, response)

	defer io.Copy(g.Writer, response)

	if deployResponse.Error != nil {
		g.Writer.WriteHeader(deployResponse.StatusCode)
		fmt.Fprintf(response, "cannot deploy application: %s\n", deployResponse.Error)
		return
	}

	g.Writer.WriteHeader(deployResponse.StatusCode)
}

func isZip(contentType string) bool {
	return contentType == "application/zip"
}

func isJSON(contentType string) bool {
	return contentType == "application/json"
}

func (c *Controller) getDeploymentInfo(body *[]byte, deploymentInfo *structs.DeploymentInfo) (*structs.DeploymentInfo, error) {
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

func (c *Controller) emitDeployFinish(deployEventData *structs.DeployEventData, response io.ReadWriter, deployResponse *I.DeployResponse, deploymentLogger logger.DeploymentLogger) {
	deploymentLogger.Debugf("emitting a %s event", constants.DeployFinishEvent)
	finishErr := c.EventManager.Emit(I.Event{Type: constants.DeployFinishEvent, Data: deployEventData})
	if finishErr != nil {
		fmt.Fprintln(response, finishErr)
		err := bluegreen.FinishDeployError{Err: fmt.Errorf("%s: %s", deployResponse.Error, deployer.EventError{constants.DeployFinishEvent, finishErr})}
		deployResponse.Error = err
		deployResponse.StatusCode = http.StatusInternalServerError
	}
}

func (c Controller) emitDeploySuccessOrFailure(deployEventData *structs.DeployEventData, response io.ReadWriter, deployResponse *I.DeployResponse, deploymentLogger logger.DeploymentLogger) {
	deployEvent := I.Event{Type: constants.DeploySuccessEvent, Data: deployEventData}
	if deployResponse.Error != nil {
		c.printErrors(response, &deployResponse.Error)

		deployEvent.Type = constants.DeployFailureEvent
		deployEvent.Error = deployResponse.Error
	}

	deploymentLogger.Debug(fmt.Sprintf("emitting a %s event", deployEvent.Type))
	eventErr := c.EventManager.Emit(deployEvent)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", deployEvent.Type, eventErr)
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
