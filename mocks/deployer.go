package mocks

import (
	"fmt"
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// Deployer handmade mock for tests.
type Deployer struct {
	DeployCall struct {
		Received struct {
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error error
		}
	}
}

// Deploy mock method.
func (d *Deployer) Deploy(deploymentInfo S.DeploymentInfo, out io.Writer) error {
	d.DeployCall.Received.DeploymentInfo = deploymentInfo
	d.DeployCall.Received.Out = out

	fmt.Fprint(out, d.DeployCall.Write.Output)

	return d.DeployCall.Returns.Error
}
