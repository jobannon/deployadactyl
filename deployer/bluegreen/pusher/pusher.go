// Package pusher handles pushing to individual Cloud Foundry instances.
package pusher

import (
	"fmt"
	"io"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	cannotDelete            = "cannot delete"
	cannotLogin             = "cannot login to"
	renamingApp             = "renaming app from %+v to %+v"
	renamedApp              = "renamed app from %+v to %+v"
	cannotRenameApp         = "rename failed"
	pushingNewApp           = "pushing new app %+v from %+v"
	mappingRoute            = "mapping route for %s to %s"
	deletedApp              = "deleted %s"
	rollingBackDeploy       = "rolling back deploy of %+v"
	unableToDelete          = "unable to delete %s: %s"
	unableToRenameVenerable = "unable to rename venerable app %s: %s"
	loggedIntoCloudFoundry  = "logged into cloud foundry"
	notRenamingNewApp       = "new app detected"
)

type Pusher struct {
	Courier I.Courier
	Log     *logging.Logger
}

// Push pushes a single application to a Clound Foundry instance using blue green deployment.
// Blue green is done by renaming the current application to appName-venerable.
// Pushes the new application to the existing appName route with an included load balanced domain if provided.
func (p Pusher) Push(appPath, foundationURL, domain string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	p.Log.Debug(renamingApp, deploymentInfo.AppName, deploymentInfo.AppName+"-venerable")
	renameOutput, err := p.Courier.Rename(deploymentInfo.AppName, deploymentInfo.AppName+"-venerable")
	if err != nil {
		if p.Courier.Exists(deploymentInfo.AppName) {
			p.Log.Errorf(cannotRenameApp)
			return errors.New(string(renameOutput))
		} else {
			p.Log.Info(notRenamingNewApp)
		}
	}
	p.Log.Info(renamedApp, deploymentInfo.AppName, deploymentInfo.AppName+"-venerable")

	p.Log.Debug(pushingNewApp, deploymentInfo.AppName, appPath)
	pushOutput, err := p.Courier.Push(deploymentInfo.AppName, appPath)
	fmt.Fprint(out, string(pushOutput))
	if err != nil {
		return errors.New(string(pushOutput))
	}
	p.Log.Info(string(pushOutput))

	p.Log.Debug(mappingRoute, deploymentInfo.AppName, domain)
	mapRouteOutput, err := p.Courier.MapRoute(deploymentInfo.AppName, domain)
	fmt.Fprint(out, string(mapRouteOutput))
	if err != nil {
		return errors.New(string(mapRouteOutput))
	}
	p.Log.Info(string(mapRouteOutput))

	return nil
}

// FinishPush will delete the venerable instance of your application.
func (p Pusher) FinishPush(foundationURL string, deploymentInfo S.DeploymentInfo) error {
	venerableName := deploymentInfo.AppName + "-venerable"

	_, err := p.Courier.Delete(deploymentInfo.AppName + "-venerable")
	if err != nil {
		return errors.Errorf("%s %s: %s", cannotDelete, venerableName, err)
	}

	p.Log.Info(deletedApp, venerableName)

	return nil
}

// Unpush will rollback Push.
// Deletes the new application and renames the appName-venerable to appName.
func (p Pusher) Unpush(foundationURL string, deploymentInfo S.DeploymentInfo) error {
	p.Log.Errorf(rollingBackDeploy, deploymentInfo.AppName)

	venerableName := deploymentInfo.AppName + "-venerable"

	_, err := p.Courier.Delete(deploymentInfo.AppName)
	if err != nil {
		p.Log.Info(unableToDelete, deploymentInfo.AppName, err)
	}
	p.Log.Info(deletedApp, deploymentInfo.AppName)

	_, err = p.Courier.Rename(venerableName, deploymentInfo.AppName)
	if err != nil {
		p.Log.Info(unableToRenameVenerable, venerableName, err)
	}
	p.Log.Info(renamedApp, venerableName, deploymentInfo.AppName)

	return nil
}

// CleanUp removes the temporary directory created by the Executor.
func (p Pusher) CleanUp() error {
	return p.Courier.CleanUp()
}

// Login will login to a Cloud Foundry instance.
func (p Pusher) Login(foundationURL string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	p.Log.Debug(
		`logging into cloud foundry with parameters:
		foundation URL: %+v
		username: %+v
		org: %+v
		space: %+v`,
		foundationURL, deploymentInfo.Username, deploymentInfo.Org, deploymentInfo.Space,
	)

	loginOutput, err := p.Courier.Login(
		foundationURL,
		deploymentInfo.Username,
		deploymentInfo.Password,
		deploymentInfo.Org,
		deploymentInfo.Space,
		deploymentInfo.SkipSSL,
	)
	out.Write(loginOutput)
	if err != nil {
		return errors.Errorf("%s %s: %s", cannotLogin, foundationURL, err)
	}
	p.Log.Info(loggedIntoCloudFoundry)

	return nil
}
