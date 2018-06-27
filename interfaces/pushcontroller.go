package interfaces

import (
	"bytes"

	"github.com/compozed/deployadactyl/structs"
)

type PushManagerFactory interface {
	PushManager(deployEventData structs.DeployEventData, auth Authorization, env structs.Environment, envVars map[string]string) ActionCreator
}

type PushController interface {
	RunDeployment(postDeploymentRequest PostDeploymentRequest, response *bytes.Buffer) (deployResponse DeployResponse)
}
