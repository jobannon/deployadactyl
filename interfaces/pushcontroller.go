package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type PushManagerFactory interface {
	PushManager(log DeploymentLogger, deployEventData structs.DeployEventData, cfContext CFContext, auth Authorization, env structs.Environment, envVars map[string]string) ActionCreator
}

type PushController interface {
	RunDeployment(postDeploymentRequest PostDeploymentRequest, response *bytes.Buffer) (deployResponse DeployResponse)
}
