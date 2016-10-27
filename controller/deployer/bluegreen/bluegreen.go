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
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless this is the first deploy and disable rollback is enabled.
func (bg BlueGreen) Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, response io.Writer) error {
	var (
		actors  = make([]actor, len(environment.Foundations))
		buffers = make([]*bytes.Buffer, len(environment.Foundations))
	)

	for i, foundationURL := range environment.Foundations {
		pusher, err := bg.PusherCreator.CreatePusher()
		if err != nil {
			return err
		}
		defer pusher.CleanUp()

		actors[i] = newActor(pusher, foundationURL)
		defer close(actors[i].commands)

		buffers[i] = &bytes.Buffer{}
	}

	defer func() {
		for _, buffer := range buffers {
			fmt.Fprintf(response, "\n%s Cloud Foundry Output %s\n", strings.Repeat("-", 19), strings.Repeat("-", 19))

			buffer.WriteTo(response)
		}
		fmt.Fprintf(response, "\n%s End Cloud Foundry Output %s\n", strings.Repeat("-", 17), strings.Repeat("-", 17))
	}()

	failed := bg.loginAll(actors, buffers, deploymentInfo)
	if failed {
		return errors.New("push failed: login failed")
	}

	bg.cleanUpAll(actors, deploymentInfo)

	failed, appExists := bg.pushAll(actors, buffers, appPath, deploymentInfo)
	if failed {
		if appExists || !environment.DisableFirstDeployRollback {
			bg.rollbackAll(actors, deploymentInfo, appExists)
			return errors.Errorf("push failed: rollback triggered")
		}

		return errors.Errorf("push failed: first deploy, rollback not enabled")
	}

	bg.finishPushAll(actors, deploymentInfo)

	return nil
}

func (bg BlueGreen) loginAll(actors []actor, buffers []*bytes.Buffer, deploymentInfo S.DeploymentInfo) bool {
	failed := false

	for i, a := range actors {
		buffer := buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Login(foundationURL, deploymentInfo, buffer)
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			failed = true
		}
	}

	return failed
}

func (bg BlueGreen) cleanUpAll(actors []actor, deploymentInfo S.DeploymentInfo) {
	for _, a := range actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			if pusher.Exists(deploymentInfo.AppName + "-venerable") {
				return pusher.DeleteVenerable(deploymentInfo)
			}
			return nil
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}

func (bg BlueGreen) pushAll(actors []actor, buffers []*bytes.Buffer, appPath string, deploymentInfo S.DeploymentInfo) (failed bool, appExists bool) {
	for i, a := range actors {
		buffer := buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {

			var exists bool

			if pusher.Exists(deploymentInfo.AppName) {
				exists = true
				appExists = true
			}

			return pusher.Push(appPath, exists, deploymentInfo, buffer)
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			failed = true
		}
	}

	return
}

func (bg BlueGreen) rollbackAll(actors []actor, deploymentInfo S.DeploymentInfo, appExists bool) {
	for _, a := range actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Rollback(appExists, deploymentInfo)
		}
	}

	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}

func (bg BlueGreen) finishPushAll(actors []actor, deploymentInfo S.DeploymentInfo) {
	for _, a := range actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.DeleteVenerable(deploymentInfo)
		}
	}

	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}
