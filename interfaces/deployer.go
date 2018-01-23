package interfaces

import (
	"io"
	"net/http"
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
		contentType DeploymentType,
		response io.ReadWriter,
		reqChan chan DeployResponse,
	)
}
