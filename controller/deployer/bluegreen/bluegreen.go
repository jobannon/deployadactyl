// Package bluegreen is responsible for concurrently pushing an application to multiple Cloud Foundry instances.
package bluegreen

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	Log I.Logger
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless it is the first deploy.
func (bg BlueGreen) Execute(actionCreator I.ActionCreator, environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, response io.ReadWriter) error {
	actors := make([]actor, len(environment.Foundations))
	buffers := make([]*bytes.Buffer, len(environment.Foundations))

	cfContext := I.CFContext{
		Environment:  environment.Name,
		Organization: deploymentInfo.Org,
		Space:        deploymentInfo.Space,
		Application:  deploymentInfo.AppName,
		UUID:         deploymentInfo.UUID,
		SkipSSL:      deploymentInfo.SkipSSL,
	}
	authorization := I.Authorization{
		Username: deploymentInfo.Username,
		Password: deploymentInfo.Password,
	}
	for i, foundationURL := range environment.Foundations {
		buffers[i] = &bytes.Buffer{}

		action, err := actionCreator.Create(deploymentInfo, cfContext, authorization, environment, buffers[i], foundationURL, appPath)
		if err != nil {
			return InitializationError{err}
		}
		defer action.Finally()

		actors[i] = NewActor(action)
		defer close(actors[i].Commands)
	}

	defer func() {
		for _, buffer := range buffers {
			fmt.Fprintf(response, "\n%s Cloud Foundry Output %s\n", strings.Repeat("-", 19), strings.Repeat("-", 19))
			buffer.WriteTo(response)
		}

		fmt.Fprintf(response, "\n%s End Cloud Foundry Output %s\n", strings.Repeat("-", 17), strings.Repeat("-", 17))
	}()

	loginErrors := bg.commands(actors, func(action I.Action) error {
		return action.Initially()
	})

	if len(loginErrors) != 0 {
		return actionCreator.InitiallyError(loginErrors)
	}

	actionErrors := bg.commands(actors, func(action I.Action) error {
		return action.Execute()
	})

	if len(actionErrors) != 0 {
		rollbackErrors := bg.commands(actors, func(action I.Action) error {
			return action.Undo()
		})

		if len(rollbackErrors) != 0 {
			return actionCreator.UndoError(actionErrors, rollbackErrors)
		}

		return actionCreator.ExecuteError(actionErrors)
	}

	finishActionErrors := bg.commands(actors, func(action I.Action) error {
		return action.Success()
	})
	if len(finishActionErrors) != 0 {
		return actionCreator.SuccessError(finishActionErrors)
	}

	return nil
}

func (bg BlueGreen) commands(actors []actor, doFunc ActorCommand) (manyErrors []error) {
	for _, a := range actors {
		a.Commands <- doFunc
	}
	for _, a := range actors {
		if err := <-a.Errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}
	return
}
