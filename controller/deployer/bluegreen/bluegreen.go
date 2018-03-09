// Package bluegreen is responsible for concurrently pushing an application to multiple Cloud Foundry instances.
package bluegreen

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	Log I.Logger
}

func (bg BlueGreen) Stop(actionCreator I.ActionCreator, environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.ReadWriter) error {
	actors := make([]actor, len(environment.Foundations))
	stopBuffers := make([]*bytes.Buffer, len(environment.Foundations))
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
		stopBuffers[i] = &bytes.Buffer{}

		stopper, err := actionCreator.Create(deploymentInfo, cfContext, authorization, stopBuffers[i], foundationURL, appPath)
		if err != nil {
			return InitializationError{err}
		}

		actors[i] = NewActor(stopper)
		defer close(actors[i].Commands)
	}

	defer func() {
		for _, buffer := range stopBuffers {
			fmt.Fprintf(out, "\n%s Cloud Foundry Output %s\n", strings.Repeat("-", 19), strings.Repeat("-", 19))
			buffer.WriteTo(out)
		}

		fmt.Fprintf(out, "\n%s End Cloud Foundry Output %s\n", strings.Repeat("-", 17), strings.Repeat("-", 17))
	}()

	loginErrors := bg.commands(actors, func(action I.Action) error {
		return action.Initially()
	})

	if len(loginErrors) != 0 {
		return LoginError{loginErrors}
	}

	stopErrors := bg.commands(actors, func(action I.Action) error {
		return action.Execute()
	})
	if len(stopErrors) > 0 {
		rollbackErrors := bg.commands(actors, func(action I.Action) error {
			return action.Undo()
		})

		if len(rollbackErrors) != 0 {
			return RollbackStopError{stopErrors, rollbackErrors}
		}

		return StopError{stopErrors}
	}
	return nil
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless it is the first deploy.
func (bg BlueGreen) Push(actionCreator I.ActionCreator, environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, response io.ReadWriter) I.DeploymentError {
	actors := make([]actor, len(environment.Foundations))
	buffers := make([]*bytes.Buffer, len(environment.Foundations))

	deploymentLogger := logger.DeploymentLogger{Log: bg.Log, UUID: deploymentInfo.UUID}
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

		pusher, err := actionCreator.Create(deploymentInfo, cfContext, authorization, buffers[i], foundationURL, appPath)
		if err != nil {
			return InitializationError{err}
		}
		defer pusher.Finally()

		actors[i] = NewActor(pusher)
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
		return LoginError{loginErrors}
	}

	pushErrors := bg.commands(actors, func(action I.Action) error {
		return action.Execute()
	})

	if len(pushErrors) != 0 {
		if !environment.EnableRollback {
			deploymentLogger.Errorf("Failed to deploy, deployment not rolled back due to EnableRollback=false")

			finishPushErrors := bg.commands(actors, func(action I.Action) error {
				return action.Success()
			})

			if len(finishPushErrors) != 0 {
				return FinishPushError{finishPushErrors}
			}

			return PushError{pushErrors}
		} else {
			rollbackErrors := bg.commands(actors, func(action I.Action) error {
				return action.Undo()
			})

			if len(rollbackErrors) != 0 {
				return RollbackError{pushErrors, rollbackErrors}
			}

			return PushError{pushErrors}
		}
	}

	finishPushErrors := bg.commands(actors, func(action I.Action) error {
		return action.Success()
	})
	if len(finishPushErrors) != 0 {
		return FinishPushError{finishPushErrors}
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
