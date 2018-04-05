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
