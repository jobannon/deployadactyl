package interfaces

import (
	S "github.com/compozed/deployadactyl/structs"
	"io"
)

type Action interface {
	Initially() error
	Execute() error
	Verify() error
	Success() error
	Undo() error
	Finally() error
}

type ActionCreator interface {
	SetUp() error
	CleanUp()
	OnStart() error
	OnFinish(environment S.Environment, response io.ReadWriter, err error) DeployResponse
	Create(environment S.Environment, response io.ReadWriter, foundationURL string) (Action, error)
	InitiallyError(initiallyErrors []error) error
	ExecuteError(executeErrors []error) error
	UndoError(executeErrors, undoErrors []error) error
	SuccessError(successErrors []error) error
}
