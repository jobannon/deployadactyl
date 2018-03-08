package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// PusherCreator interface.
type PusherCreator interface {
	CreatePusher(deploymentInfo S.DeploymentInfo, response io.ReadWriter, foundationURL, appPath string) (Action, error)
}
