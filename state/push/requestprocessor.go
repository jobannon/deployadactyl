package push

import (
	"bytes"

	"github.com/compozed/deployadactyl/interfaces"
)

type PushRequestProcessorConstructor func(log interfaces.DeploymentLogger, controller interfaces.PushController, request interfaces.PostDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor

func NewPushRequestProcessor(log interfaces.DeploymentLogger, pc interfaces.PushController, request interfaces.PostDeploymentRequest, buffer *bytes.Buffer) interfaces.RequestProcessor {
	return &PushRequestProcessor{
		PushController: pc,
		Request:        request,
		Response:       buffer,
		Log:            log,
	}
}

type PushRequestProcessor struct {
	PushController interfaces.PushController
	Request        interfaces.PostDeploymentRequest
	Response       *bytes.Buffer
	Log            interfaces.DeploymentLogger
}

func (c PushRequestProcessor) Process() interfaces.DeployResponse {
	return c.PushController.RunDeployment(c.Request, c.Response)
}
