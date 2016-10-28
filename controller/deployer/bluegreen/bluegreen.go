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
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	PusherCreator I.PusherFactory
	Log           *logging.Logger
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

	failed := bg.loginAll(deploymentInfo)
	if failed {
		return errors.New("push failed: login failed")
	}

	bg.cleanUpAll(deploymentInfo)

	bg.existsAll(deploymentInfo)

	failed = bg.pushAll(appPath, deploymentInfo)
	if failed {
		if !environment.DisableFirstDeployRollback {
			bg.rollbackAll(deploymentInfo)
			return errors.Errorf("push failed: rollback triggered")
		}

		return errors.Errorf("push failed: first deploy, rollback not enabled")
	}

	bg.finishPushAll(deploymentInfo)

	return nil
}

func (bg BlueGreen) loginAll(deploymentInfo S.DeploymentInfo) bool {
	failed := false

	for i, a := range bg.actors {
		buffer := bg.buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Login(foundationURL, deploymentInfo, buffer)
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			failed = true
		}
	}

	return failed
}

func (bg BlueGreen) cleanUpAll(deploymentInfo S.DeploymentInfo) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			pusher.Exists(deploymentInfo.AppName + "-venerable")
			return pusher.DeleteVenerable(deploymentInfo)
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}

func (bg BlueGreen) existsAll(deploymentInfo S.DeploymentInfo) (exists bool) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			pusher.Exists(deploymentInfo.AppName)
			return nil
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
		}
	}
	return
}

func (bg BlueGreen) pushAll(appPath string, deploymentInfo S.DeploymentInfo) (failed bool) {
	for i, a := range bg.actors {
		buffer := bg.buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Push(appPath, deploymentInfo, buffer)
		}
	}
	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			failed = true
		}
	}

	return
}

func (bg BlueGreen) rollbackAll(deploymentInfo S.DeploymentInfo) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Rollback(deploymentInfo)
		}
	}

	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}

func (bg BlueGreen) finishPushAll(deploymentInfo S.DeploymentInfo) {
	for _, a := range bg.actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.DeleteVenerable(deploymentInfo)
		}
	}

	for _, a := range bg.actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}
