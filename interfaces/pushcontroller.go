package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type PostRequest struct {
	ArtifactUrl          string                 `json:"artifact_url"`
	Manifest             string                 `json:"manifest"`
	EnvironmentVariables map[string]string      `json:"environment_variables"`
	HealthCheckEndpoint  string                 `json:"health_check_endpoint"`
	Data                 map[string]interface{} `json:"data"`
}

type PostDeploymentRequest struct {
	Deployment
	Request PostRequest
}

type PushManagerFactory interface {
	PushManager(log DeploymentLogger, deployEventData structs.DeployEventData, cfContext CFContext, auth Authorization, env structs.Environment, envVars map[string]string) ActionCreator
}

type PushController interface {
	RunDeployment(postDeploymentRequest PostDeploymentRequest, response *bytes.Buffer) (deployResponse DeployResponse)
}
