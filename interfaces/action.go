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
	Create(deploymentInfo S.DeploymentInfo, cfContext CFContext, authorization Authorization, response io.ReadWriter, foundationURL, appPath string) (Action, error)
	InitiallyError(initiallyErrors []error) error
	ExecuteError(executeErrors []error) error
	UndoError(executeErrors, undoErrors []error) error
	SuccessError(successErrors []error) error
}
