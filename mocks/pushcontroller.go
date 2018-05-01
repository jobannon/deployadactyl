package mocks

import (
	"bytes"
	"github.com/compozed/deployadactyl/interfaces"
)

type PushController struct {
	RunDeploymentCall struct {
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

func (c *PushController) RunDeployment(deployment *interfaces.Deployment, response *bytes.Buffer) (deployResponse interfaces.DeployResponse) {
	c.RunDeploymentCall.Called = true
	c.RunDeploymentCall.Received.Deployment = deployment
	c.RunDeploymentCall.Received.Response = response

	if c.RunDeploymentCall.Writes != "" {
		response.Write([]byte(c.RunDeploymentCall.Writes))
	}

	return c.RunDeploymentCall.Returns.DeployResponse
}
