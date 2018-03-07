// Package bluegreen is responsible for concurrently pushing an application to multiple Cloud Foundry instances.
package bluegreen

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	P "github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	StopStart "github.com/compozed/deployadactyl/controller/deployer/bluegreen/startstopper"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	PusherCreator  I.PusherCreator
	StopperCreator I.StopperCreator
	Log            I.Logger
	actors         []actor
	buffers        []*bytes.Buffer
	stopBuffers    []*bytes.Buffer
}

func (bg BlueGreen) Stop(environment S.Environment, deploymentInfo S.DeploymentInfo, out io.ReadWriter) error {
	bg.actors = make([]actor, len(environment.Foundations))
	bg.stopBuffers = make([]*bytes.Buffer, len(environment.Foundations))

	deploymentLogger := logger.DeploymentLogger{Log: bg.Log, UUID: deploymentInfo.UUID}

	for i, foundationURL := range environment.Foundations {
		bg.stopBuffers[i] = &bytes.Buffer{}

		stopper, err := bg.StopperCreator.CreateStopper(deploymentInfo, bg.stopBuffers[i])
		if err != nil {
			return InitializationError{err}
		}
		StopperAction := &StopStart.StopperAction{
			Stopper:       stopper,
			FoundationURL: foundationURL,
			AppName:       deploymentInfo.AppName,
		}
		bg.actors[i] = NewActor(StopperAction, foundationURL)
		defer close(bg.actors[i].Commands)
	}

	defer func() {
		for _, buffer := range bg.stopBuffers {
			fmt.Fprintf(out, "\n%s Cloud Foundry Output %s\n", strings.Repeat("-", 19), strings.Repeat("-", 19))

			buffer.WriteTo(out)
		}

		fmt.Fprintf(out, "\n%s End Cloud Foundry Output %s\n", strings.Repeat("-", 17), strings.Repeat("-", 17))
	}()

	loginErrors := bg.initialAll()
	if len(loginErrors) != 0 {
		return LoginError{loginErrors}
	}

	stopErrors := bg.executeAll()
	if len(stopErrors) > 0 {
		rollbackErrors := bg.undoAll(deploymentLogger)
		if len(rollbackErrors) != 0 {
			return RollbackStopError{stopErrors, rollbackErrors}
		}

		return StopError{stopErrors}
	}
	return nil
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless it is the first deploy.
func (bg BlueGreen) Push(environment S.Environment, appPath string, deploymentInfo S.DeploymentInfo, response io.ReadWriter) I.DeploymentError {
	bg.actors = make([]actor, len(environment.Foundations))
	bg.buffers = make([]*bytes.Buffer, len(environment.Foundations))

	deploymentLogger := logger.DeploymentLogger{Log: bg.Log, UUID: deploymentInfo.UUID}

	for i, foundationURL := range environment.Foundations {
		bg.buffers[i] = &bytes.Buffer{}

		pusher, err := bg.PusherCreator.CreatePusher(deploymentInfo, bg.buffers[i])
		if err != nil {
			return InitializationError{err}
		}
		defer pusher.CleanUp()

		pusherAction := &P.PusherAction{
			Pusher:        pusher,
			FoundationURL: foundationURL,
			AppPath:       appPath}
		bg.actors[i] = NewActor(pusherAction, foundationURL)
		defer close(bg.actors[i].Commands)
	}

	defer func() {
		for _, buffer := range bg.buffers {
			fmt.Fprintf(response, "\n%s Cloud Foundry Output %s\n", strings.Repeat("-", 19), strings.Repeat("-", 19))

			buffer.WriteTo(response)
		}

		fmt.Fprintf(response, "\n%s End Cloud Foundry Output %s\n", strings.Repeat("-", 17), strings.Repeat("-", 17))
	}()

	loginErrors := bg.initialAll()
	if len(loginErrors) != 0 {
		return LoginError{loginErrors}
	}

	pushErrors := bg.executeAll()
	if len(pushErrors) != 0 {
		if !environment.EnableRollback {
			deploymentLogger.Errorf("Failed to deploy, deployment not rolled back due to EnableRollback=false")

			finishPushErrors := bg.successAll()
			if len(finishPushErrors) != 0 {
				return FinishPushError{finishPushErrors}
			}

			return PushError{pushErrors}
		} else {
			rollbackErrors := bg.undoAll(deploymentLogger)
			if len(rollbackErrors) != 0 {
				return RollbackError{pushErrors, rollbackErrors}
			}

			return PushError{pushErrors}
		}
	}

	finishPushErrors := bg.successAll()
	if len(finishPushErrors) != 0 {
		return FinishPushError{finishPushErrors}
	}

	return nil
}

func (bg BlueGreen) commands(doFunc ActorCommand) (manyErrors []error) {
	for _, a := range bg.actors {
		a.Commands <- doFunc
	}
	for _, a := range bg.actors {
		if err := <-a.Errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}
	return
}

func (bg BlueGreen) initialAll() []error {
	doFunc := func(action I.Action, foundationURL string) error {
		return action.Initially()
	}

	return bg.commands(doFunc)
}

func (bg BlueGreen) executeAll() []error {
	doFunc := func(action I.Action, foundationURL string) error {
		return action.Execute()
	}

	return bg.commands(doFunc)
}

func (bg BlueGreen) successAll() []error {
	doFunc := func(action I.Action, foundationURL string) error {
		return action.Success()
	}

	return bg.commands(doFunc)
}

func (bg BlueGreen) undoAll(log I.Logger) (manyErrors []error) {
	doFunc := func(action I.Action, foundationURL string) error {
		err := action.Undo()
		if err != nil {
			log.Errorf("Could not rollback app on foundation %s with error: %s", foundationURL, err.Error())
		}
		return err
	}

	return bg.commands(doFunc)
}
