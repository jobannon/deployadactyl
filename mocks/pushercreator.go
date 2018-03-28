package mocks

import (
	"io"

	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// PusherCreator handmade mock for tests.
type PusherCreator struct {
	SetUpCall struct {
		Called   bool
		Received struct {
			Environment S.Environment
		}
		Returns struct {
			Err error
		}
	}
	OnStartCall struct {
		Called  bool
		Returns struct {
			Err error
		}
	}
	CreatePusherCall struct {
		TimesCalled int
		Returns     struct {
			Pushers []interfaces.Action
			Error   []error
		}
	}
	OnFinishCall struct {
		Called   bool
		Received struct {
			Environment S.Environment
			Response    io.ReadWriter
			Error       error
		}
		Returns struct {
			DeployResponse interfaces.DeployResponse
		}
	}
	CleanUpCall struct {
		Called bool
	}
}

type FileSystemCleaner struct {
	RemoveAllCall struct {
		Called   bool
		Received struct {
			Path string
		}
		Returns struct {
			Error error
		}
	}
}

// CreatePusher mock method.

func (p *FileSystemCleaner) RemoveAll(path string) error {
	p.RemoveAllCall.Called = true

	p.RemoveAllCall.Received.Path = path

	return p.RemoveAllCall.Returns.Error
}

func (p *PusherCreator) SetUp(environment S.Environment) error {
	p.SetUpCall.Received.Environment = environment

	p.SetUpCall.Called = true
	return p.SetUpCall.Returns.Err
}

func (p *PusherCreator) CleanUp() {
	p.CleanUpCall.Called = true
}

func (p *PusherCreator) OnStart() error {
	p.OnStartCall.Called = true

	return p.OnStartCall.Returns.Err
}

func (p *PusherCreator) OnFinish(env S.Environment, response io.ReadWriter, err error) interfaces.DeployResponse {
	p.OnFinishCall.Called = true
	p.OnFinishCall.Received.Environment = env
	p.OnFinishCall.Received.Response = response
	p.OnFinishCall.Received.Error = err

	return p.OnFinishCall.Returns.DeployResponse
}

func (p *PusherCreator) Create(environment S.Environment, response io.ReadWriter, foundationURL string) (interfaces.Action, error) {
	defer func() { p.CreatePusherCall.TimesCalled++ }()

	return p.CreatePusherCall.Returns.Pushers[p.CreatePusherCall.TimesCalled], p.CreatePusherCall.Returns.Error[p.CreatePusherCall.TimesCalled]
}

func (p *PusherCreator) InitiallyError(initiallyErrors []error) error {
	return bluegreen.LoginError{LoginErrors: initiallyErrors}
}

func (p *PusherCreator) ExecuteError(executeErrors []error) error {
	return bluegreen.PushError{PushErrors: executeErrors}
}

func (p *PusherCreator) UndoError(executeErrors, undoErrors []error) error {
	return bluegreen.RollbackError{PushErrors: executeErrors, RollbackErrors: undoErrors}
}

func (p *PusherCreator) SuccessError(successErrors []error) error {
	return bluegreen.FinishPushError{FinishPushError: successErrors}
}
