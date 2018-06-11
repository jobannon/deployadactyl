package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type StartManagerFactory interface {
	StartManager(log DeploymentLogger, deployEventData structs.DeployEventData) ActionCreator
}

type StartController interface {
	StartDeployment(deployment *Deployment, response *bytes.Buffer) (deployResponse DeployResponse)
}
