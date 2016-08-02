// Package bluegreen is responsible for concurrently pushing an application to multiple Cloud Foundry instances.
package bluegreen

import (
	"bytes"
	"fmt"
	"io"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	pushFailedRollbackTriggered     = "push failed: rollback triggered"
	pushFailedNoRollbackFirstDeploy = "push failed: first deploy, rollback not enabled"
	loginFailed                     = "push failed: login failed"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	PusherCreator I.PusherFactory
	Log           *logging.Logger
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless this is the first deploy and disable rollback is enabled.
func (bg BlueGreen) Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	var responseLogs []byte

	actors := make([]actor, len(environment.Foundations))
	buffers := make([]*bytes.Buffer, len(actors))
	firstDeploy := true

	for i, foundationURL := range environment.Foundations {
		pusher, err := bg.PusherCreator.CreatePusher()
		if err != nil {
			return errors.New(err)
		}
		defer pusher.CleanUp()

		actors[i] = newActor(pusher, foundationURL)
		defer close(actors[i].commands)

		buffers[i] = &bytes.Buffer{}
	}

	failed := bg.loginAll(actors, buffers, deploymentInfo)
	if failed {
		for _, buffer := range buffers {
			buffer.WriteTo(out)
		}
		return errors.New(loginFailed)
	}

	failed, firstDeploy, responseLogs = bg.pushAll(actors, buffers, appPath, environment.Domain, deploymentInfo)

	combinedOutput := &bytes.Buffer{}
	for _, buffer := range buffers {
		buffer.WriteTo(combinedOutput)
	}
	_, err := combinedOutput.WriteTo(out)
	if err != nil {
		return errors.New(err)
	}

	defer func() {
		fmt.Fprint(out, string(responseLogs))
	}()

	if failed && (!firstDeploy || (firstDeploy && !environment.DisableFirstDeployRollback)) {
		bg.rollbackAll(actors, deploymentInfo)
		return errors.Errorf(pushFailedRollbackTriggered + "\n" + combinedOutput.String())
	}

	bg.finishPushAll(actors, deploymentInfo)

	if failed {
		return errors.Errorf(pushFailedNoRollbackFirstDeploy + "\n" + combinedOutput.String())
	}

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

func (bg BlueGreen) pushAll(actors []actor, buffers []*bytes.Buffer, appPath, domain string, deploymentInfo S.DeploymentInfo) (bool, bool, []byte) {
	failed, firstDeploy := false, true
	var responseLogs []byte

	for i, a := range actors {
		buffer := buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			if pusher.Exists(deploymentInfo.AppName) {
				firstDeploy = false
			}
			logs, err := pusher.Push(appPath, domain, deploymentInfo, buffer)
			if logs != nil {
				responseLogs = append(responseLogs, []byte(fmt.Sprintf("\nCloud Foundry Logs for %s at %s\n------------------------------------------------------------\n%s\n", deploymentInfo.AppName, foundationURL, logs))...)
			}
			return err
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			failed = true
		}
	}

	return failed, firstDeploy, responseLogs
}

func (bg BlueGreen) rollbackAll(actors []actor, deploymentInfo S.DeploymentInfo) {
	for _, a := range actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Rollback(deploymentInfo)
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
			return pusher.FinishPush(deploymentInfo)
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}
