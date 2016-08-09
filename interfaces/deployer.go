package interfaces

import (
	"io"
	"net/http"
)

// Deployer interface.
type Deployer interface {
	Deploy(req *http.Request, environment, org, space, appName string, out io.Writer) (error, int)
	DeployZip(req *http.Request, environment, org, space, appName, appPath string, out io.Writer) (error, int)
}
