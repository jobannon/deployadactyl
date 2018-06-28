package delete

import (
	"bytes"

	"github.com/compozed/deployadactyl/interfaces"
)

type DeleteRequestProcessorConstructor func(log interfaces.DeploymentLogger, controller interfaces.DeleteController, request interfaces.DeleteDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor

func NewDeleteRequestProcessor(log interfaces.DeploymentLogger, sc interfaces.DeleteController, request interfaces.DeleteDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor {
	return &DeleteRequestProcessor{
		DeleteController: sc,
		Request:          request,
		Response:         buffer,
		Log:              log,
	}
}

type DeleteRequestProcessor struct {
	DeleteController interfaces.DeleteController
	Request          interfaces.DeleteDeploymentRequest
	Response         *bytes.Buffer
	Log              interfaces.DeploymentLogger
}

func (c DeleteRequestProcessor) Process() interfaces.DeployResponse {
	return c.DeleteController.DeleteDeployment(c.Request, c.Response)
}
