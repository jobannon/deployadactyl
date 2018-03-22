// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"os"

	I "github.com/compozed/deployadactyl/interfaces"

	"github.com/gin-gonic/gin"
)

type pusherCreatorFactory interface {
	PusherCreator(body io.Reader) I.ActionCreator
}

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer             I.Deployer
	SilentDeployer       I.Deployer
	Log                  I.Logger
	PusherCreatorFactory pusherCreatorFactory
}

func (c *Controller) RunDeployment(deployment *I.Deployment, response *bytes.Buffer) I.DeployResponse {

	bodyNotSilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	bodySilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))

	pusherCreator := c.PusherCreatorFactory.PusherCreator(bodyNotSilent)

	reqChannel1 := make(chan *I.DeployResponse)
	reqChannel2 := make(chan *I.DeployResponse)
	defer close(reqChannel1)
	defer close(reqChannel2)

	cf := deployment.CFContext
	go func() {
		reqChannel1 <- c.Deployer.Deploy(deployment.Authorization, bodyNotSilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, cf.UUID, deployment.Type, response)
	}()

	silentResponse := &bytes.Buffer{}
	if cf.Environment == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go func() {
			reqChannel2 <- c.SilentDeployer.Deploy(deployment.Authorization, bodySilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, cf.UUID, deployment.Type, silentResponse)
		}()
		<-reqChannel2
	}

	deployResponse := <-reqChannel1

	return *deployResponse
}

func (c *Controller) StopDeployment(deployment *I.Deployment, response *bytes.Buffer) I.DeployResponse {

	bodyNotSilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	bodySilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))

	pusherCreator := c.PusherCreatorFactory.PusherCreator(bodyNotSilent)

	reqChannel1 := make(chan I.DeployResponse)
	reqChannel2 := make(chan I.DeployResponse)
	defer close(reqChannel1)
	defer close(reqChannel2)

	cf := deployment.CFContext
	go c.Deployer.Deploy(deployment.Authorization, bodyNotSilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, cf.UUID, deployment.Type, response)

	silentResponse := &bytes.Buffer{}
	if cf.Environment == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go c.SilentDeployer.Deploy(deployment.Authorization, bodySilent, pusherCreator, cf.Environment, cf.Organization, cf.Space, cf.Application, cf.UUID, deployment.Type, silentResponse)
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
