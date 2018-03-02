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
	PusherCreator  I.PusherCreator
	StopperCreator I.StopperCreator
	Log            I.Logger
	actors         []actor
	stopActors     []stopActor
	buffers        []*bytes.Buffer
	stopBuffers    []*bytes.Buffer
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless it is the first deploy.
func (bg BlueGreen) Stop(environment S.Environment, deploymentInfo S.DeploymentInfo) error {
	bg.stopActors = make([]stopActor, len(environment.Foundations))
	bg.stopBuffers = make([]*bytes.Buffer, len(environment.Foundations))

	for i, foundationURL := range environment.Foundations {
		bg.stopBuffers[i] = &bytes.Buffer{}

		stopper, err := bg.StopperCreator.CreateStopper(deploymentInfo, bg.stopBuffers[i])
		if err != nil {
			return InitializationError{err}
		}
		bg.stopActors[i] = newStopActor(stopper, foundationURL)
		defer close(bg.stopActors[i].commands)
	}

	loginErrors := bg.loginAllStoppers()
	if len(loginErrors) != 0 {
		return LoginError{loginErrors}
	}

	bg.stopAll(deploymentInfo.AppName)
	return nil
}

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

		bg.actors[i] = newActor(pusher, foundationURL)
		defer close(bg.actors[i].commands)
	}

	defer func() {
		for _, buffer := range bg.buffers {
			fmt.Fprintf(response, "\n%s Cloud Foundry Output %s\n", strings.Repeat("-", 19), strings.Repeat("-", 19))

			buffer.WriteTo(response)
		}

		fmt.Fprintf(response, "\n%s End Cloud Foundry Output %s\n", strings.Repeat("-", 17), strings.Repeat("-", 17))
	}()

	loginErrors := bg.loginAll()
	if len(loginErrors) != 0 {
		return LoginError{loginErrors}
	}

	pushErrors := bg.pushAll(appPath)
	if len(pushErrors) != 0 {
		if !environment.EnableRollback {
			deploymentLogger.Errorf("Failed to deploy, deployment not rolled back due to EnableRollback=false")

			finishPushErrors := bg.finishPushAll()
			if len(finishPushErrors) != 0 {
				return FinishPushError{finishPushErrors}
			}

			return PushError{pushErrors}
		} else {
			rollbackErrors := bg.undoPushAll(deploymentLogger)
			if len(rollbackErrors) != 0 {
				return RollbackError{pushErrors, rollbackErrors}
			}

			return PushError{pushErrors}
		}
	}

	finishPushErrors := bg.finishPushAll()
	if len(finishPushErrors) != 0 {
		return FinishPushError{finishPushErrors}
	}

	return nil
}

func (bg BlueGreen) loginAll() (manyErrors []error) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Login(foundationURL)
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) loginAllStoppers() (manyErrors []error) {
	for _, a := range bg.stopActors {
		a.commands <- func(stopper I.StartStopper, foundationURL string) error {
			return stopper.Login(foundationURL)
		}
	}
	for _, a := range bg.stopActors {
		if err := <-a.errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) pushAll(appPath string) (manyErrors []error) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Push(appPath, foundationURL)
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) stopAll(appName string) (manyErrors []error) {
	for _, a := range bg.stopActors {
		a.commands <- func(stopper I.StartStopper, foundationURL string) error {
			return stopper.Stop(appName, foundationURL)
		}
	}
	for _, a := range bg.stopActors {
		if err := <-a.errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) finishPushAll() (manyErrors []error) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.FinishPush()
		}
	}

	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) undoPushAll(log I.Logger) (manyErrors []error) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			err := pusher.UndoPush()
			if err != nil {
				log.Errorf("Could not rollback app on foundation %s with error: %s", foundationURL, err.Error())
			}
			return err
		}
	}

	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			manyErrors = append(manyErrors, err)
		}
	}

	return
}
