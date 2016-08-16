package interfaces

import (
	"io"
	"net/http"
)

// Deployer interface.
type Deployer interface {
	Deploy(req *http.Request, environment, org, space, appName, appPath, contentType string, out io.Writer) (error, int)
}
