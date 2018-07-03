// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"encoding/json"

	I "github.com/compozed/deployadactyl/interfaces"

	"net/http"
	"strings"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
)

type RequestProcessorFactory func(uuid string, request interface{}, buffer *bytes.Buffer) I.RequestProcessor

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Log                     I.Logger
	RequestProcessorFactory RequestProcessorFactory
	Config                  config.Config
	ErrorFinder             I.ErrorFinder
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

	deploymentType := g.Request.Header.Get("Content-Type")

	response := &bytes.Buffer{}
	defer io.Copy(g.Writer, response)

	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)

	g.Request.Body.Close()

	deployment := I.Deployment{
		Authorization: authorization,
		CFContext:     cfContext,
		Type:          deploymentType,
		Body:          &bodyBuffer,
	}

	postRequest := I.PostRequest{}
	if deploymentType == "application/json" {
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

	deployResponse := c.RequestProcessorFactory(uuid, postDeploymentRequest, response).Process()

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

	putRequest := I.PutRequest{}
	err := json.Unmarshal(bodyBuffer, &putRequest)
	if err != nil {
		response.Write([]byte("Invalid request body."))
		g.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	deployment := I.Deployment{
		Body:          &bodyBuffer,
		Authorization: authorization,
		CFContext:     cfContext,
		Type:          g.Request.Header.Get("Content-Type"),
	}

	putDeploymentRequest := I.PutDeploymentRequest{
		Deployment: deployment,
		Request:    putRequest,
	}

	deployResponse := c.RequestProcessorFactory(uuid, putDeploymentRequest, response).Process()
	if deployResponse.Error != nil {
		fmt.Fprintf(response, "cannot deploy application: %s\n", deployResponse.Error)
	}

	g.Writer.WriteHeader(deployResponse.StatusCode)
}

func (c *Controller) DeleteRequestHandler(g *gin.Context) {
	uuid := randomizer.StringRunes(10)
	log := I.DeploymentLogger{Log: c.Log, UUID: uuid}
	log.Debugf("DELETE Request originated from: %+v", g.Request.RemoteAddr)

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

	deleteRequest := I.DeleteRequest{}
	err := json.Unmarshal(bodyBuffer, &deleteRequest)
	if err != nil {
		response.Write([]byte("Invalid request body."))
		g.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	deployment := I.Deployment{
		Body:          &bodyBuffer,
		Authorization: authorization,
		CFContext:     cfContext,
		Type:          g.Request.Header.Get("Content-Type"),
	}

	deleteDeploymentRequest := I.DeleteDeploymentRequest{
		Deployment: deployment,
		Request:    deleteRequest,
	}

	deployResponse := c.RequestProcessorFactory(uuid, deleteDeploymentRequest, response).Process()
	if deployResponse.Error != nil {
		fmt.Fprintf(response, "cannot delete application: %s\n", deployResponse.Error)
	}

	g.Writer.WriteHeader(deployResponse.StatusCode)
}
