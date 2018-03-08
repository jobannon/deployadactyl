package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

type StopperCreator interface {
	CreateStopper(cfContext CFContext, authorization Authorization, deploymentInfo S.DeploymentInfo, response io.ReadWriter, foundationURL string) (Action, error)
}
