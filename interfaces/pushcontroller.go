package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type PushManagerFactory interface {
	PushManager(deployEventData structs.DeployEventData, cfContext CFContext, auth Authorization, env structs.Environment, envVars map[string]string) ActionCreator
}

type PushController interface {
	RunDeployment(deployment *Deployment, response *bytes.Buffer) (deployResponse DeployResponse)
}
