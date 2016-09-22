// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"

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
	Fetcher      I.Fetcher
}

// Deploy checks the request content type and passes it to the Deployer.
func (c *Controller) Deploy(g *gin.Context) {
	c.Log.Info("Request originated from: %+v", g.Request.RemoteAddr)

	var (
		environmentName = g.Param("environment")
		org             = g.Param("org")
		space           = g.Param("space")
		appName         = g.Param("appName")
		buffer          = &bytes.Buffer{}
		err             error
		statusCode      int
		appPath         string
	)

	defer io.Copy(g.Writer, buffer)

	contentType := g.Request.Header.Get("Content-Type")
	if isZipRequest(contentType) {

		if g.Request.Body == nil {
			logError(requestBodyEmpty, http.StatusBadRequest, errors.New("request body required"), g, c.Log)
			return
		}

		f, err := ioutil.ReadAll(g.Request.Body)
		if err != nil {
			logError(cannotReadFileFromRequest, http.StatusInternalServerError, err, g, c.Log)
			return
		}

		appPath, err = c.Fetcher.FetchFromZip(f)
		defer os.RemoveAll(appPath)
		if err != nil {
			logError(cannotProcessZipFile, http.StatusInternalServerError, err, g, c.Log)
			return
		}
	} else if !isJsonRequest(contentType) {
		logError(contentTypeNotSupported, http.StatusBadRequest, errors.New("must be application/json or application/zip"), g, c.Log)
		return
	}

	err, statusCode = c.Deployer.Deploy(g.Request, environmentName, org, space, appName, appPath, contentType, buffer)
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

func isZipRequest(contentType string) bool {
	return contentType == "application/zip"
}

func isJsonRequest(contentType string) bool {
	return contentType == "application/json"
}
