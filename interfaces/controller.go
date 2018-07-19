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
	UUID  string                 `json:"uuid"`
}

type PutDeploymentRequest struct {
	Deployment
	Request PutRequest
}

type PostRequest struct {
	ArtifactUrl          string                 `json:"artifact_url"`
	Manifest             string                 `json:"manifest"`
	EnvironmentVariables map[string]string      `json:"environment_variables"`
	HealthCheckEndpoint  string                 `json:"health_check_endpoint"`
	Data                 map[string]interface{} `json:"data"`
	UUID                 string                 `json:"uuid"`
}

type PostDeploymentRequest struct {
	Deployment
	Request PostRequest
}

type DeleteRequest struct {
	State string                 `json:"state"`
	Data  map[string]interface{} `json:"data"`
	UUID  string                 `json:"uuid"`
}

type DeleteDeploymentRequest struct {
	Deployment
	Request DeleteRequest
}

type Controller interface {
	PostRequestHandler(g *gin.Context)

	PutRequestHandler(g *gin.Context)

	DeleteRequestHandler(g *gin.Context)
}
