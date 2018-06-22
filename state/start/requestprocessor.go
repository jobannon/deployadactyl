package start

import (
	"bytes"

	"github.com/compozed/deployadactyl/interfaces"
)

type StartRequestProcessorConstructor func(log interfaces.DeploymentLogger, controller interfaces.StartController, request interfaces.PutDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor

func NewStartRequestProcessor(log interfaces.DeploymentLogger, sc interfaces.StartController, request interfaces.PutDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor {
	return &StartRequestProcessor{
		StartController: sc,
		Request:         request,
		Response:        buffer,
		Log:             log,
	}
}

type StartRequestProcessor struct {
	StartController interfaces.StartController
	Request         interfaces.PutDeploymentRequest
	Response        *bytes.Buffer
	Log             interfaces.DeploymentLogger
}

func (c StartRequestProcessor) Process() interfaces.DeployResponse {
	return c.StartController.StartDeployment(c.Request, c.Response)
}
