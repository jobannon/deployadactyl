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
			EnvInstances uint16
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

func (p *PusherCreator) SetUp(envInstances uint16) error {
	p.SetUpCall.Received.EnvInstances = envInstances

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
