package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// Pusher interface.
type Pusher interface {
	Login(foundationURL string, deploymentInfo S.DeploymentInfo, out io.Writer) error
	Push(appPath, domain string, deploymentInfo S.DeploymentInfo, out io.Writer) ([]byte, error)
	Rollback(deploymentInfo S.DeploymentInfo, firstDeploy bool) error
	DeleteVenerable(deploymentInfo S.DeploymentInfo, foundationURL string) error
	CleanUp() error
	Exists(appName string) bool
}
