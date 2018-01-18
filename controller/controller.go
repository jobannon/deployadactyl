// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"io/ioutil"

	"os"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/gin-gonic/gin"
	"encoding/base64"
	"github.com/compozed/deployadactyl/constants"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer        I.Deployer
	SilentDeployer  I.Deployer
	Log             I.Logger
}

type Deployment struct {
	Body          *[]byte
	Type          constants.DeploymentType
	Authorization Authorization
	CFContext     CFContext
}

type Authorization struct {
	Username string
	Password string
}

type CFContext struct {
	Environment  string
	Organization string
	Space        string
	Application  string
}

func (c *Controller) DoDeploy(deployment *Deployment, response *bytes.Buffer) (*bytes.Buffer, int, error) {

	bodyNotSilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))
	bodySilent := ioutil.NopCloser(bytes.NewBuffer(*deployment.Body))

	headers := http.Header{}
	headers["Authorization"] = []string{base64.StdEncoding.EncodeToString([]byte(deployment.Authorization.Username + ":" + deployment.Authorization.Password))}
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
	//go c.NotSilentDeploy(request1, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, reqChannel1, response)

	if cf.Environment == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go c.SilentDeployer.Deploy(request2, cf.Environment, cf.Organization, cf.Space, cf.Application, deployment.Type, response, reqChannel2)
		//go c.SilentDeploy(request2, cf.Organization, cf.Space, cf.Application, reqChannel2)
		<-reqChannel2
	}

	deployResponse := <-reqChannel1

	if deployResponse.Error != nil {
		return response, http.StatusInternalServerError, deployResponse.Error
	}

	close(reqChannel1)
	close(reqChannel2)

	return response, deployResponse.StatusCode, nil
}

// Deploy checks the request content type and passes it to the Deployer.
func (c *Controller) Deploy(g *gin.Context) {
	c.Log.Debugf("Request originated from: %+v", g.Request.RemoteAddr)

	cfContext := CFContext{
		Environment: g.Param("environment"),
		Organization: g.Param("org"),
		Space: g.Param("space"),
		Application: g.Param("appName"),
	}

	user, pwd, _ := g.Request.BasicAuth()
	authorization := Authorization{
		Username: user,
		Password: pwd,
	}

	deploymentType := constants.DeploymentType{
		JSON: isJSON(g.Request.Header.Get("Content-Type")),
		ZIP: isZip(g.Request.Header.Get("Content-Type")),
	}
	response := &bytes.Buffer{}

	deployment := Deployment{
		Authorization: authorization,
		CFContext: cfContext,
		Type: deploymentType,
	}
	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)
	deployment.Body = &bodyBuffer

	response, statusCode, error := c.DoDeploy(&deployment, response)
	defer io.Copy(g.Writer, response)
	if error != nil {
		g.Writer.WriteHeader(statusCode)
		fmt.Fprintf(response, "cannot deploy application: %s\n", error)
		return
	}

	g.Writer.WriteHeader(statusCode)
}

/*
func (c *Controller) NotSilentDeploy(req *http.Request, environment, org, space, appName string, contentType constants.DeploymentType, reqChannel chan D.DeployResponse, response *bytes.Buffer) {
	deployResponse := D.DeployResponse{}
	statusCode, err := c.Deployer.Deploy(
		req,
		environment,
		org,
		space,
		appName,
		contentType,
		response,
	)

	if err != nil {
		deployResponse.StatusCode = statusCode
		deployResponse.Error = err
		reqChannel <- deployResponse
	}

	deployResponse.StatusCode = statusCode
	deployResponse.Error = err
	reqChannel <- deployResponse
}

func (c *Controller) SilentDeploy(req *http.Request, org, space, appName string, reqChannel chan D.DeployResponse) {
	url := os.Getenv("SILENT_DEPLOY_URL")
	deployResponse := D.DeployResponse{}

	request, err := http.NewRequest("POST", fmt.Sprintf(url, org, space, appName), req.Body)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer request err: %s", err))
		deployResponse.Error = err
		reqChannel <- deployResponse
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Do(request)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer response err: %s", err))
		deployResponse.StatusCode = resp.StatusCode
		deployResponse.Error = err
		reqChannel <- deployResponse
	}

	deployResponse.StatusCode = resp.StatusCode
	deployResponse.Error = err
	reqChannel <- deployResponse
}
*/
func isZip(contentType string) bool {
	return contentType == "application/zip"
}

func isJSON(contentType string) bool {
	return contentType == "application/json"
}