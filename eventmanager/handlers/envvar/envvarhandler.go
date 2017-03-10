package envvar

import (
	"github.com/spf13/afero"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

type Envvarhandler struct {
	Logger     I.Logger
	FileSystem *afero.Afero
}

func (handler Envvarhandler) OnEvent(event S.Event) error {

	handler.Logger.Debugf("Environment Variable Handler Processing Event => %+v", event)

	info := event.Data.(S.DeployEventData)

	if info.DeploymentInfo == nil || !deploymentInfoHasEnvironmentVariables(info.DeploymentInfo) {
		handler.Logger.Info("No Deployment Info or Environment Variables to process!")
		return nil
	}

	m, err := CreateManifest(info.DeploymentInfo.AppName, info.DeploymentInfo.Manifest, handler.FileSystem, handler.Logger)

	if err != nil {
		handler.Logger.Errorf("Error Parsing Manifest! Details: %v", err)
		return err
	}

	//Add any Environment variables
	addEnvResult, _ := m.AddEnvironmentVariables(info.DeploymentInfo.EnvironmentVariables)

	if m.Content.Applications[0].Path != "" || addEnvResult {

		//Ensure path is empty. We are using a local/tmp file system with exploded contents for the deploy!
		m.Content.Applications[0].Path = ""

		//Re-Write the m
		m.WriteManifest(info.DeploymentInfo.AppPath, true)
	}

	return nil
}

func deploymentInfoHasEnvironmentVariables(info *S.DeploymentInfo) bool {
	return info.EnvironmentVariables != nil && len(info.EnvironmentVariables) > 0
}
