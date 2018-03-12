package mocks

import (
	"io"

	"bytes"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreener handmade mock for tests.
type BlueGreener struct {
	ExecuteCall struct {
		Write    string
		Received struct {
			ActionCreator  I.ActionCreator
			Environment    S.Environment
			AppPath        string
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Returns struct {
			Error I.DeploymentError
		}
	}
	StopCall struct {
		Received struct {
			ActionCreator  I.ActionCreator
			Environment    S.Environment
			AppPath        string
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Returns struct {
			Error error
		}
	}
}

// Push mock method.
func (b *BlueGreener) Execute(actionCreator I.ActionCreator, environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.ReadWriter) error {
	b.ExecuteCall.Received.ActionCreator = actionCreator
	b.ExecuteCall.Received.Environment = environment
	b.ExecuteCall.Received.AppPath = appPath
	b.ExecuteCall.Received.DeploymentInfo = deploymentInfo
	b.ExecuteCall.Received.Out = out

	if b.ExecuteCall.Write != "" {
		bytes.NewBufferString(b.ExecuteCall.Write).WriteTo(out)
	}
	return b.ExecuteCall.Returns.Error
}

func (b *BlueGreener) Stop(actionCreator I.ActionCreator, environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.ReadWriter) error {
	b.StopCall.Received.ActionCreator = actionCreator
	b.StopCall.Received.Environment = environment
	b.StopCall.Received.AppPath = appPath
	b.StopCall.Received.DeploymentInfo = deploymentInfo
	b.StopCall.Received.Out = out

	return b.StopCall.Returns.Error
}
