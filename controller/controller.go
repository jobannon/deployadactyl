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
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type PushControllerFactory func(log I.DeploymentLogger) I.PushController
type StartControllerFactory func(log I.DeploymentLogger) I.StartController
type StopControllerFactory func(log I.DeploymentLogger) I.StopController

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Log                    I.Logger
	PushControllerFactory  PushControllerFactory
	StartControllerFactory StartControllerFactory
	StopControllerFactory  StopControllerFactory
	Config                 config.Config
	EventManager           I.EventManager
	ErrorFinder            I.ErrorFinder
}

func (c *Controller) PostRequestHandler(g *gin.Context) {
	uuid := randomizer.StringRunes(10)
	log := I.DeploymentLogger{Log: c.Log, UUID: uuid}
	log.Debugf("Request originated from: %+v", g.Request.RemoteAddr)

	cfContext := I.CFContext{
		Environment:  strings.ToLower(g.Param("environment")),
		Organization: strings.ToLower(g.Param("org")),
		Space:        strings.ToLower(g.Param("space")),
		Application:  strings.ToLower(g.Param("appName")),
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
	defer io.Copy(g.Writer, response)

	deployment := I.Deployment{
		Authorization: authorization,
		CFContext:     cfContext,
		Type:          deploymentType,
	}
	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)

	g.Request.Body.Close()
	deployment.Body = &bodyBuffer

	postRequest := I.PostRequest{}
	if deploymentType.JSON {
		err := json.Unmarshal(bodyBuffer, &postRequest)
		if err != nil {
			response.Write([]byte("Invalid request body."))
			g.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	postDeploymentRequest := I.PostDeploymentRequest{
		Deployment: deployment,
		Request:    postRequest,
	}

	deployResponse := c.PushControllerFactory(log).RunDeployment(postDeploymentRequest, response)

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
		Environment:  strings.ToLower(g.Param("environment")),
		Organization: strings.ToLower(g.Param("org")),
		Space:        strings.ToLower(g.Param("space")),
		Application:  strings.ToLower(g.Param("appName")),
	}

	response := &bytes.Buffer{}
	defer io.Copy(g.Writer, response)

	user, pwd, _ := g.Request.BasicAuth()
	authorization := I.Authorization{
		Username: user,
		Password: pwd,
	}

	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)
	g.Request.Body.Close()

	putRequest := &I.PutRequest{}
	err := json.Unmarshal(bodyBuffer, putRequest)
	if err != nil {
		response.Write([]byte("Invalid request body."))
		g.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	deployment := I.Deployment{
		Authorization: authorization,
		CFContext:     cfContext,
	}

	putDeploymentRequest := I.PutDeploymentRequest{
		Deployment: deployment,
		Request:    putRequest,
	}

	var deployResponse I.DeployResponse

	if putRequest.State == "stopped" {
		deployResponse = c.StopControllerFactory(log).StopDeployment(putDeploymentRequest, response)
	} else if putRequest.State == "started" {
		deployResponse = c.StartControllerFactory(log).StartDeployment(putDeploymentRequest, response)
	} else {
		response.Write([]byte("Unknown requested state: " + putRequest.State))
		deployResponse = I.DeployResponse{
			StatusCode: http.StatusBadRequest,
		}
	}

	g.Writer.WriteHeader(deployResponse.StatusCode)
}
