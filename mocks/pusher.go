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
			Domain         string
			DeploymentInfo S.DeploymentInfo
			Out            io.Writer
		}
		Write struct {
			Output string
		}
		Returns struct {
			Logs  []byte
			Error error
		}
	}

	RollbackCall struct {
		Received struct {
			DeploymentInfo S.DeploymentInfo
			FirstDeploy    bool
		}
		Returns struct {
			Error error
		}
	}

	FinishPushCall struct {
		Received struct {
			DeploymentInfo S.DeploymentInfo
			FoundationURL  string
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
		Returns struct {
			Exists bool
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
func (p *Pusher) Push(appPath, domain string, deploymentInfo S.DeploymentInfo, out io.Writer) ([]byte, error) {
	p.PushCall.Received.AppPath = appPath
	p.PushCall.Received.Domain = domain
	p.PushCall.Received.DeploymentInfo = deploymentInfo
	p.PushCall.Received.Out = out

	fmt.Fprint(out, p.PushCall.Write.Output)

	return p.PushCall.Returns.Logs, p.PushCall.Returns.Error
}

// Rollback mock method.
func (p *Pusher) Rollback(deploymentInfo S.DeploymentInfo, firstDeploy bool) error {
	p.RollbackCall.Received.DeploymentInfo = deploymentInfo
	p.RollbackCall.Received.FirstDeploy = firstDeploy

	return p.RollbackCall.Returns.Error
}

// FinishPush mock method.
func (p *Pusher) FinishPush(deploymentInfo S.DeploymentInfo, foundationURL string) error {
	p.FinishPushCall.Received.DeploymentInfo = deploymentInfo
	p.FinishPushCall.Received.FoundationURL = foundationURL

	return p.FinishPushCall.Returns.Error
}

// CleanUp mock method.
func (p *Pusher) CleanUp() error {
	return p.CleanUpCall.Returns.Error
}

// Exists mock method.
func (p *Pusher) Exists(appName string) bool {
	p.ExistsCall.Received.AppName = appName

	return p.ExistsCall.Returns.Exists
}
