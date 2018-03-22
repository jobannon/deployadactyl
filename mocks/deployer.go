package mocks

import (
	"fmt"
	"io"

	I "github.com/compozed/deployadactyl/interfaces"
)

// Deployer handmade mock for tests.
type Deployer struct {
	DeployCall struct {
		Called   int
		Received struct {
			Authorization I.Authorization
			Body          io.Reader
			ActionCreator I.ActionCreator
			Environment   string
			Org           string
			Space         string
			AppName       string
			UUID          string
			ContentType   I.DeploymentType
			Response      io.ReadWriter
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error      error
			StatusCode int
		}
	}
}

// Deploy mock method.
func (d *Deployer) Deploy(authorization I.Authorization, body io.Reader, actionCreator I.ActionCreator, environment, org, space, appName, uuid string, contentType I.DeploymentType, out io.ReadWriter) *I.DeployResponse {
	d.DeployCall.Called++

	d.DeployCall.Received.Authorization = authorization
	d.DeployCall.Received.Body = body
	d.DeployCall.Received.ActionCreator = actionCreator

	d.DeployCall.Received.Environment = environment
	d.DeployCall.Received.Org = org
	d.DeployCall.Received.Space = space
	d.DeployCall.Received.AppName = appName
	d.DeployCall.Received.UUID = uuid
	d.DeployCall.Received.ContentType = contentType
	d.DeployCall.Received.Response = out

	fmt.Fprint(out, d.DeployCall.Write.Output)

	response := &I.DeployResponse{
		StatusCode: d.DeployCall.Returns.StatusCode,
		Error:      d.DeployCall.Returns.Error,
	}

	//reqChan <- response
	return response
}
