package mocks

import (
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
)

// PushManager handmade mock for tests.
type PushManagerFactory struct {
	PushManagerCall struct {
		Called   bool
		Received struct {
			DeployEventData structs.DeployEventData
			CFContext       interfaces.CFContext
			Auth            interfaces.Authorization
			Environment     structs.Environment
		}
		Returns struct {
			ActionCreator interfaces.ActionCreator
		}
	}
}

// CreatePusher mock method.

func (p *PushManagerFactory) PushManager(deployEventData structs.DeployEventData, cf interfaces.CFContext, auth interfaces.Authorization, env structs.Environment) interfaces.ActionCreator {
	p.PushManagerCall.Called = true
	p.PushManagerCall.Received.DeployEventData = deployEventData
	p.PushManagerCall.Received.CFContext = cf
	p.PushManagerCall.Received.Auth = auth
	p.PushManagerCall.Received.Environment = env

	return p.PushManagerCall.Returns.ActionCreator
}

type StopManagerFactory struct {
	StopManagerCall struct {
		Called   bool
		Received struct {
			DeployEventData structs.DeployEventData
		}
		Returns struct {
			ActionCreater interfaces.ActionCreator
		}
	}
}

func (s *StopManagerFactory) StopManager(DeployEventData structs.DeployEventData) interfaces.ActionCreator {
	s.StopManagerCall.Called = true
	s.StopManagerCall.Received.DeployEventData = DeployEventData

	return s.StopManagerCall.Returns.ActionCreater
}

type StartManagerFactory struct {
	StartManagerCall struct {
		Called   bool
		Received struct {
			DeployEventData structs.DeployEventData
		}
		Returns struct {
			ActionCreater interfaces.ActionCreator
		}
	}
}

func (t *StartManagerFactory) StartManager(DeployEventData structs.DeployEventData) interfaces.ActionCreator {
	t.StartManagerCall.Called = true
	t.StartManagerCall.Received.DeployEventData = DeployEventData

	return t.StartManagerCall.Returns.ActionCreater
}
