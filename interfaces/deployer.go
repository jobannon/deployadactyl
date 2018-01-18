package interfaces

import (
	"io"
	"net/http"
	"github.com/compozed/deployadactyl/constants"
)

type DeployResponse struct {
	StatusCode int
	Error      error
}

// Deployer interface.
type Deployer interface {
	Deploy(
		req *http.Request,
		environment,
		org,
		space,
		appName string,
		contentType constants.DeploymentType,
		response io.ReadWriter,
		reqChan chan DeployResponse,
	)
}
