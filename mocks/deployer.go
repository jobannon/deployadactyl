package mocks

import (
	"fmt"
	"io"
	"net/http"
)

// Deployer handmade mock for tests.
type Deployer struct {
	DeployCall struct {
		Received struct {
			Request     *http.Request
			Environment string
			Org         string
			Space       string
			AppName     string
			ContentType string
			Out         io.Writer
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
func (d *Deployer) Deploy(req *http.Request, environment, org, space, appName, contentType string, out io.Writer) (err error, statusCode int) {
	d.DeployCall.Received.Request = req
	d.DeployCall.Received.Environment = environment
	d.DeployCall.Received.Org = org
	d.DeployCall.Received.Space = space
	d.DeployCall.Received.AppName = appName
	d.DeployCall.Received.ContentType = contentType
	d.DeployCall.Received.Out = out

	fmt.Fprint(out, d.DeployCall.Write.Output)

	return d.DeployCall.Returns.Error, d.DeployCall.Returns.StatusCode
}
