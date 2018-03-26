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
		deploymentInfo *structs.DeploymentInfo,
		environment structs.Environment,
		authorization Authorization,
		body io.Reader,
		actionCreator ActionCreator,
		environmentStr,
		org,
		space,
		appName string,
		contentType DeploymentType,
		response io.ReadWriter,
	) *DeployResponse
}
