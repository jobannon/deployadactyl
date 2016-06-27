package deployer

import (
	"fmt"
	"io"
	"os"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	environmentNotFound = "environment not found"
	cannotFetchArtifact = "cannot fetch artifact"
	invalidArtifact     = "invalid artifact"
	successfulDeploy    = `Your deploy was successful! (^_^)d
If you experience any problems after this point, check that you can manually push your application to Cloud Foundry on a lower environment. It is likely that it is an error with your application and not with Deployadactyl.
Thanks for using Deployadactyl! Please push down pull up on your lap bar and exit to your left.`
)

type Deployer struct {
	BlueGreener  I.BlueGreener
	Environments map[string]config.Environment
	Fetcher      I.Fetcher
	Log          *logging.Logger
	Prechecker   I.Prechecker
	EventManager I.EventManager
}

func (d Deployer) Deploy(deploymentInfo S.DeploymentInfo, out io.Writer) (err error) {
	var appPath string

	deployEventData := S.DeployEventData{
		Writer:         out,
		DeploymentInfo: &deploymentInfo,
	}

	environment, found := d.Environments[deploymentInfo.Environment]
	if !found {
		var deployEvent = S.Event{
			Type: "deploy.error",
			Data: deployEventData,
		}

		err = d.EventManager.Emit(deployEvent)
		if err != nil {
			fmt.Fprint(out, err)
		}

		err = errors.Errorf("%s: %s", environmentNotFound, deploymentInfo.Environment)
		fmt.Fprint(out, err)
		return err
	}

	err = d.Prechecker.AssertAllFoundationsUp(environment)
	if err != nil {
		fmt.Fprint(out, err)
		return errors.New(err)
	}

	appPath, err = d.Fetcher.Fetch(deploymentInfo.ArtifactURL, deploymentInfo.Manifest)
	if err != nil {
		fmt.Fprint(out, err)
		return err
	}
	defer os.RemoveAll(appPath)

	defer func() {
		var deployEvent = S.Event{
			Type: "deploy.finish",
			Data: deployEventData,
		}

		if err != nil {
			deployEvent.Type = "deploy.error"
		}

		err = d.EventManager.Emit(deployEvent)
		if err != nil {
			fmt.Fprint(out, err)
		}
	}()

	err = d.BlueGreener.Push(environment, appPath, deploymentInfo, out)
	if err != nil {
		return errors.New(err)
	}

	fmt.Fprint(out, successfulDeploy)
	return err
}
