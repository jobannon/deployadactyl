// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/geterrors"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	basicAuthHeaderNotFound   = "basic auth header not found"
	invalidPostRequest        = "invalid POST request"
	cannotOpenManifestFile    = "cannot open manifest file"
	cannotPrintToManifestFile = "cannot print to open manifest file"
	cannotDeployApplication   = "cannot deploy application"
	deployStartError          = "an error occurred in the deploy.start event"
	deployFinishError         = "an error occurred in the deploy.finish event"
	deploymentOutput          = `Deployment Parameters:
	Artifact URL: %s,
	Username:     %s,
	Enviroment:   %s,
	Org:          %s,
	Space:        %s,
	AppName:      %s

`
)

// Controller is used to control deployments using the config and event manager.
type Controller struct {
	Deployer     I.Deployer
	Log          *logging.Logger
	Config       config.Config
	EventManager I.EventManager
	Randomizer   I.Randomizer
	Fetcher      I.Fetcher
}

// Deploy parses parameters from the request, builds a DeploymentInfo and passes it to Deployer.
func (c *Controller) Deploy(g *gin.Context) {
	c.Log.Debug("Request originated from: %+v", g.Request.RemoteAddr)

	var (
		environment            = g.Param("environment")
		authenticationRequired = c.Config.Environments[environment].Authenticate
		buffer                 = &bytes.Buffer{}
		deployEventData        = S.DeployEventData{}
		deploymentInfo         = S.DeploymentInfo{}
		err                    error
	)

	username, password, ok := g.Request.BasicAuth()

	if !ok {
		if authenticationRequired {
			err = errors.New(basicAuthHeaderNotFound)
			c.Log.Error(err.Error())
			g.Writer.WriteHeader(401)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
			return
		}
		username = c.Config.Username
		password = c.Config.Password
	}

	contentType := g.Request.Header.Get("Content-Type")
	if contentType == "application/json" {
		deploymentInfo, err = getDeploymentInfo(g.Request.Body)
		if err != nil {
			c.Log.Error(err.Error())
			g.Writer.WriteHeader(500)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
			return
		}

		deploymentInfo.Username = username
		deploymentInfo.Password = password
		deploymentInfo.Environment = environment
		deploymentInfo.Org = g.Param("org")
		deploymentInfo.Space = g.Param("space")
		deploymentInfo.AppName = g.Param("appName")
		deploymentInfo.UUID = c.Randomizer.StringRunes(128)
		deploymentInfo.SkipSSL = c.Config.Environments[environment].SkipSSL

		deploymentMessage := fmt.Sprintf(deploymentOutput, deploymentInfo.ArtifactURL, deploymentInfo.Username, deploymentInfo.Environment, deploymentInfo.Org, deploymentInfo.Space, deploymentInfo.AppName)
		c.Log.Debug(deploymentMessage)
		fmt.Fprintln(buffer, deploymentMessage)

		deployEventData = S.DeployEventData{
			Writer:         buffer,
			DeploymentInfo: &deploymentInfo,
			RequestBody:    g.Request.Body,
		}

	} else if contentType == "application/zip" {
		// Get zip file from request
		f, _, err := g.Request.FormFile("application")
		if err != nil {
			c.Log.Errorf("Could not create file from request.")
			g.Writer.WriteHeader(500)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
			return
		}
		// Unzip file
		appPath, err := c.Fetcher.FetchLocal(f)
		if err != nil {
			c.Log.Errorf("Could not process zip file.")
			g.Writer.WriteHeader(500)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
			return
		}
		defer os.RemoveAll(appPath)

		// Do our deploy event
	} else {
		//Err: content type not supported
		c.Log.Errorf("Content type %s not supported", contentType)
		g.Writer.WriteHeader(500)
		g.Writer.WriteString(fmt.Sprintln(err.Error()))
		return
	}

	m, err := base64.StdEncoding.DecodeString(deploymentInfo.Manifest)
	if err != nil {
		c.Log.Errorf("%s: %s", invalidPostRequest, err)
		g.Writer.WriteHeader(500)
		g.Writer.WriteString(fmt.Sprintln(err.Error()))
		return
	}

	deploymentInfo.Manifest = string(m)

	defer func() {
		deployFinishEvent := S.Event{
			Type: "deploy.finish",
			Data: deployEventData,
		}

		err = c.EventManager.Emit(deployFinishEvent)
		if err != nil {
			c.Log.Errorf("%s: %s", deployFinishError, err)
			g.Writer.WriteHeader(500)
			g.Writer.WriteString(fmt.Sprintln(err.Error()))
		}

		io.Copy(g.Writer, buffer)
	}()

	deployStartEvent := S.Event{
		Type: "deploy.start",
		Data: deployEventData,
	}

	err = c.EventManager.Emit(deployStartEvent)
	if err != nil {
		c.Log.Errorf("%s: %s", deployStartError, err)
		g.Writer.WriteHeader(500)
		g.Writer.WriteString(fmt.Sprintln(err.Error()))
		return
	}

	err = c.Deployer.Deploy(deploymentInfo, buffer)
	if err != nil {
		c.Log.Errorf("%s: %s", cannotDeployApplication, err)
		g.Writer.WriteHeader(500)
		g.Error(err)
	}
}

func getDeploymentInfo(reader io.Reader) (S.DeploymentInfo, error) {
	deploymentInfo := S.DeploymentInfo{}
	err := json.NewDecoder(reader).Decode(&deploymentInfo)
	if err != nil {
		return deploymentInfo, err
	}

	getter := geterrors.WrapFunc(func(key string) string {
		if key == "artifact_url" {
			return deploymentInfo.ArtifactURL
		}
		return ""
	})
	getter.Get("artifact_url")

	err = getter.Err("The following properties are missing")
	if err != nil {
		return S.DeploymentInfo{}, err
	}
	return deploymentInfo, nil
}
