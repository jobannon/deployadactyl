// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"errors"
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
	contentTypeNotSupported   = "content type not supported"
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
		appName         = g.Param("appName")
		buffer          = &bytes.Buffer{}
		err             error
		statusCode      int
	)

	defer io.Copy(g.Writer, buffer)

	contentType := g.Request.Header.Get("Content-Type")
	if contentType == "application/json" {
		err, statusCode = c.Deployer.Deploy(g.Request, environmentName, org, space, appName, "", contentType, buffer)
		if err != nil {
			logError(cannotDeployApplication, statusCode, err, g, c.Log)
			return
		}
		g.Writer.WriteHeader(statusCode)
		g.Writer.WriteString(successfulDeploy)
		return
	} else if contentType == "application/zip" {
		if g.Request.Body != nil {
			f, err := ioutil.ReadAll(g.Request.Body)
			if err != nil {
				logError(cannotReadFileFromRequest, 500, err, g, c.Log)
				return
			}

			appPath, err := c.Fetcher.FetchFromZip(f)
			if err != nil {
				logError(cannotProcessZipFile, 500, err, g, c.Log)
				return
			}
			defer os.RemoveAll(appPath)

			err, statusCode = c.Deployer.Deploy(g.Request, environmentName, org, space, appName, appPath, contentType, buffer)

			if err != nil {
				logError(cannotDeployApplication, statusCode, err, g, c.Log)
				return
			}
			g.Writer.WriteHeader(statusCode)
			g.Writer.WriteString(successfulDeploy)
			return
		}
		logError(requestBodyEmpty, 400, errors.New("request body required"), g, c.Log)
		return
	}
	logError(contentTypeNotSupported, 400, errors.New("must be application/json or application/zip"), g, c.Log)
}

func logError(message string, statusCode int, err error, g *gin.Context, l *logging.Logger) {
	l.Errorf("%s: %s", message, err)
	g.Writer.WriteHeader(statusCode)
	g.Writer.WriteString(message + " - " + err.Error())
	g.Error(err)
}
