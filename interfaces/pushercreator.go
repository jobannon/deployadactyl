package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// PusherCreator interface.
type PusherCreator interface {
	CreatePusher(deploymentInfo S.DeploymentInfo, cfContext CFContext, authorization Authorization, response io.ReadWriter, foundationURL, appPath string) (Action, error)
}
