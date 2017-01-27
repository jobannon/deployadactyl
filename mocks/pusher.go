package mocks

import (
	"fmt"
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

// Pusher handmade mock for tests.
type Pusher struct {
	LoginCall struct {
		Received struct {
			FoundationURL  string
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error error
		}
	}

	PushCall struct {
		Received struct {
			AppPath        string
			AppExists      bool
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error error
		}
	}

	RollbackCall struct {
		Received struct {
			AppExists      bool
			DeploymentInfo S.DeploymentInfo
		}
		Returns struct {
			Error error
		}
	}

	FinishPushCall struct {
		Received struct {
			DeploymentInfo S.DeploymentInfo
		}
		Returns struct {
			Error error
		}
	}

	CleanUpCall struct {
		Returns struct {
			Error error
		}
	}

	ExistsCall struct {
		Received struct {
			AppName string
		}
	}
}

// Login mock method.
func (p *Pusher) Login(foundationURL string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	p.LoginCall.Received.FoundationURL = foundationURL
	p.LoginCall.Received.DeploymentInfo = deploymentInfo
	p.LoginCall.Received.Out = out

	fmt.Fprint(out, p.LoginCall.Write.Output)

	return p.LoginCall.Returns.Error
}

// Push mock method.
func (p *Pusher) Push(appPath string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	p.PushCall.Received.AppPath = appPath
	p.PushCall.Received.DeploymentInfo = deploymentInfo
	p.PushCall.Received.Out = out

	fmt.Fprint(out, p.PushCall.Write.Output)

	return p.PushCall.Returns.Error
}

// Rollback mock method.
func (p *Pusher) Rollback(deploymentInfo S.DeploymentInfo) error {
	p.RollbackCall.Received.DeploymentInfo = deploymentInfo

	return p.RollbackCall.Returns.Error
}

// FinishPush mock method.
func (p *Pusher) FinishPush(deploymentInfo S.DeploymentInfo) error {
	p.FinishPushCall.Received.DeploymentInfo = deploymentInfo

	return p.FinishPushCall.Returns.Error
}

// CleanUp mock method.
func (p *Pusher) CleanUp() error {
	return p.CleanUpCall.Returns.Error
}

// Exists mock method.
func (p *Pusher) Exists(appName string) {
	p.ExistsCall.Received.AppName = appName
}
