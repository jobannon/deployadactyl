// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer I.Deployer
	Log      *logging.Logger
}

// Deploy checks the request content type and passes it to the Deployer.
func (c *Controller) Deploy(g *gin.Context) {
	c.Log.Info("Request originated from: %+v", g.Request.RemoteAddr)

	buffer := &bytes.Buffer{}

	defer io.Copy(g.Writer, buffer)

	statusCode, err := c.Deployer.Deploy(
		g.Request,
		g.Param("environment"),
		g.Param("org"),
		g.Param("space"),
		g.Param("appName"),
		g.Request.Header.Get("Content-Type"),
		buffer,
	)
	if err != nil {
		c.Log.Errorf("%s: %s", "cannot deploy application", err)
		g.Writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(buffer, fmt.Sprintf("%s: %s", "cannot deploy application", err.Error()))
		g.Error(err)
		return
	}

	g.Writer.WriteHeader(statusCode)
}
