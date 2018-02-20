package interfaces

import (
	"github.com/compozed/deployadactyl/structs"
	"io"
	"net/http"
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
		appName string,
		contentType DeploymentType,
		response io.ReadWriter,
		reqChan chan DeployResponse,
	)
}
