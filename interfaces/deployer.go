package interfaces

import (
	"io"

	"github.com/compozed/deployadactyl/structs"
)

type DeployResponse struct {
	StatusCode     int
	DeploymentInfo *structs.DeploymentInfo
	Error          error
}

// Deployer interface.
type Deployer interface {
	Deploy(
		authorization Authorization,
		body io.Reader,
		actionCreator ActionCreator,
		environment,
		org,
		space,
		appName,
		uuid string,
		contentType DeploymentType,
		response io.ReadWriter,
	) *DeployResponse
}
