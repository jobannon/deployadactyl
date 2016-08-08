// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

const (
	cannotDeployApplication = "cannot deploy application"
)

// Controller is used to control deployments using the config and event manager.
type Controller struct {
	Config       config.Config
	Deployer     I.Deployer
	Log          *logging.Logger
	EventManager I.EventManager
	Fetcher      I.Fetcher
}

// Deploy parses parameters from the request, builds a DeploymentInfo and passes it to Deployer.
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
			g.Error(err)
		}
	} else if contentType == "application/zip" {
		f, _, err := g.Request.FormFile("application")
		if err != nil {
			c.Log.Errorf("Could not create file from request.")
			g.Writer.WriteHeader(500)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
			g.Error(err)
		}

		appPath, err := c.Fetcher.FetchFromZip(f)
		if err != nil {
			c.Log.Errorf("Could not process zip file.")
			g.Writer.WriteHeader(500)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
			g.Error(err)
		}
		defer os.RemoveAll(appPath)

		err, statusCode = c.Deployer.DeployZip(g.Request, environmentName, org, space, appName, buffer)
		if err != nil {
			c.Log.Errorf("%s: %s", cannotDeployApplication, err)
			g.Writer.WriteHeader(statusCode)
			g.Error(err)
		}
	} else {
		c.Log.Errorf("Content type '%s' not supported", contentType)
		g.Writer.WriteHeader(400)
		g.Writer.WriteString(fmt.Sprintln(err.Error()))
		g.Error(err)
	}

	io.Copy(g.Writer, buffer)
}
