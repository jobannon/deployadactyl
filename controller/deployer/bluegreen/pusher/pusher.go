// Package pusher handles pushing to individual Cloud Foundry instances.
package pusher

import (
	"fmt"
	"io"
	"strings"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

// Pusher has a courier used to push applications to Cloud Foundry.
type Pusher struct {
	Courier I.Courier
	Log     *logging.Logger
}

// Push pushes a single application to a Clound Foundry instance using blue green deployment.
// Blue green is done by renaming the current application to appName-venerable.
// Pushes the new application to the existing appName route with an included load balanced domain if provided.
//
// Returns Cloud Foundry logs if there is an error.
func (p Pusher) Push(appPath string, appExists bool, deploymentInfo S.DeploymentInfo, response io.Writer) error {
	if appExists {
		_, err := p.Courier.Rename(deploymentInfo.AppName, deploymentInfo.AppName+"-venerable")
		if err != nil {
			return errors.Errorf("rename failed: %s", err)
		}

		p.Log.Infof("renamed app from %s to %s", deploymentInfo.AppName, deploymentInfo.AppName+"-venerable")
	} else {
		p.Log.Infof("new app detected")
	}

	p.Log.Debugf("pushing app %s to %s", deploymentInfo.AppName, deploymentInfo.Domain)
	p.Log.Debugf("tempdir for app %s: %s", deploymentInfo.AppName, appPath)

	pushOutput, err := p.Courier.Push(deploymentInfo.AppName, appPath, deploymentInfo.Instances)
	fmt.Fprint(response, string(pushOutput))
	if err != nil {
		logs, newErr := p.Courier.Logs(deploymentInfo.AppName)
		fmt.Fprintf(response, "\n%s", string(logs))
		if newErr != nil {
			return errors.Errorf("%s: cannot get Cloud Foundry logs: %s", err, newErr)
		}
		return err
	}

	p.Log.Infof(fmt.Sprintf("output from Cloud Foundry:\n%s\n%s\n%s", strings.Repeat("-", 60), string(pushOutput), strings.Repeat("-", 60)))
	p.Log.Debugf("mapping route for %s to %s", deploymentInfo.AppName, deploymentInfo.Domain)

	mapRouteOutput, err := p.Courier.MapRoute(deploymentInfo.AppName, deploymentInfo.Domain)
	fmt.Fprint(response, string(mapRouteOutput))
	if err != nil {
		logs, newErr := p.Courier.Logs(deploymentInfo.AppName)
		fmt.Fprintf(response, "\n%s", string(logs))
		if newErr != nil {
			return errors.Errorf("cannot get Cloud Foundry logs: %s", newErr)
		}
		return err
	}
	p.Log.Debugf(string(mapRouteOutput))
	p.Log.Infof("application route created at %s.%s", deploymentInfo.AppName, deploymentInfo.Domain)

	return nil
}

// DeleteVenerable will delete the venerable instance of your application.
func (p Pusher) DeleteVenerable(deploymentInfo S.DeploymentInfo, foundationURL string) error {
	venerableName := deploymentInfo.AppName + "-venerable"

	_, err := p.Courier.Delete(deploymentInfo.AppName + "-venerable")
	if err != nil {
		return errors.Errorf("cannot delete %s: %s", venerableName, err)
	}

	p.Log.Infof("deleted %s", venerableName)
	p.Log.Infof("finished push successfully on %s", foundationURL)

	return nil
}

// Rollback will rollback Push.
// Deletes the new application.
// Renames appName-venerable back to appName if this is not the first deploy.
func (p Pusher) Rollback(appExists bool, deploymentInfo S.DeploymentInfo) error {
	p.Log.Errorf("rolling back deploy of %s", deploymentInfo.AppName)
	venerableName := deploymentInfo.AppName + "-venerable"

	_, err := p.Courier.Delete(deploymentInfo.AppName)
	if err != nil {
		p.Log.Infof("unable to delete %s: %s", deploymentInfo.AppName, err)
	}
	p.Log.Infof("deleted %s", deploymentInfo.AppName)

	if appExists {
		_, err = p.Courier.Rename(venerableName, deploymentInfo.AppName)
		if err != nil {
			p.Log.Infof("unable to rename venerable app %s: %s", venerableName, err)
		}
		p.Log.Infof("renamed app from %s to %s", venerableName, deploymentInfo.AppName)
	}

	return nil
}

// Login will login to a Cloud Foundry instance.
func (p Pusher) Login(foundationURL string, deploymentInfo S.DeploymentInfo, response io.Writer) error {
	p.Log.Debugf(
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
	response.Write(loginOutput)
	if err != nil {
		return errors.Errorf("cannot login to %s: %s", foundationURL, err)
	}
	p.Log.Infof("logged into cloud foundry %s", foundationURL)

	return nil
}

// CleanUp removes the temporary directory created by the Executor.
func (p Pusher) CleanUp() error {
	return p.Courier.CleanUp()
}

// Exists uses the courier to check if the application exists.
func (p Pusher) Exists(appName string) bool {
	return p.Courier.Exists(appName)
}
