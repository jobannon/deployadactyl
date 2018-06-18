package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type StopManagerFactory interface {
	StopManager(log DeploymentLogger, deployEventData structs.DeployEventData) ActionCreator
}

type StopController interface {
	StopDeployment(request PutDeploymentRequest, response *bytes.Buffer) (deployResponse DeployResponse)
}
