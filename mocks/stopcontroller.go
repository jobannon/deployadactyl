package mocks

import (
	"bytes"
	"github.com/compozed/deployadactyl/interfaces"
)

type StopController struct {
	StopDeploymentCall struct {
		Received struct {
			Deployment *interfaces.Deployment
			Response   *bytes.Buffer
		}
		Returns struct {
			DeployResponse interfaces.DeployResponse
		}
		Writes string
		Called bool
	}
}

func (c *StopController) StopDeployment(deployment *interfaces.Deployment, response *bytes.Buffer) (deployResponse interfaces.DeployResponse) {
	c.StopDeploymentCall.Called = true
	c.StopDeploymentCall.Received.Deployment = deployment
	c.StopDeploymentCall.Received.Response = response

	if c.StopDeploymentCall.Writes != "" {
		response.Write([]byte(c.StopDeploymentCall.Writes))
	}

	return c.StopDeploymentCall.Returns.DeployResponse
}
