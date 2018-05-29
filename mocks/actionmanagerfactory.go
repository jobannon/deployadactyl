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
			Log             interfaces.DeploymentLogger
			DeployEventData structs.DeployEventData
			CFContext       interfaces.CFContext
			Auth            interfaces.Authorization
			Environment     structs.Environment
			EnvVars         map[string]string
		}
		Returns struct {
			ActionCreator interfaces.ActionCreator
		}
	}
}

// CreatePusher mock method.

func (p *PushManagerFactory) PushManager(log interfaces.DeploymentLogger, deployEventData structs.DeployEventData, cf interfaces.CFContext, auth interfaces.Authorization, env structs.Environment, envVars map[string]string) interfaces.ActionCreator {
	p.PushManagerCall.Called = true
	p.PushManagerCall.Received.Log = log
	p.PushManagerCall.Received.DeployEventData = deployEventData
	p.PushManagerCall.Received.CFContext = cf
	p.PushManagerCall.Received.Auth = auth
	p.PushManagerCall.Received.Environment = env
	p.PushManagerCall.Received.EnvVars = envVars

	return p.PushManagerCall.Returns.ActionCreator
}

type StopManagerFactory struct {
	StopManagerCall struct {
		Called   bool
		Received struct {
			Log             interfaces.DeploymentLogger
			DeployEventData structs.DeployEventData
		}
		Returns struct {
			ActionCreater interfaces.ActionCreator
		}
	}
}

func (s *StopManagerFactory) StopManager(log interfaces.DeploymentLogger, DeployEventData structs.DeployEventData) interfaces.ActionCreator {
	s.StopManagerCall.Called = true
	s.StopManagerCall.Received.Log = log
	s.StopManagerCall.Received.DeployEventData = DeployEventData

	return s.StopManagerCall.Returns.ActionCreater
}

type StartManagerFactory struct {
	StartManagerCall struct {
		Called   bool
		Received struct {
			Log             interfaces.DeploymentLogger
			DeployEventData structs.DeployEventData
		}
		Returns struct {
			ActionCreater interfaces.ActionCreator
		}
	}
}

func (t *StartManagerFactory) StartManager(log interfaces.DeploymentLogger, DeployEventData structs.DeployEventData) interfaces.ActionCreator {
	t.StartManagerCall.Called = true
	t.StartManagerCall.Received.Log = log
	t.StartManagerCall.Received.DeployEventData = DeployEventData

	return t.StartManagerCall.Returns.ActionCreater
}
