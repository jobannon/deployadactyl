package interfaces

import (
	"io"
	"net/http"

	"github.com/compozed/deployadactyl/structs"
)

type DeployResponse struct {
	StatusCode     int
	Error          error
	DeploymentInfo *structs.DeploymentInfo
}

// Deployer interface.
type Deployer interface {
	Deploy(
		req *http.Request,
		environment,
		org,
		space,
		appName,
		uuid string,
		contentType DeploymentType,
		response io.ReadWriter,
		reqChan chan DeployResponse,
	)
}
