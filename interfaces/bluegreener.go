package interfaces

import (
	"io"

	"github.com/compozed/deployadactyl/config"
	S "github.com/compozed/deployadactyl/structs"
)

type BlueGreener interface {
	Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.Writer) error
}
