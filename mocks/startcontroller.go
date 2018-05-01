package mocks

import (
	"bytes"
	"github.com/compozed/deployadactyl/interfaces"
)

type StartController struct {
	StartDeploymentCall struct {
		Received struct {
			Deployment *interfaces.Deployment
			Data       map[string]interface{}
			Response   *bytes.Buffer
		}
		Returns struct {
			DeployResponse interfaces.DeployResponse
		}
		Writes string
		Called bool
	}
}

func (c *StartController) StartDeployment(deployment *interfaces.Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse interfaces.DeployResponse) {
	c.StartDeploymentCall.Called = true
	c.StartDeploymentCall.Received.Deployment = deployment
	c.StartDeploymentCall.Received.Data = data
	c.StartDeploymentCall.Received.Response = response

	if c.StartDeploymentCall.Writes != "" {
		response.Write([]byte(c.StartDeploymentCall.Writes))
	}

	return c.StartDeploymentCall.Returns.DeployResponse
}
