// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"io/ioutil"

	"os"

	"encoding/base64"

	I "github.com/compozed/deployadactyl/interfaces"

	"github.com/gin-gonic/gin"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer       I.Deployer
	SilentDeployer I.Deployer
	Log            I.Logger
}

func (c *Controller) RunDeployment(deployment *I.Deployment, response *bytes.Buffer) (int, error) {

	bodyNotSilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	bodySilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))

	headers := http.Header{}
	if deployment.Authorization.Username != "" && deployment.Authorization.Password != "" {
		headers["Authorization"] = []string{"Basic " + base64.StdEncoding.EncodeToString([]byte(deployment.Authorization.Username+":"+deployment.Authorization.Password))}

	} else {
		headers["Authorization"] = []string{}
	}

	request1 := &http.Request{
		Header: headers,
		Body:   bodyNotSilent,
	}

	request2 := &http.Request{
		Header: headers,
		Body:   bodySilent,
	}

	reqChannel1 := make(chan I.DeployResponse)
	reqChannel2 := make(chan I.DeployResponse)

	cf := deployment.CFContext
	go c.Deployer.Deploy(request1, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, response, reqChannel1)

	silentResponse := &bytes.Buffer{}
	if cf.Environment == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go c.SilentDeployer.Deploy(request2, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, silentResponse, reqChannel2)
		<-reqChannel2
	}

	deployResponse := <-reqChannel1

	if deployResponse.Error != nil {
		return http.StatusInternalServerError, deployResponse.Error
	}

	close(reqChannel1)
	close(reqChannel2)

	return deployResponse.StatusCode, nil
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

	statusCode, error := c.RunDeployment(&deployment, response)

	defer io.Copy(g.Writer, response)

	if error != nil {
		g.Writer.WriteHeader(statusCode)
		fmt.Fprintf(response, "cannot deploy application: %s\n", error)
		return
	}

	g.Writer.WriteHeader(statusCode)
}

func isZip(contentType string) bool {
	return contentType == "application/zip"
}

func isJSON(contentType string) bool {
	return contentType == "application/json"
}
