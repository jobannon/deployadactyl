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
	"github.com/compozed/deployadactyl/randomizer"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Log             I.Logger
	PushControllerFactory func(log I.DeploymentLogger) I.PushController
	StartControllerFactory func(log I.DeploymentLogger) I.StartController
	StopControllerFactory  func(log I.DeploymentLogger) I.StopController
	Config          config.Config
	EventManager    I.EventManager
	ErrorFinder     I.ErrorFinder
}

type PutRequest struct {
	State string                 `json:"state"`
	Data  map[string]interface{} `json:"data"`
}

func (c *Controller) RunDeployment(deployment *I.Deployment, response *bytes.Buffer) I.DeployResponse {
	if deployment.CFContext.UUID == "" {
		deployment.CFContext.UUID = randomizer.StringRunes(10)
	}
	log := I.DeploymentLogger{Log: c.Log, UUID: deployment.CFContext.UUID}
	return c.PushControllerFactory(log).RunDeployment(deployment, response)
}

// RunDeploymentViaHttp checks the request content type and passes it to the Deployer.
func (c *Controller) RunDeploymentViaHttp(g *gin.Context) {
	uuid := randomizer.StringRunes(10)
	log := I.DeploymentLogger{Log: c.Log, UUID: uuid}
	log.Debugf("Request originated from: %+v", g.Request.RemoteAddr)

	cfContext := I.CFContext{
		Environment:  g.Param("environment"),
		Organization: g.Param("org"),
		Space:        g.Param("space"),
		Application:  g.Param("appName"),
		UUID: uuid,
	}

	user, pwd, _ := g.Request.BasicAuth()
	authorization := I.Authorization{
		Username: user,
		Password: pwd,
	}

	deploymentType := I.DeploymentType{
		JSON: g.Request.Header.Get("Content-Type") == "application/json",
		ZIP:  g.Request.Header.Get("Content-Type") == "application/zip",
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

	deployResponse := c.PushControllerFactory(log).RunDeployment(&deployment, response)

	defer io.Copy(g.Writer, response)

	if deployResponse.Error != nil {
		g.Writer.WriteHeader(deployResponse.StatusCode)
		fmt.Fprintf(response, "cannot deploy application: %s\n", deployResponse.Error)
		return
	}

	g.Writer.WriteHeader(deployResponse.StatusCode)
}

func (c *Controller) PutRequestHandler(g *gin.Context) {
	uuid := randomizer.StringRunes(10)
	log := I.DeploymentLogger{Log: c.Log, UUID: uuid}
	log.Debugf("PUT Request originated from: %+v", g.Request.RemoteAddr)

	cfContext := I.CFContext{
		Environment:  g.Param("environment"),
		Organization: g.Param("org"),
		Space:        g.Param("space"),
		Application:  g.Param("appName"),
		UUID: uuid,
	}

	response := &bytes.Buffer{}
	defer io.Copy(g.Writer, response)

	user, pwd, _ := g.Request.BasicAuth()
	authorization := I.Authorization{
		Username: user,
		Password: pwd,
	}

	deployment := I.Deployment{
		Authorization: authorization,
		CFContext:     cfContext,
	}

	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)
	g.Request.Body.Close()

	putRequest := &PutRequest{}
	err := json.Unmarshal(bodyBuffer, putRequest)
	if err != nil {
		response.Write([]byte("Invalid request body."))
		g.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	var deployResponse I.DeployResponse

	if putRequest.State == "stopped" {
		deployResponse = c.StopControllerFactory(log).StopDeployment(&deployment, putRequest.Data, response)
	} else if putRequest.State == "started" {
		deployResponse = c.StartControllerFactory(log).StartDeployment(&deployment, putRequest.Data, response)
	} else {
		response.Write([]byte("Unknown requested state: " + putRequest.State))
		deployResponse = I.DeployResponse{
			StatusCode: http.StatusBadRequest,
		}
	}

	g.Writer.WriteHeader(deployResponse.StatusCode)
}
