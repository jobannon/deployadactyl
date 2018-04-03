package interfaces

import (
	"bytes"
	"github.com/compozed/deployadactyl/structs"
)

type StopManagerFactory interface {
	StopManager(deployEventData structs.DeployEventData) ActionCreator
}

type StopController interface {
	StopDeployment(deployment *Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse DeployResponse)
}
