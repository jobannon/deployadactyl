// Package pusher handles pushing to individual Cloud Foundry instances.
package pusher

import (
	"fmt"
	"io"
	"strings"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

const TemporaryNameSuffix = "-new-build-"

// Pusher has a courier used to push applications to Cloud Foundry.
// It represents logging into a single foundation to perform operations.
type Pusher struct {
	Courier   I.Courier
	Log       I.Logger
	appExists bool
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

	output, err := p.Courier.Login(
		foundationURL,
		deploymentInfo.Username,
		deploymentInfo.Password,
		deploymentInfo.Org,
		deploymentInfo.Space,
		deploymentInfo.SkipSSL,
	)
	response.Write(output)
	if err != nil {
		return LoginError{foundationURL, output}
	}
	p.Log.Infof("logged into cloud foundry %s", foundationURL)

	return nil
}

// Push pushes a single application to a Clound Foundry instance using blue green deployment.
// Blue green is done by pushing a new application with the appName+TemporaryNameSuffix+UUID.
// It pushes the new application with the existing appName route.
// It will map a load balanced domain if provided in the config.yml.
//
// Returns Cloud Foundry logs if there is an error.
func (p Pusher) Push(appPath string, deploymentInfo S.DeploymentInfo, response io.Writer) error {

	tempAppWithUUID := deploymentInfo.AppName + TemporaryNameSuffix + deploymentInfo.UUID

	if !p.appExists {
		p.Log.Infof("new app detected")
	}

	p.Log.Debugf("pushing app %s to %s", tempAppWithUUID, deploymentInfo.Domain)
	p.Log.Debugf("tempdir for app %s: %s", tempAppWithUUID, appPath)

	pushOutput, err := p.Courier.Push(tempAppWithUUID, appPath, deploymentInfo.AppName, deploymentInfo.Instances)
	response.Write(pushOutput)
	p.Log.Infof(fmt.Sprintf("output from Cloud Foundry:\n%s\n%s\n%s", strings.Repeat("-", 60), string(pushOutput), strings.Repeat("-", 60)))
	if err != nil {
		logs, newErr := p.Courier.Logs(tempAppWithUUID)
		fmt.Fprintf(response, "\n%s", string(logs))
		p.Log.Debugf(fmt.Sprintf("logs from %s:\n%s\n%s\n%s", tempAppWithUUID, strings.Repeat("-", 60), string(pushOutput), strings.Repeat("-", 60)))
		if newErr != nil {
			return CloudFoundryGetLogsError{err, newErr}
		}

		return PushError{}
	}

	p.Log.Debugf("mapping route for %s to %s", deploymentInfo.AppName, deploymentInfo.Domain)

	if deploymentInfo.Domain != "" {
		mapRouteOutput, err := p.Courier.MapRoute(tempAppWithUUID, deploymentInfo.Domain, deploymentInfo.AppName)
		response.Write(mapRouteOutput)
		if err != nil {
			logs, newErr := p.Courier.Logs(tempAppWithUUID)
			fmt.Fprintf(response, "\n%s", string(logs))
			if newErr != nil {
				return CloudFoundryGetLogsError{err, newErr}
			}

			return MapRouteError{}
		}
		p.Log.Debugf(string(mapRouteOutput))
		p.Log.Infof("application route created at %s.%s", deploymentInfo.AppName, deploymentInfo.Domain)
	}

	return nil
}

// FinishPush will delete the original application if it existed. It will always
// rename the the newly pushed application to the appName.
func (p Pusher) FinishPush(deploymentInfo S.DeploymentInfo) error {
	if p.appExists {
		out, err := p.Courier.Delete(deploymentInfo.AppName)
		if err != nil {
			p.Log.Errorf("could not delete %s", deploymentInfo.AppName)
			return DeleteApplicationError{deploymentInfo.AppName, out}
		}
		p.Log.Infof("deleted %s", deploymentInfo.AppName)
	}

	out, err := p.Courier.Rename(deploymentInfo.AppName+TemporaryNameSuffix+deploymentInfo.UUID, deploymentInfo.AppName)
	if err != nil {
		p.Log.Errorf("could not rename %s to %s", deploymentInfo.AppName+TemporaryNameSuffix+deploymentInfo.UUID, deploymentInfo.AppName)
		return RenameError{deploymentInfo.AppName + TemporaryNameSuffix + deploymentInfo.UUID, out}
	}
	p.Log.Infof("renamed %s to %s", deploymentInfo.AppName+TemporaryNameSuffix+deploymentInfo.UUID, deploymentInfo.AppName)

	return nil
}

// UndoPush is only called when a Push fails. If it is not the first deployment, UndoPush will
// delete the temporary application that was pushed.
// If is the first deployment, UndoPush will rename the failed push to have the appName.
func (p Pusher) UndoPush(deploymentInfo S.DeploymentInfo) error {

	tempAppWithUUID := deploymentInfo.AppName + TemporaryNameSuffix + deploymentInfo.UUID

	if p.appExists {
		p.Log.Errorf("rolling back deploy of %s", tempAppWithUUID)

		out, err := p.Courier.Delete(tempAppWithUUID)
		if err != nil {
			p.Log.Infof("unable to delete %s: %s", tempAppWithUUID, out)
		} else {
			p.Log.Infof("deleted %s", tempAppWithUUID)
		}

	} else {
		out, err := p.Courier.Rename(tempAppWithUUID, deploymentInfo.AppName)
		if err != nil {
			p.Log.Infof("unable to rename venerable app %s: %s", tempAppWithUUID, out)
			return RenameError{tempAppWithUUID, out}
		}
		p.Log.Infof("renamed app from %s to %s", tempAppWithUUID, deploymentInfo.AppName)

		p.Log.Infof("app %s did not previously exist: not rolling back", deploymentInfo.AppName)
	}

	return nil
}

// CleanUp removes the temporary directory created by the Executor.
func (p Pusher) CleanUp() error {
	return p.Courier.CleanUp()
}

// Exists uses the courier to check if the application already exists, meaning this is not the
// first time it has been pushed to Cloud Foundry.
func (p *Pusher) Exists(appName string) {
	p.appExists = p.Courier.Exists(appName)
}
