package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

type StopperCreator interface {
	CreateStopper(deploymentInfo S.DeploymentInfo, response io.ReadWriter) (StartStopper, error)
}
