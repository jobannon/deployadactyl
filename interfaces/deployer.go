package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// Deployer interface.
type Deployer interface {
	Deploy(deploymentInfo S.DeploymentInfo, out io.Writer) error
}
