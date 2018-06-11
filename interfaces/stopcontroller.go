package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type StopManagerFactory interface {
	StopManager(log DeploymentLogger, deployEventData structs.DeployEventData) ActionCreator
}

type StopController interface {
	StopDeployment(deployment *Deployment, response *bytes.Buffer) (deployResponse DeployResponse)
}
