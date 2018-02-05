package mocks

import (
	"io"

	"bytes"

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
			Error error
		}
	}
}

// Push mock method.
func (b *BlueGreener) Push(environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.ReadWriter) error {
	b.PushCall.Received.Environment = environment
	b.PushCall.Received.AppPath = appPath
	b.PushCall.Received.DeploymentInfo = deploymentInfo
	b.PushCall.Received.Out = out

	if b.PushCall.Write != "" {
		bytes.NewBufferString(b.PushCall.Write).WriteTo(out)
	}
	return b.PushCall.Returns.Error
}
