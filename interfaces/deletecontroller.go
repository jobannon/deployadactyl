package interfaces

import (
	"bytes"

	"github.com/compozed/deployadactyl/structs"
)

type DeleteManagerFactory interface {
	DeleteManager(log DeploymentLogger, deployEventData structs.DeployEventData) ActionCreator
}

type DeleteController interface {
	DeleteDeployment(deployment *Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse DeployResponse)
}
