package mocks

import (
	"io"

	"github.com/compozed/deployadactyl/config"
	S "github.com/compozed/deployadactyl/structs"
)

type BlueGreener struct {
	PushCall struct {
		Received struct {
			Environment    config.Environment
			AppPath        string
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Returns struct {
			Error error
		}
	}
}

func (b *BlueGreener) Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	b.PushCall.Received.Environment = environment
	b.PushCall.Received.AppPath = appPath
	b.PushCall.Received.DeploymentInfo = deploymentInfo
	b.PushCall.Received.Out = out

	return b.PushCall.Returns.Error
}
