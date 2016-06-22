package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

type Deployer interface {
	Deploy(deploymentInfo S.DeploymentInfo, out io.Writer) error
}
