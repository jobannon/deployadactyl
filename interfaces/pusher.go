package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

type Pusher interface {
	Login(foundationURL string, deploymentInfo S.DeploymentInfo, out io.Writer) error
	Push(appPath, foundationURL, domain string, deploymentInfo S.DeploymentInfo, out io.Writer) error
	Rollback(foundationURL string, deploymentInfo S.DeploymentInfo) error
	FinishPush(foundationURL string, deploymentInfo S.DeploymentInfo) error
	CleanUp() error
	AppExists(appName string) bool
}
