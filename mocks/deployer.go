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
			Request *http.Request
			EnvironmentName, Org, Space, AppName string
			Out            io.Writer
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error error
			StatusCode int
		}
	}

	DeployZipCall struct {
			   Received struct {
					    Request *http.Request
					    EnvironmentName, Org, Space, AppName string
					    Out            io.Writer
				    }
			   Write struct {
					    Output string
				    }
			   Returns struct {
					    Error error
					    StatusCode int
				    }
		   }
}

// Deploy mock method.
func (d *Deployer) Deploy(req *http.Request, environmentName, org, space, appName string, out io.Writer) (err error, statusCode int) {
	d.DeployCall.Received.Request = req
	d.DeployCall.Received.EnvironmentName = environmentName
	d.DeployCall.Received.Org = org
	d.DeployCall.Received.Space = space
	d.DeployCall.Received.AppName = appName
	d.DeployCall.Received.Out = out


	fmt.Fprint(out, d.DeployCall.Write.Output)

	return d.DeployCall.Returns.Error, d.DeployCall.Returns.StatusCode
}

// DeployZip mock method.
func (d *Deployer) DeployZip(req *http.Request, environmentName, org, space, appName string, out io.Writer) (err error, statusCode int) {
	d.DeployZipCall.Received.Request = req
	d.DeployZipCall.Received.EnvironmentName = environmentName
	d.DeployZipCall.Received.Org = org
	d.DeployZipCall.Received.Space = space
	d.DeployZipCall.Received.AppName = appName
	d.DeployZipCall.Received.Out = out


	fmt.Fprint(out, d.DeployZipCall.Write.Output)

	return d.DeployZipCall.Returns.Error, d.DeployZipCall.Returns.StatusCode
}
