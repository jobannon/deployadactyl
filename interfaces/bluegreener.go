package interfaces

import (
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

type BlueGreener interface {
	Execute(
		actionCreator ActionCreator,
		environment S.Environment,
		deploymentInfo S.DeploymentInfo,
		response io.ReadWriter,
	) error
}
