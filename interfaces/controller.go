package interfaces

import (
	"github.com/gin-gonic/gin"
)

type Deployment struct {
	Body          *[]byte
	Type          string
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
	SkipSSL      bool
}

type PutRequest struct {
	State string                 `json:"state"`
	Data  map[string]interface{} `json:"data"`
}

type PutDeploymentRequest struct {
	Deployment
	Request *PutRequest
}

type Controller interface {
	PostRequestHandler(g *gin.Context)

	PutRequestHandler(g *gin.Context)
}
