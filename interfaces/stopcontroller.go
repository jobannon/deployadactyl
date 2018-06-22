package interfaces

import (
	"bytes"

	"github.com/compozed/deployadactyl/structs"
)

type StopManagerFactory interface {
	StopManager(deployEventData structs.DeployEventData) ActionCreator
}

type StopController interface {
	StopDeployment(request PutDeploymentRequest, response *bytes.Buffer) (deployResponse DeployResponse)
}
