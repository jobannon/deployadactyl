package mocks

import (
	"fmt"
	"io"

	S "github.com/compozed/deployadactyl/structs"
)

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
			FoundationURL  string
			Domain         string
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
			FoundationURL  string
			DeploymentInfo S.DeploymentInfo
		}
		Returns struct {
			Error error
		}
	}

	FinishPushCall struct {
		Received struct {
			FoundationURL  string
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

	AppExistsCall struct {
		Received struct {
			AppName string
		}
		Returns struct {
			Exists bool
		}
	}
}

func (p *Pusher) Login(foundationURL string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	p.LoginCall.Received.FoundationURL = foundationURL
	p.LoginCall.Received.DeploymentInfo = deploymentInfo
	p.LoginCall.Received.Out = out

	fmt.Fprint(out, p.LoginCall.Write.Output)

	return p.LoginCall.Returns.Error
}

func (p *Pusher) Push(appPath, foundationURL, domain string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	p.PushCall.Received.AppPath = appPath
	p.PushCall.Received.FoundationURL = foundationURL
	p.PushCall.Received.Domain = domain
	p.PushCall.Received.DeploymentInfo = deploymentInfo
	p.PushCall.Received.Out = out

	fmt.Fprint(out, p.PushCall.Write.Output)

	return p.PushCall.Returns.Error
}

func (p *Pusher) Rollback(foundationURL string, deploymentInfo S.DeploymentInfo) error {
	p.RollbackCall.Received.FoundationURL = foundationURL
	p.RollbackCall.Received.DeploymentInfo = deploymentInfo

	return p.RollbackCall.Returns.Error
}

func (p *Pusher) FinishPush(foundationURL string, deploymentInfo S.DeploymentInfo) error {
	p.FinishPushCall.Received.FoundationURL = foundationURL
	p.FinishPushCall.Received.DeploymentInfo = deploymentInfo

	return p.FinishPushCall.Returns.Error
}

func (p *Pusher) CleanUp() error {
	return p.CleanUpCall.Returns.Error
}

func (p *Pusher) AppExists(appName string) bool {
	p.AppExistsCall.Received.AppName = appName

	return p.AppExistsCall.Returns.Exists
}
