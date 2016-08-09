// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

const (
	successfulDeploy          = "deploy successful"
	cannotDeployApplication   = "cannot deploy application"
	requestBodyEmpty          = "request body is empty"
	cannotReadFileFromRequest = "cannot read file from request"
	cannotProcessZipFile      = "cannot process zip file"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Config       config.Config
	Deployer     I.Deployer
	Log          *logging.Logger
	EventManager I.EventManager
	Fetcher      I.Fetcher
}

// Deploy checks the request content type and passes it to the Deployer.
func (c *Controller) Deploy(g *gin.Context) {
	c.Log.Debug("Request originated from: %+v", g.Request.RemoteAddr)

	var (
		environmentName = g.Param("environment")
		org             = g.Param("org")
		space           = g.Param("space")
		appName         = g.Param("appname")
		buffer          = &bytes.Buffer{}
		err             error
		statusCode      int
	)

	contentType := g.Request.Header.Get("Content-Type")

	if contentType == "application/json" {
		err, statusCode = c.Deployer.Deploy(g.Request, environmentName, org, space, appName, buffer)
		if err != nil {
			c.Log.Errorf("%s: %s", cannotDeployApplication, err)
			g.Writer.WriteHeader(statusCode)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
			g.Error(err)
			return
		} else {
			g.Writer.WriteHeader(statusCode)
			g.Writer.WriteString(successfulDeploy)
		}
	} else if contentType == "application/zip" {
		if g.Request.Body != nil {
			f, err := ioutil.ReadAll(g.Request.Body)
			if err != nil {
				c.Log.Errorf(cannotReadFileFromRequest)
				g.Writer.WriteHeader(500)
				g.Writer.WriteString(fmt.Sprintln(cannotReadFileFromRequest + " - " + err.Error()))
				g.Error(err)
				return
			}

			appPath, err := c.Fetcher.FetchFromZip(f)
			if err != nil {
				c.Log.Errorf(cannotProcessZipFile)
				g.Writer.WriteHeader(500)
				g.Writer.WriteString(fmt.Sprintln(cannotProcessZipFile + " - " + err.Error()))
				g.Error(err)
				return
			}
			defer os.RemoveAll(appPath)

			err, statusCode = c.Deployer.DeployZip(g.Request, environmentName, org, space, appName, buffer)
			if err != nil {
				c.Log.Errorf("%s: %s", cannotDeployApplication, err)
				g.Writer.WriteHeader(statusCode)
				g.Writer.WriteString(fmt.Sprintln(cannotDeployApplication + " - " + err.Error()))
				g.Error(err)
				return
			} else {
				g.Writer.WriteHeader(statusCode)
				g.Writer.WriteString(successfulDeploy)
			}
		} else {
			c.Log.Errorf(requestBodyEmpty)
			g.Writer.WriteHeader(400)
			g.Writer.WriteString(requestBodyEmpty)
			return
		}
	} else {
		c.Log.Errorf("content type '%s' not supported", contentType)
		g.Writer.WriteHeader(400)
		g.Writer.WriteString(fmt.Sprintf("content type '%s' not supported", contentType))
		return
	}

	io.Copy(g.Writer, buffer)
}
