// Package pusher handles pushing to individual Cloud Foundry instances.
package pusher

import (
	"fmt"
	"io"
	"strings"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

const temporaryNameSuffix = "-venerable"

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
// Blue green is done by renaming the current application to appName-venerable+UUID.
// It pushes the new application to the existing appName route with an included load balanced domain if provided.
//
// Returns Cloud Foundry logs if there is an error.
func (p Pusher) Push(appPath string, deploymentInfo S.DeploymentInfo, response io.Writer) error {

	var (
		appNameWithUUID          = deploymentInfo.AppName + deploymentInfo.UUID
		appNameVenerableWithUUID = deploymentInfo.AppName + temporaryNameSuffix + deploymentInfo.UUID
	)

	if p.appExists {
		output, err := p.Courier.Rename(deploymentInfo.AppName, appNameVenerableWithUUID)
		if err != nil {
			return RenameError{deploymentInfo.AppName, output}
		}

		p.Log.Infof("renamed app from %s to %s", deploymentInfo.AppName, appNameVenerableWithUUID)
	} else {
		p.Log.Infof("new app detected")
	}

	p.Log.Debugf("pushing app %s to %s", appNameWithUUID, deploymentInfo.Domain)
	p.Log.Debugf("tempdir for app %s: %s", appNameWithUUID, appPath)

	pushOutput, err := p.Courier.Push(appNameWithUUID, appPath, deploymentInfo.AppName, deploymentInfo.Instances)
	response.Write(pushOutput)
	if err != nil {
		logs, newErr := p.Courier.Logs(appNameWithUUID)
		fmt.Fprintf(response, "\n%s", string(logs))
		if newErr != nil {
			return CloudFoundryGetLogsError{err, newErr}
		}
		return err
	}

	p.Log.Infof(fmt.Sprintf("output from Cloud Foundry:\n%s\n%s\n%s", strings.Repeat("-", 60), string(pushOutput), strings.Repeat("-", 60)))
	p.Log.Debugf("mapping route for %s to %s", deploymentInfo.AppName, deploymentInfo.Domain)

	mapRouteOutput, err := p.Courier.MapRoute(appNameWithUUID, deploymentInfo.Domain, deploymentInfo.AppName)
	response.Write(mapRouteOutput)
	if err != nil {
		logs, newErr := p.Courier.Logs(appNameWithUUID)
		fmt.Fprintf(response, "\n%s", string(logs))
		if newErr != nil {
			return CloudFoundryGetLogsError{err, newErr}
		}
		return err
	}
	p.Log.Debugf(string(mapRouteOutput))
	p.Log.Infof("application route created at %s.%s", deploymentInfo.AppName, deploymentInfo.Domain)

	return nil
}

// FinishPush will rename appName+UUID to appName. If this is not the first time the application
// has been deployed it will delete appName-venerable+UUID from the blue green operation.
func (p Pusher) FinishPush(deploymentInfo S.DeploymentInfo) error {
	out, err := p.Courier.Rename(deploymentInfo.AppName+deploymentInfo.UUID, deploymentInfo.AppName)
	if err != nil {
		p.Log.Errorf("could not rename %s to %s", deploymentInfo.AppName+deploymentInfo.UUID, deploymentInfo.AppName)
		return RenameError{deploymentInfo.AppName + deploymentInfo.UUID, out}
	}
	p.Log.Infof("renamed %s to %s", deploymentInfo.AppName+deploymentInfo.UUID, deploymentInfo.AppName)

	if p.appExists {
		out, err = p.Courier.Delete(deploymentInfo.AppName + temporaryNameSuffix + deploymentInfo.UUID)
		if err != nil {
			p.Log.Errorf("could not delete %s", deploymentInfo.AppName+temporaryNameSuffix+deploymentInfo.UUID)
			return DeleteApplicationError{deploymentInfo.AppName + temporaryNameSuffix + deploymentInfo.UUID, out}
		}
		p.Log.Infof("deleted %s", deploymentInfo.AppName+temporaryNameSuffix+deploymentInfo.UUID)
	}

	return nil
}

// Rollback will delete appName+UUID and rename appName-venerable+UUID to appName on a failed Push.
// It performs no operation if this is the first time an application has been deployed in order to
// preserve the application for debugging the deployment.
func (p Pusher) Rollback(deploymentInfo S.DeploymentInfo) error {

	if p.appExists {
		var (
			appNameWithUUID          = deploymentInfo.AppName + deploymentInfo.UUID
			appNameVenerableWithUUID = deploymentInfo.AppName + temporaryNameSuffix + deploymentInfo.UUID
		)

		p.Log.Errorf("rolling back deploy of %s", appNameWithUUID)

		out, err := p.Courier.Delete(appNameWithUUID)
		if err != nil {
			p.Log.Infof("unable to delete %s: %s", appNameWithUUID, out)
		} else {
			p.Log.Infof("deleted %s", appNameWithUUID)
		}

		out, err = p.Courier.Rename(appNameVenerableWithUUID, deploymentInfo.AppName)
		if err != nil {
			p.Log.Infof("unable to rename venerable app %s: %s", appNameVenerableWithUUID, out)
			return RenameError{appNameVenerableWithUUID, out}
		}

		p.Log.Infof("renamed app from %s to %s", appNameVenerableWithUUID, deploymentInfo.AppName)
		return nil
	}

	p.Log.Infof("app %s did not previously exist: not rolling back", deploymentInfo.AppName)
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
