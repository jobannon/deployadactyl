package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

type Pusher interface {
	Login(foundationURL string, deploymentInfo S.DeploymentInfo, out io.Writer) error
	Push(appPath, domain string, deploymentInfo S.DeploymentInfo, out io.Writer) ([]byte, error)
	Rollback(deploymentInfo S.DeploymentInfo) error
	FinishPush(deploymentInfo S.DeploymentInfo) error
	CleanUp() error
	Exists(appName string) bool
}
