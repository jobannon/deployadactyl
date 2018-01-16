// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"io/ioutil"

	"os"

	"log"

	"crypto/tls"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/gin-gonic/gin"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer I.Deployer
	Log      I.Logger
}

type DeployResponse struct {
	StatusCode int
	Error      error
}

// Deploy checks the request content type and passes it to the Deployer.
func (c *Controller) Deploy(g *gin.Context) {
	c.Log.Debugf("Request originated from: %+v", g.Request.RemoteAddr)

	bodyBuffer, _ := ioutil.ReadAll(g.Request.Body)
	bodyNotSilent := ioutil.NopCloser(bytes.NewBuffer(bodyBuffer))
	bodySilent := ioutil.NopCloser(bytes.NewBuffer(bodyBuffer))
	reqChannel1 := make(chan DeployResponse)
	reqChannel2 := make(chan DeployResponse)

	request1 := &http.Request{
		Method:        g.Request.Method,
		URL:           g.Request.URL,
		Proto:         g.Request.Proto,
		ProtoMajor:    g.Request.ProtoMajor,
		ProtoMinor:    g.Request.ProtoMinor,
		Header:        g.Request.Header,
		Body:          bodyNotSilent,
		Host:          g.Request.Host,
		ContentLength: g.Request.ContentLength,
		Close:         true,
	}

	request2 := &http.Request{
		Method:        g.Request.Method,
		URL:           g.Request.URL,
		Proto:         g.Request.Proto,
		ProtoMajor:    g.Request.ProtoMajor,
		ProtoMinor:    g.Request.ProtoMinor,
		Header:        g.Request.Header,
		Body:          bodySilent,
		Host:          g.Request.Host,
		ContentLength: g.Request.ContentLength,
		Close:         true,
	}

	response := &bytes.Buffer{}

	defer io.Copy(g.Writer, response)

	go c.NotSilentDeploy(request1, g.Param("environment"), g.Param("org"), g.Param("space"), g.Param("appName"), g.Request.Header.Get("Content-Type"), reqChannel1, response)

	if g.Param("environment") == os.Getenv("SILENT_DEPLOY_ENVIRONMENT") {
		go c.SilentDeploy(request2, g.Param("org"), g.Param("space"), g.Param("appName"), reqChannel2)
		<-reqChannel2
	}

	deployResponse := <-reqChannel1

	if deployResponse.Error != nil {
		g.Writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(response, "cannot deploy application: %s\n", deployResponse.Error)
		return
	}

	close(reqChannel1)
	close(reqChannel2)
	g.Writer.WriteHeader(deployResponse.StatusCode)
}

func (c *Controller) NotSilentDeploy(req *http.Request, environment, org, space, appName, contentType string, reqChannel chan DeployResponse, response *bytes.Buffer) {
	deployResponse := DeployResponse{}
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

func (c *Controller) SilentDeploy(req *http.Request, org, space, appName string, reqChannel chan DeployResponse) {
	url := os.Getenv("SILENT_DEPLOY_URL")
	deployResponse := DeployResponse{}

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
