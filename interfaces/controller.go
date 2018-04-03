package interfaces

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

type DeploymentType struct {
	JSON bool
	ZIP  bool
}

type Deployment struct {
	Body          *[]byte
	Type          DeploymentType
	Authorization Authorization
	CFContext     CFContext
}

type Authorization struct {
	Username string
	Password string
}

type CFContext struct {
	Environment  string
	Organization string
	Space        string
	Application  string
	UUID         string
	SkipSSL      bool
}

type Controller interface {
	RunDeployment(deployment *Deployment, response *bytes.Buffer) DeployResponse

	RunDeploymentViaHttp(g *gin.Context)

	PutRequestHandler(g *gin.Context)
}
