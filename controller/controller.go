// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"io"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

const (
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
}

// Deploy checks the request content type and passes it to the Deployer.
func (c *Controller) Deploy(g *gin.Context) {

	c.Log.Info("Request originated from: %+v", g.Request.RemoteAddr)

	buffer := &bytes.Buffer{}

	defer io.Copy(g.Writer, buffer)

	err, statusCode := c.Deployer.Deploy(
		g.Request,
		g.Param("environment"),
		g.Param("org"),
		g.Param("space"),
		g.Param("appName"),
		g.Request.Header.Get("Content-Type"),
		buffer,
	)
	if err != nil {
		logError(cannotDeployApplication, statusCode, err, g, c.Log)
		return
	}

	g.Writer.WriteHeader(statusCode)
}

func logError(message string, statusCode int, err error, g *gin.Context, l *logging.Logger) {
	l.Errorf("%s: %s", message, err)
	g.Writer.WriteHeader(statusCode)
	g.Writer.WriteString(message + " - " + err.Error() + "\n")
	g.Error(err)
}
