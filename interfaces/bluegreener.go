package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreener interface.
type BlueGreener interface {
	Push(
		pusherCreator PusherCreator,
		environment S.Environment,
		appPath string,
		deploymentInfo S.DeploymentInfo,
		response io.ReadWriter,
	) DeploymentError
	Stop(
		stopperCreator StopperCreator,
		environment S.Environment,
		deploymentInfo S.DeploymentInfo,
		response io.ReadWriter,
	) error
}
