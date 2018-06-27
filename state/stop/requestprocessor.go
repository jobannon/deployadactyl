package stop

import (
	"bytes"

	"github.com/compozed/deployadactyl/interfaces"
)

type StopRequestProcessorConstructor func(log interfaces.DeploymentLogger, controller interfaces.StopController, request interfaces.PutDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor

func NewStopRequestProcessor(log interfaces.DeploymentLogger, sc interfaces.StopController, request interfaces.PutDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor {
	return &StopRequestProcessor{
		StopController: sc,
		Request:        request,
		Response:       buffer,
		Log:            log,
	}
}

type StopRequestProcessor struct {
	StopController interfaces.StopController
	Request        interfaces.PutDeploymentRequest
	Response       *bytes.Buffer
	Log            interfaces.DeploymentLogger
}

func (c StopRequestProcessor) Process() interfaces.DeployResponse {
	return c.StopController.StopDeployment(c.Request, c.Response)
}
