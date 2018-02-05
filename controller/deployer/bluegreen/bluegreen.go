// Package bluegreen is responsible for concurrently pushing an application to multiple Cloud Foundry instances.
package bluegreen

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	PusherCreator I.PusherCreator
	Log           I.Logger
	actors        []actor
	buffers       []*bytes.Buffer
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless it is the first deploy.
func (bg BlueGreen) Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, response io.ReadWriter) error {
	bg.actors = make([]actor, len(environment.Foundations))
	bg.buffers = make([]*bytes.Buffer, len(environment.Foundations))

	deploymentLogger := logger.DeploymentLogger{bg.Log, deploymentInfo.UUID}

	for i, foundationURL := range environment.Foundations {
		bg.buffers[i] = &bytes.Buffer{}

		pusher, err := bg.PusherCreator.CreatePusher(deploymentInfo, bg.buffers[i])
		if err != nil {
			return err
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
