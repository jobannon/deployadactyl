package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type PushManagerFactory interface {
	PushManager(deployEventData structs.DeployEventData, cfContext CFContext, auth Authorization, env structs.Environment) ActionCreator
}

type PushController interface {
	RunDeployment(deployment *Deployment, response *bytes.Buffer) (deployResponse DeployResponse)
}
