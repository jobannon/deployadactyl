package interfaces

import (
	"io"
	"net/http"
)

// Deployer interface.
type Deployer interface {
	Deploy(
		req *http.Request,
		environment,
		org,
		space,
		appName,
		contentType string,
		response io.Writer,
	) (int, error)
}
