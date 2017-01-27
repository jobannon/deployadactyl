// Package bluegreen is responsible for concurrently pushing an application to multiple Cloud Foundry instances.
package bluegreen

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	PusherCreator I.PusherFactory
	Log           I.Logger
	actors        []actor
	buffers       []*bytes.Buffer
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless this is the first deploy and disable rollback is enabled.
func (bg BlueGreen) Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, response io.Writer) error {
	bg.actors = make([]actor, len(environment.Foundations))
	bg.buffers = make([]*bytes.Buffer, len(environment.Foundations))

	for i, foundationURL := range environment.Foundations {
		pusher, err := bg.PusherCreator.CreatePusher()
		if err != nil {
			return err
		}
		defer pusher.CleanUp()

		bg.actors[i] = newActor(pusher, foundationURL)
		defer close(bg.actors[i].commands)

		bg.buffers[i] = &bytes.Buffer{}
	}

	defer func() {
		for _, buffer := range bg.buffers {
			fmt.Fprintf(response, "\n%s Cloud Foundry Output %s\n", strings.Repeat("-", 19), strings.Repeat("-", 19))

			buffer.WriteTo(response)
		}
		fmt.Fprintf(response, "\n%s End Cloud Foundry Output %s\n", strings.Repeat("-", 17), strings.Repeat("-", 17))
	}()

	loginErrors := bg.loginAll(deploymentInfo)
	if len(loginErrors) != 0 {
		return LoginError{loginErrors}
	}

	bg.existsAll(deploymentInfo)

	pushErrors := bg.pushAll(appPath, deploymentInfo)
	if len(pushErrors) != 0 {
		rollbackErrors := bg.rollbackAll(deploymentInfo)
		if len(rollbackErrors) != 0 {
			return RollbackError{pushErrors, rollbackErrors}
		}

		return PushError{pushErrors}
	}

	finishPushErrors := bg.finishPushAll(deploymentInfo)
	if len(finishPushErrors) != 0 {
		return FinishPushError{finishPushErrors}
	}

	return nil
}

func (bg BlueGreen) loginAll(deploymentInfo S.DeploymentInfo) (manyErrors []error) {
	for i, a := range bg.actors {
		buffer := bg.buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Login(foundationURL, deploymentInfo, buffer)
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) existsAll(deploymentInfo S.DeploymentInfo) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			pusher.Exists(deploymentInfo.AppName)
			return nil
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			// noop
		}
	}
}

func (bg BlueGreen) pushAll(appPath string, deploymentInfo S.DeploymentInfo) (manyErrors []error) {
	for i, a := range bg.actors {
		buffer := bg.buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Push(appPath, deploymentInfo, buffer)
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) rollbackAll(deploymentInfo S.DeploymentInfo) (manyErrors []error) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Rollback(deploymentInfo)
		}
	}

	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			manyErrors = append(manyErrors, err)
		}
	}

	return
}

func (bg BlueGreen) finishPushAll(deploymentInfo S.DeploymentInfo) (manyErrors []error) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.FinishPush(deploymentInfo)
		}
	}

	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			manyErrors = append(manyErrors, err)
		}
	}

	return
}
