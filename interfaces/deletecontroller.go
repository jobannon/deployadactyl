package interfaces

import (
	"bytes"

	"github.com/compozed/deployadactyl/structs"
)

type DeleteManagerFactory interface {
	DeleteManager(deployEventData structs.DeployEventData) ActionCreator
}

type DeleteController interface {
	DeleteDeployment(request DeleteDeploymentRequest, response *bytes.Buffer) (deployResponse DeployResponse)
}
