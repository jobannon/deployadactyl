package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

type StopperCreator interface {
	CreateStopper(deploymentInfo S.DeploymentInfo, cfContext CFContext, authorization Authorization, response io.ReadWriter, foundationURL, appPath string) (Action, error)
}
