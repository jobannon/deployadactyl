package mocks

import (
	"bytes"

	"github.com/compozed/deployadactyl/interfaces"
)

type DeleteController struct {
	DeleteDeploymentCall struct {
		Received struct {
			Deployment interfaces.DeleteDeploymentRequest
			Response   *bytes.Buffer
		}
		Returns struct {
			DeployResponse interfaces.DeployResponse
		}
		Writes string
		Called bool
	}
}

func (c *DeleteController) DeleteDeployment(deployment interfaces.DeleteDeploymentRequest, response *bytes.Buffer) (deployResponse interfaces.DeployResponse) {
	c.DeleteDeploymentCall.Called = true
	c.DeleteDeploymentCall.Received.Deployment = deployment
	c.DeleteDeploymentCall.Received.Deployment.Request = deployment.Request
	c.DeleteDeploymentCall.Received.Response = response

	if c.DeleteDeploymentCall.Writes != "" {
		response.Write([]byte(c.DeleteDeploymentCall.Writes))
	}

	return c.DeleteDeploymentCall.Returns.DeployResponse
}
