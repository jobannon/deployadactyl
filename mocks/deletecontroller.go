package mocks

import (
	"bytes"

	"github.com/compozed/deployadactyl/interfaces"
)

type DeleteController struct {
	DeleteDeploymentCall struct {
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

func (c *DeleteController) DeleteDeployment(deployment *interfaces.Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse interfaces.DeployResponse) {
	c.DeleteDeploymentCall.Called = true
	c.DeleteDeploymentCall.Received.Deployment = deployment
	c.DeleteDeploymentCall.Received.Data = data
	c.DeleteDeploymentCall.Received.Response = response

	if c.DeleteDeploymentCall.Writes != "" {
		response.Write([]byte(c.DeleteDeploymentCall.Writes))
	}

	return c.DeleteDeploymentCall.Returns.DeployResponse
}
