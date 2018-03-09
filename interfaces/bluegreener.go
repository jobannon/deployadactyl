package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreener interface.
type BlueGreener interface {
	Push(
		actionCreator ActionCreator,
		environment S.Environment,
		appPath string,
		deploymentInfo S.DeploymentInfo,
		response io.ReadWriter,
	) error
	Stop(
		actionCreator ActionCreator,
		environment S.Environment,
		appPath string,
		deploymentInfo S.DeploymentInfo,
		response io.ReadWriter,
	) error
}
