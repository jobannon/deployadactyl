package mocks

import (
	"io"

	"bytes"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreener handmade mock for tests.
type BlueGreener struct {
	PushCall struct {
		Write    string
		Received struct {
			Environment    S.Environment
			AppPath        string
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Returns struct {
			Error I.DeploymentError
		}
	}
}

// Push mock method.
func (b *BlueGreener) Push(environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.ReadWriter) I.DeploymentError {
	b.PushCall.Received.Environment = environment
	b.PushCall.Received.AppPath = appPath
	b.PushCall.Received.DeploymentInfo = deploymentInfo
	b.PushCall.Received.Out = out

	if b.PushCall.Write != "" {
		bytes.NewBufferString(b.PushCall.Write).WriteTo(out)
	}
	return b.PushCall.Returns.Error
}
