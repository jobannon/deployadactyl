package mocks

import (
	"fmt"
	"io"
	"net/http"

	I "github.com/compozed/deployadactyl/interfaces"
)

// Deployer handmade mock for tests.
type Deployer struct {
	DeployCall struct {
		Called int
		Received struct {
			Request     *http.Request
			Environment string
			Org         string
			Space       string
			AppName     string
			ContentType I.DeploymentType
			Response    io.ReadWriter
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
func (d *Deployer) Deploy(req *http.Request, environment, org, space, appName string, contentType I.DeploymentType, out io.ReadWriter, reqChan chan I.DeployResponse) {
	d.DeployCall.Called++

	d.DeployCall.Received.Request = req
	d.DeployCall.Received.Environment = environment
	d.DeployCall.Received.Org = org
	d.DeployCall.Received.Space = space
	d.DeployCall.Received.AppName = appName
	d.DeployCall.Received.ContentType = contentType
	d.DeployCall.Received.Response = out

	fmt.Fprint(out, d.DeployCall.Write.Output)

	response := I.DeployResponse{
		StatusCode: d.DeployCall.Returns.StatusCode,
		Error: d.DeployCall.Returns.Error,
	}

	reqChan <- response
}
